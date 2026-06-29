package repositories

import (
	"context"

	"boxengage/backend/internal/domain"
)

type EmailSettingsRepository interface {
	FindByBoxID(ctx context.Context, boxID domain.ID) (*domain.EmailSettings, error)
	Upsert(ctx context.Context, settings *domain.EmailSettings) error
}

type EmailRepository interface {
	ListTemplates(ctx context.Context, boxID domain.ID) ([]domain.EmailTemplate, error)
	FindTemplateByID(ctx context.Context, boxID, id domain.ID) (*domain.EmailTemplate, error)
	SaveTemplate(ctx context.Context, template *domain.EmailTemplate) error
	UpdateTemplate(ctx context.Context, template domain.EmailTemplate) error
	DeleteTemplate(ctx context.Context, boxID, id domain.ID) error

	ListCampaigns(ctx context.Context, boxID domain.ID) ([]domain.EmailCampaign, error)
	FindCampaignByID(ctx context.Context, boxID, id domain.ID) (*domain.EmailCampaign, error)
	SaveCampaign(ctx context.Context, campaign *domain.EmailCampaign) error
	UpdateCampaign(ctx context.Context, campaign domain.EmailCampaign) error
	ListRecipients(ctx context.Context, emailCampaignID domain.ID) ([]domain.EmailRecipient, error)
	SaveRecipients(ctx context.Context, recipients []domain.EmailRecipient) error
	UpdateRecipient(ctx context.Context, recipient domain.EmailRecipient) error
}
