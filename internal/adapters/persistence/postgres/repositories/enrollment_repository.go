package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/app/memberships"
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
	db := databaseForContext(r.db, ctx)
	row := enrollmentToModel(*enrollment)
	if err := db.Create(&row).Error; err != nil {
		return enrollmentWriteError(err)
	}
	for index := range enrollment.Addons {
		addon := &enrollment.Addons[index]
		if err := ensureID(&addon.ID); err != nil {
			return err
		}
		addon.OrganizationID = enrollment.OrganizationID
		addon.EnrollmentID = enrollment.ID
		addonRow := enrollmentAddonToModel(*addon)
		if err := db.Create(&addonRow).Error; err != nil {
			return enrollmentWriteError(err)
		}
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

func (r EnrollmentGormRepository) ResolveReservationEntitlements(ctx context.Context, organizationID, unitID, personID domain.ID, targetKind domain.ScheduleTargetKind, targetID domain.ID, occursAt time.Time) ([]domain.ReservationEntitlement, error) {
	targetColumn := "modality_id"
	if targetKind == domain.ScheduleTargetService {
		targetColumn = "service_offering_id"
	}
	if targetKind != domain.ScheduleTargetModality && targetKind != domain.ScheduleTargetService {
		return nil, nil
	}
	query := fmt.Sprintf(`
		SELECT e.id AS enrollment_id, mpe.id AS entitlement_id, mpe.access_type, mpe.quantity, mpe.period_unit, mpe.daily_limit
		FROM enrollments e JOIN boxes b ON b.id = e.home_unit_id AND b.organization_id = e.organization_id JOIN membership_plan_entitlements mpe ON mpe.plan_version_id = e.plan_version_id AND mpe.organization_id = e.organization_id
		WHERE e.organization_id = ? AND e.home_unit_id = ? AND e.person_id = ? AND e.lifecycle_status = 'active'
		  AND e.starts_on <= (?::timestamptz AT TIME ZONE b.timezone)::date AND (e.valid_until IS NULL OR e.valid_until >= (?::timestamptz AT TIME ZONE b.timezone)::date) AND mpe.%s = ?
		UNION ALL
		SELECT e.id, mae.id, mae.access_type, mae.quantity, mae.period_unit, mae.daily_limit
		FROM enrollments e JOIN boxes b ON b.id = e.home_unit_id AND b.organization_id = e.organization_id JOIN enrollment_addons ea ON ea.enrollment_id = e.id AND ea.organization_id = e.organization_id
		JOIN membership_plan_version_addon_entitlements mae ON mae.addon_id = ea.plan_version_addon_id AND mae.organization_id = e.organization_id
		WHERE e.organization_id = ? AND e.home_unit_id = ? AND e.person_id = ? AND e.lifecycle_status = 'active'
		  AND e.starts_on <= (?::timestamptz AT TIME ZONE b.timezone)::date AND (e.valid_until IS NULL OR e.valid_until >= (?::timestamptz AT TIME ZONE b.timezone)::date) AND mae.%s = ?`, targetColumn, targetColumn)
	values := []domain.ReservationEntitlement{}
	err := databaseForContext(r.db, ctx).Raw(query, stringID(organizationID), stringID(unitID), stringID(personID), occursAt, occursAt, stringID(targetID), stringID(organizationID), stringID(unitID), stringID(personID), occursAt, occursAt, stringID(targetID)).Scan(&values).Error
	return values, err
}

func (r EnrollmentGormRepository) ResolveMembershipAccessCandidates(ctx context.Context, request domain.MembershipAccessRequest) ([]domain.MembershipAccessCandidate, error) {
	if request.TargetID != "" && request.TargetKind != domain.ScheduleTargetModality && request.TargetKind != domain.ScheduleTargetService {
		return []domain.MembershipAccessCandidate{}, nil
	}
	query := `
		SELECT e.id AS enrollment_id, rights.entitlement_id, e.home_unit_id,
			rights.target_kind, rights.target_id, rights.access_type, rights.quantity,
			rights.period_unit, rights.daily_limit, rights.booking_required,
			e.starts_on, e.valid_until, b.timezone
		FROM enrollments e
		JOIN boxes b ON b.id = e.home_unit_id AND b.organization_id = e.organization_id
		JOIN (
			SELECT mpe.organization_id, mpe.plan_version_id, NULL::uuid AS enrollment_id,
				mpe.id AS entitlement_id, mpe.access_type, mpe.quantity, mpe.period_unit,
				mpe.daily_limit, mpe.booking_required,
				CASE WHEN mpe.modality_id IS NOT NULL THEN 'modality' ELSE 'service_offering' END AS target_kind,
				COALESCE(mpe.modality_id, mpe.service_offering_id) AS target_id
			FROM membership_plan_entitlements mpe
			UNION ALL
			SELECT mae.organization_id, NULL::uuid, ea.enrollment_id, mae.id,
				mae.access_type, mae.quantity, mae.period_unit, mae.daily_limit,
				mae.booking_required,
				CASE WHEN mae.modality_id IS NOT NULL THEN 'modality' ELSE 'service_offering' END,
				COALESCE(mae.modality_id, mae.service_offering_id)
			FROM enrollment_addons ea
			JOIN membership_plan_version_addon_entitlements mae
			  ON mae.addon_id = ea.plan_version_addon_id AND mae.organization_id = ea.organization_id
		) rights ON rights.organization_id = e.organization_id
		  AND ((rights.plan_version_id = e.plan_version_id AND rights.enrollment_id IS NULL) OR rights.enrollment_id = e.id)
		WHERE e.organization_id = ? AND e.home_unit_id = ? AND e.person_id = ?
		  AND e.lifecycle_status = 'active'
		  AND e.starts_on <= (?::timestamptz AT TIME ZONE b.timezone)::date
		  AND (e.valid_until IS NULL OR e.valid_until >= (?::timestamptz AT TIME ZONE b.timezone)::date)
		  AND (NULLIF(?, '') IS NULL OR (rights.target_kind = ? AND rights.target_id = NULLIF(?, '')::uuid))
		ORDER BY e.starts_on DESC, rights.entitlement_id`
	values := []domain.MembershipAccessCandidate{}
	err := databaseForContext(r.db, ctx).Raw(query, stringID(request.OrganizationID), stringID(request.UnitID), stringID(request.PersonID), request.OccurredAt, request.OccurredAt, string(request.TargetID), string(request.TargetKind), string(request.TargetID)).Scan(&values).Error
	return values, err
}

func (r EnrollmentGormRepository) ListEnrollmentRoster(ctx context.Context, organizationID, unitID domain.ID, at time.Time) ([]domain.EnrollmentRosterItem, error) {
	type rosterRow struct {
		EnrollmentID        string
		StudentID           string
		PersonID            string
		PersonName          string
		Source              string
		PlanID              string
		PlanName            string
		LifecycleStatus     string
		StartsOn            time.Time
		ValidUntil          *time.Time
		EndedOn             *time.Time
		TargetID            string
		TargetKind          string
		TargetName          string
		ContractAccepted    bool
		HealthFormSubmitted bool
		InitialPaymentPaid  bool
		PaymentOverdue      bool
	}
	query := `
		WITH rights AS (
			SELECT mpe.organization_id, mpe.plan_version_id, NULL::uuid AS enrollment_id,
				CASE WHEN mpe.modality_id IS NOT NULL THEN 'modality' ELSE 'service_offering' END AS target_kind,
				COALESCE(mpe.modality_id, mpe.service_offering_id) AS target_id
			FROM membership_plan_entitlements mpe
			UNION ALL
			SELECT mae.organization_id, NULL::uuid, ea.enrollment_id,
				CASE WHEN mae.modality_id IS NOT NULL THEN 'modality' ELSE 'service_offering' END,
				COALESCE(mae.modality_id, mae.service_offering_id)
			FROM enrollment_addons ea
			JOIN membership_plan_version_addon_entitlements mae
			  ON mae.addon_id = ea.plan_version_addon_id AND mae.organization_id = ea.organization_id
		)
		SELECT e.id::text AS enrollment_id, '' AS student_id,
			e.person_id::text AS person_id, '' AS person_name,
			'' AS source, mp.id::text AS plan_id, mpv.name AS plan_name,
			e.lifecycle_status, e.starts_on, e.valid_until, e.ended_on,
			COALESCE(rights.target_id::text, '') AS target_id, COALESCE(rights.target_kind, '') AS target_kind,
			COALESCE(m.name, so.name, '') AS target_name,
			EXISTS (
				SELECT 1 FROM contract_instances ci
				WHERE ci.organization_id=e.organization_id AND ci.unit_id=e.home_unit_id
				  AND ci.person_id=e.person_id AND ci.enrollment_id=e.id AND ci.status='accepted'
			) AS contract_accepted,
			EXISTS (
				SELECT 1 FROM health_form_submissions hf
				WHERE hf.organization_id=e.organization_id AND hf.unit_id=e.home_unit_id
				  AND hf.person_id=e.person_id AND hf.enrollment_id=e.id AND hf.status='submitted'
			) AS health_form_submitted,
			COALESCE((
				SELECT c.status='paid' AND (c.paid_cents-c.refunded_cents)>=c.amount_cents
				FROM member_billing_contracts bc
				JOIN member_charges c ON c.organization_id=bc.organization_id AND c.contract_id=bc.id
				WHERE bc.organization_id=e.organization_id AND bc.unit_id=e.home_unit_id
				  AND bc.person_id=e.person_id AND bc.enrollment_id=e.id AND bc.status='active'
				ORDER BY c.sequence LIMIT 1
			), FALSE) AS initial_payment_paid,
			EXISTS (
				SELECT 1 FROM member_billing_contracts bc
				JOIN member_charges c ON c.organization_id=bc.organization_id AND c.contract_id=bc.id
				WHERE bc.organization_id=e.organization_id AND bc.unit_id=e.home_unit_id
				  AND bc.person_id=e.person_id AND bc.enrollment_id=e.id AND bc.status='active'
				  AND c.status IN ('pending','partially_paid')
				  AND (c.amount_cents-c.paid_cents+c.refunded_cents)>0
				  AND c.due_on < (?::timestamptz AT TIME ZONE b.timezone)::date
			) AS payment_overdue
		FROM enrollments e
		JOIN boxes b ON b.id=e.home_unit_id AND b.organization_id=e.organization_id
		JOIN membership_plan_versions mpv ON mpv.id=e.plan_version_id AND mpv.organization_id=e.organization_id
		JOIN membership_plans mp ON mp.id=mpv.plan_id AND mp.organization_id=e.organization_id
		LEFT JOIN rights ON rights.organization_id=e.organization_id
		  AND ((rights.plan_version_id=e.plan_version_id AND rights.enrollment_id IS NULL) OR rights.enrollment_id=e.id)
		LEFT JOIN modalities m ON rights.target_kind='modality' AND m.id=rights.target_id AND m.organization_id=e.organization_id
		LEFT JOIN service_offerings so ON rights.target_kind='service_offering' AND so.id=rights.target_id AND so.organization_id=e.organization_id
		WHERE e.organization_id=? AND e.home_unit_id=?
		ORDER BY e.created_at DESC, target_name`
	rows := []rosterRow{}
	if err := databaseForContext(r.db, ctx).Raw(query, at, stringID(organizationID), stringID(unitID)).Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.EnrollmentRosterItem, 0, len(rows))
	positions := make(map[string]int, len(rows))
	for _, row := range rows {
		index, exists := positions[row.EnrollmentID]
		if !exists {
			index = len(items)
			positions[row.EnrollmentID] = index
			items = append(items, domain.EnrollmentRosterItem{
				EnrollmentID: domain.ID(row.EnrollmentID), StudentID: domain.ID(row.StudentID),
				PersonID: domain.ID(row.PersonID), PersonName: row.PersonName, Source: domain.Source(row.Source),
				PlanID: domain.ID(row.PlanID), PlanName: row.PlanName,
				LifecycleStatus: domain.EnrollmentLifecycleStatus(row.LifecycleStatus), StartsOn: row.StartsOn,
				ValidUntil: row.ValidUntil, EndedOn: row.EndedOn, Targets: []domain.EnrollmentRosterTarget{},
				Prerequisites: domain.MembershipAccessPrerequisites{
					ContractAccepted: row.ContractAccepted, HealthFormSubmitted: row.HealthFormSubmitted,
					InitialPaymentPaid: row.InitialPaymentPaid, PaymentOverdue: row.PaymentOverdue,
				},
			})
		}
		if row.TargetID != "" {
			target := domain.EnrollmentRosterTarget{ID: domain.ID(row.TargetID), Kind: domain.ScheduleTargetKind(row.TargetKind), Name: row.TargetName}
			duplicate := false
			for _, existing := range items[index].Targets {
				if existing.ID == target.ID && existing.Kind == target.Kind {
					duplicate = true
					break
				}
			}
			if !duplicate {
				items[index].Targets = append(items[index].Targets, target)
			}
		}
	}
	return items, nil
}

