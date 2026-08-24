package repositories

import (
	"context"
	"errors"

	"boxengage/backend/internal/domain"
)

var (
	ErrSchedulingUnitNotFound           = errors.New("scheduling unit not found")
	ErrSchedulingProfessionalNotFound   = errors.New("scheduling professional not found")
	ErrSchedulingRevisionConflict       = errors.New("scheduling revision conflict")
	ErrSchedulingAccountConflict        = errors.New("scheduling account conflict")
	ErrSchedulingProfessionalArchived   = errors.New("scheduling professional archived")
	ErrSchedulingProfessionalAnonymized = errors.New("scheduling professional anonymized")
	ErrSchedulingIdempotencyKeyReused   = errors.New("scheduling idempotency key reused")
	ErrSchedulingProfessionalInvalid    = errors.New("scheduling professional invalid")
	ErrSchedulingAccountUnavailable     = errors.New("scheduling account unavailable")
	ErrSchedulingAnonymizationBlocked   = errors.New("scheduling anonymization blocked")
)

type SchedulingRepository interface {
	LockCommand(ctx context.Context, organizationID domain.ID, command, idempotencyKey string) error
	FindCommand(ctx context.Context, organizationID domain.ID, command, idempotencyKey string) (*domain.SchedulingCommandRequest, error)
	CreateCommand(ctx context.Context, request *domain.SchedulingCommandRequest) error

	LockProfessional(ctx context.Context, organizationID, professionalID domain.ID) error
	LockAccount(ctx context.Context, organizationID, userID domain.ID) error

	ListProfessionals(ctx context.Context, organizationID, unitID domain.ID, includeArchived bool, query string) ([]domain.Professional, error)
	GetByID(ctx context.Context, organizationID, unitID, professionalID domain.ID) (*domain.Professional, error)
	GetByIDForUpdate(ctx context.Context, organizationID, professionalID domain.ID) (*domain.Professional, error)
	FindByLinkedUser(ctx context.Context, organizationID, userID domain.ID) (*domain.Professional, error)

	CreateProfessional(ctx context.Context, professional *domain.Professional, unitID domain.ID) error
	UpdateProfessional(ctx context.Context, professional *domain.Professional) error
	SetUnitActive(ctx context.Context, organizationID, unitID, professionalID domain.ID, active bool) error

	CreateAuditEvent(ctx context.Context, event *domain.SchedulingAuditEvent) error
	ListAuditEventsForProfessional(ctx context.Context, organizationID, unitID, professionalID domain.ID) ([]domain.SchedulingAuditEvent, error)
	CountActiveInUnit(ctx context.Context, organizationID, unitID domain.ID) (int64, error)
}

type SchedulingUnitContextResolver interface {
	ResolveSchedulingUnitContext(ctx context.Context, authenticatedUserID, unitID domain.ID) (*domain.SchedulingUnitContext, error)
}

type SchedulingAccountResolver interface {
	ListProfessionalAccountCandidates(ctx context.Context, organizationID, unitID domain.ID) ([]domain.ProfessionalAccountCandidate, error)
	ResolveProfessionalAccount(ctx context.Context, organizationID, unitID, userID domain.ID) (*domain.ProfessionalAccountCandidate, error)
}

type SchedulingProfessionalResolver interface {
	ResolveProfessionalForUser(ctx context.Context, organizationID, unitID, userID domain.ID) (*domain.ID, error)
}
