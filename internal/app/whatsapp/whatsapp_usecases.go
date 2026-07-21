package whatsapp

import (
	"context"
	"errors"
	"strings"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
	"gorm.io/gorm"
)

type SettingsResolver struct {
	settings repositories.WhatsappSettingsRepository
	platform domain.WhatsappSettings
}

func NewSettingsResolver(settings repositories.WhatsappSettingsRepository, platform domain.WhatsappSettings) SettingsResolver {
	platform.ConnectionMode = domain.WhatsappConnectionPlatform
	if strings.TrimSpace(platform.Provider) == "" {
		platform.Provider = "twilio"
	}
	return SettingsResolver{settings: settings, platform: platform}
}

func (r SettingsResolver) Resolve(ctx context.Context, boxID domain.ID) (*domain.WhatsappSettings, error) {
	settings, err := r.ResolveMetadata(ctx, boxID)
	if err != nil {
		return nil, err
	}
	if settings.ConnectionMode == domain.WhatsappConnectionPlatform && !platformSettingsAvailable(*settings) {
		return nil, errors.New("a conexão compartilhada do EngageFit ainda não está disponível; configure uma conta própria ou contate o suporte")
	}
	return settings, nil
}

func (r SettingsResolver) ResolveMetadata(ctx context.Context, boxID domain.ID) (*domain.WhatsappSettings, error) {
	settings, err := r.settings.FindByBoxID(ctx, boxID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		settings = &domain.WhatsappSettings{BoxID: boxID, ConnectionMode: domain.WhatsappConnectionPlatform}
	} else if err != nil {
		return nil, err
	}
	mode := normalizeConnectionMode(settings.ConnectionMode)
	if mode == domain.WhatsappConnectionDedicated {
		settings.ConnectionMode = mode
		return settings, nil
	}
	platform := r.platform
	platform.BoxID = boxID
	return &platform, nil
}

func (r SettingsResolver) ResolveSettings(settings domain.WhatsappSettings) (*domain.WhatsappSettings, error) {
	mode := normalizeConnectionMode(settings.ConnectionMode)
	if mode == domain.WhatsappConnectionDedicated {
		settings.ConnectionMode = mode
		return &settings, nil
	}
	platform := r.platform
	platform.BoxID = settings.BoxID
	if !platformSettingsAvailable(platform) {
		return nil, errors.New("a conexão compartilhada do EngageFit ainda não está disponível; configure uma conta própria ou contate o suporte")
	}
	return &platform, nil
}

func (r SettingsResolver) PlatformAvailable() bool {
	return platformSettingsAvailable(r.platform)
}

func (r SettingsResolver) PlatformSender() string { return r.platform.InstanceName }

func platformSettingsAvailable(settings domain.WhatsappSettings) bool {
	accountSID, authToken, ok := strings.Cut(settings.APIKeyEncrypted, ":")
	return settings.Enabled && strings.TrimSpace(settings.InstanceName) != "" && ok && strings.TrimSpace(accountSID) != "" && strings.TrimSpace(authToken) != ""
}

func normalizeConnectionMode(mode domain.WhatsappConnectionMode) domain.WhatsappConnectionMode {
	if mode == domain.WhatsappConnectionDedicated {
		return mode
	}
	return domain.WhatsappConnectionPlatform
}

type GetSettingsUseCase struct {
	settings repositories.WhatsappSettingsRepository
	resolver SettingsResolver
}

func NewGetSettingsUseCase(settings repositories.WhatsappSettingsRepository, resolver SettingsResolver) GetSettingsUseCase {
	return GetSettingsUseCase{settings: settings, resolver: resolver}
}

func (uc GetSettingsUseCase) Execute(ctx context.Context, boxID domain.ID) (*domain.WhatsappSettings, error) {
	settings, err := uc.settings.FindByBoxID(ctx, boxID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		settings = &domain.WhatsappSettings{BoxID: boxID, ConnectionMode: domain.WhatsappConnectionPlatform}
	} else if err != nil {
		return nil, err
	}
	settings.ConnectionMode = normalizeConnectionMode(settings.ConnectionMode)
	settings.PlatformAvailable = uc.resolver.PlatformAvailable()
	settings.PlatformSender = uc.resolver.PlatformSender()
	return settings, nil
}

type UpdateSettingsUseCase struct {
	settings repositories.WhatsappSettingsRepository
}

func NewUpdateSettingsUseCase(settings repositories.WhatsappSettingsRepository) UpdateSettingsUseCase {
	return UpdateSettingsUseCase{settings: settings}
}

func (uc UpdateSettingsUseCase) Execute(ctx context.Context, settings *domain.WhatsappSettings) error {
	return uc.settings.Upsert(ctx, settings)
}

type TestSettingsUseCase struct {
	gateway  services.WhatsappGateway
	resolver SettingsResolver
}

func NewTestSettingsUseCase(gateway services.WhatsappGateway, resolver SettingsResolver) TestSettingsUseCase {
	return TestSettingsUseCase{gateway: gateway, resolver: resolver}
}

func (uc TestSettingsUseCase) Execute(ctx context.Context, settings domain.WhatsappSettings) error {
	resolved, err := uc.resolver.ResolveSettings(settings)
	if err != nil {
		return err
	}
	return uc.gateway.Test(ctx, *resolved)
}
