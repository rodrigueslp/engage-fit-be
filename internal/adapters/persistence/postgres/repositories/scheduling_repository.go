package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type SchedulingGormRepository struct {
	db *gorm.DB
}

func NewSchedulingGormRepository(db *gorm.DB) SchedulingGormRepository {
	return SchedulingGormRepository{db: db}
}

func (r SchedulingGormRepository) LockCommand(ctx context.Context, organizationID domain.ID, command, idempotencyKey string) error {
	key := fmt.Sprintf("scheduling-cmd:%s:%s:%s", organizationID, command, idempotencyKey)
	return databaseForContext(r.db, ctx).Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", key).Error
}

func (r SchedulingGormRepository) FindCommand(ctx context.Context, organizationID domain.ID, command, idempotencyKey string) (*domain.SchedulingCommandRequest, error) {
	var row models.SchedulingCommandRequestModel
	err := databaseForContext(r.db, ctx).Where("organization_id = ? AND command = ? AND idempotency_key = ?", stringID(organizationID), command, idempotencyKey).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	response := map[string]any{}
	if len(row.ResponseData) > 0 {
		if err := json.Unmarshal(row.ResponseData, &response); err != nil {
			return nil, err
		}
	}
	return &domain.SchedulingCommandRequest{
		ID:             domainID(row.ID),
		OrganizationID: domainID(row.OrganizationID),
		UnitID:         domainID(row.UnitID),
		Command:        row.Command,
		IdempotencyKey: row.IdempotencyKey,
		Fingerprint:    row.RequestFingerprint,
		ProfessionalID: domainID(row.ProfessionalID),
		ResponseData:   response,
		CreatedAt:      row.CreatedAt,
	}, nil
}

func (r SchedulingGormRepository) CreateCommand(ctx context.Context, request *domain.SchedulingCommandRequest) error {
	if err := ensureID(&request.ID); err != nil {
		return err
	}
	responseData, err := json.Marshal(nonNilAuditData(request.ResponseData))
	if err != nil {
		return err
	}
	row := models.SchedulingCommandRequestModel{
		ID:                 stringID(request.ID),
		OrganizationID:     stringID(request.OrganizationID),
		UnitID:             stringID(request.UnitID),
		Command:            request.Command,
		IdempotencyKey:     request.IdempotencyKey,
		RequestFingerprint: request.Fingerprint,
		ProfessionalID:     stringID(request.ProfessionalID),
		ResponseData:       responseData,
		CreatedAt:          request.CreatedAt,
	}
	return schedulingWriteError(databaseForContext(r.db, ctx).Create(&row).Error)
}

func (r SchedulingGormRepository) LockProfessional(ctx context.Context, organizationID, professionalID domain.ID) error {
	key := fmt.Sprintf("scheduling-prof:%s:%s", organizationID, professionalID)
	return databaseForContext(r.db, ctx).Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", key).Error
}

func (r SchedulingGormRepository) LockAccount(ctx context.Context, organizationID, userID domain.ID) error {
	key := fmt.Sprintf("scheduling-account:%s:%s", organizationID, userID)
	return databaseForContext(r.db, ctx).Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", key).Error
}

type professionalWithUnitModel struct {
	models.SchedulingProfessionalModel
	ActiveInUnit bool `gorm:"column:active_in_unit"`
}

func (r SchedulingGormRepository) ListProfessionals(ctx context.Context, organizationID, unitID domain.ID, includeArchived bool, query string) ([]domain.Professional, error) {
	db := databaseForContext(r.db, ctx)
	q := db.Table("scheduling_professionals p").
		Select("p.*, COALESCE(u.active, false) AS active_in_unit").
		Joins("JOIN scheduling_professional_units u ON u.professional_id = p.id AND u.unit_id = ? AND u.organization_id = p.organization_id", stringID(unitID)).
		Where("p.organization_id = ?", stringID(organizationID))

	if !includeArchived {
		q = q.Where("p.status = ?", string(domain.ProfessionalStatusActive))
	}
	if query != "" {
		q = q.Where("p.display_name_key LIKE ?", "%"+strings.ToLower(query)+"%")
	}

	var rows []professionalWithUnitModel
	if err := q.Order("p.display_name_key ASC, p.id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]domain.Professional, 0, len(rows))
	for _, row := range rows {
		result = append(result, professionalToDomain(row.SchedulingProfessionalModel, row.ActiveInUnit))
	}
	return result, nil
}

func (r SchedulingGormRepository) GetByID(ctx context.Context, organizationID, unitID, professionalID domain.ID) (*domain.Professional, error) {
	db := databaseForContext(r.db, ctx)
	var row professionalWithUnitModel
	err := db.Table("scheduling_professionals p").
		Select("p.*, COALESCE(u.active, false) AS active_in_unit").
		Joins("JOIN scheduling_professional_units u ON u.professional_id = p.id AND u.unit_id = ? AND u.organization_id = p.organization_id", stringID(unitID)).
		Where("p.id = ? AND p.organization_id = ?", stringID(professionalID), stringID(organizationID)).
		Take(&row).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, portrepo.ErrSchedulingProfessionalNotFound
	}
	if err != nil {
		return nil, err
	}
	prof := professionalToDomain(row.SchedulingProfessionalModel, row.ActiveInUnit)
	return &prof, nil
}

