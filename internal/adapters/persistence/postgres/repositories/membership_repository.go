package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

func (r MembershipGormRepository) ListPlans(ctx context.Context, organizationID, unitID domain.ID, includeArchived bool) ([]domain.MembershipPlan, error) {
	db := databaseForContext(r.db, ctx)
	query := db.Where("organization_id = ? AND home_unit_id = ?", stringID(organizationID), stringID(unitID))
	if !includeArchived {
		query = query.Where("status = ?", string(domain.MembershipPlanActive))
	}
	var rows []models.MembershipPlanModel
	if err := query.Order("CASE WHEN status = 'active' THEN 0 ELSE 1 END, name_key ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.MembershipPlan, 0, len(rows))
	for _, row := range rows {
		plan, err := loadMembershipPlan(db, row)
		if err != nil {
			return nil, err
		}
		result = append(result, *plan)
	}
	return result, nil
}

func (r MembershipGormRepository) GetPlan(ctx context.Context, organizationID, unitID, planID domain.ID) (*domain.MembershipPlan, error) {
	db := databaseForContext(r.db, ctx)
	var row models.MembershipPlanModel
	if err := db.Where("id = ? AND organization_id = ? AND home_unit_id = ?", stringID(planID), stringID(organizationID), stringID(unitID)).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, portrepo.ErrMembershipPlanNotFound
		}
		return nil, err
	}
	return loadMembershipPlan(db, row)
}

func (r MembershipGormRepository) LockCommand(ctx context.Context, organizationID domain.ID, command, idempotencyKey string) error {
	key := fmt.Sprintf("membership:%s:%s:%s", organizationID, command, idempotencyKey)
	return databaseForContext(r.db, ctx).Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", key).Error
}

func (r MembershipGormRepository) FindCommand(ctx context.Context, organizationID domain.ID, command, idempotencyKey string) (*domain.MembershipCommandRequest, error) {
	var row models.MembershipPlanCommandRequestModel
	err := databaseForContext(r.db, ctx).Where("organization_id = ? AND command = ? AND idempotency_key = ?", stringID(organizationID), command, idempotencyKey).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	response := map[string]any{}
	if err := json.Unmarshal(row.ResponseData, &response); err != nil {
		return nil, err
	}
	return &domain.MembershipCommandRequest{ID: domainID(row.ID), OrganizationID: domainID(row.OrganizationID), Command: row.Command, IdempotencyKey: row.IdempotencyKey, Fingerprint: row.RequestFingerprint, PlanID: domainID(row.PlanID), PlanVersionID: domainID(row.PlanVersionID), ResponseData: response, CreatedAt: row.CreatedAt}, nil
}

func (r MembershipGormRepository) CreateCommand(ctx context.Context, request *domain.MembershipCommandRequest) error {
	if err := ensureID(&request.ID); err != nil {
		return err
	}
	response, err := json.Marshal(nonNilAuditData(request.ResponseData))
	if err != nil {
		return err
	}
	row := models.MembershipPlanCommandRequestModel{ID: stringID(request.ID), OrganizationID: stringID(request.OrganizationID), Command: request.Command, IdempotencyKey: request.IdempotencyKey, RequestFingerprint: request.Fingerprint, PlanID: stringID(request.PlanID), PlanVersionID: stringID(request.PlanVersionID), ResponseData: response, CreatedAt: request.CreatedAt}
	if err := databaseForContext(r.db, ctx).Create(&row).Error; err != nil {
		return membershipWriteError(err)
	}
	return nil
}

func (r MembershipGormRepository) CreatePlan(ctx context.Context, plan *domain.MembershipPlan, version *domain.MembershipPlanVersion, entitlements []domain.MembershipPlanEntitlement, addons []domain.MembershipPlanVersionAddon) error {
	if err := ensureID(&plan.ID); err != nil {
		return err
	}
	if err := ensureID(&version.ID); err != nil {
		return err
	}
	db := databaseForContext(r.db, ctx)
	planRow := membershipPlanToModel(*plan)
	if err := db.Create(&planRow).Error; err != nil {
		return membershipWriteError(err)
	}
	version.PlanID = plan.ID
	version.OrganizationID = plan.OrganizationID
	versionRow := membershipVersionToModel(*version)
	if err := db.Create(&versionRow).Error; err != nil {
		return membershipWriteError(err)
	}
	if err := createMembershipEntitlements(db, plan.OrganizationID, version.ID, entitlements); err != nil {
		return err
	}
	return createMembershipAddons(db, plan.OrganizationID, version.ID, addons)
}

