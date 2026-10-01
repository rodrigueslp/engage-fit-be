package repositories

import (
	"context"
	"encoding/json"
	"time"

	"boxengage/backend/internal/domain"
)

type athletePortalScopeRow struct {
	MembershipID   string
	UnitID         string
	UnitName       string
	OrganizationID string
	PersonID       string
}

func (r AthleteGormRepository) ResolveAthleteScope(ctx context.Context, athleteID, membershipID domain.ID) (*domain.AthletePortalScope, error) {
	var row athletePortalScopeRow
	result := r.db.WithContext(ctx).Raw(`
		SELECT abm.id AS membership_id, abm.box_id AS unit_id, b.organization_id, s.person_id
		FROM athlete_box_memberships abm
		JOIN athlete_student_links asl ON asl.membership_id = abm.id
		JOIN students s ON s.id = asl.student_id AND s.box_id = abm.box_id AND s.anonymized_at IS NULL
		JOIN boxes b ON b.id = abm.box_id AND b.status = 'active'
		WHERE abm.id = ? AND abm.athlete_account_id = ? AND abm.status = 'active' AND s.person_id IS NOT NULL
		ORDER BY asl.linked_at DESC LIMIT 1`, string(membershipID), string(athleteID)).Scan(&row)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &domain.AthletePortalScope{MembershipID: domain.ID(row.MembershipID), OrganizationID: domain.ID(row.OrganizationID), UnitID: domain.ID(row.UnitID), PersonID: domain.ID(row.PersonID)}, nil
}

func (r AthleteGormRepository) LoadAthletePortal(ctx context.Context, athleteID domain.ID, from, until time.Time) ([]domain.AthletePortalUnit, error) {
	var scopes []athletePortalScopeRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT ON (abm.id)
			abm.id AS membership_id, abm.box_id AS unit_id, b.name AS unit_name,
			b.organization_id, s.person_id
		FROM athlete_box_memberships abm
		JOIN athlete_student_links asl ON asl.membership_id = abm.id
		JOIN students s ON s.id = asl.student_id AND s.box_id = abm.box_id AND s.anonymized_at IS NULL
		JOIN boxes b ON b.id = abm.box_id AND b.status = 'active'
		WHERE abm.athlete_account_id = ? AND abm.status = 'active' AND s.person_id IS NOT NULL
		ORDER BY abm.id, asl.linked_at DESC`, string(athleteID)).Scan(&scopes).Error
	if err != nil {
		return nil, err
	}

	units := make([]domain.AthletePortalUnit, 0, len(scopes))
	for _, scope := range scopes {
		unit := domain.AthletePortalUnit{
			MembershipID: domain.ID(scope.MembershipID), UnitID: domain.ID(scope.UnitID), UnitName: scope.UnitName,
			Billing: domain.AthleteBillingSummary{FinancialStatus: "not_configured", Charges: []domain.AthleteChargeSummary{}},
			Agenda:  []domain.AthleteScheduleSession{},
		}
		if err := r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM checkins c JOIN students s ON s.id = c.student_id AND s.box_id = c.box_id WHERE c.box_id = ? AND s.person_id = ? AND s.anonymized_at IS NULL`, scope.UnitID, scope.PersonID).Scan(&unit.CheckinCount).Error; err != nil {
			return nil, err
		}
		if err := r.loadAthletePlan(ctx, scope, from, &unit); err != nil {
			return nil, err
		}
		if err := r.loadAthleteContract(ctx, scope, &unit); err != nil {
			return nil, err
		}
		if err := r.loadAthleteBilling(ctx, scope, from, &unit); err != nil {
			return nil, err
		}
		if err := r.loadAthleteAgenda(ctx, scope, from, until, &unit); err != nil {
			return nil, err
		}
		units = append(units, unit)
	}
	return units, nil
}

