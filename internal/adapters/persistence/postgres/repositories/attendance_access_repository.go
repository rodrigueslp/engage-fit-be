package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"boxengage/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (r AttendanceGormRepository) LockAttendanceCommand(ctx context.Context, organizationID domain.ID, key string) error {
	return databaseForContext(r.db, ctx).Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", fmt.Sprintf("attendance:%s:%s", organizationID, key)).Error
}

func (r AttendanceGormRepository) FindAttendanceCommand(ctx context.Context, organizationID domain.ID, key string) (*domain.AttendanceCommandRequest, error) {
	var row struct{ OrganizationID, IdempotencyKey, RequestFingerprint, DecisionID string }
	err := databaseForContext(r.db, ctx).Table("attendance_command_requests").Where("organization_id = ? AND idempotency_key = ?", stringID(organizationID), key).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.AttendanceCommandRequest{OrganizationID: domainID(row.OrganizationID), IdempotencyKey: row.IdempotencyKey, Fingerprint: row.RequestFingerprint, DecisionID: domainID(row.DecisionID)}, nil
}

func (r AttendanceGormRepository) ResolveAttendanceSubject(ctx context.Context, organizationID, unitID, personID domain.ID, at time.Time) (*domain.AttendanceSubject, error) {
	var row struct {
		PersonID, StudentID, Name, Source, RelationshipStatus string
		ActiveEnrollmentID                                    *string
	}
	err := databaseForContext(r.db, ctx).Raw(`
		SELECT p.id AS person_id, s.id AS student_id, COALESCE(p.preferred_name, p.name) AS name,
			s.source, pur.relationship_status, active_enrollment.id AS active_enrollment_id
		FROM persons p
		JOIN person_unit_relationships pur ON pur.person_id = p.id AND pur.organization_id = p.organization_id AND pur.unit_id = ?
		JOIN LATERAL (
			SELECT candidate.id, candidate.source FROM students candidate
			WHERE candidate.person_id = p.id AND candidate.box_id = ? AND candidate.anonymized_at IS NULL
			ORDER BY CASE candidate.source WHEN 'wellhub' THEN 0 WHEN 'totalpass' THEN 1 ELSE 2 END, candidate.created_at
			LIMIT 1
		) s ON true
		LEFT JOIN LATERAL (
			SELECT e.id FROM enrollments e JOIN boxes b ON b.id = e.home_unit_id AND b.organization_id = e.organization_id
			WHERE e.organization_id = p.organization_id AND e.home_unit_id = ? AND e.person_id = p.id AND e.lifecycle_status = 'active'
			AND e.starts_on <= (?::timestamptz AT TIME ZONE b.timezone)::date
			AND (e.valid_until IS NULL OR e.valid_until >= (?::timestamptz AT TIME ZONE b.timezone)::date)
			ORDER BY e.starts_on DESC LIMIT 1
		) active_enrollment ON true
		WHERE p.id = ? AND p.organization_id = ? AND p.profile_status = 'active'`, stringID(unitID), stringID(unitID), stringID(unitID), at, at, stringID(personID), stringID(organizationID)).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.PersonID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	value := &domain.AttendanceSubject{PersonID: domainID(row.PersonID), StudentID: domainID(row.StudentID), Name: row.Name, Source: domain.Source(row.Source), RelationshipActive: row.RelationshipStatus == string(domain.PersonRelationshipActive)}
	if row.ActiveEnrollmentID != nil {
		value.ActiveEnrollmentID = domainID(*row.ActiveEnrollmentID)
	}
	return value, nil
}

func (r AttendanceGormRepository) HasLegacyMembership(ctx context.Context, organizationID, unitID, personID domain.ID) (bool, error) {
	var count int64
	err := databaseForContext(r.db, ctx).Table("students s").Joins("JOIN boxes b ON b.id = s.box_id").Where("b.organization_id = ? AND s.box_id = ? AND s.person_id = ? AND s.source = 'box_member' AND s.legacy_access_eligible = true AND s.anonymized_at IS NULL", stringID(organizationID), stringID(unitID), stringID(personID)).Count(&count).Error
	return count > 0, err
}

func (r AttendanceGormRepository) FindAttendanceReservation(ctx context.Context, organizationID, unitID, sessionID, personID domain.ID) (*domain.Reservation, error) {
	var value domain.Reservation
	err := databaseForContext(r.db, ctx).Table("reservations").Where("organization_id = ? AND unit_id = ? AND session_id = ? AND person_id = ?", stringID(organizationID), stringID(unitID), stringID(sessionID), stringID(personID)).Order("created_at DESC").Take(&value).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &value, err
}

