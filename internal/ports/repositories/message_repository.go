package repositories

import (
	"context"

	"boxengage/backend/internal/domain"
)

type MessageRepository interface {
	ListTemplates(ctx context.Context, boxID domain.ID) ([]domain.MessageTemplate, error)
	FindTemplateByID(ctx context.Context, boxID, id domain.ID) (*domain.MessageTemplate, error)
	FindTemplateByType(ctx context.Context, boxID domain.ID, templateType domain.MessageTemplateType) (*domain.MessageTemplate, error)
	SaveTemplate(ctx context.Context, template *domain.MessageTemplate) error
	UpdateTemplate(ctx context.Context, template domain.MessageTemplate) error
	DeleteTemplate(ctx context.Context, boxID, id domain.ID) error

	ListCampaigns(ctx context.Context, boxID domain.ID) ([]domain.MessageCampaign, error)
	FindCampaignByID(ctx context.Context, boxID, id domain.ID) (*domain.MessageCampaign, error)
	SaveCampaign(ctx context.Context, campaign *domain.MessageCampaign) error
	UpdateCampaign(ctx context.Context, campaign domain.MessageCampaign) error
	ListRecipients(ctx context.Context, messageCampaignID domain.ID) ([]domain.MessageRecipient, error)
	SaveRecipients(ctx context.Context, recipients []domain.MessageRecipient) error
	UpdateRecipient(ctx context.Context, recipient domain.MessageRecipient) error
}