func (r MembershipGormRepository) UpdateDraft(ctx context.Context, organizationID, unitID, planID domain.ID, expectedRevision int, version domain.MembershipPlanVersion, entitlements []domain.MembershipPlanEntitlement, addons []domain.MembershipPlanVersionAddon) (*domain.MembershipPlan, error) {
	db := databaseForContext(r.db, ctx)
	planRow, err := lockMembershipPlan(db, organizationID, unitID, planID)
	if err != nil {
		return nil, err
	}
	var draft models.MembershipPlanVersionModel
	if err := db.Raw("SELECT * FROM membership_plan_versions WHERE plan_id = ? AND organization_id = ? AND status = 'draft' FOR UPDATE", stringID(planID), stringID(organizationID)).Scan(&draft).Error; err != nil {
		return nil, err
	}
	if draft.ID == "" {
		return nil, portrepo.ErrMembershipDraftNotFound
	}
	if draft.Revision != expectedRevision {
		return nil, portrepo.ErrMembershipRevisionConflict
	}
	updates := map[string]any{"revision": draft.Revision + 1, "name": version.Name, "description": version.Description, "price_cents": version.PriceCents, "currency": version.Currency, "billing_cycle": string(version.BillingCycle), "validity_count": version.ValidityCount, "validity_unit": membershipValidityValue(version.ValidityUnit), "updated_at": version.UpdatedAt}
	if err := db.Model(&models.MembershipPlanVersionModel{}).Where("id = ? AND organization_id = ? AND status = 'draft' AND revision = ?", draft.ID, stringID(organizationID), expectedRevision).Updates(updates).Error; err != nil {
		return nil, membershipWriteError(err)
	}
	if err := db.Where("plan_version_id = ? AND organization_id = ?", draft.ID, stringID(organizationID)).Delete(&models.MembershipPlanEntitlementModel{}).Error; err != nil {
		return nil, err
	}
	if err := createMembershipEntitlements(db, organizationID, domainID(draft.ID), entitlements); err != nil {
		return nil, err
	}
	var existingAddons []models.MembershipPlanVersionAddonModel
	if err := db.Where("plan_version_id = ? AND organization_id = ?", draft.ID, stringID(organizationID)).Find(&existingAddons).Error; err != nil {
		return nil, err
	}
	for _, a := range existingAddons {
		if err := db.Where("addon_id = ? AND organization_id = ?", a.ID, stringID(organizationID)).Delete(&models.MembershipPlanVersionAddonEntitlementModel{}).Error; err != nil {
			return nil, err
		}
	}
	if err := db.Where("plan_version_id = ? AND organization_id = ?", draft.ID, stringID(organizationID)).Delete(&models.MembershipPlanVersionAddonModel{}).Error; err != nil {
		return nil, err
	}
	if err := createMembershipAddons(db, organizationID, domainID(draft.ID), addons); err != nil {
		return nil, err
	}
	if draft.VersionNumber == 1 {
		_, nameKey, _ := domain.NormalizeMembershipPlanName(version.Name)
		if err := db.Model(&models.MembershipPlanModel{}).Where("id = ? AND organization_id = ?", planRow.ID, stringID(organizationID)).Updates(map[string]any{"name": version.Name, "name_key": nameKey, "updated_at": version.UpdatedAt}).Error; err != nil {
			return nil, membershipWriteError(err)
		}
	}
	return r.GetPlan(ctx, organizationID, unitID, planID)
}