func (r SchedulingGormRepository) GetByIDForUpdate(ctx context.Context, organizationID, professionalID domain.ID) (*domain.Professional, error) {
	db := databaseForContext(r.db, ctx)
	var row models.SchedulingProfessionalModel
	err := db.Raw("SELECT * FROM scheduling_professionals WHERE id = ? AND organization_id = ? FOR UPDATE", stringID(professionalID), stringID(organizationID)).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == "" {
		return nil, portrepo.ErrSchedulingProfessionalNotFound
	}
	var unitRow models.SchedulingProfessionalUnitModel
	_ = db.Where("professional_id = ? AND organization_id = ?", row.ID, row.OrganizationID).Take(&unitRow).Error
	prof := professionalToDomain(row, unitRow.Active)
	return &prof, nil
}

func (r SchedulingGormRepository) FindByLinkedUser(ctx context.Context, organizationID, userID domain.ID) (*domain.Professional, error) {
	db := databaseForContext(r.db, ctx)
	var row models.SchedulingProfessionalModel
	err := db.Where("organization_id = ? AND linked_user_id = ?", stringID(organizationID), stringID(userID)).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var unitRow models.SchedulingProfessionalUnitModel
	_ = db.Where("professional_id = ? AND organization_id = ?", row.ID, row.OrganizationID).Take(&unitRow).Error
	prof := professionalToDomain(row, unitRow.Active)
	return &prof, nil
}

func (r SchedulingGormRepository) CreateProfessional(ctx context.Context, professional *domain.Professional, unitID domain.ID) error {
	if err := ensureID(&professional.ID); err != nil {
		return err
	}
	db := databaseForContext(r.db, ctx)
	profRow := professionalToModel(*professional)
	if err := db.Create(&profRow).Error; err != nil {
		return schedulingWriteError(err)
	}
	unitRow := models.SchedulingProfessionalUnitModel{
		OrganizationID: stringID(professional.OrganizationID),
		UnitID:         stringID(unitID),
		ProfessionalID: stringID(professional.ID),
		Active:         professional.ActiveInUnit,
		CreatedAt:      professional.CreatedAt,
		UpdatedAt:      professional.UpdatedAt,
	}
	if err := db.Create(&unitRow).Error; err != nil {
		return schedulingWriteError(err)
	}
	return nil
}

func (r SchedulingGormRepository) UpdateProfessional(ctx context.Context, professional *domain.Professional) error {
	db := databaseForContext(r.db, ctx)
	profRow := professionalToModel(*professional)
	if err := db.Save(&profRow).Error; err != nil {
		return schedulingWriteError(err)
	}
	return nil
}

func (r SchedulingGormRepository) SetUnitActive(ctx context.Context, organizationID, unitID, professionalID domain.ID, active bool) error {
	db := databaseForContext(r.db, ctx)
	return db.Model(&models.SchedulingProfessionalUnitModel{}).
		Where("organization_id = ? AND unit_id = ? AND professional_id = ?", stringID(organizationID), stringID(unitID), stringID(professionalID)).
		Updates(map[string]any{"active": active, "updated_at": time.Now().UTC()}).Error
}

func (r SchedulingGormRepository) CreateAuditEvent(ctx context.Context, event *domain.SchedulingAuditEvent) error {
	if err := ensureID(&event.ID); err != nil {
		return err
	}
	changedFields, err := json.Marshal(event.ChangedFields)
	if err != nil {
		return err
	}
	beforeData, err := json.Marshal(nonNilAuditData(event.BeforeData))
	if err != nil {
		return err
	}
	afterData, err := json.Marshal(nonNilAuditData(event.AfterData))
	if err != nil {
		return err
	}
	var reason *string
	if event.Reason != "" {
		r := event.Reason
		reason = &r
	}
	row := models.SchedulingAuditEventModel{
		ID:             stringID(event.ID),
		OrganizationID: stringID(event.OrganizationID),
		UnitID:         stringID(event.UnitID),
		ActorUserID:    stringID(event.ActorUserID),
		Action:         event.Action,
		EntityType:     event.EntityType,
		EntityID:       stringID(event.EntityID),
		Revision:       event.Revision,
		ChangedFields:  changedFields,
		BeforeData:     beforeData,
		AfterData:      afterData,
		Reason:         reason,
		CreatedAt:      event.CreatedAt,
	}
	return schedulingWriteError(databaseForContext(r.db, ctx).Create(&row).Error)
}

