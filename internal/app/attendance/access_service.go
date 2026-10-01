package attendance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"time"

	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"github.com/google/uuid"
)

var (
	ErrAccessUnavailable = errors.New("attendance access unavailable")
	ErrIdempotencyReused = errors.New("attendance idempotency key reused")
)

var attendanceKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)

type AdmitInput struct {
	PersonID   domain.ID                 `json:"person_id"`
	SessionID  *domain.ID                `json:"session_id,omitempty"`
	TargetKind domain.ScheduleTargetKind `json:"target_kind,omitempty"`
	TargetID   domain.ID                 `json:"target_id,omitempty"`
}

type AccessService struct {
	repo     portrepo.AttendanceAccessRepository
	contexts portrepo.SchedulingUnitContextResolver
	sessions portrepo.PublishedSessionReader
	tx       portrepo.TransactionManager
	now      func() time.Time
	access   membershipAccessEvaluator
	units    accessControlUnitReader
}

type membershipAccessEvaluator interface {
	Evaluate(context.Context, domain.MembershipAccessRequest) (*domain.MembershipAccessDecision, error)
}

type accessControlUnitReader interface {
	FindByID(context.Context, domain.ID) (*domain.Box, error)
}

func NewAccessService(repo portrepo.AttendanceAccessRepository, contexts portrepo.SchedulingUnitContextResolver, sessions portrepo.PublishedSessionReader, tx portrepo.TransactionManager) *AccessService {
	return &AccessService{repo: repo, contexts: contexts, sessions: sessions, tx: tx, now: func() time.Time { return time.Now().UTC() }}
}

func (s *AccessService) WithMembershipAccess(value membershipAccessEvaluator) *AccessService {
	s.access = value
	return s
}

func (s *AccessService) WithUnitAccessControl(value accessControlUnitReader) *AccessService {
	s.units = value
	return s
}

func (s *AccessService) Admit(ctx context.Context, actorID, unitID domain.ID, key string, input AdmitInput) (*domain.AttendanceAdmission, error) {
	if s.units != nil {
		unitSettings, err := s.units.FindByID(ctx, unitID)
		if err != nil || unitSettings == nil || unitSettings.EffectiveAccessControlMode() != domain.AccessControlManual {
			return nil, ErrAccessUnavailable
		}
	}
	unit, err := s.contexts.ResolveSchedulingUnitContext(ctx, actorID, unitID)
	if err != nil || unit == nil || !unit.Authorized || (unit.Role != "OWNER" && unit.Role != "COACH") {
		return nil, ErrAccessUnavailable
	}
	if input.PersonID == "" || !attendanceKeyPattern.MatchString(key) {
		return nil, ErrInvalidInput
	}
	fingerprint := attendanceFingerprint(unitID, input)
	var result *domain.AttendanceAdmission
	err = s.tx.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.repo.LockAttendanceCommand(tx, unit.OrganizationID, key); err != nil {
			return err
		}
		command, err := s.repo.FindAttendanceCommand(tx, unit.OrganizationID, key)
		if err != nil {
			return err
		}
		if command != nil {
			if command.Fingerprint != fingerprint {
				return ErrIdempotencyReused
			}
			result, err = s.repo.GetAttendanceAdmission(tx, unit.OrganizationID, command.DecisionID)
			if result != nil {
				result.Decision.IdempotentReplay = true
			}
			return err
		}

		now := s.now()
		subject, err := s.repo.ResolveAttendanceSubject(tx, unit.OrganizationID, unitID, input.PersonID, now)
		if err != nil || subject == nil {
			return ErrAccessUnavailable
		}
		decision := domain.AttendanceDecision{
			ID: domain.ID(uuid.NewString()), OrganizationID: unit.OrganizationID, UnitID: unitID,
			PersonID: subject.PersonID, StudentID: subject.StudentID, PersonName: subject.Name,
			Source: subject.Source, SessionID: input.SessionID, Outcome: domain.AttendanceDenied,
			ReasonCode: domain.AttendanceReasonPersonUnavailable, Reason: "Pessoa indisponível para entrada nesta unidade.",
			OccurredAt: now, ActorUserID: actorID,
		}
		if subject.ActiveEnrollmentID != "" {
			id := subject.ActiveEnrollmentID
			decision.EnrollmentID = &id
		}
		if err := s.evaluate(tx, &decision, *subject, input, now); err != nil {
			return err
		}
		if err := s.repo.CreateAttendanceDecision(tx, &decision); err != nil {
			return err
		}
		var record *domain.AttendanceRecord
		if decision.Outcome == domain.AttendanceAllowed {
			location, loadErr := time.LoadLocation(unit.Timezone)
			if loadErr != nil {
				return loadErr
			}
			record = &domain.AttendanceRecord{ID: domain.ID(uuid.NewString()), OrganizationID: unit.OrganizationID, UnitID: unitID, DecisionID: decision.ID, PersonID: subject.PersonID, StudentID: subject.StudentID, SessionID: input.SessionID, LocalDate: now.In(location).Format("2006-01-02"), OccurredAt: now}
			if err := s.repo.CreateAttendanceRecord(tx, record, subject.Source, actorID); err != nil {
				return err
			}
		}
		if err := s.repo.CreateAttendanceCommand(tx, &domain.AttendanceCommandRequest{OrganizationID: unit.OrganizationID, IdempotencyKey: key, Fingerprint: fingerprint, DecisionID: decision.ID}); err != nil {
			return err
		}
		result = &domain.AttendanceAdmission{Decision: decision, Attendance: record}
		return nil
	})
	return result, err
}

