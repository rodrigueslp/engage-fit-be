package repositories

import (
	"context"
	"errors"

	"boxengage/backend/internal/domain"
)

var (
	ErrOrganizationAccessNotFound   = errors.New("organization access not found")
	ErrOrganizationUnitNotFound     = errors.New("organization unit not found")
	ErrOrganizationMemberNotFound   = errors.New("organization member not found")
	ErrOrganizationUnitNameConflict = errors.New("organization unit name conflict")
)

type UnitAccessAuthorizer interface {
	ResolveUserUnitAccess(ctx context.Context, userID, unitID domain.ID) (*domain.UnitAccessContext, error)
}

type OrganizationRepository interface {
	UnitAccessAuthorizer
	ListAccessibleUnits(ctx context.Context, userID, organizationID domain.ID) ([]domain.Box, error)
	FindOrganization(ctx context.Context, organizationID domain.ID) (*domain.Organization, error)
	FindOrganizationOwner(ctx context.Context, organizationID domain.ID) (*domain.User, error)
	CreateUnit(ctx context.Context, unit *domain.Box, grantUserID, actorUserID domain.ID, at domain.OrganizationAuditEvent) error
	CountNonArchivedUnits(ctx context.Context, organizationID domain.ID) (int64, error)
	UpdateUnit(ctx context.Context, unit domain.Box, event domain.OrganizationAuditEvent) error
	FindOrganizationMember(ctx context.Context, organizationID, userID domain.ID) (*domain.User, error)
	ListMemberUnitIDs(ctx context.Context, organizationID, userID domain.ID) ([]domain.ID, error)
	ReplaceMemberUnitAccess(ctx context.Context, organizationID, userID domain.ID, unitIDs []domain.ID, actorUserID domain.ID, event domain.OrganizationAuditEvent) error
}
