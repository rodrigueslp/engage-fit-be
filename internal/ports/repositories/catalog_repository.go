package repositories

import (
	"context"
	"errors"

	"boxengage/backend/internal/domain"
)

var (
	ErrCatalogOrganizationNotFound = errors.New("catalog organization not found")
	ErrCatalogUnitNotFound         = errors.New("catalog unit not found")
	ErrCatalogModalityNotFound     = errors.New("catalog modality not found")
	ErrCatalogSpaceNotFound        = errors.New("catalog space not found")
	ErrCatalogModalityNameConflict = errors.New("catalog modality name conflict")
	ErrCatalogSpaceNameConflict    = errors.New("catalog space name conflict")
)

type CatalogRepository interface {
	GetContext(ctx context.Context, unitID domain.ID) (*domain.CatalogContext, error)
	ListModalities(ctx context.Context, organizationID, unitID domain.ID, includeArchived bool) ([]domain.Modality, error)
	FindModality(ctx context.Context, organizationID, unitID, modalityID domain.ID) (*domain.Modality, error)
	CreateModality(ctx context.Context, modality *domain.Modality) error
	UpdateModality(ctx context.Context, modality domain.Modality) error
	CreateUnitModality(ctx context.Context, association *domain.UnitModality) error
	SetUnitModalityActive(ctx context.Context, organizationID, unitID, modalityID domain.ID, active bool) error
	ListSpaces(ctx context.Context, unitID domain.ID, includeArchived bool) ([]domain.Space, error)
	FindSpace(ctx context.Context, unitID, spaceID domain.ID) (*domain.Space, error)
	CreateSpace(ctx context.Context, space *domain.Space) error
	UpdateSpace(ctx context.Context, space domain.Space) error
	CreateAuditEvent(ctx context.Context, event *domain.CatalogAuditEvent) error
}
