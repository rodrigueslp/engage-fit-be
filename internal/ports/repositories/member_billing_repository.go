package repositories

import (
	"context"
	"errors"
	"time"

	"boxengage/backend/internal/domain"
)

var (
	ErrMemberBillingNotFound          = errors.New("member billing not found")
	ErrMemberBillingConflict          = errors.New("member billing conflict")
	ErrMemberBillingIdempotencyReused = errors.New("member billing idempotency reused")
)

type MemberBillingRepository interface {
	LockCommand(context.Context, domain.ID, string, string) error
	LockContract(context.Context, domain.ID, domain.ID) error
	FindCommand(context.Context, domain.ID, string, string) (*domain.MemberBillingCommand, error)
	CreateCommand(context.Context, *domain.MemberBillingCommand) error
	GetContractByEnrollment(context.Context, domain.ID, domain.ID, domain.ID) (*domain.MemberBillingContract, error)
	GetContract(context.Context, domain.ID, domain.ID, domain.ID) (*domain.MemberBillingContract, error)
	CreateContract(context.Context, *domain.MemberBillingContract) error
	UpdateNextDue(context.Context, domain.ID, domain.ID, *time.Time, time.Time) error
	ListCharges(context.Context, domain.ID, domain.ID, domain.ID) ([]domain.MemberCharge, error)
	GetCharge(context.Context, domain.ID, domain.ID, domain.ID) (*domain.MemberCharge, error)
	GetChargeForUpdate(context.Context, domain.ID, domain.ID, domain.ID) (*domain.MemberCharge, error)
	CreateCharge(context.Context, *domain.MemberCharge) error
	UpdateCharge(context.Context, *domain.MemberCharge) error
	CreatePayment(context.Context, *domain.MemberPayment) error
	ListPrivacy(context.Context, domain.ID, domain.ID) ([]domain.MemberBillingSnapshot, error)
}

type MemberBillingPrivacyReader interface {
	ListPrivacy(context.Context, domain.ID, domain.ID) ([]domain.MemberBillingSnapshot, error)
}
