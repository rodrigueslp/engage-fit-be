package repositories

import (
	"context"

	"boxengage/backend/internal/domain"
)

type WhatsappSettingsRepository interface {
	FindByBoxID(ctx context.Context, boxID domain.ID) (*domain.WhatsappSettings, error)
	Upsert(ctx context.Context, settings *domain.WhatsappSettings) error
}
