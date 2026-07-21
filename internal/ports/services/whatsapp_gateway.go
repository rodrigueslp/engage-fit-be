package services

import (
	"context"

	"boxengage/backend/internal/domain"
)

type WhatsappMessage struct {
	Phone            string
	Body             string
	ContentSID       string
	ContentVariables map[string]string
}

type WhatsappSendResult struct {
	ProviderMessageID string
	InitialStatus     string
}

type WhatsappGateway interface {
	Test(ctx context.Context, settings domain.WhatsappSettings) error
	Send(ctx context.Context, settings domain.WhatsappSettings, message WhatsappMessage) (*WhatsappSendResult, error)
}

type WhatsappSettingsResolver interface {
	Resolve(ctx context.Context, boxID domain.ID) (*domain.WhatsappSettings, error)
	ResolveMetadata(ctx context.Context, boxID domain.ID) (*domain.WhatsappSettings, error)
}
