package services

import (
	"context"

	"boxengage/backend/internal/domain"
)

type MessagingReservationRequest struct {
	BoxID             domain.ID
	RequestedByUserID domain.ID
	SourceType        string
	SourceID          domain.ID
	ConnectionMode    domain.WhatsappConnectionMode
	Recipients        int
}

type MessagingGovernance interface {
	Reserve(ctx context.Context, request MessagingReservationRequest) (*domain.MessageDispatch, error)
	Complete(ctx context.Context, dispatchID domain.ID, accepted, failed int) error
}
