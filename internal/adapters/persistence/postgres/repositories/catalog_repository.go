package repositories

import (
	"context"
	"encoding/json"
	"errors"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

func (r CatalogGormRepository) GetContext(ctx context.Context, unitID domain.ID) (*domain.CatalogContext, error) {
	db := databaseForContext(r.db, ctx)
	var unitModel models.BoxModel
	if err := db.Where("id = ?", stringID(unitID)).First(&unitModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, portrepo.ErrCatalogUnitNotFound
		}
		return nil, err
	}
	if unitModel.OrganizationID == nil || *unitModel.OrganizationID == "" {
		return nil, portrepo.ErrCatalogOrganizationNotFound
	}
	var organizationModel models.OrganizationModel
	if err := db.Where("id = ?", *unitModel.OrganizationID).First(&organizationModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, portrepo.ErrCatalogOrganizationNotFound
		}
		return nil, err
	}
	organization := organizationToDomain(organizationModel)
	unit := boxToDomain(unitModel)
	return &domain.CatalogContext{Organization: organization, Unit: unit}, nil
}

func (r CatalogGormRepository) ListModalities(ctx context.Context, organizationID, unitID domain.ID, includeArchived bool) ([]domain.Modality, error) {
	query := databaseForContext(r.db, ctx).Table("modalities m").
		Select("m.*, um.active AS active_in_unit").
		Joins("JOIN unit_modalities um ON um.modality_id = m.id AND um.organization_id = m.organization_id").
		Where("m.organization_id = ? AND um.unit_id = ? AND um.organization_id = ?", stringID(organizationID), stringID(unitID), stringID(organizationID))
	if !includeArchived {
		query = query.Where("m.status = ?", string(domain.CatalogStatusActive))
	}
	var rows []modalityRow
	if err := query.Order("CASE WHEN m.status = 'active' THEN 0 ELSE 1 END, m.name_key ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.Modality, 0, len(rows))
	for _, row := range rows {
		result = append(result, modalityRowToDomain(row))
	}
	return result, nil
}

func (r CatalogGormRepository) FindModality(ctx context.Context, organizationID, unitID, modalityID domain.ID) (*domain.Modality, error) {
	var row modalityRow
	err := databaseForContext(r.db, ctx).Table("modalities m").
		Select("m.*, um.active AS active_in_unit").
		Joins("JOIN unit_modalities um ON um.modality_id = m.id AND um.organization_id = m.organization_id").
		Where("m.id = ? AND m.organization_id = ? AND um.unit_id = ? AND um.organization_id = ?", stringID(modalityID), stringID(organizationID), stringID(unitID), stringID(organizationID)).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, portrepo.ErrCatalogModalityNotFound
		}
		return nil, err
	}
	item := modalityRowToDomain(row)
	return &item, nil
}

func (r CatalogGormRepository) CreateModality(ctx context.Context, modality *domain.Modality) error {
	if err := ensureID(&modality.ID); err != nil {
		return err
	}
	model := modalityToModel(*modality)
	if err := databaseForContext(r.db, ctx).Create(&model).Error; err != nil {
		return catalogWriteError(err)
	}
	return nil
}

func (r CatalogGormRepository) UpdateModality(ctx context.Context, modality domain.Modality) error {
	result := databaseForContext(r.db, ctx).Model(&models.ModalityModel{}).
		Where("id = ? AND organization_id = ?", stringID(modality.ID), stringID(modality.OrganizationID)).
		Updates(map[string]any{"name": modality.Name, "name_key": modality.NameKey, "color": modality.Color, "status": string(modality.Status), "updated_at": modality.UpdatedAt})
	if result.Error != nil {
		return catalogWriteError(result.Error)
	}
	if result.RowsAffected == 0 {
		return portrepo.ErrCatalogModalityNotFound
	}
	return nil
}

func (r CatalogGormRepository) CreateUnitModality(ctx context.Context, association *domain.UnitModality) error {
	model := unitModalityToModel(*association)
	return databaseForContext(r.db, ctx).Create(&model).Error
}

func (r CatalogGormRepository) SetUnitModalityActive(ctx context.Context, organizationID, unitID, modalityID domain.ID, active bool) error {
	result := databaseForContext(r.db, ctx).Model(&models.UnitModalityModel{}).
		Where("organization_id = ? AND unit_id = ? AND modality_id = ?", stringID(organizationID), stringID(unitID), stringID(modalityID)).
		Update("active", active)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return portrepo.ErrCatalogModalityNotFound
	}
	return nil
}

func (r CatalogGormRepository) ListSpaces(ctx context.Context, unitID domain.ID, includeArchived bool) ([]domain.Space, error) {
	query := databaseForContext(r.db, ctx).Where("unit_id = ?", stringID(unitID))
	if !includeArchived {
		query = query.Where("status = ?", string(domain.CatalogStatusActive))
	}
	var rows []models.SpaceModel
	if err := query.Order("CASE WHEN status = 'active' THEN 0 ELSE 1 END, name_key ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.Space, 0, len(rows))
	for _, row := range rows {
		result = append(result, spaceToDomain(row))
	}
	return result, nil
}

func (r CatalogGormRepository) FindSpace(ctx context.Context, unitID, spaceID domain.ID) (*domain.Space, error) {
	var model models.SpaceModel
	if err := databaseForContext(r.db, ctx).Where("id = ? AND unit_id = ?", stringID(spaceID), stringID(unitID)).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, portrepo.ErrCatalogSpaceNotFound
		}
		return nil, err
	}
	item := spaceToDomain(model)
	return &item, nil
}

