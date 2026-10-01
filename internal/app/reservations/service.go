package reservations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"time"

	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"github.com/google/uuid"
)

var (
	ErrNotFound          = portrepo.ErrReservationNotFound
	ErrInvalid           = portrepo.ErrReservationInvalid
	ErrAlreadyExists     = portrepo.ErrReservationAlreadyExists
	ErrNotEligible       = portrepo.ErrReservationNotEligible
	ErrLimitReached      = portrepo.ErrReservationLimitReached
	ErrRevisionConflict  = portrepo.ErrReservationRevisionConflict
	ErrAlreadyFinal      = portrepo.ErrReservationAlreadyFinal
	ErrIdempotencyReused = portrepo.ErrReservationIdempotencyReused
	ErrMakeupInvalid     = portrepo.ErrReservationMakeupInvalid
	ErrContractPending   = errors.New("reservation contract pending")
	ErrHealthFormPending = errors.New("reservation health form pending")
	ErrPaymentRequired   = errors.New("reservation payment required")
	ErrPaymentOverdue    = errors.New("reservation payment overdue")
)

type BookInput struct {
	SessionID              domain.ID  `json:"session_id"`
	PersonID               domain.ID  `json:"person_id"`
	MakeupForReservationID *domain.ID `json:"makeup_for_reservation_id,omitempty"`
}

type CancelInput struct {
	ExpectedRevision int    `json:"expected_revision"`
	Reason           string `json:"reason"`
}

type Service struct {
	repo        portrepo.ReservationRepository
	sessions    portrepo.PublishedSessionReader
	eligibility portrepo.ReservationEligibilityResolver
	people      portrepo.ReservationPersonResolver
	contexts    portrepo.SchedulingUnitContextResolver
	tx          portrepo.TransactionManager
	now         func() time.Time
	access      membershipAccessEvaluator
}

type membershipAccessEvaluator interface {
	Evaluate(context.Context, domain.MembershipAccessRequest) (*domain.MembershipAccessDecision, error)
}

type Option func(*Service)

func WithClock(clock func() time.Time) Option { return func(s *Service) { s.now = clock } }
func WithAccessEvaluator(value membershipAccessEvaluator) Option {
	return func(s *Service) { s.access = value }
}

func NewService(repo portrepo.ReservationRepository, sessions portrepo.PublishedSessionReader, eligibility portrepo.ReservationEligibilityResolver, people portrepo.ReservationPersonResolver, contexts portrepo.SchedulingUnitContextResolver, tx portrepo.TransactionManager, options ...Option) *Service {
	s := &Service{repo: repo, sessions: sessions, eligibility: eligibility, people: people, contexts: contexts, tx: tx, now: func() time.Time { return time.Now().UTC() }}
	for _, option := range options {
		option(s)
	}
	return s
}

func (s *Service) authorize(ctx context.Context, actorID, unitID domain.ID) (*domain.SchedulingUnitContext, error) {
	value, err := s.contexts.ResolveSchedulingUnitContext(ctx, actorID, unitID)
	if err != nil || value == nil || !value.Authorized || (value.Role != "OWNER" && value.Role != "COACH") {
		return nil, ErrNotFound
	}
	return value, nil
}

func (s *Service) Book(ctx context.Context, actorID, unitID domain.ID, key string, input BookInput) (*domain.Reservation, bool, error) {
	unit, err := s.authorize(ctx, actorID, unitID)
	if err != nil {
		return nil, false, err
	}
	return s.book(ctx, unit, actorID, "", key, input)
}

func (s *Service) BookForAthlete(ctx context.Context, scope domain.AthletePortalScope, athleteID domain.ID, sessionID domain.ID, key string) (*domain.Reservation, bool, error) {
	if scope.OrganizationID == "" || scope.UnitID == "" || scope.PersonID == "" || athleteID == "" {
		return nil, false, ErrNotFound
	}
	unit := &domain.SchedulingUnitContext{OrganizationID: scope.OrganizationID, UnitID: scope.UnitID, Authorized: true}
	return s.book(ctx, unit, "", athleteID, key, BookInput{SessionID: sessionID, PersonID: scope.PersonID})
}

