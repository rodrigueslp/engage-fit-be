package repositories

import (
	"context"
	"errors"
	"time"

	"boxengage/backend/internal/domain"
)

var (
	ErrMembershipPlanNotFound      = errors.New("membership plan not found")
	ErrMembershipPlanNameConflict  = errors.New("membership plan name conflict")
	ErrMembershipDraftNotFound     = errors.New("membership plan draft not found")
	ErrMembershipDraftExists       = errors.New("membership plan draft exists")
	ErrMembershipRevisionConflict  = errors.New("membership plan revision conflict")
	ErrMembershipAlreadyPublished  = errors.New("membership plan already published")
	ErrMembershipPlanArchived      = errors.New("membership plan archived")
	ErrMembershipIdempotencyReused = errors.New("membership idempotency key reused")
	ErrMembershipAddonNotFound     = errors.New("membership plan addon not found")
	ErrMembershipAddonDuplicate    = errors.New("membership plan addon duplicate")
)

type MembershipRepository interface {
	ListPlans(ctx context.Context, organizationID, unitID domain.ID, includeArchived bool) ([]domain.MembershipPlan, error)
	GetPlan(ctx context.Context, organizationID, unitID, planID domain.ID) (*domain.MembershipPlan, error)
	LockCommand(ctx context.Context, organizationID domain.ID, command, idempotencyKey string) error
	FindCommand(ctx context.Context, organizationID domain.ID, command, idempotencyKey string) (*domain.MembershipCommandRequest, error)
	CreateCommand(ctx context.Context, request *domain.MembershipCommandRequest) error
	CreatePlan(ctx context.Context, plan *domain.MembershipPlan, version *domain.MembershipPlanVersion, entitlements []domain.MembershipPlanEntitlement, addons []domain.MembershipPlanVersionAddon) error
	UpdateDraft(ctx context.Context, organizationID, unitID, planID domain.ID, expectedRevision int, version domain.MembershipPlanVersion, entitlements []domain.MembershipPlanEntitlement, addons []domain.MembershipPlanVersionAddon) (*domain.MembershipPlan, error)
	PublishDraft(ctx context.Context, organizationID, unitID, planID domain.ID, draftRevision int, publishedAt time.Time) (*domain.MembershipPlan, error)
	CreateNewVersion(ctx context.Context, organizationID, unitID, planID domain.ID, createdAt time.Time) (*domain.MembershipPlan, error)
	SetPlanStatus(ctx context.Context, organizationID, unitID, planID domain.ID, status domain.MembershipPlanStatus, updatedAt time.Time) (*domain.MembershipPlan, error)
	CreateAuditEvent(ctx context.Context, event *domain.MembershipPlanAuditEvent) error
}
