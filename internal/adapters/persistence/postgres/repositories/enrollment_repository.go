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

func (r EnrollmentGormRepository) LockCommand(ctx context.Context, organizationID domain.ID, command, idempotencyKey string) error {
	key := fmt.Sprintf("enrollment-command:%s:%s:%s", organizationID, command, idempotencyKey)
	return databaseForContext(r.db, ctx).Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", key).Error
}

func (r EnrollmentGormRepository) FindCommand(ctx context.Context, organizationID domain.ID, command, idempotencyKey string) (*domain.EnrollmentCommandRequest, error) {
	var row models.EnrollmentCommandRequestModel
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
	return &domain.EnrollmentCommandRequest{ID: domainID(row.ID), OrganizationID: domainID(row.OrganizationID), Command: row.Command, IdempotencyKey: row.IdempotencyKey, Fingerprint: row.RequestFingerprint, EnrollmentID: domainID(row.EnrollmentID), ResponseData: response, CreatedAt: row.CreatedAt}, nil
}

func (r EnrollmentGormRepository) CreateCommand(ctx context.Context, request *domain.EnrollmentCommandRequest) error {
	if err := ensureID(&request.ID); err != nil {
		return err
	}
	response, err := json.Marshal(nonNilAuditData(request.ResponseData))
	if err != nil {
		return err
	}
	row := models.EnrollmentCommandRequestModel{ID: stringID(request.ID), OrganizationID: stringID(request.OrganizationID), Command: request.Command, IdempotencyKey: request.IdempotencyKey, RequestFingerprint: request.Fingerprint, EnrollmentID: stringID(request.EnrollmentID), ResponseData: response, CreatedAt: request.CreatedAt}
	return enrollmentWriteError(databaseForContext(r.db, ctx).Create(&row).Error)
}

func (r EnrollmentGormRepository) LockSubject(ctx context.Context, organizationID, personID, unitID domain.ID) error {
	key := fmt.Sprintf("enrollment-subject:%s:%s:%s", organizationID, personID, unitID)
	return databaseForContext(r.db, ctx).Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", key).Error
}

func (r EnrollmentGormRepository) ListForPerson(ctx context.Context, organizationID, unitID, personID domain.ID) (*domain.EnrollmentHistory, error) {
	db := databaseForContext(r.db, ctx)
	var rows []models.EnrollmentModel
	if err := db.Where("organization_id = ? AND home_unit_id = ? AND person_id = ?", stringID(organizationID), stringID(unitID), stringID(personID)).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := &domain.EnrollmentHistory{History: []domain.Enrollment{}}
	for _, row := range rows {
		enrollment, err := loadEnrollment(db, row)
		if err != nil {
			return nil, err
		}
		if enrollment.LifecycleStatus == domain.EnrollmentLifecycleActive && result.Current == nil {
			copy := enrollment
			result.Current = &copy
		} else {
			result.History = append(result.History, enrollment)
		}
	}
	return result, nil
}

func (r EnrollmentGormRepository) GetByID(ctx context.Context, organizationID, unitID, enrollmentID domain.ID) (*domain.Enrollment, error) {
	db := databaseForContext(r.db, ctx)
	var row models.EnrollmentModel
	if err := db.Where("id = ? AND organization_id = ? AND home_unit_id = ?", stringID(enrollmentID), stringID(organizationID), stringID(unitID)).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, portrepo.ErrEnrollmentNotFound
		}
		return nil, err
	}
	item, err := loadEnrollment(db, row)
	return &item, err
}

func (r EnrollmentGormRepository) GetOpenForUpdate(ctx context.Context, organizationID, unitID, personID domain.ID) (*domain.Enrollment, error) {
	db := databaseForContext(r.db, ctx)
	var row models.EnrollmentModel
	if err := db.Raw("SELECT * FROM enrollments WHERE organization_id = ? AND home_unit_id = ? AND person_id = ? AND lifecycle_status = 'active' FOR UPDATE", stringID(organizationID), stringID(unitID), stringID(personID)).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == "" {
		return nil, nil
	}
	item, err := loadEnrollment(db, row)
	return &item, err
}