func (r SchedulingGormRepository) ListAuditEventsForProfessional(ctx context.Context, organizationID, unitID, professionalID domain.ID) ([]domain.SchedulingAuditEvent, error) {
	db := databaseForContext(r.db, ctx)
	var rows []models.SchedulingAuditEventModel
	if err := db.Where("organization_id = ? AND entity_id = ?", stringID(organizationID), stringID(professionalID)).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.SchedulingAuditEvent, 0, len(rows))
	for _, row := range rows {
		var changedFields []string
		if len(row.ChangedFields) > 0 {
			_ = json.Unmarshal(row.ChangedFields, &changedFields)
		}
		var beforeData, afterData map[string]any
		if len(row.BeforeData) > 0 {
			_ = json.Unmarshal(row.BeforeData, &beforeData)
		}
		if len(row.AfterData) > 0 {
			_ = json.Unmarshal(row.AfterData, &afterData)
		}
		reason := ""
		if row.Reason != nil {
			reason = *row.Reason
		}
		result = append(result, domain.SchedulingAuditEvent{
			ID:             domainID(row.ID),
			OrganizationID: domainID(row.OrganizationID),
			UnitID:         domainID(row.UnitID),
			ActorUserID:    domainID(row.ActorUserID),
			Action:         row.Action,
			EntityType:     row.EntityType,
			EntityID:       domainID(row.EntityID),
			Revision:       row.Revision,
			ChangedFields:  changedFields,
			BeforeData:     beforeData,
			AfterData:      afterData,
			Reason:         reason,
			CreatedAt:      row.CreatedAt,
		})
	}
	return result, nil
}

func (r SchedulingGormRepository) CountActiveInUnit(ctx context.Context, organizationID, unitID domain.ID) (int64, error) {
	db := databaseForContext(r.db, ctx)
	var count int64
	err := db.Table("scheduling_professionals p").
		Joins("JOIN scheduling_professional_units u ON u.professional_id = p.id AND u.unit_id = ? AND u.organization_id = p.organization_id", stringID(unitID)).
		Where("p.organization_id = ? AND p.status = ? AND u.active = true", stringID(organizationID), string(domain.ProfessionalStatusActive)).
		Count(&count).Error
	return count, err
}

func professionalToDomain(model models.SchedulingProfessionalModel, activeInUnit bool) domain.Professional {
	var linkedUserID *domain.ID
	if model.LinkedUserID != nil && *model.LinkedUserID != "" {
		id := domainID(*model.LinkedUserID)
		linkedUserID = &id
	}
	var archivedByUserID *domain.ID
	if model.ArchivedByUserID != nil && *model.ArchivedByUserID != "" {
		id := domainID(*model.ArchivedByUserID)
		archivedByUserID = &id
	}
	return domain.Professional{
		ID:               domainID(model.ID),
		OrganizationID:   domainID(model.OrganizationID),
		DisplayName:      model.DisplayName,
		DisplayNameKey:   model.DisplayNameKey,
		LinkedUserID:     linkedUserID,
		Status:           domain.ProfessionalStatus(model.Status),
		Revision:         model.Revision,
		AnonymizedAt:     model.AnonymizedAt,
		CreatedByUserID:  domainID(model.CreatedByUserID),
		UpdatedByUserID:  domainID(model.UpdatedByUserID),
		ArchivedAt:       model.ArchivedAt,
		ArchivedByUserID: archivedByUserID,
		ActiveInUnit:     activeInUnit,
		CreatedAt:        model.CreatedAt,
		UpdatedAt:        model.UpdatedAt,
	}
}

func professionalToModel(prof domain.Professional) models.SchedulingProfessionalModel {
	var linkedUserID *string
	if prof.LinkedUserID != nil && *prof.LinkedUserID != "" {
		s := stringID(*prof.LinkedUserID)
		linkedUserID = &s
	}
	var archivedByUserID *string
	if prof.ArchivedByUserID != nil && *prof.ArchivedByUserID != "" {
		s := stringID(*prof.ArchivedByUserID)
		archivedByUserID = &s
	}
	return models.SchedulingProfessionalModel{
		ID:               stringID(prof.ID),
		OrganizationID:   stringID(prof.OrganizationID),
		DisplayName:      prof.DisplayName,
		DisplayNameKey:   prof.DisplayNameKey,
		LinkedUserID:     linkedUserID,
		Status:           string(prof.Status),
		Revision:         prof.Revision,
		AnonymizedAt:     prof.AnonymizedAt,
		CreatedByUserID:  stringID(prof.CreatedByUserID),
		UpdatedByUserID:  stringID(prof.UpdatedByUserID),
		ArchivedAt:       prof.ArchivedAt,
		ArchivedByUserID: archivedByUserID,
		CreatedAt:        prof.CreatedAt,
		UpdatedAt:        prof.UpdatedAt,
	}
}

func schedulingWriteError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			if pgErr.ConstraintName == "idx_scheduling_professionals_org_linked_user" ||
				pgErr.ConstraintName == "scheduling_professionals_org_linked_user_key" {
				return portrepo.ErrSchedulingAccountConflict
			}
			if pgErr.ConstraintName == "scheduling_command_requests_scope_key" {
				return portrepo.ErrSchedulingIdempotencyKeyReused
			}
			return portrepo.ErrSchedulingProfessionalInvalid
		case "23503":
			return portrepo.ErrSchedulingUnitNotFound
		case "23514":
			return portrepo.ErrSchedulingProfessionalInvalid
		}
	}
	return err
}