func (r CatalogGormRepository) CreateSpace(ctx context.Context, space *domain.Space) error {
	if err := ensureID(&space.ID); err != nil {
		return err
	}
	model := spaceToModel(*space)
	if err := databaseForContext(r.db, ctx).Create(&model).Error; err != nil {
		return catalogWriteError(err)
	}
	return nil
}

func (r CatalogGormRepository) UpdateSpace(ctx context.Context, space domain.Space) error {
	result := databaseForContext(r.db, ctx).Model(&models.SpaceModel{}).
		Where("id = ? AND unit_id = ?", stringID(space.ID), stringID(space.UnitID)).
		Updates(map[string]any{"name": space.Name, "name_key": space.NameKey, "default_capacity": space.DefaultCapacity, "status": string(space.Status), "updated_at": space.UpdatedAt})
	if result.Error != nil {
		return catalogWriteError(result.Error)
	}
	if result.RowsAffected == 0 {
		return portrepo.ErrCatalogSpaceNotFound
	}
	return nil
}

func (r CatalogGormRepository) CreateAuditEvent(ctx context.Context, event *domain.CatalogAuditEvent) error {
	if err := ensureID(&event.ID); err != nil {
		return err
	}
	before, err := json.Marshal(nonNilAuditData(event.BeforeData))
	if err != nil {
		return err
	}
	after, err := json.Marshal(nonNilAuditData(event.AfterData))
	if err != nil {
		return err
	}
	var unitID *string
	if event.UnitID != "" {
		value := stringID(event.UnitID)
		unitID = &value
	}
	model := models.CatalogAuditEventModel{ID: stringID(event.ID), OrganizationID: stringID(event.OrganizationID), UnitID: unitID, ActorUserID: stringID(event.ActorUserID), Action: event.Action, EntityType: event.EntityType, EntityID: stringID(event.EntityID), BeforeData: before, AfterData: after, CreatedAt: event.CreatedAt}
	return databaseForContext(r.db, ctx).Create(&model).Error
}

type modalityRow struct {
	models.ModalityModel
	ActiveInUnit bool `gorm:"column:active_in_unit"`
}

func modalityRowToDomain(row modalityRow) domain.Modality {
	item := modalityToDomain(row.ModalityModel)
	item.ActiveInUnit = row.ActiveInUnit
	return item
}

func modalityToDomain(model models.ModalityModel) domain.Modality {
	return domain.Modality{ID: domainID(model.ID), OrganizationID: domainID(model.OrganizationID), Name: model.Name, NameKey: model.NameKey, Color: model.Color, Status: domain.CatalogStatus(model.Status), CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt}
}

func modalityToModel(item domain.Modality) models.ModalityModel {
	return models.ModalityModel{ID: stringID(item.ID), OrganizationID: stringID(item.OrganizationID), Name: item.Name, NameKey: item.NameKey, Color: item.Color, Status: string(item.Status), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func unitModalityToModel(item domain.UnitModality) models.UnitModalityModel {
	return models.UnitModalityModel{OrganizationID: stringID(item.OrganizationID), UnitID: stringID(item.UnitID), ModalityID: stringID(item.ModalityID), Active: item.Active, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func spaceToDomain(model models.SpaceModel) domain.Space {
	return domain.Space{ID: domainID(model.ID), UnitID: domainID(model.UnitID), Name: model.Name, NameKey: model.NameKey, DefaultCapacity: model.DefaultCapacity, Status: domain.CatalogStatus(model.Status), CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt}
}

func spaceToModel(item domain.Space) models.SpaceModel {
	return models.SpaceModel{ID: stringID(item.ID), UnitID: stringID(item.UnitID), Name: item.Name, NameKey: item.NameKey, DefaultCapacity: item.DefaultCapacity, Status: string(item.Status), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func organizationToDomain(model models.OrganizationModel) domain.Organization {
	var changedBy domain.ID
	if model.StatusChangedBy != nil {
		changedBy = domainID(*model.StatusChangedBy)
	}
	return domain.Organization{ID: domainID(model.ID), Name: model.Name, Status: domain.OrganizationStatus(model.Status), StatusReason: model.StatusReason, StatusChangedAt: model.StatusChangedAt, StatusChangedBy: changedBy, CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt}
}

func catalogWriteError(err error) error {
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) || postgresError.Code != "23505" {
		return err
	}
	switch postgresError.ConstraintName {
	case "modalities_organization_name_key_key":
		return portrepo.ErrCatalogModalityNameConflict
	case "spaces_unit_name_key_key":
		return portrepo.ErrCatalogSpaceNameConflict
	default:
		return err
	}
}

func nonNilAuditData(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}
