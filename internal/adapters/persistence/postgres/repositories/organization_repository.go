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
	"gorm.io/gorm/clause"
)

func (r OrganizationGormRepository) ResolveUserUnitAccess(ctx context.Context, userID, unitID domain.ID) (*domain.UnitAccessContext, error) {
	var row struct {
		models.BoxModel
		OrganizationName   string `gorm:"column:organization_name"`
		OrganizationStatus string `gorm:"column:organization_status"`
		MembershipRole     string `gorm:"column:membership_role"`
	}
	err := databaseForContext(r.db, ctx).Table("boxes b").
		Select("b.*, o.name AS organization_name, o.status AS organization_status, oum.role AS membership_role").
		Joins("JOIN organizations o ON o.id = b.organization_id").
		Joins("JOIN organization_user_memberships oum ON oum.organization_id = b.organization_id AND oum.user_id = ? AND oum.active = TRUE", stringID(userID)).
		Joins("JOIN user_unit_access uua ON uua.organization_id = b.organization_id AND uua.unit_id = b.id AND uua.user_id = ? AND uua.active = TRUE", stringID(userID)).
		Where("b.id = ?", stringID(unitID)).Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, portrepo.ErrOrganizationAccessNotFound
		}
		return nil, err
	}
	unit := boxToDomain(row.BoxModel)
	organization := domain.Organization{ID: unit.OrganizationID, Name: row.OrganizationName, Status: domain.OrganizationStatus(row.OrganizationStatus)}
	return &domain.UnitAccessContext{Organization: organization, Unit: unit, Role: domain.UserRole(row.MembershipRole)}, nil
}

func (r OrganizationGormRepository) ListAccessibleUnits(ctx context.Context, userID, organizationID domain.ID) ([]domain.Box, error) {
	var rows []models.BoxModel
	err := databaseForContext(r.db, ctx).Table("boxes b").
		Joins("JOIN user_unit_access uua ON uua.unit_id = b.id AND uua.organization_id = b.organization_id").
		Joins("JOIN organization_user_memberships oum ON oum.organization_id = b.organization_id AND oum.user_id = uua.user_id AND oum.active = TRUE").
		Where("b.organization_id = ? AND uua.user_id = ? AND uua.active = TRUE", stringID(organizationID), stringID(userID)).
		Order("CASE WHEN b.status = 'active' THEN 0 WHEN b.status = 'suspended' THEN 1 ELSE 2 END, b.name_key ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]domain.Box, 0, len(rows))
	for _, row := range rows {
		result = append(result, boxToDomain(row))
	}
	return result, nil
}

func (r OrganizationGormRepository) FindOrganization(ctx context.Context, organizationID domain.ID) (*domain.Organization, error) {
	var row models.OrganizationModel
	if err := databaseForContext(r.db, ctx).Where("id = ?", stringID(organizationID)).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, portrepo.ErrOrganizationAccessNotFound
		}
		return nil, err
	}
	item := organizationToDomain(row)
	return &item, nil
}

func (r OrganizationGormRepository) FindOrganizationOwner(ctx context.Context, organizationID domain.ID) (*domain.User, error) {
	var row models.UserModel
	err := databaseForContext(r.db, ctx).Table("users u").Joins("JOIN organization_user_memberships oum ON oum.user_id = u.id AND oum.active = TRUE").
		Where("oum.organization_id = ? AND u.role = ?", stringID(organizationID), string(domain.UserRoleOwner)).Order("u.created_at ASC").First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, portrepo.ErrOrganizationMemberNotFound
		}
		return nil, err
	}
	item := userToDomain(row)
	return &item, nil
}

func (r OrganizationGormRepository) CreateUnit(ctx context.Context, unit *domain.Box, grantUserID, actorUserID domain.ID, event domain.OrganizationAuditEvent) error {
	return databaseForContext(r.db, ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureID(&unit.ID); err != nil {
			return err
		}
		unitModel := boxToModel(*unit)
		if err := tx.Create(&unitModel).Error; err != nil {
			return organizationWriteError(err)
		}
		now := unit.CreatedAt
		actor := stringID(actorUserID)
		access := models.UserUnitAccessModel{OrganizationID: stringID(unit.OrganizationID), UserID: stringID(grantUserID), UnitID: stringID(unit.ID), Active: true, GrantedByUserID: &actor, CreatedAt: now, UpdatedAt: now}
		if err := tx.Create(&access).Error; err != nil {
			return err
		}
		event.UnitID, event.EntityID = unit.ID, unit.ID
		return saveOrganizationAudit(tx, &event)
	})
}