func (r MembershipGormRepository) PublishDraft(ctx context.Context, organizationID, unitID, planID domain.ID, draftRevision int, publishedAt time.Time) (*domain.MembershipPlan, error) {
	db := databaseForContext(r.db, ctx)
	planRow, err := lockMembershipPlan(db, organizationID, unitID, planID)
	if err != nil {
		return nil, err
	}
	if planRow.Status == string(domain.MembershipPlanArchived) {
		return nil, portrepo.ErrMembershipPlanArchived
	}
	var draft models.MembershipPlanVersionModel
	if err := db.Raw("SELECT * FROM membership_plan_versions WHERE plan_id = ? AND organization_id = ? AND status = 'draft' FOR UPDATE", stringID(planID), stringID(organizationID)).Scan(&draft).Error; err != nil {
		return nil, err
	}
	if draft.ID == "" {
		return nil, portrepo.ErrMembershipDraftNotFound
	}
	if draft.Revision != draftRevision {
		return nil, portrepo.ErrMembershipRevisionConflict
	}
	if err := db.Model(&models.MembershipPlanVersionModel{}).Where("plan_id = ? AND organization_id = ? AND status = 'published'", stringID(planID), stringID(organizationID)).Update("status", string(domain.MembershipVersionRetired)).Error; err != nil {
		return nil, err
	}
	result := db.Model(&models.MembershipPlanVersionModel{}).Where("id = ? AND organization_id = ? AND status = 'draft' AND revision = ?", draft.ID, stringID(organizationID), draftRevision).Updates(map[string]any{"status": string(domain.MembershipVersionPublished), "published_at": publishedAt, "updated_at": publishedAt})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, portrepo.ErrMembershipRevisionConflict
	}
	_, nameKey, _ := domain.NormalizeMembershipPlanName(draft.Name)
	if err := db.Model(&models.MembershipPlanModel{}).Where("id = ? AND organization_id = ?", planRow.ID, stringID(organizationID)).Updates(map[string]any{"name": draft.Name, "name_key": nameKey, "updated_at": publishedAt}).Error; err != nil {
		return nil, membershipWriteError(err)
	}
	return r.GetPlan(ctx, organizationID, unitID, planID)
}

func (r MembershipGormRepository) CreateNewVersion(ctx context.Context, organizationID, unitID, planID domain.ID, createdAt time.Time) (*domain.MembershipPlan, error) {
	db := databaseForContext(r.db, ctx)
	if _, err := lockMembershipPlan(db, organizationID, unitID, planID); err != nil {
		return nil, err
	}
	var draftCount int64
	if err := db.Model(&models.MembershipPlanVersionModel{}).Where("plan_id = ? AND organization_id = ? AND status = 'draft'", stringID(planID), stringID(organizationID)).Count(&draftCount).Error; err != nil {
		return nil, err
	}
	if draftCount > 0 {
		return nil, portrepo.ErrMembershipDraftExists
	}
	var published models.MembershipPlanVersionModel
	if err := db.Where("plan_id = ? AND organization_id = ? AND status = 'published'", stringID(planID), stringID(organizationID)).Take(&published).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, portrepo.ErrMembershipAlreadyPublished
		}
		return nil, err
	}
	newVersionID, err := newID()
	if err != nil {
		return nil, err
	}
	clone := published
	clone.ID = stringID(newVersionID)
	clone.VersionNumber++
	clone.Revision = 1
	clone.Status = string(domain.MembershipVersionDraft)
	clone.PublishedAt = nil
	clone.CreatedAt = createdAt
	clone.UpdatedAt = createdAt
	if err := db.Create(&clone).Error; err != nil {
		return nil, membershipWriteError(err)
	}
	var previous []models.MembershipPlanEntitlementModel
	if err := db.Where("plan_version_id = ? AND organization_id = ?", published.ID, stringID(organizationID)).Order("position ASC").Find(&previous).Error; err != nil {
		return nil, err
	}
	for _, item := range previous {
		entitlementID, idErr := newID()
		if idErr != nil {
			return nil, idErr
		}
		item.ID = stringID(entitlementID)
		item.PlanVersionID = clone.ID
		item.CreatedAt = createdAt
		item.UpdatedAt = createdAt
		if err := db.Create(&item).Error; err != nil {
			return nil, err
		}
	}
	var previousAddons []models.MembershipPlanVersionAddonModel
	if err := db.Where("plan_version_id = ? AND organization_id = ?", published.ID, stringID(organizationID)).Order("position ASC").Find(&previousAddons).Error; err != nil {
		return nil, err
	}
	for _, item := range previousAddons {
		oldAddonID := item.ID
		newAddonID, idErr := newID()
		if idErr != nil {
			return nil, idErr
		}
		item.ID = stringID(newAddonID)
		item.PlanVersionID = clone.ID
		item.CreatedAt = createdAt
		item.UpdatedAt = createdAt
		if err := db.Create(&item).Error; err != nil {
			return nil, err
		}
		var previousAddonEntitlements []models.MembershipPlanVersionAddonEntitlementModel
		if err := db.Where("addon_id = ? AND organization_id = ?", oldAddonID, stringID(organizationID)).Order("position ASC").Find(&previousAddonEntitlements).Error; err != nil {
			return nil, err
		}
		for _, e := range previousAddonEntitlements {
			entID, eErr := newID()
			if eErr != nil {
				return nil, eErr
			}
			e.ID = stringID(entID)
			e.AddonID = item.ID
			e.CreatedAt = createdAt
			e.UpdatedAt = createdAt
			if err := db.Create(&e).Error; err != nil {
				return nil, err
			}
		}
	}
	return r.GetPlan(ctx, organizationID, unitID, planID)
}