func (r EnrollmentGormRepository) GetPlanVersionForEnrollment(ctx context.Context, organizationID, unitID, planID, versionID domain.ID) (*domain.MembershipPlan, *domain.MembershipPlanVersion, error) {
	db := databaseForContext(r.db, ctx)
	var planRow models.MembershipPlanModel
	if err := db.Raw("SELECT * FROM membership_plans WHERE id = ? AND organization_id = ? AND home_unit_id = ? FOR SHARE", stringID(planID), stringID(organizationID), stringID(unitID)).Scan(&planRow).Error; err != nil {
		return nil, nil, err
	}
	if planRow.ID == "" || planRow.Status != string(domain.MembershipPlanActive) {
		return nil, nil, portrepo.ErrMembershipPlanNotFound
	}
	var versionRow models.MembershipPlanVersionModel
	if err := db.Raw("SELECT * FROM membership_plan_versions WHERE id = ? AND plan_id = ? AND organization_id = ? FOR SHARE", stringID(versionID), stringID(planID), stringID(organizationID)).Scan(&versionRow).Error; err != nil {
		return nil, nil, err
	}
	if versionRow.ID == "" || versionRow.Status != string(domain.MembershipVersionPublished) {
		return nil, nil, portrepo.ErrPlanVersionChanged
	}
	version, err := loadMembershipVersion(db, versionRow, planRow.HomeUnitID)
	if err != nil {
		return nil, nil, err
	}
	plan := membershipPlanToDomain(planRow)
	return &plan, &version, nil
}

func (r EnrollmentGormRepository) CreateEnrollment(ctx context.Context, enrollment *domain.Enrollment) error {
	if err := ensureID(&enrollment.ID); err != nil {
		return err
	}
	row := enrollmentToModel(*enrollment)
	if err := databaseForContext(r.db, ctx).Create(&row).Error; err != nil {
		return enrollmentWriteError(err)
	}
	return nil
}

func (r EnrollmentGormRepository) EndEnrollment(ctx context.Context, enrollmentID, organizationID, unitID domain.ID, expectedRevision int, endedOn, endedAt time.Time, reason string, actorID domain.ID) (*domain.Enrollment, error) {
	db := databaseForContext(r.db, ctx)
	result := db.Model(&models.EnrollmentModel{}).
		Where("id = ? AND organization_id = ? AND home_unit_id = ? AND lifecycle_status = 'active' AND revision = ?", stringID(enrollmentID), stringID(organizationID), stringID(unitID), expectedRevision).
		Updates(map[string]any{"lifecycle_status": string(domain.EnrollmentLifecycleEnded), "ended_on": dateOnly(endedOn), "ended_at": endedAt, "ended_reason": reason, "ended_by_user_id": stringID(actorID), "revision": expectedRevision + 1, "updated_at": endedAt})
	if result.Error != nil {
		return nil, enrollmentWriteError(result.Error)
	}
	if result.RowsAffected == 0 {
		var row models.EnrollmentModel
		err := db.Where("id = ? AND organization_id = ? AND home_unit_id = ?", stringID(enrollmentID), stringID(organizationID), stringID(unitID)).Take(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, portrepo.ErrEnrollmentNotFound
		}
		if err != nil {
			return nil, err
		}
		if row.LifecycleStatus == string(domain.EnrollmentLifecycleEnded) {
			return nil, portrepo.ErrEnrollmentAlreadyEnded
		}
		return nil, portrepo.ErrEnrollmentRevisionConflict
	}
	return r.GetByID(ctx, organizationID, unitID, enrollmentID)
}

func (r EnrollmentGormRepository) CreateEvent(ctx context.Context, event *domain.EnrollmentEvent) error {
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
	row := models.EnrollmentEventModel{ID: stringID(event.ID), OrganizationID: stringID(event.OrganizationID), UnitID: stringID(event.UnitID), PersonID: stringID(event.PersonID), EnrollmentID: stringID(event.EnrollmentID), ActorUserID: stringID(event.ActorUserID), Action: event.Action, BeforeData: before, AfterData: after, Reason: enrollmentNullableString(event.Reason), CreatedAt: event.CreatedAt}
	return databaseForContext(r.db, ctx).Create(&row).Error
}

func (r EnrollmentGormRepository) ListPrivacy(ctx context.Context, unitID, personID domain.ID) ([]domain.EnrollmentPrivacyExport, error) {
	db := databaseForContext(r.db, ctx)
	var organizationID string
	if err := db.Table("boxes").Select("organization_id").Where("id = ?", stringID(unitID)).Scan(&organizationID).Error; err != nil || organizationID == "" {
		if err != nil {
			return nil, err
		}
		return nil, nil
	}
	history, err := r.ListForPerson(ctx, domainID(organizationID), unitID, personID)
	if err != nil {
		return nil, err
	}
	items := make([]domain.Enrollment, 0, len(history.History)+1)
	if history.Current != nil {
		items = append(items, *history.Current)
	}
	items = append(items, history.History...)
	result := make([]domain.EnrollmentPrivacyExport, 0, len(items))
	for _, item := range items {
		var rows []models.EnrollmentEventModel
		if err := db.Where("organization_id = ? AND unit_id = ? AND enrollment_id = ?", organizationID, stringID(unitID), stringID(item.ID)).Order("created_at ASC").Find(&rows).Error; err != nil {
			return nil, err
		}
		events := make([]domain.EnrollmentEvent, 0, len(rows))
		for _, row := range rows {
			events = append(events, enrollmentEventToDomain(row))
		}
		result = append(result, domain.EnrollmentPrivacyExport{Enrollment: item, Events: events})
	}
	return result, nil
}

