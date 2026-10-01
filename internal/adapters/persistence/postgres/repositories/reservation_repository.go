package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

func (r ReservationGormRepository) LockCommand(ctx context.Context, organizationID domain.ID, command, key string) error {
	return databaseForContext(r.db, ctx).Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", fmt.Sprintf("reservation-command:%s:%s:%s", organizationID, command, key)).Error
}

func (r ReservationGormRepository) FindCommand(ctx context.Context, organizationID domain.ID, command, key string) (*domain.ReservationCommandRequest, error) {
	var row struct {
		ID, OrganizationID, Command, IdempotencyKey, RequestFingerprint, ReservationID string
		ResponseData                                                                   []byte
		CreatedAt                                                                      time.Time
	}
	err := databaseForContext(r.db, ctx).Table("reservation_command_requests").Where("organization_id = ? AND command = ? AND idempotency_key = ?", stringID(organizationID), command, key).Take(&row).Error
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
	return &domain.ReservationCommandRequest{ID: domainID(row.ID), OrganizationID: domainID(row.OrganizationID), Command: row.Command, IdempotencyKey: row.IdempotencyKey, Fingerprint: row.RequestFingerprint, ReservationID: domainID(row.ReservationID), ResponseData: response, CreatedAt: row.CreatedAt}, nil
}

func (r ReservationGormRepository) CreateCommand(ctx context.Context, value *domain.ReservationCommandRequest) error {
	response, err := json.Marshal(value.ResponseData)
	if err != nil {
		return err
	}
	return reservationWriteError(databaseForContext(r.db, ctx).Exec(`INSERT INTO reservation_command_requests (id, organization_id, command, idempotency_key, request_fingerprint, reservation_id, response_data, created_at) VALUES (?, ?, ?, ?, ?, ?, ?::jsonb, ?)`, stringID(value.ID), stringID(value.OrganizationID), value.Command, value.IdempotencyKey, value.Fingerprint, stringID(value.ReservationID), string(response), value.CreatedAt).Error)
}

func (r ReservationGormRepository) LockSession(ctx context.Context, organizationID, sessionID domain.ID) error {
	return databaseForContext(r.db, ctx).Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", fmt.Sprintf("reservation-session:%s:%s", organizationID, sessionID)).Error
}

func reservationSelect() string {
	return `SELECT r.id, r.organization_id, r.unit_id, r.session_id, r.person_id, COALESCE(p.preferred_name, p.name) AS person_name, r.enrollment_id, r.entitlement_id, r.status, r.makeup_for_reservation_id, r.revision, COALESCE(r.cancelled_reason, '') AS cancelled_reason, r.created_at, r.updated_at FROM reservations r JOIN persons p ON p.id = r.person_id AND p.organization_id = r.organization_id`
}

func (r ReservationGormRepository) Get(ctx context.Context, organizationID, unitID, reservationID domain.ID, forUpdate bool) (*domain.Reservation, error) {
	query := reservationSelect() + ` WHERE r.id = ? AND r.organization_id = ? AND r.unit_id = ?`
	if forUpdate {
		query += ` FOR UPDATE OF r`
	}
	var value domain.Reservation
	if err := databaseForContext(r.db, ctx).Raw(query, stringID(reservationID), stringID(organizationID), stringID(unitID)).Scan(&value).Error; err != nil {
		return nil, err
	}
	if value.ID == "" {
		return nil, portrepo.ErrReservationNotFound
	}
	return &value, nil
}

func (r ReservationGormRepository) FindActive(ctx context.Context, organizationID, unitID, sessionID, personID domain.ID) (*domain.Reservation, error) {
	var value domain.Reservation
	err := databaseForContext(r.db, ctx).Raw(reservationSelect()+` WHERE r.organization_id = ? AND r.unit_id = ? AND r.session_id = ? AND r.person_id = ? AND r.status IN ('confirmed','waitlisted')`, stringID(organizationID), stringID(unitID), stringID(sessionID), stringID(personID)).Scan(&value).Error
	if err != nil {
		return nil, err
	}
	if value.ID == "" {
		return nil, nil
	}
	return &value, nil
}

func (r ReservationGormRepository) ListSession(ctx context.Context, organizationID, unitID, sessionID domain.ID) ([]domain.Reservation, error) {
	values := []domain.Reservation{}
	err := databaseForContext(r.db, ctx).Raw(reservationSelect()+` WHERE r.organization_id = ? AND r.unit_id = ? AND r.session_id = ? ORDER BY CASE r.status WHEN 'confirmed' THEN 0 WHEN 'waitlisted' THEN 1 WHEN 'no_show' THEN 2 ELSE 3 END, r.created_at`, stringID(organizationID), stringID(unitID), stringID(sessionID)).Scan(&values).Error
	return values, err
}

func (r ReservationGormRepository) ListPerson(ctx context.Context, organizationID, unitID, personID domain.ID, from, until time.Time) ([]domain.Reservation, error) {
	values := []domain.Reservation{}
	err := databaseForContext(r.db, ctx).Raw(reservationSelect()+` JOIN scheduling_sessions s ON s.id = r.session_id AND s.organization_id = r.organization_id AND s.unit_id = r.unit_id WHERE r.organization_id = ? AND r.unit_id = ? AND r.person_id = ? AND s.starts_at >= ? AND s.starts_at < ? ORDER BY s.starts_at`, stringID(organizationID), stringID(unitID), stringID(personID), from, until).Scan(&values).Error
	return values, err
}