func (r AthleteGormRepository) loadAthletePlan(ctx context.Context, scope athletePortalScopeRow, now time.Time, unit *domain.AthletePortalUnit) error {
	type row struct {
		EnrollmentID, Name, Description, LifecycleStatus, BillingCycle, Currency string
		StartsOn                                                                 time.Time
		ValidUntil                                                               *time.Time
		PriceCents                                                               int
	}
	var value row
	result := r.db.WithContext(ctx).Raw(`
		SELECT e.id AS enrollment_id, pv.name, pv.description, e.lifecycle_status,
			e.starts_on, e.valid_until, pv.billing_cycle, pv.price_cents, pv.currency
		FROM enrollments e
		JOIN membership_plan_versions pv ON pv.id = e.plan_version_id AND pv.organization_id = e.organization_id
		WHERE e.organization_id = ? AND e.home_unit_id = ? AND e.person_id = ?
		ORDER BY (e.lifecycle_status = 'active') DESC, e.created_at DESC
		LIMIT 1`, scope.OrganizationID, scope.UnitID, scope.PersonID).Scan(&value)
	if result.Error != nil || result.RowsAffected == 0 {
		return result.Error
	}
	status := value.LifecycleStatus
	if status == "active" && value.ValidUntil != nil && value.ValidUntil.Before(dayStart(now)) {
		status = "expired"
	}
	var validUntil *string
	if value.ValidUntil != nil {
		formatted := value.ValidUntil.Format("2006-01-02")
		validUntil = &formatted
	}
	unit.Plan = &domain.AthletePlanSummary{
		EnrollmentID: domain.ID(value.EnrollmentID), Name: value.Name, Description: value.Description,
		Status: status, StartsOn: value.StartsOn.Format("2006-01-02"), ValidUntil: validUntil,
		BillingCycle: domain.MembershipBillingCycle(value.BillingCycle), PriceCents: value.PriceCents, Currency: value.Currency,
	}
	return nil
}

func (r AthleteGormRepository) loadAthleteContract(ctx context.Context, scope athletePortalScopeRow, unit *domain.AthletePortalUnit) error {
	type row struct {
		ID, EnrollmentID, DocumentTitle, Status, DocumentBodyMarkdown, AcceptanceStatement string
		IssuedAt, AcceptedAt                                                               *time.Time
	}
	var value row
	result := r.db.WithContext(ctx).Raw(`
		SELECT id, enrollment_id, document_title, status, document_body_markdown,
			acceptance_statement, issued_at, accepted_at
		FROM contract_instances
		WHERE organization_id = ? AND unit_id = ? AND person_id = ? AND status <> 'discarded'
		ORDER BY created_at DESC LIMIT 1`, scope.OrganizationID, scope.UnitID, scope.PersonID).Scan(&value)
	if result.Error != nil || result.RowsAffected == 0 {
		return result.Error
	}
	unit.Contract = &domain.AthleteContractSummary{
		ID: domain.ID(value.ID), EnrollmentID: domain.ID(value.EnrollmentID), Title: value.DocumentTitle,
		Status: value.Status, BodyMarkdown: value.DocumentBodyMarkdown, AcceptanceStatement: value.AcceptanceStatement,
		IssuedAt: value.IssuedAt, AcceptedAt: value.AcceptedAt,
	}
	return nil
}

func (r AthleteGormRepository) loadAthleteBilling(ctx context.Context, scope athletePortalScopeRow, now time.Time, unit *domain.AthletePortalUnit) error {
	type row struct {
		ID, EnrollmentID, Currency, Status, PaymentMethod, PaymentInstructions string
		DueOn                                                                  time.Time
		AmountCents, PaidCents, RefundedCents                                  int
		PaidAt                                                                 *time.Time
	}
	var rows []row
	if err := r.db.WithContext(ctx).Raw(`
		SELECT id, enrollment_id, due_on, amount_cents, paid_cents, refunded_cents,
			currency, status, payment_method, payment_instructions::text, paid_at
		FROM member_charges
		WHERE organization_id = ? AND unit_id = ? AND person_id = ?
		ORDER BY due_on DESC, sequence DESC LIMIT 24`, scope.OrganizationID, scope.UnitID, scope.PersonID).Scan(&rows).Error; err != nil {
		return err
	}
	open := 0
	financialStatus := "up_to_date"
	charges := make([]domain.AthleteChargeSummary, 0, len(rows))
	for _, value := range rows {
		instructions := map[string]any{}
		_ = json.Unmarshal([]byte(value.PaymentInstructions), &instructions)
		instructions = safePaymentInstructions(instructions)
		outstanding := value.AmountCents - value.PaidCents
		if value.Status == "pending" || value.Status == "partially_paid" {
			open += outstanding
			if value.DueOn.Before(dayStart(now)) {
				financialStatus = "overdue"
			} else if financialStatus != "overdue" {
				financialStatus = "pending"
			}
		}
		charges = append(charges, domain.AthleteChargeSummary{
			ID: domain.ID(value.ID), EnrollmentID: domain.ID(value.EnrollmentID), DueOn: value.DueOn.Format("2006-01-02"),
			AmountCents: value.AmountCents, PaidCents: value.PaidCents, RefundedCents: value.RefundedCents,
			Currency: value.Currency, Status: value.Status, PaymentMethod: value.PaymentMethod,
			PaymentInstructions: instructions, PaidAt: value.PaidAt,
		})
	}
	if len(rows) == 0 {
		financialStatus = "not_configured"
	}
	unit.Billing = domain.AthleteBillingSummary{FinancialStatus: financialStatus, OpenAmountCents: open, Charges: charges}
	return nil
}