func (s *Service) book(ctx context.Context, unit *domain.SchedulingUnitContext, actorID, athleteID domain.ID, key string, input BookInput) (*domain.Reservation, bool, error) {
	unitID := unit.UnitID
	if input.SessionID == "" || input.PersonID == "" || !validKey(key) {
		return nil, false, ErrInvalid
	}
	fingerprint := hash(map[string]any{"command": "book", "unit_id": unitID, "session_id": input.SessionID, "person_id": input.PersonID, "makeup_for": input.MakeupForReservationID})
	var result *domain.Reservation
	fresh := false
	err := s.tx.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.repo.LockCommand(tx, unit.OrganizationID, "book", key); err != nil {
			return err
		}
		command, err := s.repo.FindCommand(tx, unit.OrganizationID, "book", key)
		if err != nil {
			return err
		}
		if command != nil {
			if command.Fingerprint != fingerprint {
				return ErrIdempotencyReused
			}
			result, err = reservationFromResponse(command.ResponseData["reservation"])
			return err
		}
		if err := s.repo.LockSession(tx, unit.OrganizationID, input.SessionID); err != nil {
			return err
		}
		session, err := s.sessions.GetPublishedSession(tx, unit.OrganizationID, unitID, input.SessionID)
		if err != nil || session == nil {
			return ErrNotFound
		}
		if session.Status != domain.ScheduleSessionScheduled || !session.StartsAt.After(s.now()) {
			return ErrInvalid
		}
		name, err := s.people.ResolveReservationPerson(tx, unit.OrganizationID, unitID, input.PersonID)
		if err != nil || name == "" {
			return ErrNotEligible
		}
		if active, err := s.repo.FindActive(tx, unit.OrganizationID, unitID, input.SessionID, input.PersonID); err != nil {
			return err
		} else if active != nil {
			return ErrAlreadyExists
		}
		entitlement, err := s.resolveEntitlement(tx, unit.OrganizationID, input.PersonID, *session, input.MakeupForReservationID)
		if err != nil {
			return err
		}
		confirmed, err := s.repo.CountSession(tx, unit.OrganizationID, unitID, input.SessionID, domain.ReservationConfirmed)
		if err != nil {
			return err
		}
		status := domain.ReservationConfirmed
		if confirmed >= session.Capacity {
			status = domain.ReservationWaitlisted
		}
		now := s.now()
		value := &domain.Reservation{ID: domain.ID(uuid.NewString()), OrganizationID: unit.OrganizationID, UnitID: unitID, SessionID: input.SessionID, PersonID: input.PersonID, PersonName: name, EnrollmentID: entitlement.EnrollmentID, EntitlementID: entitlement.EntitlementID, Status: status, MakeupForReservationID: input.MakeupForReservationID, Revision: 1, CreatedAt: now, UpdatedAt: now}
		if err := s.repo.Create(tx, value); err != nil {
			return err
		}
		if err := s.repo.CreateAuditEvent(tx, &domain.ReservationAuditEvent{ID: domain.ID(uuid.NewString()), OrganizationID: unit.OrganizationID, UnitID: unitID, SessionID: input.SessionID, ReservationID: value.ID, PersonID: input.PersonID, ActorUserID: actorID, ActorAthleteID: athleteID, Action: "reservation." + string(status), CreatedAt: now}); err != nil {
			return err
		}
		if err := s.repo.CreateCommand(tx, &domain.ReservationCommandRequest{ID: domain.ID(uuid.NewString()), OrganizationID: unit.OrganizationID, Command: "book", IdempotencyKey: key, Fingerprint: fingerprint, ReservationID: value.ID, ResponseData: map[string]any{"reservation": value}, CreatedAt: now}); err != nil {
			return err
		}
		result, fresh = value, true
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return result, fresh, nil
}

