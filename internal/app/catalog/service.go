package catalog

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

var (
	ErrInvalidInput                = errors.New("invalid catalog input")
	ErrOrganizationNotFound        = repositories.ErrCatalogOrganizationNotFound
	ErrUnitNotFound                = repositories.ErrCatalogUnitNotFound
	ErrModalityNotFound            = repositories.ErrCatalogModalityNotFound
	ErrSpaceNotFound               = repositories.ErrCatalogSpaceNotFound
	ErrModalityNameConflict        = repositories.ErrCatalogModalityNameConflict
	ErrSpaceNameConflict           = repositories.ErrCatalogSpaceNameConflict
	ErrModalityArchived            = errors.New("catalog modality archived")
	ErrServiceOfferingNotFound     = repositories.ErrCatalogServiceOfferingNotFound
	ErrServiceOfferingNameConflict = repositories.ErrCatalogServiceOfferingNameConflict
	ErrServiceOfferingArchived     = errors.New("catalog service offering archived")
)

type ModalityChanges struct {
	Name   *string
	Color  *string
	Status *domain.CatalogStatus
}

type SpaceChanges struct {
	Name                 *string
	DefaultCapacity      *int
	ClearDefaultCapacity bool
	Status               *domain.CatalogStatus
}

type ServiceOfferingChanges struct {
	Name        *string
	Description *string
	Color       *string
	Status      *domain.CatalogStatus
}

type Service struct {
	repository   repositories.CatalogRepository
	transactions repositories.TransactionManager
	now          func() time.Time
}

func NewService(repository repositories.CatalogRepository, transactions repositories.TransactionManager) *Service {
	return &Service{repository: repository, transactions: transactions, now: time.Now}
}

func (s Service) Organization(ctx context.Context, unitID domain.ID) (*domain.CatalogContext, error) {
	return s.repository.GetContext(ctx, unitID)
}

func (s Service) ListModalities(ctx context.Context, sessionUnitID, requestedUnitID domain.ID, includeArchived bool) ([]domain.Modality, error) {
	if requestedUnitID != "" && requestedUnitID != sessionUnitID {
		return nil, ErrUnitNotFound
	}
	current, err := s.repository.GetContext(ctx, sessionUnitID)
	if err != nil {
		return nil, err
	}
	return s.repository.ListModalities(ctx, current.Organization.ID, sessionUnitID, includeArchived)
}

