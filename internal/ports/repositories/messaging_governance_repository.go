package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
)

type MessagingReservation struct {
	BoxID             domain.ID
	RequestedByUserID domain.ID
	SourceType        string
	SourceID          domain.ID
	ConnectionMode    domain.WhatsappConnectionMode
	Recipients        int
}

type MessagingGovernanceRepository interface {
	Reserve(ctx context.Context, request MessagingReservation) (*domain.MessageDispatch, error)
	Complete(ctx context.Context, dispatchID domain.ID, accepted, failed int) error
	GetBoxPolicy(ctx context.Context, boxID domain.ID) (*domain.MessagingPolicy, error)
	GetPlatformPolicy(ctx context.Context) (*domain.MessagingPolicy, error)
	UpsertPolicy(ctx context.Context, policy *domain.MessagingPolicy) error
	GetBoxUsage(ctx context.Context, boxID domain.ID, atTime time.Time) (*domain.MessagingUsage, error)
	GetPlatformUsage(ctx context.Context, atTime time.Time) (*domain.MessagingUsage, error)
	SaveAuditLog(ctx context.Context, log *domain.AdminAuditLog) error
}