func (r EnrollmentGormRepository) ResolveMembershipAccessPrerequisites(ctx context.Context, organizationID, unitID, personID, enrollmentID domain.ID, occursAt time.Time) (domain.MembershipAccessPrerequisites, error) {
	query := `
		SELECT
			EXISTS (
				SELECT 1 FROM contract_instances ci
				WHERE ci.organization_id = ? AND ci.unit_id = ? AND ci.person_id = ? AND ci.enrollment_id = ? AND ci.status = 'accepted'
			) AS contract_accepted,
			EXISTS (
				SELECT 1 FROM health_form_submissions hf
				WHERE hf.organization_id = ? AND hf.unit_id = ? AND hf.person_id = ? AND hf.enrollment_id = ? AND hf.status = 'submitted'
			) AS health_form_submitted,
			COALESCE((
				SELECT c.status = 'paid' AND (c.paid_cents - c.refunded_cents) >= c.amount_cents
				FROM member_billing_contracts bc
				JOIN member_charges c ON c.organization_id = bc.organization_id AND c.contract_id = bc.id
				WHERE bc.organization_id = ? AND bc.unit_id = ? AND bc.person_id = ? AND bc.enrollment_id = ? AND bc.status = 'active'
				ORDER BY c.sequence
				LIMIT 1
			), FALSE) AS initial_payment_paid,
			EXISTS (
				SELECT 1 FROM member_billing_contracts bc
				JOIN member_charges c ON c.organization_id = bc.organization_id AND c.contract_id = bc.id
				JOIN boxes b ON b.id = bc.unit_id AND b.organization_id = bc.organization_id
				WHERE bc.organization_id = ? AND bc.unit_id = ? AND bc.person_id = ? AND bc.enrollment_id = ? AND bc.status = 'active'
				  AND c.status IN ('pending', 'partially_paid')
				  AND (c.amount_cents - c.paid_cents + c.refunded_cents) > 0
				  AND c.due_on < (?::timestamptz AT TIME ZONE b.timezone)::date
			) AS payment_overdue`
	value := domain.MembershipAccessPrerequisites{}
	err := databaseForContext(r.db, ctx).Raw(query,
		stringID(organizationID), stringID(unitID), stringID(personID), stringID(enrollmentID),
		stringID(organizationID), stringID(unitID), stringID(personID), stringID(enrollmentID),
		stringID(organizationID), stringID(unitID), stringID(personID), stringID(enrollmentID),
		stringID(organizationID), stringID(unitID), stringID(personID), stringID(enrollmentID), occursAt,
	).Scan(&value).Error
	return value, err
}