func (r AttendanceGormRepository) CreateAttendanceDecision(ctx context.Context, value *domain.AttendanceDecision) error {
	return databaseForContext(r.db, ctx).Exec(`INSERT INTO attendance_access_decisions (id, organization_id, unit_id, person_id, student_id, session_id, enrollment_id, entitlement_id, reservation_id, source, outcome, reason_code, reason, remaining_usage, actor_user_id, occurred_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, stringID(value.ID), stringID(value.OrganizationID), stringID(value.UnitID), stringID(value.PersonID), stringID(value.StudentID), nullableDomainID(value.SessionID), nullableDomainID(value.EnrollmentID), nullableDomainID(value.EntitlementID), nullableDomainID(value.ReservationID), string(value.Source), string(value.Outcome), string(value.ReasonCode), value.Reason, value.RemainingUsage, stringID(value.ActorUserID), value.OccurredAt).Error
}

func (r AttendanceGormRepository) CreateAttendanceRecord(ctx context.Context, value *domain.AttendanceRecord, source domain.Source, actorID domain.ID) error {
	db := databaseForContext(r.db, ctx)
	query := db.Table("attendance_records").Where("organization_id = ? AND unit_id = ? AND person_id = ?", stringID(value.OrganizationID), stringID(value.UnitID), stringID(value.PersonID))
	if value.SessionID == nil {
		query = query.Where("session_id IS NULL AND local_date = ?", value.LocalDate)
	} else {
		query = query.Where("session_id = ?", stringID(*value.SessionID))
	}
	var existing struct {
		ID, DecisionID, StudentID, LocalDate string
		SessionID, CheckinID                 *string
		OccurredAt                           time.Time
	}
	err := query.Take(&existing).Error
	if err == nil {
		value.ID, value.DecisionID, value.StudentID, value.LocalDate, value.OccurredAt, value.AlreadyRecorded = domainID(existing.ID), domainID(existing.DecisionID), domainID(existing.StudentID), existing.LocalDate, existing.OccurredAt, true
		if existing.SessionID != nil {
			id := domainID(*existing.SessionID)
			value.SessionID = &id
		}
		if existing.CheckinID != nil {
			id := domainID(*existing.CheckinID)
			value.CheckinID = &id
		}
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	checkinID := domain.ID(uuid.NewString())
	checkinTime := value.OccurredAt
	insert := db.Exec(`INSERT INTO checkins (id, box_id, student_id, checkin_date, checkin_time, source, entry_method, recorded_by_user_id, created_at)
		SELECT ?, ?, ?, ?::date, ?, ?, 'access_decision', ?, ? WHERE NOT EXISTS (
			SELECT 1 FROM checkins WHERE box_id = ? AND student_id = ? AND checkin_date = ?::date
		)`, stringID(checkinID), stringID(value.UnitID), stringID(value.StudentID), value.LocalDate, checkinTime.Format("15:04:05"), string(source), stringID(actorID), value.OccurredAt, stringID(value.UnitID), stringID(value.StudentID), value.LocalDate)
	if insert.Error != nil {
		return insert.Error
	}
	if insert.RowsAffected == 0 {
		var existingCheckin string
		if err := db.Table("checkins").Select("id").Where("box_id = ? AND student_id = ? AND checkin_date = ?", stringID(value.UnitID), stringID(value.StudentID), value.LocalDate).Order("created_at").Limit(1).Scan(&existingCheckin).Error; err != nil {
			return err
		}
		checkinID = domainID(existingCheckin)
	}
	value.CheckinID = &checkinID
	return db.Exec(`INSERT INTO attendance_records (id, organization_id, unit_id, decision_id, person_id, student_id, session_id, checkin_id, local_date, occurred_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?::date, ?)`, stringID(value.ID), stringID(value.OrganizationID), stringID(value.UnitID), stringID(value.DecisionID), stringID(value.PersonID), stringID(value.StudentID), nullableDomainID(value.SessionID), stringID(checkinID), value.LocalDate, value.OccurredAt).Error
}

func (r AttendanceGormRepository) CreateAttendanceCommand(ctx context.Context, value *domain.AttendanceCommandRequest) error {
	return databaseForContext(r.db, ctx).Exec(`INSERT INTO attendance_command_requests (id, organization_id, idempotency_key, request_fingerprint, decision_id, created_at) VALUES (?, ?, ?, ?, ?, NOW())`, uuid.NewString(), stringID(value.OrganizationID), value.IdempotencyKey, value.Fingerprint, stringID(value.DecisionID)).Error
}

type attendanceAdmissionRow struct {
	DecisionID, OrganizationID, UnitID, PersonID, StudentID, PersonName, Source, Outcome, ReasonCode, Reason                                       string
	SessionID, EnrollmentID, EntitlementID, ReservationID, AttendanceID, AttendanceDecisionID, AttendanceStudentID, AttendanceLocalDate, CheckinID *string
	RemainingUsage                                                                                                                                 *int
	OccurredAt                                                                                                                                     time.Time
	AttendanceOccurredAt                                                                                                                           *time.Time
}

func attendanceAdmissionSelect() string {
	return `SELECT d.id AS decision_id, d.organization_id, d.unit_id, d.person_id, d.student_id, COALESCE(p.preferred_name,p.name) AS person_name, d.source, d.session_id, d.enrollment_id, d.entitlement_id, d.reservation_id, d.outcome, d.reason_code, d.reason, d.remaining_usage, d.occurred_at, a.id AS attendance_id, a.decision_id AS attendance_decision_id, a.student_id AS attendance_student_id, a.local_date::text AS attendance_local_date, a.checkin_id, a.occurred_at AS attendance_occurred_at FROM attendance_access_decisions d JOIN persons p ON p.id=d.person_id AND p.organization_id=d.organization_id LEFT JOIN attendance_records a ON a.decision_id=d.id`
}

func admissionFromRow(row attendanceAdmissionRow) domain.AttendanceAdmission {
	d := domain.AttendanceDecision{ID: domainID(row.DecisionID), OrganizationID: domainID(row.OrganizationID), UnitID: domainID(row.UnitID), PersonID: domainID(row.PersonID), StudentID: domainID(row.StudentID), PersonName: row.PersonName, Source: domain.Source(row.Source), Outcome: domain.AttendanceDecisionOutcome(row.Outcome), ReasonCode: domain.AttendanceReasonCode(row.ReasonCode), Reason: row.Reason, OccurredAt: row.OccurredAt}
	if row.SessionID != nil {
		id := domainID(*row.SessionID)
		d.SessionID = &id
	}
	if row.EnrollmentID != nil {
		id := domainID(*row.EnrollmentID)
		d.EnrollmentID = &id
	}
	if row.EntitlementID != nil {
		id := domainID(*row.EntitlementID)
		d.EntitlementID = &id
	}
	d.RemainingUsage = row.RemainingUsage
	if row.ReservationID != nil {
		id := domainID(*row.ReservationID)
		d.ReservationID = &id
	}
	result := domain.AttendanceAdmission{Decision: d}
	if row.AttendanceID != nil && row.AttendanceOccurredAt != nil {
		a := &domain.AttendanceRecord{ID: domainID(*row.AttendanceID), OrganizationID: d.OrganizationID, UnitID: d.UnitID, PersonID: d.PersonID, StudentID: d.StudentID, LocalDate: stringValue(row.AttendanceLocalDate), OccurredAt: *row.AttendanceOccurredAt}
		if row.AttendanceDecisionID != nil {
			a.DecisionID = domainID(*row.AttendanceDecisionID)
		}
		if row.SessionID != nil {
			id := domainID(*row.SessionID)
			a.SessionID = &id
		}
		if row.CheckinID != nil {
			id := domainID(*row.CheckinID)
			a.CheckinID = &id
		}
		result.Attendance = a
	}
	return result
}

func (r AttendanceGormRepository) GetAttendanceAdmission(ctx context.Context, organizationID, decisionID domain.ID) (*domain.AttendanceAdmission, error) {
	var row attendanceAdmissionRow
	if err := databaseForContext(r.db, ctx).Raw(attendanceAdmissionSelect()+` WHERE d.organization_id=? AND d.id=?`, stringID(organizationID), stringID(decisionID)).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.DecisionID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	value := admissionFromRow(row)
	return &value, nil
}

func (r AttendanceGormRepository) ListAttendanceForDate(ctx context.Context, organizationID, unitID domain.ID, date string) ([]domain.AttendanceAdmission, error) {
	var rows []attendanceAdmissionRow
	if err := databaseForContext(r.db, ctx).Raw(attendanceAdmissionSelect()+` WHERE d.organization_id=? AND d.unit_id=? AND d.occurred_at >= (?::date AT TIME ZONE (SELECT timezone FROM boxes WHERE id=?)) AND d.occurred_at < ((?::date + 1) AT TIME ZONE (SELECT timezone FROM boxes WHERE id=?)) ORDER BY d.occurred_at DESC`, stringID(organizationID), stringID(unitID), date, stringID(unitID), date, stringID(unitID)).Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.AttendanceAdmission, 0, len(rows))
	for _, row := range rows {
		result = append(result, admissionFromRow(row))
	}
	return result, nil
}

func nullableDomainID(value *domain.ID) any {
	if value == nil || *value == "" {
		return nil
	}
	return stringID(*value)
}
func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
