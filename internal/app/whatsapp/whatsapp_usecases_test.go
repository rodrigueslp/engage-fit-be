package whatsapp

import (
	"context"
	"testing"

	"gorm.io/gorm"

	"boxengage/backend/internal/domain"
)

type whatsappSettingsRepositoryStub struct {
	settings *domain.WhatsappSettings
	err      error
}

func (r whatsappSettingsRepositoryStub) FindByBoxID(context.Context, domain.ID) (*domain.WhatsappSettings, error) {
	if r.settings == nil {
		return nil, r.err
	}
	copy := *r.settings
	return &copy, r.err
}

func (r whatsappSettingsRepositoryStub) Upsert(context.Context, *domain.WhatsappSettings) error {
	return nil
}

func TestSettingsResolverUsesPlatformConnectionByDefault(t *testing.T) {
	repository := whatsappSettingsRepositoryStub{err: gorm.ErrRecordNotFound}
	resolver := NewSettingsResolver(repository, domain.WhatsappSettings{
		Provider:        "twilio",
		InstanceName:    "+5511000000000",
		APIKeyEncrypted: "AC-platform:token",
		Enabled:         true,
	})

	settings, err := resolver.Resolve(context.Background(), domain.ID("box-1"))
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.ConnectionMode != domain.WhatsappConnectionPlatform || settings.InstanceName != "+5511000000000" {
		t.Fatalf("Resolve() = %#v, want platform settings", settings)
	}
}

func TestSettingsResolverUsesDedicatedConnectionWhenSelected(t *testing.T) {
	dedicated := &domain.WhatsappSettings{
		BoxID:           domain.ID("box-1"),
		ConnectionMode:  domain.WhatsappConnectionDedicated,
		Provider:        "twilio",
		InstanceName:    "+5522000000000",
		APIKeyEncrypted: "AC-dedicated:token",
		Enabled:         true,
	}
	resolver := NewSettingsResolver(whatsappSettingsRepositoryStub{settings: dedicated}, domain.WhatsappSettings{
		Provider:        "twilio",
		InstanceName:    "+5511000000000",
		APIKeyEncrypted: "AC-platform:token",
		Enabled:         true,
	})

	settings, err := resolver.Resolve(context.Background(), dedicated.BoxID)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if settings.InstanceName != dedicated.InstanceName || settings.APIKeyEncrypted != dedicated.APIKeyEncrypted {
		t.Fatalf("Resolve() = %#v, want dedicated settings", settings)
	}
}

func TestSettingsResolverRejectsUnavailablePlatformConnection(t *testing.T) {
	resolver := NewSettingsResolver(whatsappSettingsRepositoryStub{err: gorm.ErrRecordNotFound}, domain.WhatsappSettings{Provider: "twilio"})

	if _, err := resolver.Resolve(context.Background(), domain.ID("box-1")); err == nil {
		t.Fatal("Resolve() error = nil, want unavailable platform error")
	}
}