func (r EnrollmentGormRepository) HasActiveEnrollmentInUnit(ctx context.Context, organizationID, unitID, personID domain.ID, at time.Time) (bool, error) {
	var count int64
	err := databaseForContext(r.db, ctx).Table("enrollments e").Joins("JOIN boxes b ON b.id = e.home_unit_id AND b.organization_id = e.organization_id").Where(`e.organization_id = ? AND e.home_unit_id = ? AND e.person_id = ? AND e.lifecycle_status = 'active' AND e.starts_on <= (?::timestamptz AT TIME ZONE b.timezone)::date AND (e.valid_until IS NULL OR e.valid_until >= (?::timestamptz AT TIME ZONE b.timezone)::date)`, stringID(organizationID), stringID(unitID), stringID(personID), at, at).Count(&count).Error
	return count > 0, err
}

func (r EnrollmentGormRepository) HasEnrollmentInUnit(ctx context.Context, organizationID, unitID, personID domain.ID) (bool, error) {
	var count int64
	err := databaseForContext(r.db, ctx).Table("enrollments").Where("organization_id = ? AND home_unit_id = ? AND person_id = ?", stringID(organizationID), stringID(unitID), stringID(personID)).Count(&count).Error
	return count > 0, err
}

func (r EnrollmentGormRepository) HasEnrollmentOutsideRequestedUnit(ctx context.Context, organizationID, unitID, personID domain.ID, at time.Time) (bool, error) {
	var count int64
	err := databaseForContext(r.db, ctx).Table("enrollments e").Joins("JOIN boxes b ON b.id = e.home_unit_id AND b.organization_id = e.organization_id").Where(`e.organization_id = ? AND e.home_unit_id <> ? AND e.person_id = ? AND e.lifecycle_status = 'active' AND e.starts_on <= (?::timestamptz AT TIME ZONE b.timezone)::date AND (e.valid_until IS NULL OR e.valid_until >= (?::timestamptz AT TIME ZONE b.timezone)::date)`, stringID(organizationID), stringID(unitID), stringID(personID), at, at).Count(&count).Error
	return count > 0, err
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
	var addonRows []models.EnrollmentAddonModel
	if err := db.Where("enrollment_id = ? AND organization_id = ?", row.ID, row.OrganizationID).Order("created_at ASC").Find(&addonRows).Error; err != nil {
		return item, err
	}
	addons := make([]domain.EnrollmentAddon, 0, len(addonRows))
	addonsPriceCents := 0
	for _, aRow := range addonRows {
		addon := enrollmentAddonToDomain(aRow)
		addons = append(addons, addon)
		addonsPriceCents += addon.PriceDeltaCents
	}
	item.Addons = addons
	item.BasePriceCents = version.PriceCents
	item.AddonsPriceCents = addonsPriceCents
	item.TotalPriceCents = item.BasePriceCents + item.AddonsPriceCents
	item.Terms = domain.EnrollmentTerms{
		PlanID:           domainID(plan.ID),
		PlanVersionID:    domainID(version.ID),
		VersionNumber:    version.VersionNumber,
		Name:             version.Name,
		Description:      version.Description,
		PriceCents:       item.TotalPriceCents,
		BasePriceCents:   item.BasePriceCents,
		AddonsPriceCents: item.AddonsPriceCents,
		TotalPriceCents:  item.TotalPriceCents,
		Currency:         version.Currency,
		BillingCycle:     domain.MembershipBillingCycle(version.BillingCycle),
		Summary:          loadedVersion.Summary,
		BaseSummary:      loadedVersion.Summary,
	}
	return item, nil
}