func (r MembershipGormRepository) SetPlanStatus(ctx context.Context, organizationID, unitID, planID domain.ID, status domain.MembershipPlanStatus, updatedAt time.Time) (*domain.MembershipPlan, error) {
	db := databaseForContext(r.db, ctx)
	result := db.Model(&models.MembershipPlanModel{}).Where("id = ? AND organization_id = ? AND home_unit_id = ?", stringID(planID), stringID(organizationID), stringID(unitID)).Updates(map[string]any{"status": string(status), "updated_at": updatedAt})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, portrepo.ErrMembershipPlanNotFound
	}
	return r.GetPlan(ctx, organizationID, unitID, planID)
}

func (r MembershipGormRepository) CreateAuditEvent(ctx context.Context, event *domain.MembershipPlanAuditEvent) error {
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
	var versionID *string
	if event.PlanVersionID != "" {
		value := stringID(event.PlanVersionID)
		versionID = &value
	}
	row := models.MembershipPlanAuditEventModel{ID: stringID(event.ID), OrganizationID: stringID(event.OrganizationID), UnitID: stringID(event.UnitID), ActorUserID: stringID(event.ActorUserID), Action: event.Action, PlanID: stringID(event.PlanID), PlanVersionID: versionID, BeforeData: before, AfterData: after, CreatedAt: event.CreatedAt}
	return databaseForContext(r.db, ctx).Create(&row).Error
}

func loadMembershipPlan(db *gorm.DB, row models.MembershipPlanModel) (*domain.MembershipPlan, error) {
	plan := membershipPlanToDomain(row)
	var versions []models.MembershipPlanVersionModel
	if err := db.Where("plan_id = ? AND organization_id = ?", row.ID, row.OrganizationID).Order("version_number DESC").Find(&versions).Error; err != nil {
		return nil, err
	}
	for _, versionRow := range versions {
		version, err := loadMembershipVersion(db, versionRow, row.HomeUnitID)
		if err != nil {
			return nil, err
		}
		plan.Versions = append(plan.Versions, version)
		copyVersion := version
		if version.Status == domain.MembershipVersionPublished {
			plan.Published = &copyVersion
		}
		if version.Status == domain.MembershipVersionDraft {
			plan.Draft = &copyVersion
		}
	}
	return &plan, nil
}

