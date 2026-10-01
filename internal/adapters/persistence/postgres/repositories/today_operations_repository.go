package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
	"gorm.io/gorm"
)

// TodayOperationsGormRepository is an explicit cross-module read model for
// the operational dashboard. It never writes source module tables.
type TodayOperationsGormRepository struct{ db *gorm.DB }

func NewTodayOperationsGormRepository(db *gorm.DB) TodayOperationsGormRepository {
	return TodayOperationsGormRepository{db: db}
}

func (r TodayOperationsGormRepository) ReadToday(ctx context.Context, organizationID, unitID domain.ID, date string, from, until time.Time) (*domain.TodayOperations, error) {
	db := databaseForContext(r.db, ctx)
	result := &domain.TodayOperations{Date: date, Sessions: []domain.TodaySession{}, Charges: []domain.TodayCharge{}, PeopleActions: []domain.TodayPeopleAction{}}
	if err := db.Raw(`
		SELECT s.id AS session_id, s.target_name_snapshot AS target_name, s.professional_name_snapshot AS professional_name, COALESCE(s.space_name_snapshot, '') AS space_name,
		       s.starts_at, s.ends_at, s.capacity,
		       COUNT(DISTINCT r.id) FILTER (WHERE r.status = 'confirmed') AS confirmed_count,
		       COUNT(DISTINCT r.id) FILTER (WHERE r.status = 'waitlisted') AS waitlist_count,
		       COUNT(DISTINCT a.id) AS attendance_count
		FROM scheduling_sessions s
		LEFT JOIN reservations r ON r.organization_id=s.organization_id AND r.unit_id=s.unit_id AND r.session_id=s.id
		LEFT JOIN attendance_records a ON a.organization_id=s.organization_id AND a.unit_id=s.unit_id AND a.session_id=s.id
		WHERE s.organization_id=? AND s.unit_id=? AND s.status='scheduled' AND s.starts_at < ? AND s.ends_at > ?
		GROUP BY s.id, s.target_name_snapshot, s.professional_name_snapshot, s.space_name_snapshot, s.starts_at, s.ends_at, s.capacity
		ORDER BY s.starts_at, s.id`, stringID(organizationID), stringID(unitID), until, from).Scan(&result.Sessions).Error; err != nil {
		return nil, err
	}
	if err := db.Raw(`
		SELECT c.id AS charge_id, c.person_id, c.enrollment_id, COALESCE(p.preferred_name, p.name) AS person_name,
		       c.due_on::text AS due_on, (c.amount_cents-c.paid_cents) AS open_amount_cents,
		       CASE WHEN c.due_on < ?::date THEN 'overdue' ELSE c.status END AS status
		FROM member_charges c JOIN persons p ON p.id=c.person_id AND p.organization_id=c.organization_id
		WHERE c.organization_id=? AND c.unit_id=? AND c.due_on <= ?::date
		  AND c.status IN ('pending','partially_paid') AND c.amount_cents > c.paid_cents
		ORDER BY c.due_on, person_name, c.id`, date, stringID(organizationID), stringID(unitID), date).Scan(&result.Charges).Error; err != nil {
		return nil, err
	}
	if err := db.Raw(`
		SELECT 'pending_contract' AS kind, c.person_id, c.enrollment_id, COALESCE(p.preferred_name,p.name) AS person_name,
		       c.id AS reference_id, 'Contrato aguardando aceite' AS label
		FROM contract_instances c JOIN persons p ON p.id=c.person_id AND p.organization_id=c.organization_id
		WHERE c.organization_id=? AND c.unit_id=? AND c.status='pending_acceptance'
		UNION ALL
		SELECT 'health_review' AS kind, h.person_id, h.enrollment_id, COALESCE(p.preferred_name,p.name) AS person_name,
		       h.id AS reference_id, 'PAR-Q requer revisão' AS label
		FROM health_form_submissions h JOIN persons p ON p.id=h.person_id AND p.organization_id=h.organization_id
		WHERE h.organization_id=? AND h.unit_id=? AND h.status='submitted' AND h.requires_review=TRUE
		ORDER BY person_name, kind`, stringID(organizationID), stringID(unitID), stringID(organizationID), stringID(unitID)).Scan(&result.PeopleActions).Error; err != nil {
		return nil, err
	}
	var totals struct{ AttendanceTotal, DeniedAccessTotal int }
	if err := db.Raw(`SELECT
		(SELECT COUNT(*) FROM attendance_records WHERE organization_id=? AND unit_id=? AND local_date=?) AS attendance_total,
		(SELECT COUNT(*) FROM attendance_access_decisions WHERE organization_id=? AND unit_id=? AND outcome='denied' AND occurred_at>=? AND occurred_at<?) AS denied_access_total`,
		stringID(organizationID), stringID(unitID), date, stringID(organizationID), stringID(unitID), from, until).Scan(&totals).Error; err != nil {
		return nil, err
	}
	result.AttendanceTotal, result.DeniedAccessTotal = totals.AttendanceTotal, totals.DeniedAccessTotal
	return result, nil
}