func (r OrganizationGormRepository) CountNonArchivedUnits(ctx context.Context, organizationID domain.ID) (int64, error) {
	var count int64
	err := databaseForContext(r.db, ctx).Model(&models.BoxModel{}).Where("organization_id = ? AND status <> 'archived'", stringID(organizationID)).Count(&count).Error
	return count, err
}

func (r OrganizationGormRepository) UpdateUnit(ctx context.Context, unit domain.Box, event domain.OrganizationAuditEvent) error {
	return databaseForContext(r.db, ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.BoxModel{}).Where("id = ? AND organization_id = ?", stringID(unit.ID), stringID(unit.OrganizationID)).Updates(map[string]any{
			"name": unit.Name, "name_key": unit.NameKey, "timezone": unit.Timezone, "status": string(unit.EffectiveStatus()), "status_reason": unit.StatusReason,
			"status_changed_at": unit.StatusChangedAt, "status_changed_by": nullableStringID(unit.StatusChangedBy), "updated_at": unit.UpdatedAt,
		})
		if result.Error != nil {
			return organizationWriteError(result.Error)
		}
		if result.RowsAffected == 0 {
			return portrepo.ErrOrganizationUnitNotFound
		}
		return saveOrganizationAudit(tx, &event)
	})
}

func (r OrganizationGormRepository) FindOrganizationMember(ctx context.Context, organizationID, userID domain.ID) (*domain.User, error) {
	var row models.UserModel
	err := databaseForContext(r.db, ctx).Table("users u").
		Joins("JOIN organization_user_memberships oum ON oum.user_id = u.id").
		Where("u.id = ? AND oum.organization_id = ?", stringID(userID), stringID(organizationID)).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, portrepo.ErrOrganizationMemberNotFound
		}
		return nil, err
	}
	item := userToDomain(row)
	return &item, nil
}

func (r OrganizationGormRepository) ListMemberUnitIDs(ctx context.Context, organizationID, userID domain.ID) ([]domain.ID, error) {
	var rows []string
	err := databaseForContext(r.db, ctx).Model(&models.UserUnitAccessModel{}).
		Where("organization_id = ? AND user_id = ? AND active = TRUE", stringID(organizationID), stringID(userID)).Order("unit_id ASC").Pluck("unit_id", &rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]domain.ID, 0, len(rows))
	for _, row := range rows {
		result = append(result, domainID(row))
	}
	return result, nil
}

func (r OrganizationGormRepository) ReplaceMemberUnitAccess(ctx context.Context, organizationID, userID domain.ID, unitIDs []domain.ID, actorUserID domain.ID, event domain.OrganizationAuditEvent) error {
	return databaseForContext(r.db, ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.UserUnitAccessModel{}).Where("organization_id = ? AND user_id = ?", stringID(organizationID), stringID(userID)).Update("active", false).Error; err != nil {
			return err
		}
		now := event.CreatedAt
		actor := stringID(actorUserID)
		for _, unitID := range unitIDs {
			item := models.UserUnitAccessModel{OrganizationID: stringID(organizationID), UserID: stringID(userID), UnitID: stringID(unitID), Active: true, GrantedByUserID: &actor, CreatedAt: now, UpdatedAt: now}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "unit_id"}}, DoUpdates: clause.Assignments(map[string]any{"active": true, "granted_by_user_id": actor, "updated_at": now})}).Create(&item).Error; err != nil {
				return err
			}
		}
		return saveOrganizationAudit(tx, &event)
	})
}

func saveOrganizationAudit(db *gorm.DB, event *domain.OrganizationAuditEvent) error {
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
	model := models.OrganizationAuditEventModel{ID: stringID(event.ID), OrganizationID: stringID(event.OrganizationID), UnitID: unitID, ActorUserID: stringID(event.ActorUserID), Action: event.Action, EntityType: event.EntityType, EntityID: stringID(event.EntityID), BeforeData: before, AfterData: after, CreatedAt: event.CreatedAt}
	return db.Create(&model).Error
}

func organizationWriteError(err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23505" && postgresError.ConstraintName == "boxes_organization_name_key_key" {
		return portrepo.ErrOrganizationUnitNameConflict
	}
	return err
}

func nullableStringID(value domain.ID) any {
	if value == "" {
		return nil
	}
	return stringID(value)
}