func (s *Service) resolveEntitlement(ctx context.Context, organizationID, personID domain.ID, session domain.ScheduleSession, makeupID *domain.ID) (domain.ReservationEntitlement, error) {
	if s.access != nil {
		request := domain.MembershipAccessRequest{OrganizationID: organizationID, UnitID: session.UnitID, PersonID: personID, TargetKind: session.TargetKind, TargetID: session.TargetID, OccurredAt: session.StartsAt, Purpose: domain.MembershipAccessBooking}
		if makeupID != nil {
			source, err := s.repo.GetMakeupSource(ctx, organizationID, session.UnitID, personID, *makeupID)
			if err != nil || source == nil {
				return domain.ReservationEntitlement{}, ErrMakeupInvalid
			}
			request.PreferredEnrollment, request.IgnoreUsage = source.EnrollmentID, true
		}
		decision, err := s.access.Evaluate(ctx, request)
		if err != nil {
			return domain.ReservationEntitlement{}, err
		}
		if decision.Outcome != domain.MembershipAccessAllowed {
			switch decision.ReasonCode {
			case domain.MembershipAccessReasonLimitReached:
				return domain.ReservationEntitlement{}, ErrLimitReached
			case domain.MembershipAccessReasonContractPending:
				return domain.ReservationEntitlement{}, ErrContractPending
			case domain.MembershipAccessReasonHealthFormPending:
				return domain.ReservationEntitlement{}, ErrHealthFormPending
			case domain.MembershipAccessReasonPaymentRequired:
				return domain.ReservationEntitlement{}, ErrPaymentRequired
			case domain.MembershipAccessReasonPaymentOverdue:
				return domain.ReservationEntitlement{}, ErrPaymentOverdue
			}
			return domain.ReservationEntitlement{}, ErrNotEligible
		}
		return domain.ReservationEntitlement{EnrollmentID: decision.EnrollmentID, EntitlementID: decision.EntitlementID}, nil
	}
	entitlements, err := s.eligibility.ResolveReservationEntitlements(ctx, organizationID, session.UnitID, personID, session.TargetKind, session.TargetID, session.StartsAt)
	if err != nil || len(entitlements) == 0 {
		if err == nil {
			err = ErrNotEligible
		}
		return domain.ReservationEntitlement{}, err
	}
	return s.selectEntitlement(ctx, organizationID, personID, session, entitlements, makeupID)
}

func (s *Service) selectEntitlement(ctx context.Context, organizationID, personID domain.ID, session domain.ScheduleSession, values []domain.ReservationEntitlement, makeupID *domain.ID) (domain.ReservationEntitlement, error) {
	if makeupID != nil {
		source, err := s.repo.GetMakeupSource(ctx, organizationID, session.UnitID, personID, *makeupID)
		if err != nil || source == nil {
			return domain.ReservationEntitlement{}, ErrMakeupInvalid
		}
		for _, item := range values {
			if item.EnrollmentID == source.EnrollmentID {
				return item, nil
			}
		}
		return domain.ReservationEntitlement{}, ErrMakeupInvalid
	}
	limited := false
	for _, item := range values {
		from, until := entitlementWindow(session.StartsAt, session.Timezone, item.PeriodUnit)
		used, err := s.repo.CountConsumed(ctx, organizationID, personID, item.EnrollmentID, item.EntitlementID, from, until)
		if err != nil {
			return item, err
		}
		if item.DailyLimit != nil {
			dayFrom, dayUntil := entitlementWindow(session.StartsAt, session.Timezone, ptrPeriod(domain.MembershipPeriodDay))
			daily, err := s.repo.CountConsumed(ctx, organizationID, personID, item.EnrollmentID, item.EntitlementID, dayFrom, dayUntil)
			if err != nil {
				return item, err
			}
			if daily >= *item.DailyLimit {
				limited = true
				continue
			}
		}
		if item.AccessType != domain.MembershipAccessOpen && item.Quantity != nil && used >= *item.Quantity {
			limited = true
			continue
		}
		return item, nil
	}
	if limited {
		return domain.ReservationEntitlement{}, ErrLimitReached
	}
	return domain.ReservationEntitlement{}, ErrNotEligible
}

func ptrPeriod(value domain.MembershipPeriodUnit) *domain.MembershipPeriodUnit { return &value }

func entitlementWindow(at time.Time, timezone string, period *domain.MembershipPeriodUnit) (time.Time, time.Time) {
	if period == nil {
		return time.Unix(0, 0).UTC(), time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		location = time.UTC
	}
	local := at.In(location)
	var start time.Time
	switch *period {
	case domain.MembershipPeriodWeek:
		offset := (int(local.Weekday()) + 6) % 7
		start = time.Date(local.Year(), local.Month(), local.Day()-offset, 0, 0, 0, 0, location)
	case domain.MembershipPeriodMonth:
		start = time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, location)
	default:
		start = time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	}
	var end time.Time
	if *period == domain.MembershipPeriodWeek {
		end = start.AddDate(0, 0, 7)
	} else if *period == domain.MembershipPeriodMonth {
		end = start.AddDate(0, 1, 0)
	} else {
		end = start.AddDate(0, 0, 1)
	}
	return start.UTC(), end.UTC()
}

