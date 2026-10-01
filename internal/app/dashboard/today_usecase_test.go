package dashboard

import (
	"context"
	"testing"
	"time"

	"boxengage/backend/internal/domain"
)

type todayBoxes struct{ box domain.Box }

func (f todayBoxes) FindByID(context.Context, domain.ID) (*domain.Box, error)     { return &f.box, nil }
func (todayBoxes) ListAll(context.Context) ([]domain.Box, error)                  { return nil, nil }
func (todayBoxes) Save(context.Context, *domain.Box) error                        { return nil }
func (todayBoxes) SaveWithOwner(context.Context, *domain.Box, *domain.User) error { return nil }
func (todayBoxes) Update(context.Context, domain.Box) error                       { return nil }

type todayReader struct {
	date           string
	from, until    time.Time
	organizationID domain.ID
}

func (f *todayReader) ReadToday(_ context.Context, organizationID, _ domain.ID, date string, from, until time.Time) (*domain.TodayOperations, error) {
	f.organizationID, f.date, f.from, f.until = organizationID, date, from, until
	return &domain.TodayOperations{}, nil
}

func TestTodayOperationsUsesUnitLocalDay(t *testing.T) {
	reader := &todayReader{}
	uc := NewGetTodayOperationsUseCase(todayBoxes{domain.Box{ID: "unit", OrganizationID: "org", Timezone: "America/Sao_Paulo"}}, reader)
	uc.now = func() time.Time { return time.Date(2026, 9, 17, 2, 30, 0, 0, time.UTC) }
	result, err := uc.Execute(context.Background(), "unit")
	if err != nil {
		t.Fatal(err)
	}
	if reader.date != "2026-09-16" || reader.organizationID != "org" {
		t.Fatalf("wrong scope/day: %#v", reader)
	}
	if got := reader.until.Sub(reader.from); got != 24*time.Hour {
		t.Fatalf("unexpected local day: %v", got)
	}
	if result.Timezone != "America/Sao_Paulo" || result.GeneratedAt.IsZero() {
		t.Fatalf("metadata missing: %#v", result)
	}
}