func loadMembershipVersion(db *gorm.DB, row models.MembershipPlanVersionModel, homeUnitID string) (domain.MembershipPlanVersion, error) {
	version := membershipVersionToDomain(row)
	var entitlements []models.MembershipPlanEntitlementModel
	if err := db.Where("plan_version_id = ? AND organization_id = ?", row.ID, row.OrganizationID).Order("position ASC").Find(&entitlements).Error; err != nil {
		return version, err
	}
	for _, item := range entitlements {
		entitlement := membershipEntitlementToDomain(item)
		if item.ModalityID != nil {
			var modality struct {
				ID, Name, Status string
				ActiveInUnit     bool `gorm:"column:active_in_unit"`
			}
			if err := db.Table("modalities m").Select("m.id, m.name, m.status, COALESCE(um.active, false) AS active_in_unit").Joins("LEFT JOIN unit_modalities um ON um.modality_id = m.id AND um.organization_id = m.organization_id AND um.unit_id = ?", homeUnitID).Where("m.id = ? AND m.organization_id = ?", *item.ModalityID, row.OrganizationID).Take(&modality).Error; err == nil {
				entitlement.TargetName, entitlement.TargetType = modality.Name, "modality"
				if modality.Status != string(domain.CatalogStatusActive) || !modality.ActiveInUnit {
					version.Warnings = append(version.Warnings, modality.Name+" está indisponível na unidade")
				}
			}
		} else if item.ServiceOfferingID != nil {
			var service struct {
				ID, Name, Status string
				ActiveInUnit     bool `gorm:"column:active_in_unit"`
			}
			if err := db.Table("service_offerings so").Select("so.id, so.name, so.status, COALESCE(uso.active, false) AS active_in_unit").Joins("LEFT JOIN unit_service_offerings uso ON uso.service_offering_id = so.id AND uso.organization_id = so.organization_id AND uso.unit_id = ?", homeUnitID).Where("so.id = ? AND so.organization_id = ?", *item.ServiceOfferingID, row.OrganizationID).Take(&service).Error; err == nil {
				entitlement.TargetName, entitlement.TargetType = service.Name, "service_offering"
				if service.Status != string(domain.CatalogStatusActive) || !service.ActiveInUnit {
					version.Warnings = append(version.Warnings, service.Name+" está indisponível na unidade")
				}
			}
		}
		version.Entitlements = append(version.Entitlements, entitlement)
	}
	var addonModels []models.MembershipPlanVersionAddonModel
	if err := db.Where("plan_version_id = ? AND organization_id = ?", row.ID, row.OrganizationID).Order("position ASC").Find(&addonModels).Error; err != nil {
		return version, err
	}
	for _, addonRow := range addonModels {
		addon := membershipAddonToDomain(addonRow)
		var addonEntitlements []models.MembershipPlanVersionAddonEntitlementModel
		if err := db.Where("addon_id = ? AND organization_id = ?", addonRow.ID, row.OrganizationID).Order("position ASC").Find(&addonEntitlements).Error; err != nil {
			return version, err
		}
		for _, item := range addonEntitlements {
			entitlement := membershipAddonEntitlementToDomain(item)
			if item.ModalityID != nil {
				var modality struct {
					ID, Name, Status string
					ActiveInUnit     bool `gorm:"column:active_in_unit"`
				}
				if err := db.Table("modalities m").Select("m.id, m.name, m.status, COALESCE(um.active, false) AS active_in_unit").Joins("LEFT JOIN unit_modalities um ON um.modality_id = m.id AND um.organization_id = m.organization_id AND um.unit_id = ?", homeUnitID).Where("m.id = ? AND m.organization_id = ?", *item.ModalityID, row.OrganizationID).Take(&modality).Error; err == nil {
					entitlement.TargetName, entitlement.TargetType = modality.Name, "modality"
					if modality.Status != string(domain.CatalogStatusActive) || !modality.ActiveInUnit {
						addon.Warnings = append(addon.Warnings, modality.Name+" está indisponível na unidade")
					}
				}
			} else if item.ServiceOfferingID != nil {
				var service struct {
					ID, Name, Status string
					ActiveInUnit     bool `gorm:"column:active_in_unit"`
				}
				if err := db.Table("service_offerings so").Select("so.id, so.name, so.status, COALESCE(uso.active, false) AS active_in_unit").Joins("LEFT JOIN unit_service_offerings uso ON uso.service_offering_id = so.id AND uso.organization_id = so.organization_id AND uso.unit_id = ?", homeUnitID).Where("so.id = ? AND so.organization_id = ?", *item.ServiceOfferingID, row.OrganizationID).Take(&service).Error; err == nil {
					entitlement.TargetName, entitlement.TargetType = service.Name, "service_offering"
					if service.Status != string(domain.CatalogStatusActive) || !service.ActiveInUnit {
						addon.Warnings = append(addon.Warnings, service.Name+" está indisponível na unidade")
					}
				}
			}
			addon.Entitlements = append(addon.Entitlements, entitlement)
		}
		addon.Summary = domain.MembershipAddonSummary(addon)
		version.Addons = append(version.Addons, addon)
	}
	version.Summary = domain.MembershipPlanSummary(version)
	return version, nil
}

