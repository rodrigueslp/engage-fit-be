package repositories

import (
	"context"
	"errors"
	"time"

	"boxengage/backend/internal/domain"
)

var (
	ErrReservationNotFound          = errors.New("reservation not found")
	ErrReservationInvalid           = errors.New("reservation invalid")
	ErrReservationAlreadyExists     = errors.New("reservation already exists")
	ErrReservationNotEligible       = errors.New("reservation not eligible")
	ErrReservationLimitReached      = errors.New("reservation entitlement limit reached")
	ErrReservationRevisionConflict  = errors.New("reservation revision conflict")
	ErrReservationAlreadyFinal      = errors.New("reservation already final")
	ErrReservationIdempotencyReused = errors.New("reservation idempotency reused")
	ErrReservationMakeupInvalid     = errors.New("reservation makeup invalid")
)

type ReservationRepository interface {
	LockCommand(context.Context, domain.ID, string, string) error
	FindCommand(context.Context, domain.ID, string, string) (*domain.ReservationCommandRequest, error)
	CreateCommand(context.Context, *domain.ReservationCommandRequest) error
	LockSession(context.Context, domain.ID, domain.ID) error
	Get(context.Context, domain.ID, domain.ID, domain.ID, bool) (*domain.Reservation, error)
	FindActive(context.Context, domain.ID, domain.ID, domain.ID, domain.ID) (*domain.Reservation, error)
	ListSession(context.Context, domain.ID, domain.ID, domain.ID) ([]domain.Reservation, error)
	ListPerson(context.Context, domain.ID, domain.ID, domain.ID, time.Time, time.Time) ([]domain.Reservation, error)
	CountSession(context.Context, domain.ID, domain.ID, domain.ID, domain.ReservationStatus) (int, error)
	CountConsumed(context.Context, domain.ID, domain.ID, domain.ID, domain.ID, time.Time, time.Time) (int, error)
	Create(context.Context, *domain.Reservation) error
	Update(context.Context, *domain.Reservation, int) error
	FirstWaitlisted(context.Context, domain.ID, domain.ID, domain.ID) (*domain.Reservation, error)
	GetMakeupSource(context.Context, domain.ID, domain.ID, domain.ID, domain.ID) (*domain.Reservation, error)
	CreateAuditEvent(context.Context, *domain.ReservationAuditEvent) error
}

type ReservationEligibilityResolver interface {
	ResolveReservationEntitlements(context.Context, domain.ID, domain.ID, domain.ID, domain.ScheduleTargetKind, domain.ID, time.Time) ([]domain.ReservationEntitlement, error)
}

type ReservationPersonResolver interface {
	ResolveReservationPerson(context.Context, domain.ID, domain.ID, domain.ID) (string, error)
}

type ReservationPrivacyReader interface {
	ListReservationPrivacy(context.Context, domain.ID, domain.ID) ([]domain.Reservation, error)
}