func (s *AccessService) evaluate(ctx context.Context, decision *domain.AttendanceDecision, subject domain.AttendanceSubject, input AdmitInput, now time.Time) error {
	if !subject.RelationshipActive {
		decision.ReasonCode, decision.Reason = domain.AttendanceReasonInactiveRelationship, "Vínculo inativo nesta unidade."
		return nil
	}
	if decision.SessionID != nil {
		session, err := s.sessions.GetPublishedSession(ctx, decision.OrganizationID, decision.UnitID, *decision.SessionID)
		if err != nil && !errors.Is(err, portrepo.ErrScheduleSessionNotFound) {
			return err
		}
		if err != nil || session == nil || session.Status != domain.ScheduleSessionScheduled {
			decision.ReasonCode, decision.Reason = domain.AttendanceReasonSessionUnavailable, "Aula indisponível para presença."
			return nil
		}
		if now.Before(session.StartsAt.Add(-60*time.Minute)) || now.After(session.EndsAt.Add(120*time.Minute)) {
			decision.ReasonCode, decision.Reason = domain.AttendanceReasonOutsideAttendanceWindow, "Check-in fora da janela da aula."
			return nil
		}
		reservation, err := s.repo.FindAttendanceReservation(ctx, decision.OrganizationID, decision.UnitID, *decision.SessionID, subject.PersonID)
		if err != nil {
			return err
		}
		if reservation == nil {
			decision.ReasonCode, decision.Reason = domain.AttendanceReasonSessionRequiresReservation, "Esta aula exige reserva confirmada."
			return nil
		}
		decision.ReservationID = &reservation.ID
		decision.EnrollmentID = &reservation.EnrollmentID
		decision.EntitlementID = &reservation.EntitlementID
		if reservation.Status == domain.ReservationWaitlisted {
			decision.ReasonCode, decision.Reason = domain.AttendanceReasonWaitlisted, "Pessoa ainda está na lista de espera."
			return nil
		}
		if reservation.Status != domain.ReservationConfirmed {
			decision.ReasonCode, decision.Reason = domain.AttendanceReasonSessionRequiresReservation, "Esta aula exige reserva confirmada."
			return nil
		}
		if s.access != nil {
			access, err := s.access.Evaluate(ctx, domain.MembershipAccessRequest{OrganizationID: decision.OrganizationID, UnitID: decision.UnitID, PersonID: subject.PersonID, TargetKind: session.TargetKind, TargetID: session.TargetID, OccurredAt: session.StartsAt, Purpose: domain.MembershipAccessCheckin, PreferredEnrollment: reservation.EnrollmentID, PreferredEntitlement: reservation.EntitlementID, UsageAlreadyCounted: true})
			if err != nil {
				return err
			}
			if access.Outcome != domain.MembershipAccessAllowed {
				decision.ReasonCode, decision.Reason = domain.AttendanceReasonNoSessionEntitlement, "A reserva não possui mais um direito de acesso válido."
				applyMembershipPrerequisiteDenial(decision, access)
				return nil
			}
			decision.RemainingUsage = access.RemainingUsage
		}
		decision.Outcome, decision.ReasonCode, decision.Reason = domain.AttendanceAllowed, domain.AttendanceReasonConfirmedReservation, "Reserva confirmada; entrada liberada."
		return nil
	}
	if subject.Source == domain.SourceWellhub || subject.Source == domain.SourceTotalPass {
		decision.Outcome, decision.ReasonCode, decision.Reason = domain.AttendanceAllowed, domain.AttendanceReasonCorporateRelationship, "Vínculo corporativo ativo; entrada liberada localmente."
		return nil
	}
	if s.access != nil {
		access, err := s.access.Evaluate(ctx, domain.MembershipAccessRequest{OrganizationID: decision.OrganizationID, UnitID: decision.UnitID, PersonID: subject.PersonID, TargetKind: input.TargetKind, TargetID: input.TargetID, OccurredAt: now, Purpose: domain.MembershipAccessCheckin})
		if err != nil {
			return err
		}
		if access.EnrollmentID != "" {
			id := access.EnrollmentID
			decision.EnrollmentID = &id
		}
		if access.EntitlementID != "" {
			id := access.EntitlementID
			decision.EntitlementID = &id
		}
		decision.RemainingUsage = access.RemainingUsage
		if access.Outcome == domain.MembershipAccessAllowed {
			decision.Outcome = domain.AttendanceAllowed
			if access.LegacyFallback {
				decision.ReasonCode, decision.Reason = domain.AttendanceReasonLegacyTransition, access.Reason
			} else {
				decision.ReasonCode, decision.Reason = domain.AttendanceReasonActiveEnrollment, access.Reason
			}
			return nil
		}
		switch access.ReasonCode {
		case domain.MembershipAccessReasonLimitReached:
			decision.ReasonCode = domain.AttendanceReasonUsageLimitReached
		case domain.MembershipAccessReasonBookingRequired:
			decision.ReasonCode = domain.AttendanceReasonBookingRequired
		case domain.MembershipAccessReasonTargetNotIncluded, domain.MembershipAccessReasonUnitNotIncluded:
			decision.ReasonCode = domain.AttendanceReasonTargetNotIncluded
		default:
			decision.ReasonCode = domain.AttendanceReasonNoActiveEnrollment
		}
		decision.Reason = access.Reason
		applyMembershipPrerequisiteDenial(decision, access)
		return nil
	}
	if subject.ActiveEnrollmentID != "" {
		decision.Outcome, decision.ReasonCode, decision.Reason = domain.AttendanceAllowed, domain.AttendanceReasonActiveEnrollment, "Matrícula ativa; entrada liberada."
		return nil
	}
	decision.ReasonCode, decision.Reason = domain.AttendanceReasonNoActiveEnrollment, "Nenhuma matrícula ativa encontrada."
	return nil
}