func (s Service) CreateModality(ctx context.Context, unitID, actorUserID domain.ID, name, color string) (*domain.Modality, error) {
	normalizedName, nameKey, ok := domain.NormalizeCatalogName(name)
	if !ok {
		return nil, ErrInvalidInput
	}
	normalizedColor, ok := domain.NormalizeModalityColor(color)
	if !ok {
		return nil, ErrInvalidInput
	}
	var created domain.Modality
	err := s.transactions.WithinTransaction(ctx, func(txctx context.Context) error {
		current, err := s.repository.GetContext(txctx, unitID)
		if err != nil {
			return err
		}
		now := s.now().UTC()
		created = domain.Modality{OrganizationID: current.Organization.ID, Name: normalizedName, NameKey: nameKey, Color: normalizedColor, Status: domain.CatalogStatusActive, ActiveInUnit: true, CreatedAt: now, UpdatedAt: now}
		if err := s.repository.CreateModality(txctx, &created); err != nil {
			return err
		}
		association := domain.UnitModality{OrganizationID: current.Organization.ID, UnitID: unitID, ModalityID: created.ID, Active: true, CreatedAt: now, UpdatedAt: now}
		if err := s.repository.CreateUnitModality(txctx, &association); err != nil {
			return err
		}
		if err := s.audit(txctx, current.Organization.ID, unitID, actorUserID, "modality.created", "modality", created.ID, nil, modalityAuditData(created), now); err != nil {
			return err
		}
		return s.audit(txctx, current.Organization.ID, unitID, actorUserID, "unit_modality.activated", "unit_modality", created.ID, nil, unitModalityAuditData(true), now)
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func (s Service) UpdateModality(ctx context.Context, unitID, actorUserID, modalityID domain.ID, changes ModalityChanges) (*domain.Modality, error) {
	if changes.Name == nil && changes.Color == nil && changes.Status == nil {
		return nil, ErrInvalidInput
	}
	var updated domain.Modality
	err := s.transactions.WithinTransaction(ctx, func(txctx context.Context) error {
		current, err := s.repository.GetContext(txctx, unitID)
		if err != nil {
			return err
		}
		item, err := s.repository.FindModality(txctx, current.Organization.ID, unitID, modalityID)
		if err != nil {
			return err
		}
		before := *item
		if changes.Name != nil {
			name, key, valid := domain.NormalizeCatalogName(*changes.Name)
			if !valid {
				return ErrInvalidInput
			}
			item.Name, item.NameKey = name, key
		}
		if changes.Color != nil {
			color, valid := domain.NormalizeModalityColor(*changes.Color)
			if !valid {
				return ErrInvalidInput
			}
			item.Color = color
		}
		if changes.Status != nil {
			if !domain.ValidCatalogStatus(*changes.Status) {
				return ErrInvalidInput
			}
			item.Status = *changes.Status
		}
		item.UpdatedAt = s.now().UTC()
		if err := s.repository.UpdateModality(txctx, *item); err != nil {
			return err
		}
		action := "modality.updated"
		if before.Status != item.Status {
			if item.Status == domain.CatalogStatusArchived {
				action = "modality.archived"
			} else {
				action = "modality.reactivated"
			}
		}
		if err := s.audit(txctx, current.Organization.ID, unitID, actorUserID, action, "modality", item.ID, modalityAuditData(before), modalityAuditData(*item), item.UpdatedAt); err != nil {
			return err
		}
		updated = *item
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s Service) SetModalityActive(ctx context.Context, sessionUnitID, requestedUnitID, actorUserID, modalityID domain.ID, active bool) (*domain.Modality, error) {
	if requestedUnitID != sessionUnitID {
		return nil, ErrUnitNotFound
	}
	var updated domain.Modality
	err := s.transactions.WithinTransaction(ctx, func(txctx context.Context) error {
		current, err := s.repository.GetContext(txctx, sessionUnitID)
		if err != nil {
			return err
		}
		item, err := s.repository.FindModality(txctx, current.Organization.ID, sessionUnitID, modalityID)
		if err != nil {
			return err
		}
		if active && item.Status == domain.CatalogStatusArchived {
			return ErrModalityArchived
		}
		before := item.ActiveInUnit
		if err := s.repository.SetUnitModalityActive(txctx, current.Organization.ID, sessionUnitID, modalityID, active); err != nil {
			return err
		}
		item.ActiveInUnit = active
		item.UpdatedAt = s.now().UTC()
		action := "unit_modality.deactivated"
		if active {
			action = "unit_modality.activated"
		}
		if err := s.audit(txctx, current.Organization.ID, sessionUnitID, actorUserID, action, "unit_modality", modalityID, unitModalityAuditData(before), unitModalityAuditData(active), item.UpdatedAt); err != nil {
			return err
		}
		updated = *item
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s Service) ListServiceOfferings(ctx context.Context, sessionUnitID, requestedUnitID domain.ID, includeArchived bool) ([]domain.ServiceOffering, error) {
	if requestedUnitID != "" && requestedUnitID != sessionUnitID {
		return nil, ErrUnitNotFound
	}
	current, err := s.repository.GetContext(ctx, sessionUnitID)
	if err != nil {
		return nil, err
	}
	return s.repository.ListServiceOfferings(ctx, current.Organization.ID, sessionUnitID, includeArchived)
}

func (s Service) CreateServiceOffering(ctx context.Context, unitID, actorUserID domain.ID, name, description, color string) (*domain.ServiceOffering, error) {
	normalizedName, nameKey, ok := domain.NormalizeCatalogName(name)
	if !ok {
		return nil, ErrInvalidInput
	}
	normalizedDescription, ok := domain.NormalizeServiceOfferingDescription(description)
	if !ok {
		return nil, ErrInvalidInput
	}
	normalizedColor, ok := domain.NormalizeCatalogColor(color)
	if !ok {
		return nil, ErrInvalidInput
	}
	var created domain.ServiceOffering
	err := s.transactions.WithinTransaction(ctx, func(txctx context.Context) error {
		current, err := s.repository.GetContext(txctx, unitID)
		if err != nil {
			return err
		}
		now := s.now().UTC()
		created = domain.ServiceOffering{OrganizationID: current.Organization.ID, Name: normalizedName, NameKey: nameKey, Description: normalizedDescription, Color: normalizedColor, Status: domain.CatalogStatusActive, ActiveInUnit: true, CreatedAt: now, UpdatedAt: now}
		if err := s.repository.CreateServiceOffering(txctx, &created); err != nil {
			return err
		}
		association := domain.UnitServiceOffering{OrganizationID: current.Organization.ID, UnitID: unitID, ServiceOfferingID: created.ID, Active: true, CreatedAt: now, UpdatedAt: now}
		if err := s.repository.CreateUnitServiceOffering(txctx, &association); err != nil {
			return err
		}
		if err := s.audit(txctx, current.Organization.ID, unitID, actorUserID, "service_offering.created", "service_offering", created.ID, nil, serviceOfferingAuditData(created), now); err != nil {
			return err
		}
		return s.audit(txctx, current.Organization.ID, unitID, actorUserID, "unit_service_offering.activated", "unit_service_offering", created.ID, nil, unitServiceOfferingAuditData(true), now)
	})
	if err != nil {
		return nil, err
	}
	logServiceOfferingAction("service_offering.created", created.OrganizationID, unitID, actorUserID, created.ID)
	return &created, nil
}

func (s Service) UpdateServiceOffering(ctx context.Context, unitID, actorUserID, serviceOfferingID domain.ID, changes ServiceOfferingChanges) (*domain.ServiceOffering, error) {
	if changes.Name == nil && changes.Description == nil && changes.Color == nil && changes.Status == nil {
		return nil, ErrInvalidInput
	}
	var updated domain.ServiceOffering
	action := "service_offering.updated"
	err := s.transactions.WithinTransaction(ctx, func(txctx context.Context) error {
		current, err := s.repository.GetContext(txctx, unitID)
		if err != nil {
			return err
		}
		item, err := s.repository.FindServiceOffering(txctx, current.Organization.ID, unitID, serviceOfferingID)
		if err != nil {
			return err
		}
		before := *item
		if changes.Name != nil {
			name, key, valid := domain.NormalizeCatalogName(*changes.Name)
			if !valid {
				return ErrInvalidInput
			}
			item.Name, item.NameKey = name, key
		}
		if changes.Description != nil {
			description, valid := domain.NormalizeServiceOfferingDescription(*changes.Description)
			if !valid {
				return ErrInvalidInput
			}
			item.Description = description
		}
		if changes.Color != nil {
			color, valid := domain.NormalizeCatalogColor(*changes.Color)
			if !valid {
				return ErrInvalidInput
			}
			item.Color = color
		}
		if changes.Status != nil {
			if !domain.ValidCatalogStatus(*changes.Status) {
				return ErrInvalidInput
			}
			item.Status = *changes.Status
		}
		item.UpdatedAt = s.now().UTC()
		if err := s.repository.UpdateServiceOffering(txctx, *item); err != nil {
			return err
		}
		if before.Status != item.Status {
			if item.Status == domain.CatalogStatusArchived {
				action = "service_offering.archived"
			} else {
				action = "service_offering.reactivated"
			}
		}
		if err := s.audit(txctx, current.Organization.ID, unitID, actorUserID, action, "service_offering", item.ID, serviceOfferingAuditData(before), serviceOfferingAuditData(*item), item.UpdatedAt); err != nil {
			return err
		}
		updated = *item
		return nil
	})
	if err != nil {
		return nil, err
	}
	logServiceOfferingAction(action, updated.OrganizationID, unitID, actorUserID, updated.ID)
	return &updated, nil
}

func (s Service) SetServiceOfferingActive(ctx context.Context, sessionUnitID, requestedUnitID, actorUserID, serviceOfferingID domain.ID, active bool) (*domain.ServiceOffering, error) {
	if requestedUnitID != sessionUnitID {
		return nil, ErrUnitNotFound
	}
	var updated domain.ServiceOffering
	action := "unit_service_offering.deactivated"
	if active {
		action = "unit_service_offering.activated"
	}
	err := s.transactions.WithinTransaction(ctx, func(txctx context.Context) error {
		current, err := s.repository.GetContext(txctx, sessionUnitID)
		if err != nil {
			return err
		}
		item, err := s.repository.FindServiceOffering(txctx, current.Organization.ID, sessionUnitID, serviceOfferingID)
		if err != nil {
			return err
		}
		if active && item.Status == domain.CatalogStatusArchived {
			return ErrServiceOfferingArchived
		}
		before := item.ActiveInUnit
		if err := s.repository.SetUnitServiceOfferingActive(txctx, current.Organization.ID, sessionUnitID, serviceOfferingID, active); err != nil {
			return err
		}
		item.ActiveInUnit = active
		item.UpdatedAt = s.now().UTC()
		if err := s.audit(txctx, current.Organization.ID, sessionUnitID, actorUserID, action, "unit_service_offering", serviceOfferingID, unitServiceOfferingAuditData(before), unitServiceOfferingAuditData(active), item.UpdatedAt); err != nil {
			return err
		}
		updated = *item
		return nil
	})
	if err != nil {
		return nil, err
	}
	logServiceOfferingAction(action, updated.OrganizationID, sessionUnitID, actorUserID, updated.ID)
	return &updated, nil
}

func (s Service) ListSpaces(ctx context.Context, sessionUnitID, requestedUnitID domain.ID, includeArchived bool) ([]domain.Space, error) {
	if requestedUnitID != sessionUnitID {
		return nil, ErrUnitNotFound
	}
	if _, err := s.repository.GetContext(ctx, sessionUnitID); err != nil {
		return nil, err
	}
	return s.repository.ListSpaces(ctx, sessionUnitID, includeArchived)
}

func (s Service) CreateSpace(ctx context.Context, sessionUnitID, requestedUnitID, actorUserID domain.ID, name string, capacity *int) (*domain.Space, error) {
	if requestedUnitID != sessionUnitID {
		return nil, ErrUnitNotFound
	}
	normalizedName, nameKey, ok := domain.NormalizeCatalogName(name)
	if !ok || !validCapacity(capacity) {
		return nil, ErrInvalidInput
	}
	var created domain.Space
	err := s.transactions.WithinTransaction(ctx, func(txctx context.Context) error {
		current, err := s.repository.GetContext(txctx, sessionUnitID)
		if err != nil {
			return err
		}
		now := s.now().UTC()
		created = domain.Space{UnitID: sessionUnitID, Name: normalizedName, NameKey: nameKey, DefaultCapacity: capacity, Status: domain.CatalogStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := s.repository.CreateSpace(txctx, &created); err != nil {
			return err
		}
		return s.audit(txctx, current.Organization.ID, sessionUnitID, actorUserID, "space.created", "space", created.ID, nil, spaceAuditData(created), now)
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func (s Service) UpdateSpace(ctx context.Context, sessionUnitID, requestedUnitID, actorUserID, spaceID domain.ID, changes SpaceChanges) (*domain.Space, error) {
	if requestedUnitID != sessionUnitID {
		return nil, ErrUnitNotFound
	}
	if changes.Name == nil && changes.DefaultCapacity == nil && !changes.ClearDefaultCapacity && changes.Status == nil {
		return nil, ErrInvalidInput
	}
	if changes.DefaultCapacity != nil && !validCapacity(changes.DefaultCapacity) {
		return nil, ErrInvalidInput
	}
	var updated domain.Space
	err := s.transactions.WithinTransaction(ctx, func(txctx context.Context) error {
		current, err := s.repository.GetContext(txctx, sessionUnitID)
		if err != nil {
			return err
		}
		item, err := s.repository.FindSpace(txctx, sessionUnitID, spaceID)
		if err != nil {
			return err
		}
		before := *item
		if changes.Name != nil {
			name, key, valid := domain.NormalizeCatalogName(*changes.Name)
			if !valid {
				return ErrInvalidInput
			}
			item.Name, item.NameKey = name, key
		}
		if changes.ClearDefaultCapacity {
			item.DefaultCapacity = nil
		} else if changes.DefaultCapacity != nil {
			item.DefaultCapacity = changes.DefaultCapacity
		}
		if changes.Status != nil {
			if !domain.ValidCatalogStatus(*changes.Status) {
				return ErrInvalidInput
			}
			item.Status = *changes.Status
		}
		item.UpdatedAt = s.now().UTC()
		if err := s.repository.UpdateSpace(txctx, *item); err != nil {
			return err
		}
		action := "space.updated"
		if before.Status != item.Status {
			if item.Status == domain.CatalogStatusArchived {
				action = "space.archived"
			} else {
				action = "space.reactivated"
			}
		}
		if err := s.audit(txctx, current.Organization.ID, sessionUnitID, actorUserID, action, "space", item.ID, spaceAuditData(before), spaceAuditData(*item), item.UpdatedAt); err != nil {
			return err
		}
		updated = *item
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s Service) audit(ctx context.Context, organizationID, unitID, actorID domain.ID, action, entityType string, entityID domain.ID, before, after map[string]any, at time.Time) error {
	event := domain.CatalogAuditEvent{OrganizationID: organizationID, UnitID: unitID, ActorUserID: actorID, Action: action, EntityType: entityType, EntityID: entityID, BeforeData: before, AfterData: after, CreatedAt: at}
	return s.repository.CreateAuditEvent(ctx, &event)
}

func validCapacity(value *int) bool { return value == nil || (*value >= 1 && *value <= 10000) }

func modalityAuditData(item domain.Modality) map[string]any {
	return map[string]any{"name": item.Name, "color": item.Color, "status": item.Status}
}
func unitModalityAuditData(active bool) map[string]any { return map[string]any{"active": active} }
func serviceOfferingAuditData(item domain.ServiceOffering) map[string]any {
	return map[string]any{"name": item.Name, "description": item.Description, "color": item.Color, "status": item.Status}
}
func unitServiceOfferingAuditData(active bool) map[string]any {
	return map[string]any{"active": active}
}
func spaceAuditData(item domain.Space) map[string]any {
	return map[string]any{"name": item.Name, "default_capacity": item.DefaultCapacity, "status": item.Status}
}

func logServiceOfferingAction(action string, organizationID, unitID, actorUserID, serviceOfferingID domain.ID) {
	slog.Info("catalog service offering changed", "action", action, "organization_id", organizationID, "unit_id", unitID, "actor_user_id", actorUserID, "service_offering_id", serviceOfferingID)
}