func enrollmentToModel(value domain.Enrollment) models.EnrollmentModel {
	return models.EnrollmentModel{ID: stringID(value.ID), OrganizationID: stringID(value.OrganizationID), PersonID: stringID(value.PersonID), HomeUnitID: stringID(value.HomeUnitID), PlanVersionID: stringID(value.PlanVersionID), LifecycleStatus: string(value.LifecycleStatus), StartsOn: dateOnly(value.StartsOn), ValidUntil: datePointer(value.ValidUntil), EndedOn: datePointer(value.EndedOn), EndedAt: value.EndedAt, EndedReason: enrollmentNullableString(value.EndedReason), Revision: value.Revision, CreatedByUserID: stringID(value.CreatedByUserID), EndedByUserID: optionalIDString(value.EndedByUserID), CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func enrollmentToDomain(value models.EnrollmentModel) domain.Enrollment {
	return domain.Enrollment{ID: domainID(value.ID), OrganizationID: domainID(value.OrganizationID), PersonID: domainID(value.PersonID), HomeUnitID: domainID(value.HomeUnitID), PlanVersionID: domainID(value.PlanVersionID), LifecycleStatus: domain.EnrollmentLifecycleStatus(value.LifecycleStatus), StartsOn: dateOnly(value.StartsOn), ValidUntil: datePointer(value.ValidUntil), EndedOn: datePointer(value.EndedOn), EndedAt: value.EndedAt, EndedReason: optionalString(value.EndedReason), Revision: value.Revision, CreatedByUserID: domainID(value.CreatedByUserID), EndedByUserID: optionalDomainID(value.EndedByUserID), CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func enrollmentAddonToModel(value domain.EnrollmentAddon) models.EnrollmentAddonModel {
	return models.EnrollmentAddonModel{
		ID:                 stringID(value.ID),
		OrganizationID:     stringID(value.OrganizationID),
		EnrollmentID:       stringID(value.EnrollmentID),
		PlanVersionAddonID: stringID(value.PlanVersionAddonID),
		AddonName:          value.Name,
		PriceDeltaCents:    value.PriceDeltaCents,
		EntitlementSummary: value.Summary,
		CreatedAt:          value.CreatedAt,
	}
}

func enrollmentAddonToDomain(value models.EnrollmentAddonModel) domain.EnrollmentAddon {
	return domain.EnrollmentAddon{
		ID:                 domainID(value.ID),
		OrganizationID:     domainID(value.OrganizationID),
		EnrollmentID:       domainID(value.EnrollmentID),
		PlanVersionAddonID: domainID(value.PlanVersionAddonID),
		Name:               value.AddonName,
		PriceDeltaCents:    value.PriceDeltaCents,
		Summary:            value.EntitlementSummary,
		CreatedAt:          value.CreatedAt,
	}
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
		case "enrollment_addons_enrollment_addon_key":
			return portrepo.ErrEnrollmentAddonDuplicate
		}
	}
	if pgErr.Code == "23514" && pgErr.ConstraintName == "enrollments_person_phone_required" {
		return memberships.ErrEnrollmentPhoneRequired
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