func (s *Service) Cancel(ctx context.Context, actorID, unitID, reservationID domain.ID, key string, input CancelInput) (*domain.Reservation, *domain.Reservation, bool, error) {
	return s.transition(ctx, actorID, unitID, reservationID, key, "cancel", input, false, nil, "", "")
}

func (s *Service) CancelForAthlete(ctx context.Context, scope domain.AthletePortalScope, athleteID, sessionID, reservationID domain.ID, key string) (*domain.Reservation, *domain.Reservation, bool, error) {
	if scope.OrganizationID == "" || scope.UnitID == "" || scope.PersonID == "" || athleteID == "" || sessionID == "" {
		return nil, nil, false, ErrNotFound
	}
	return s.transition(ctx, "", scope.UnitID, reservationID, key, "cancel", CancelInput{Reason: "Cancelada pelo aluno"}, false, &scope, athleteID, sessionID)
}

func (s *Service) MarkNoShow(ctx context.Context, actorID, unitID, reservationID domain.ID, key string, expectedRevision int) (*domain.Reservation, bool, error) {
	value, _, fresh, err := s.transition(ctx, actorID, unitID, reservationID, key, "no_show", CancelInput{ExpectedRevision: expectedRevision}, true, nil, "", "")
	return value, fresh, err
}

func (s *Service) transition(ctx context.Context, actorID, unitID, reservationID domain.ID, key, commandName string, input CancelInput, noShow bool, athleteScope *domain.AthletePortalScope, athleteID, sessionID domain.ID) (*domain.Reservation, *domain.Reservation, bool, error) {
	var unit *domain.SchedulingUnitContext
	var err error
	if athleteScope != nil {
		unit = &domain.SchedulingUnitContext{OrganizationID: athleteScope.OrganizationID, UnitID: athleteScope.UnitID, Authorized: true}
	} else {
		unit, err = s.authorize(ctx, actorID, unitID)
		if err != nil {
			return nil, nil, false, err
		}
	}
	if reservationID == "" || (athleteScope == nil && input.ExpectedRevision < 1) || !validKey(key) || (!noShow && (len(input.Reason) < 3 || len(input.Reason) > 240)) {
		return nil, nil, false, ErrInvalid
	}
	fingerprintInput := map[string]any{"command": commandName, "unit_id": unitID, "reservation_id": reservationID, "revision": input.ExpectedRevision, "reason": input.Reason}
	if athleteScope != nil {
		fingerprintInput["person_id"] = athleteScope.PersonID
		fingerprintInput["session_id"] = sessionID
	}
	fingerprint := hash(fingerprintInput)
	var result, promoted *domain.Reservation
	fresh := false
	err = s.tx.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.repo.LockCommand(tx, unit.OrganizationID, commandName, key); err != nil {
			return err
		}
		cmd, err := s.repo.FindCommand(tx, unit.OrganizationID, commandName, key)
		if err != nil {
			return err
		}
		if cmd != nil {
			if cmd.Fingerprint != fingerprint {
				return ErrIdempotencyReused
			}
			result, err = reservationFromResponse(cmd.ResponseData["reservation"])
			if err != nil {
				return err
			}
			if cmd.ResponseData["promoted"] != nil {
				promoted, err = reservationFromResponse(cmd.ResponseData["promoted"])
				if err != nil {
					return err
				}
			}
			return nil
		}
		current, err := s.repo.Get(tx, unit.OrganizationID, unitID, reservationID, true)
		if err != nil {
			return err
		}
		if athleteScope != nil && (current.PersonID != athleteScope.PersonID || current.SessionID != sessionID) {
			return ErrNotFound
		}
		if athleteScope == nil && current.Revision != input.ExpectedRevision {
			return ErrRevisionConflict
		}
		if err := s.repo.LockSession(tx, unit.OrganizationID, current.SessionID); err != nil {
			return err
		}
		session, err := s.sessions.GetPublishedSession(tx, unit.OrganizationID, unitID, current.SessionID)
		if err != nil || session == nil {
			return ErrNotFound
		}
		if athleteScope != nil && (session.Status != domain.ScheduleSessionScheduled || !session.StartsAt.After(s.now())) {
			return ErrAlreadyFinal
		}
		wasConfirmed := current.Status == domain.ReservationConfirmed
		if noShow {
			if current.Status != domain.ReservationConfirmed || s.now().Before(session.StartsAt) {
				return ErrAlreadyFinal
			}
			current.Status = domain.ReservationNoShow
		} else {
			if current.Status != domain.ReservationConfirmed && current.Status != domain.ReservationWaitlisted {
				return ErrAlreadyFinal
			}
			current.Status, current.CancelledReason = domain.ReservationCancelled, input.Reason
		}
		oldRevision := current.Revision
		current.Revision++
		current.UpdatedAt = s.now()
		if err := s.repo.Update(tx, current, oldRevision); err != nil {
			return err
		}
		action := "reservation.cancelled"
		if noShow {
			action = "reservation.no_show"
		}
		if err := s.repo.CreateAuditEvent(tx, &domain.ReservationAuditEvent{ID: domain.ID(uuid.NewString()), OrganizationID: unit.OrganizationID, UnitID: unitID, SessionID: current.SessionID, ReservationID: current.ID, PersonID: current.PersonID, ActorUserID: actorID, ActorAthleteID: athleteID, Action: action, Reason: input.Reason, CreatedAt: s.now()}); err != nil {
			return err
		}
		if !noShow && wasConfirmed && session.StartsAt.After(s.now()) {
			promoted, err = s.promoteFirstEligible(tx, unit.OrganizationID, unitID, actorID, athleteID, *session)
			if err != nil {
				return err
			}
		}
		response := map[string]any{"reservation": current}
		if promoted != nil {
			response["promoted"] = promoted
		}
		if err := s.repo.CreateCommand(tx, &domain.ReservationCommandRequest{ID: domain.ID(uuid.NewString()), OrganizationID: unit.OrganizationID, Command: commandName, IdempotencyKey: key, Fingerprint: fingerprint, ReservationID: current.ID, ResponseData: response, CreatedAt: s.now()}); err != nil {
			return err
		}
		result, fresh = current, true
		return nil
	})
	if err != nil {
		return nil, nil, false, err
	}
	return result, promoted, fresh, nil
}

