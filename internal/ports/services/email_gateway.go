package services

import (
	"context"

	"boxengage/backend/internal/domain"
)

type EmailMessage struct {
	ToEmail string
	Subject string
	Body    string
}

type EmailGateway interface {
	Test(ctx context.Context, settings domain.EmailSettings) error
	Send(ctx context.Context, settings domain.EmailSettings, message EmailMessage) error
}