func applyMembershipPrerequisiteDenial(decision *domain.AttendanceDecision, access *domain.MembershipAccessDecision) {
	switch access.ReasonCode {
	case domain.MembershipAccessReasonContractPending:
		decision.ReasonCode, decision.Reason = domain.AttendanceReasonContractPending, access.Reason
	case domain.MembershipAccessReasonHealthFormPending:
		decision.ReasonCode, decision.Reason = domain.AttendanceReasonHealthFormPending, access.Reason
	case domain.MembershipAccessReasonPaymentRequired:
		decision.ReasonCode, decision.Reason = domain.AttendanceReasonPaymentRequired, access.Reason
	case domain.MembershipAccessReasonPaymentOverdue:
		decision.ReasonCode, decision.Reason = domain.AttendanceReasonPaymentOverdue, access.Reason
	case domain.MembershipAccessReasonLimitReached:
		decision.ReasonCode, decision.Reason = domain.AttendanceReasonUsageLimitReached, access.Reason
	}
}

func (s *AccessService) ListToday(ctx context.Context, actorID, unitID domain.ID) ([]domain.AttendanceAdmission, error) {
	if s.units != nil {
		unitSettings, err := s.units.FindByID(ctx, unitID)
		if err != nil || unitSettings == nil || unitSettings.EffectiveAccessControlMode() != domain.AccessControlManual {
			return nil, ErrAccessUnavailable
		}
	}
	unit, err := s.contexts.ResolveSchedulingUnitContext(ctx, actorID, unitID)
	if err != nil || unit == nil || !unit.Authorized {
		return nil, ErrAccessUnavailable
	}
	location, err := time.LoadLocation(unit.Timezone)
	if err != nil {
		return nil, err
	}
	return s.repo.ListAttendanceForDate(ctx, unit.OrganizationID, unitID, s.now().In(location).Format("2006-01-02"))
}

func attendanceFingerprint(unitID domain.ID, input AdmitInput) string {
	session := ""
	if input.SessionID != nil {
		session = string(*input.SessionID)
	}
	sum := sha256.Sum256([]byte(string(unitID) + "\n" + string(input.PersonID) + "\n" + session + "\n" + string(input.TargetKind) + "\n" + string(input.TargetID)))
	return hex.EncodeToString(sum[:])
}
