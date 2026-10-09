package featureflags

import (
	"context"
	"errors"
	"testing"
)

type memoryStore struct{ values map[string]bool }

func (m *memoryStore) List(context.Context) (map[string]bool, error) { return m.values, nil }
func (m *memoryStore) Set(_ context.Context, key string, enabled bool, _, _, _ string) error {
	m.values[key] = enabled
	return nil
}

func TestOverridesRespectNonEditableFlagsAndDependencies(t *testing.T) {
	store := &memoryStore{values: map[string]bool{"catalog": false, "people": true}}
	service := NewService(map[string]bool{"catalog": true, "memberships": true, "enrollments": true, "people": false}, store).WithAvailability(map[string]bool{"people": false})
	values, err := service.Effective(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if values["catalog"] || values["memberships"] || values["enrollments"] || values["people"] {
		t.Fatalf("unexpected effective flags: %#v", values)
	}
	if err := service.Set(context.Background(), "people", true, "test", "admin", "127.0.0.1"); !errors.Is(err, ErrNotEditable) {
		t.Fatalf("expected startup limit, got %v", err)
	}
	if err := service.Set(context.Background(), "catalog", true, "", "admin", "127.0.0.1"); !errors.Is(err, ErrReasonRequired) {
		t.Fatalf("expected reason requirement, got %v", err)
	}
	if err := service.Set(context.Background(), "catalog", true, "restore", "admin", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	values, _ = service.Effective(context.Background())
	if !values["catalog"] || !values["memberships"] || values["enrollments"] {
		t.Fatalf("unexpected restored flags: %#v", values)
	}
	if err := service.Set(context.Background(), "enrollments", true, "enable", "admin", "127.0.0.1"); !errors.Is(err, ErrDependencyDisabled) {
		t.Fatalf("expected dependency check, got %v", err)
	}
	other := NewService(map[string]bool{"catalog": false}, &memoryStore{values: map[string]bool{}})
	if err := other.Set(context.Background(), "catalog", true, "enable", "admin", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	enabled, err := other.Effective(context.Background())
	if err != nil || !enabled["catalog"] {
		t.Fatalf("expected runtime activation: %#v, %v", enabled, err)
	}
	ready := NewService(map[string]bool{"people": false, "workouts": false, "llm": false}, &memoryStore{values: map[string]bool{}}).WithAvailability(map[string]bool{"people": true, "llm": false})
	if err := ready.Set(context.Background(), "people", true, "enable", "admin", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if err := ready.Set(context.Background(), "llm", true, "enable", "admin", "127.0.0.1"); !errors.Is(err, ErrNotEditable) {
		t.Fatalf("expected unavailable LLM, got %v", err)
	}
	values, err = ready.Effective(context.Background())
	if err != nil || !values["people"] || values["llm"] {
		t.Fatalf("unexpected ready flags: %#v, %v", values, err)
	}
	unavailable := NewService(map[string]bool{"whatsapp": true}, &memoryStore{values: map[string]bool{"whatsapp": false}}).WithAvailability(map[string]bool{"whatsapp": false})
	values, err = unavailable.Effective(context.Background())
	if err != nil || values["whatsapp"] {
		t.Fatalf("disabled override must remain effective when editing becomes unavailable: %#v, %v", values, err)
	}
}