func (r AthleteGormRepository) loadAthleteAgenda(ctx context.Context, scope athletePortalScopeRow, from, until time.Time, unit *domain.AthletePortalUnit) error {
	type row struct {
		ID, TargetName, ProfessionalName, SpaceName, Timezone string
		StartsAt, EndsAt                                      time.Time
		Capacity, ConfirmedCount, AvailableCount              int
		ReservationID, ReservationStatus                      *string
		WaitlistPosition                                      *int
		AttendedAt                                            *time.Time
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT ss.id, ss.target_name_snapshot AS target_name,
			ss.professional_name_snapshot AS professional_name,
			COALESCE(ss.space_name_snapshot, '') AS space_name,
			ss.starts_at, ss.ends_at, ss.timezone, ss.capacity,
			(SELECT COUNT(*) FROM reservations rc WHERE rc.organization_id = ss.organization_id AND rc.unit_id = ss.unit_id AND rc.session_id = ss.id AND rc.status = 'confirmed')::int AS confirmed_count,
			GREATEST(ss.capacity - (SELECT COUNT(*) FROM reservations rc WHERE rc.organization_id = ss.organization_id AND rc.unit_id = ss.unit_id AND rc.session_id = ss.id AND rc.status = 'confirmed'), 0)::int AS available_count,
			r.id AS reservation_id, r.status AS reservation_status,
			CASE WHEN r.status = 'waitlisted' THEN (SELECT COUNT(*)::int + 1 FROM reservations rw WHERE rw.organization_id = r.organization_id AND rw.unit_id = r.unit_id AND rw.session_id = r.session_id AND rw.status = 'waitlisted' AND (rw.created_at < r.created_at OR (rw.created_at = r.created_at AND rw.id < r.id))) END AS waitlist_position,
			ar.occurred_at AS attended_at
		FROM scheduling_sessions ss
		LEFT JOIN LATERAL (
			SELECT rr.* FROM reservations rr
			WHERE rr.organization_id = ss.organization_id AND rr.unit_id = ss.unit_id
				AND rr.session_id = ss.id AND rr.person_id = ?
			ORDER BY (rr.status IN ('confirmed','waitlisted')) DESC, rr.updated_at DESC LIMIT 1
		) r ON true
		LEFT JOIN attendance_records ar ON ar.organization_id = ss.organization_id AND ar.unit_id = ss.unit_id
			AND ar.session_id = ss.id AND ar.person_id = ?
		WHERE ss.organization_id = ? AND ss.unit_id = ? AND ss.status = 'scheduled'
			AND ss.starts_at >= ? AND ss.starts_at < ?
		ORDER BY ss.starts_at, ss.id`, scope.PersonID, scope.PersonID, scope.OrganizationID, scope.UnitID, from, until).Scan(&rows).Error
	if err != nil {
		return err
	}
	for _, value := range rows {
		var reservationID *domain.ID
		if value.ReservationID != nil {
			id := domain.ID(*value.ReservationID)
			reservationID = &id
		}
		status := ""
		if value.ReservationStatus != nil {
			status = *value.ReservationStatus
		}
		unit.Agenda = append(unit.Agenda, domain.AthleteScheduleSession{
			ID: domain.ID(value.ID), TargetName: value.TargetName, ProfessionalName: value.ProfessionalName,
			SpaceName: value.SpaceName, StartsAt: value.StartsAt, EndsAt: value.EndsAt, Timezone: value.Timezone,
			Capacity: value.Capacity, ConfirmedCount: value.ConfirmedCount, AvailableCount: value.AvailableCount,
			ReservationID: reservationID, ReservationStatus: status, WaitlistPosition: value.WaitlistPosition, AttendedAt: value.AttendedAt,
		})
	}
	return nil
}

func safePaymentInstructions(value map[string]any) map[string]any {
	allowed := map[string]bool{"simulation": true, "message": true, "pix_copy_paste": true, "digitable_line": true, "checkout_url": true, "expires_at": true}
	result := make(map[string]any)
	for key, item := range value {
		if allowed[key] {
			result[key] = item
		}
	}
	return result
}

func dayStart(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}