func lockMembershipPlan(db *gorm.DB, organizationID, unitID, planID domain.ID) (models.MembershipPlanModel, error) {
	var row models.MembershipPlanModel
	if err := db.Raw("SELECT * FROM membership_plans WHERE id = ? AND organization_id = ? AND home_unit_id = ? FOR UPDATE", stringID(planID), stringID(organizationID), stringID(unitID)).Scan(&row).Error; err != nil {
		return row, err
	}
	if row.ID == "" {
		return row, portrepo.ErrMembershipPlanNotFound
	}
	return row, nil
}

func createMembershipEntitlements(db *gorm.DB, organizationID, versionID domain.ID, entitlements []domain.MembershipPlanEntitlement) error {
	for index := range entitlements {
		item := entitlements[index]
		if err := ensureID(&item.ID); err != nil {
			return err
		}
		item.OrganizationID = organizationID
		item.PlanVersionID = versionID
		row := membershipEntitlementToModel(item)
		if err := db.Create(&row).Error; err != nil {
			return membershipWriteError(err)
		}
	}
	return nil
}

func createMembershipAddons(db *gorm.DB, organizationID, versionID domain.ID, addons []domain.MembershipPlanVersionAddon) error {
	for index := range addons {
		addon := &addons[index]
		if err := ensureID(&addon.ID); err != nil {
			return err
		}
		addon.OrganizationID = organizationID
		addon.PlanVersionID = versionID
		addonRow := membershipAddonToModel(*addon)
		if err := db.Create(&addonRow).Error; err != nil {
			return membershipWriteError(err)
		}
		for eIdx := range addon.Entitlements {
			item := &addon.Entitlements[eIdx]
			if err := ensureID(&item.ID); err != nil {
				return err
			}
			item.OrganizationID = organizationID
			item.AddonID = addon.ID
			itemRow := membershipAddonEntitlementToModel(*item)
			if err := db.Create(&itemRow).Error; err != nil {
				return membershipWriteError(err)
			}
		}
	}
	return nil
}