func (s *Service) promoteFirstEligible(ctx context.Context, organizationID, unitID, actorID, athleteID domain.ID, session domain.ScheduleSession) (*domain.Reservation, error) {
	for {
		candidate, err := s.repo.FirstWaitlisted(ctx, organizationID, unitID, session.ID)
		if err != nil || candidate == nil {
			return candidate, err
		}
		eligible := false
		if selected, selectErr := s.resolveEntitlement(ctx, organizationID, candidate.PersonID, session, candidate.MakeupForReservationID); selectErr == nil {
			candidate.EnrollmentID, candidate.EntitlementID, eligible = selected.EnrollmentID, selected.EntitlementID, true
		}
		previous := candidate.Revision
		candidate.Revision++
		candidate.UpdatedAt = s.now()
		if eligible {
			candidate.Status = domain.ReservationConfirmed
			if err := s.repo.Update(ctx, candidate, previous); err != nil {
				return nil, err
			}
			if err := s.repo.CreateAuditEvent(ctx, &domain.ReservationAuditEvent{ID: domain.ID(uuid.NewString()), OrganizationID: organizationID, UnitID: unitID, SessionID: session.ID, ReservationID: candidate.ID, PersonID: candidate.PersonID, ActorUserID: actorID, ActorAthleteID: athleteID, Action: "reservation.promoted", CreatedAt: s.now()}); err != nil {
				return nil, err
			}
			return candidate, nil
		}
		candidate.Status, candidate.CancelledReason = domain.ReservationCancelled, "Elegibilidade expirada durante a promoção da fila"
		if err := s.repo.Update(ctx, candidate, previous); err != nil {
			return nil, err
		}
		if err := s.repo.CreateAuditEvent(ctx, &domain.ReservationAuditEvent{ID: domain.ID(uuid.NewString()), OrganizationID: organizationID, UnitID: unitID, SessionID: session.ID, ReservationID: candidate.ID, PersonID: candidate.PersonID, ActorUserID: actorID, ActorAthleteID: athleteID, Action: "reservation.cancelled", Reason: candidate.CancelledReason, CreatedAt: s.now()}); err != nil {
			return nil, err
		}
	}
}

