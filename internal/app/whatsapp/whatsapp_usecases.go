package whatsapp

import (
	"context"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

type GetSettingsUseCase struct {
	settings repositories.WhatsappSettingsRepository
}

func NewGetSettingsUseCase(settings repositories.WhatsappSettingsRepository) GetSettingsUseCase {
	return GetSettingsUseCase{settings: settings}
}

func (uc GetSettingsUseCase) Execute(ctx context.Context, boxID domain.ID) (*domain.WhatsappSettings, error) {
	return uc.settings.FindByBoxID(ctx, boxID)
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
	gateway services.WhatsappGateway
}

func NewTestSettingsUseCase(gateway services.WhatsappGateway) TestSettingsUseCase {
	return TestSettingsUseCase{gateway: gateway}
}

func (uc TestSettingsUseCase) Execute(ctx context.Context, settings domain.WhatsappSettings) error {
	return uc.gateway.Test(ctx, settings)
}