func membershipPlanToModel(value domain.MembershipPlan) models.MembershipPlanModel {
	return models.MembershipPlanModel{ID: stringID(value.ID), OrganizationID: stringID(value.OrganizationID), HomeUnitID: stringID(value.HomeUnitID), Name: value.Name, NameKey: value.NameKey, Status: string(value.Status), CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func membershipPlanToDomain(value models.MembershipPlanModel) domain.MembershipPlan {
	return domain.MembershipPlan{ID: domainID(value.ID), OrganizationID: domainID(value.OrganizationID), HomeUnitID: domainID(value.HomeUnitID), Name: value.Name, NameKey: value.NameKey, Status: domain.MembershipPlanStatus(value.Status), CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt, Versions: []domain.MembershipPlanVersion{}}
}

func membershipVersionToModel(value domain.MembershipPlanVersion) models.MembershipPlanVersionModel {
	return models.MembershipPlanVersionModel{ID: stringID(value.ID), OrganizationID: stringID(value.OrganizationID), PlanID: stringID(value.PlanID), VersionNumber: value.VersionNumber, Revision: value.Revision, Status: string(value.Status), Name: value.Name, Description: value.Description, PriceCents: value.PriceCents, Currency: value.Currency, BillingCycle: string(value.BillingCycle), ValidityCount: value.ValidityCount, ValidityUnit: membershipValidityValue(value.ValidityUnit), PublishedAt: value.PublishedAt, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func membershipVersionToDomain(value models.MembershipPlanVersionModel) domain.MembershipPlanVersion {
	var validity *domain.MembershipValidityUnit
	if value.ValidityUnit != nil {
		converted := domain.MembershipValidityUnit(*value.ValidityUnit)
		validity = &converted
	}
	return domain.MembershipPlanVersion{ID: domainID(value.ID), OrganizationID: domainID(value.OrganizationID), PlanID: domainID(value.PlanID), VersionNumber: value.VersionNumber, Revision: value.Revision, Status: domain.MembershipPlanVersionStatus(value.Status), Name: value.Name, Description: value.Description, PriceCents: value.PriceCents, Currency: value.Currency, BillingCycle: domain.MembershipBillingCycle(value.BillingCycle), ValidityCount: value.ValidityCount, ValidityUnit: validity, PublishedAt: value.PublishedAt, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt, Entitlements: []domain.MembershipPlanEntitlement{}, Addons: []domain.MembershipPlanVersionAddon{}, Warnings: []string{}}
}

func membershipEntitlementToModel(value domain.MembershipPlanEntitlement) models.MembershipPlanEntitlementModel {
	return models.MembershipPlanEntitlementModel{ID: stringID(value.ID), OrganizationID: stringID(value.OrganizationID), PlanVersionID: stringID(value.PlanVersionID), ModalityID: optionalIDString(value.ModalityID), ServiceOfferingID: optionalIDString(value.ServiceOfferingID), AccessType: string(value.AccessType), Quantity: value.Quantity, PeriodUnit: membershipPeriodValue(value.PeriodUnit), DailyLimit: value.DailyLimit, DeliveryMode: membershipDeliveryValue(value.DeliveryMode), BookingRequired: value.BookingRequired, Position: value.Position, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func membershipEntitlementToDomain(value models.MembershipPlanEntitlementModel) domain.MembershipPlanEntitlement {
	var period *domain.MembershipPeriodUnit
	if value.PeriodUnit != nil {
		converted := domain.MembershipPeriodUnit(*value.PeriodUnit)
		period = &converted
	}
	var delivery *domain.MembershipDeliveryMode
	if value.DeliveryMode != nil {
		converted := domain.MembershipDeliveryMode(*value.DeliveryMode)
		delivery = &converted
	}
	return domain.MembershipPlanEntitlement{ID: domainID(value.ID), OrganizationID: domainID(value.OrganizationID), PlanVersionID: domainID(value.PlanVersionID), ModalityID: optionalDomainID(value.ModalityID), ServiceOfferingID: optionalDomainID(value.ServiceOfferingID), AccessType: domain.MembershipAccessType(value.AccessType), Quantity: value.Quantity, PeriodUnit: period, DailyLimit: value.DailyLimit, DeliveryMode: delivery, BookingRequired: value.BookingRequired, Position: value.Position, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func membershipAddonToModel(value domain.MembershipPlanVersionAddon) models.MembershipPlanVersionAddonModel {
	return models.MembershipPlanVersionAddonModel{
		ID:              stringID(value.ID),
		OrganizationID:  stringID(value.OrganizationID),
		PlanVersionID:   stringID(value.PlanVersionID),
		Name:            value.Name,
		Description:     value.Description,
		PriceDeltaCents: value.PriceDeltaCents,
		Position:        value.Position,
		CreatedAt:       value.CreatedAt,
		UpdatedAt:       value.UpdatedAt,
	}
}

func membershipAddonToDomain(value models.MembershipPlanVersionAddonModel) domain.MembershipPlanVersionAddon {
	return domain.MembershipPlanVersionAddon{
		ID:              domainID(value.ID),
		OrganizationID:  domainID(value.OrganizationID),
		PlanVersionID:   domainID(value.PlanVersionID),
		Name:            value.Name,
		Description:     value.Description,
		PriceDeltaCents: value.PriceDeltaCents,
		Position:        value.Position,
		CreatedAt:       value.CreatedAt,
		UpdatedAt:       value.UpdatedAt,
		Entitlements:    []domain.MembershipPlanAddonEntitlement{},
		Warnings:        []string{},
	}
}

func membershipAddonEntitlementToModel(value domain.MembershipPlanAddonEntitlement) models.MembershipPlanVersionAddonEntitlementModel {
	return models.MembershipPlanVersionAddonEntitlementModel{
		ID:                stringID(value.ID),
		OrganizationID:    stringID(value.OrganizationID),
		AddonID:           stringID(value.AddonID),
		ModalityID:        optionalIDString(value.ModalityID),
		ServiceOfferingID: optionalIDString(value.ServiceOfferingID),
		AccessType:        string(value.AccessType),
		Quantity:          value.Quantity,
		PeriodUnit:        membershipPeriodValue(value.PeriodUnit),
		DailyLimit:        value.DailyLimit,
		DeliveryMode:      membershipDeliveryValue(value.DeliveryMode),
		BookingRequired:   value.BookingRequired,
		Position:          value.Position,
		CreatedAt:         value.CreatedAt,
		UpdatedAt:         value.UpdatedAt,
	}
}

func membershipAddonEntitlementToDomain(value models.MembershipPlanVersionAddonEntitlementModel) domain.MembershipPlanAddonEntitlement {
	var period *domain.MembershipPeriodUnit
	if value.PeriodUnit != nil {
		converted := domain.MembershipPeriodUnit(*value.PeriodUnit)
		period = &converted
	}
	var delivery *domain.MembershipDeliveryMode
	if value.DeliveryMode != nil {
		converted := domain.MembershipDeliveryMode(*value.DeliveryMode)
		delivery = &converted
	}
	return domain.MembershipPlanAddonEntitlement{
		ID:                domainID(value.ID),
		OrganizationID:    domainID(value.OrganizationID),
		AddonID:           domainID(value.AddonID),
		ModalityID:        optionalDomainID(value.ModalityID),
		ServiceOfferingID: optionalDomainID(value.ServiceOfferingID),
		AccessType:        domain.MembershipAccessType(value.AccessType),
		Quantity:          value.Quantity,
		PeriodUnit:        period,
		DailyLimit:        value.DailyLimit,
		DeliveryMode:      delivery,
		BookingRequired:   value.BookingRequired,
		Position:          value.Position,
		CreatedAt:         value.CreatedAt,
		UpdatedAt:         value.UpdatedAt,
	}
}

func membershipWriteError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	if pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "membership_plans_organization_name_key_key":
			return portrepo.ErrMembershipPlanNameConflict
		case "idx_membership_plan_versions_one_draft":
			return portrepo.ErrMembershipDraftExists
		case "membership_plan_command_requests_scope_key":
			return portrepo.ErrMembershipIdempotencyReused
		case "membership_plan_version_addons_version_name_key":
			return portrepo.ErrMembershipAddonDuplicate
		case "membership_addon_entitlements_target_unique":
			return portrepo.ErrMembershipAddonDuplicate
		}
	}
	return err
}

func optionalIDString(value domain.ID) *string {
	if value == "" {
		return nil
	}
	converted := stringID(value)
	return &converted
}
func optionalDomainID(value *string) domain.ID {
	if value == nil {
		return ""
	}
	return domainID(*value)
}
func membershipValidityValue(value *domain.MembershipValidityUnit) *string {
	if value == nil {
		return nil
	}
	converted := string(*value)
	return &converted
}
func membershipPeriodValue(value *domain.MembershipPeriodUnit) *string {
	if value == nil {
		return nil
	}
	converted := string(*value)
	return &converted
}
func membershipDeliveryValue(value *domain.MembershipDeliveryMode) *string {
	if value == nil {
		return nil
	}
	converted := string(*value)
	return &converted
}