func (s *Service) ListSession(ctx context.Context, actorID, unitID, sessionID domain.ID) ([]domain.Reservation, domain.ReservationSummary, error) {
	unit, err := s.authorize(ctx, actorID, unitID)
	if err != nil {
		return nil, domain.ReservationSummary{}, err
	}
	session, err := s.sessions.GetPublishedSession(ctx, unit.OrganizationID, unitID, sessionID)
	if err != nil || session == nil {
		return nil, domain.ReservationSummary{}, ErrNotFound
	}
	items, err := s.repo.ListSession(ctx, unit.OrganizationID, unitID, sessionID)
	if err != nil {
		return nil, domain.ReservationSummary{}, err
	}
	confirmed, waitlisted := 0, 0
	for i := range items {
		if items[i].Status == domain.ReservationConfirmed {
			confirmed++
		}
		if items[i].Status == domain.ReservationWaitlisted {
			waitlisted++
			p := waitlisted
			items[i].WaitlistPosition = &p
		}
	}
	available := max(0, session.Capacity-confirmed)
	if session.Status == domain.ScheduleSessionCancelled {
		available = 0
	}
	return items, domain.ReservationSummary{SessionID: sessionID, Capacity: session.Capacity, ConfirmedCount: confirmed, WaitlistCount: waitlisted, AvailableCount: available}, nil
}

func (s *Service) SessionSummaries(ctx context.Context, actorID, unitID domain.ID, sessionIDs []domain.ID) ([]domain.ReservationSummary, error) {
	if len(sessionIDs) == 0 || len(sessionIDs) > 100 {
		return nil, ErrInvalid
	}
	seen := map[domain.ID]struct{}{}
	result := make([]domain.ReservationSummary, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		if sessionID == "" {
			return nil, ErrInvalid
		}
		if _, exists := seen[sessionID]; exists {
			continue
		}
		seen[sessionID] = struct{}{}
		_, summary, err := s.ListSession(ctx, actorID, unitID, sessionID)
		if err != nil {
			return nil, err
		}
		result = append(result, summary)
	}
	return result, nil
}

func (s *Service) ListPerson(ctx context.Context, actorID, unitID, personID domain.ID, from, until time.Time) ([]domain.Reservation, error) {
	unit, err := s.authorize(ctx, actorID, unitID)
	if err != nil {
		return nil, err
	}
	if !until.After(from) || until.Sub(from) > 366*24*time.Hour {
		return nil, ErrInvalid
	}
	return s.repo.ListPerson(ctx, unit.OrganizationID, unitID, personID, from, until)
}

// CancelSessionReservations is the public reservations port consumed by scheduling.
// It is safe inside the scheduling transaction and never promotes a queue for a cancelled session.
func (s *Service) CancelSessionReservations(ctx context.Context, organizationID, unitID, sessionID, actorID domain.ID, reason string) error {
	if err := s.repo.LockSession(ctx, organizationID, sessionID); err != nil {
		return err
	}
	items, err := s.repo.ListSession(ctx, organizationID, unitID, sessionID)
	if err != nil {
		return err
	}
	runes := []rune(reason)
	if len(runes) > 220 {
		reason = string(runes[:220])
	}
	reason = "Sessão cancelada: " + reason
	for i := range items {
		item := &items[i]
		if item.Status != domain.ReservationConfirmed && item.Status != domain.ReservationWaitlisted {
			continue
		}
		previous := item.Revision
		item.Status = domain.ReservationCancelled
		item.CancelledReason = reason
		item.Revision++
		item.UpdatedAt = s.now()
		if err := s.repo.Update(ctx, item, previous); err != nil {
			return err
		}
		if err := s.repo.CreateAuditEvent(ctx, &domain.ReservationAuditEvent{ID: domain.ID(uuid.NewString()), OrganizationID: organizationID, UnitID: unitID, SessionID: sessionID, ReservationID: item.ID, PersonID: item.PersonID, ActorUserID: actorID, Action: "reservation.cancelled", Reason: reason, CreatedAt: s.now()}); err != nil {
			return err
		}
	}
	return nil
}

func validKey(value string) bool {
	return regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`).MatchString(value)
}
func hash(value any) string {
	payload, _ := json.Marshal(value)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
func reservationFromResponse(value any) (*domain.Reservation, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var result domain.Reservation
	if err = json.Unmarshal(payload, &result); err != nil {
		return nil, err
	}
	if result.ID == "" {
		return nil, ErrNotFound
	}
	return &result, nil
}
func SortWaitlist(values []domain.Reservation) {
	sort.SliceStable(values, func(i, j int) bool { return values[i].CreatedAt.Before(values[j].CreatedAt) })
}