func (r ReservationGormRepository) CountSession(ctx context.Context, organizationID, unitID, sessionID domain.ID, status domain.ReservationStatus) (int, error) {
	var count int64
	err := databaseForContext(r.db, ctx).Table("reservations").Where("organization_id = ? AND unit_id = ? AND session_id = ? AND status = ?", stringID(organizationID), stringID(unitID), stringID(sessionID), string(status)).Count(&count).Error
	return int(count), err
}

func (r ReservationGormRepository) CountConsumed(ctx context.Context, organizationID, personID, enrollmentID, entitlementID domain.ID, from, until time.Time) (int, error) {
	var count int64
	err := databaseForContext(r.db, ctx).Table("reservations r").Joins("JOIN scheduling_sessions s ON s.id = r.session_id AND s.organization_id = r.organization_id").Where("r.organization_id = ? AND r.person_id = ? AND r.enrollment_id = ? AND r.entitlement_id = ? AND r.status IN ('confirmed','no_show') AND s.starts_at >= ? AND s.starts_at < ?", stringID(organizationID), stringID(personID), stringID(enrollmentID), stringID(entitlementID), from, until).Count(&count).Error
	return int(count), err
}

func (r ReservationGormRepository) Create(ctx context.Context, value *domain.Reservation) error {
	var makeup any
	if value.MakeupForReservationID != nil {
		makeup = stringID(*value.MakeupForReservationID)
	}
	err := databaseForContext(r.db, ctx).Exec(`INSERT INTO reservations (id, organization_id, unit_id, session_id, person_id, enrollment_id, entitlement_id, status, makeup_for_reservation_id, revision, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, stringID(value.ID), stringID(value.OrganizationID), stringID(value.UnitID), stringID(value.SessionID), stringID(value.PersonID), stringID(value.EnrollmentID), stringID(value.EntitlementID), string(value.Status), makeup, value.Revision, value.CreatedAt, value.UpdatedAt).Error
	return reservationWriteError(err)
}

func (r ReservationGormRepository) Update(ctx context.Context, value *domain.Reservation, expectedRevision int) error {
	result := databaseForContext(r.db, ctx).Exec(`UPDATE reservations SET status = ?, enrollment_id = ?, entitlement_id = ?, revision = ?, cancelled_reason = NULLIF(?, ''), updated_at = ? WHERE id = ? AND organization_id = ? AND unit_id = ? AND revision = ?`, string(value.Status), stringID(value.EnrollmentID), stringID(value.EntitlementID), value.Revision, value.CancelledReason, value.UpdatedAt, stringID(value.ID), stringID(value.OrganizationID), stringID(value.UnitID), expectedRevision)
	if result.Error != nil {
		return reservationWriteError(result.Error)
	}
	if result.RowsAffected == 0 {
		return portrepo.ErrReservationRevisionConflict
	}
	return nil
}

func (r ReservationGormRepository) FirstWaitlisted(ctx context.Context, organizationID, unitID, sessionID domain.ID) (*domain.Reservation, error) {
	var value domain.Reservation
	err := databaseForContext(r.db, ctx).Raw(reservationSelect()+` WHERE r.organization_id = ? AND r.unit_id = ? AND r.session_id = ? AND r.status = 'waitlisted' ORDER BY r.created_at, r.id LIMIT 1 FOR UPDATE OF r`, stringID(organizationID), stringID(unitID), stringID(sessionID)).Scan(&value).Error
	if err != nil {
		return nil, err
	}
	if value.ID == "" {
		return nil, nil
	}
	return &value, nil
}

func (r ReservationGormRepository) GetMakeupSource(ctx context.Context, organizationID, unitID, personID, reservationID domain.ID) (*domain.Reservation, error) {
	var value domain.Reservation
	err := databaseForContext(r.db, ctx).Raw(reservationSelect()+` WHERE r.id = ? AND r.organization_id = ? AND r.unit_id = ? AND r.person_id = ? AND r.status = 'no_show' AND NOT EXISTS (SELECT 1 FROM reservations child WHERE child.makeup_for_reservation_id = r.id AND child.status <> 'cancelled') FOR SHARE OF r`, stringID(reservationID), stringID(organizationID), stringID(unitID), stringID(personID)).Scan(&value).Error
	if err != nil {
		return nil, err
	}
	if value.ID == "" {
		return nil, portrepo.ErrReservationMakeupInvalid
	}
	return &value, nil
}

func (r ReservationGormRepository) CreateAuditEvent(ctx context.Context, value *domain.ReservationAuditEvent) error {
	return databaseForContext(r.db, ctx).Exec(`INSERT INTO reservation_audit_events (id, organization_id, unit_id, session_id, reservation_id, person_id, actor_user_id, actor_athlete_account_id, action, reason, created_at) VALUES (?, ?, ?, ?, ?, ?, NULLIF(?, '')::uuid, NULLIF(?, '')::uuid, ?, NULLIF(?, ''), ?)`, stringID(value.ID), stringID(value.OrganizationID), stringID(value.UnitID), stringID(value.SessionID), stringID(value.ReservationID), stringID(value.PersonID), stringID(value.ActorUserID), stringID(value.ActorAthleteID), value.Action, value.Reason, value.CreatedAt).Error
}

func (r ReservationGormRepository) ListReservationPrivacy(ctx context.Context, unitID, personID domain.ID) ([]domain.Reservation, error) {
	values := []domain.Reservation{}
	err := databaseForContext(r.db, ctx).Raw(reservationSelect()+` JOIN boxes b ON b.id = r.unit_id AND b.organization_id = r.organization_id WHERE r.unit_id = ? AND r.person_id = ? ORDER BY r.created_at`, stringID(unitID), stringID(personID)).Scan(&values).Error
	return values, err
}

func reservationWriteError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return portrepo.ErrReservationAlreadyExists
	}
	return err
}