func loadEnrollment(db *gorm.DB, row models.EnrollmentModel) (domain.Enrollment, error) {
	item := enrollmentToDomain(row)
	var version models.MembershipPlanVersionModel
	if err := db.Where("id = ? AND organization_id = ?", row.PlanVersionID, row.OrganizationID).Take(&version).Error; err != nil {
		return item, err
	}
	var plan models.MembershipPlanModel
	if err := db.Where("id = ? AND organization_id = ?", version.PlanID, row.OrganizationID).Take(&plan).Error; err != nil {
		return item, err
	}
	loadedVersion, err := loadMembershipVersion(db, version, row.HomeUnitID)
	if err != nil {
		return item, err
	}
	item.Terms = domain.EnrollmentTerms{PlanID: domainID(plan.ID), PlanVersionID: domainID(version.ID), VersionNumber: version.VersionNumber, Name: version.Name, Description: version.Description, PriceCents: version.PriceCents, Currency: version.Currency, BillingCycle: domain.MembershipBillingCycle(version.BillingCycle), Summary: loadedVersion.Summary}
	return item, nil
}

func enrollmentToModel(value domain.Enrollment) models.EnrollmentModel {
	return models.EnrollmentModel{ID: stringID(value.ID), OrganizationID: stringID(value.OrganizationID), PersonID: stringID(value.PersonID), HomeUnitID: stringID(value.HomeUnitID), PlanVersionID: stringID(value.PlanVersionID), LifecycleStatus: string(value.LifecycleStatus), StartsOn: dateOnly(value.StartsOn), ValidUntil: datePointer(value.ValidUntil), EndedOn: datePointer(value.EndedOn), EndedAt: value.EndedAt, EndedReason: enrollmentNullableString(value.EndedReason), Revision: value.Revision, CreatedByUserID: stringID(value.CreatedByUserID), EndedByUserID: optionalIDString(value.EndedByUserID), CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func enrollmentToDomain(value models.EnrollmentModel) domain.Enrollment {
	return domain.Enrollment{ID: domainID(value.ID), OrganizationID: domainID(value.OrganizationID), PersonID: domainID(value.PersonID), HomeUnitID: domainID(value.HomeUnitID), PlanVersionID: domainID(value.PlanVersionID), LifecycleStatus: domain.EnrollmentLifecycleStatus(value.LifecycleStatus), StartsOn: dateOnly(value.StartsOn), ValidUntil: datePointer(value.ValidUntil), EndedOn: datePointer(value.EndedOn), EndedAt: value.EndedAt, EndedReason: optionalString(value.EndedReason), Revision: value.Revision, CreatedByUserID: domainID(value.CreatedByUserID), EndedByUserID: optionalDomainID(value.EndedByUserID), CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func enrollmentEventToDomain(row models.EnrollmentEventModel) domain.EnrollmentEvent {
	before, after := map[string]any{}, map[string]any{}
	_ = json.Unmarshal(row.BeforeData, &before)
	_ = json.Unmarshal(row.AfterData, &after)
	return domain.EnrollmentEvent{ID: domainID(row.ID), OrganizationID: domainID(row.OrganizationID), UnitID: domainID(row.UnitID), PersonID: domainID(row.PersonID), EnrollmentID: domainID(row.EnrollmentID), ActorUserID: domainID(row.ActorUserID), Action: row.Action, BeforeData: before, AfterData: after, Reason: optionalString(row.Reason), CreatedAt: row.CreatedAt}
}

func enrollmentWriteError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	if pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "idx_enrollments_one_open_per_person_unit":
			return portrepo.ErrActiveEnrollmentExists
		case "enrollment_command_requests_scope_key":
			return portrepo.ErrEnrollmentIdempotencyReused
		}
	}
	return err
}

func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func datePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	converted := dateOnly(*value)
	return &converted
}

func enrollmentNullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
