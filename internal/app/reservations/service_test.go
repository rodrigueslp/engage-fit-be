package reservations

import (
	"context"
	"testing"
	"time"

	"boxengage/backend/internal/domain"
)

type fakeTx struct{}

func (fakeTx) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type fakeContexts struct{}

func (fakeContexts) ResolveSchedulingUnitContext(context.Context, domain.ID, domain.ID) (*domain.SchedulingUnitContext, error) {
	return &domain.SchedulingUnitContext{OrganizationID: "org", UnitID: "unit", Role: "OWNER", Authorized: true}, nil
}

type fakeSessions struct{ session domain.ScheduleSession }

func (f fakeSessions) GetPublishedSession(context.Context, domain.ID, domain.ID, domain.ID) (*domain.ScheduleSession, error) {
	copy := f.session
	return &copy, nil
}
func (f fakeSessions) ListPublishedSessions(context.Context, domain.ID, domain.ID, time.Time, time.Time) ([]domain.ScheduleSession, error) {
	return []domain.ScheduleSession{f.session}, nil
}
func (fakeSessions) ListSessionEvents(context.Context, domain.ID, domain.ID, int64, int) ([]domain.SessionEventV1, error) {
	return nil, nil
}

type fakeEligibility struct{}

func (fakeEligibility) ResolveReservationEntitlements(_ context.Context, _, _, person domain.ID, _ domain.ScheduleTargetKind, _ domain.ID, _ time.Time) ([]domain.ReservationEntitlement, error) {
	q := 10
	return []domain.ReservationEntitlement{{EnrollmentID: "enroll-" + person, EntitlementID: "right-" + person, AccessType: domain.MembershipAccessClassPack, Quantity: &q}}, nil
}

type fakePeople struct{}

func (fakePeople) ResolveReservationPerson(_ context.Context, _, _, person domain.ID) (string, error) {
	return "Pessoa " + string(person), nil
}

type reservationAccessStub struct{ reason domain.MembershipAccessReason }

func (s reservationAccessStub) Evaluate(context.Context, domain.MembershipAccessRequest) (*domain.MembershipAccessDecision, error) {
	return &domain.MembershipAccessDecision{Outcome: domain.MembershipAccessDenied, ReasonCode: s.reason}, nil
}

type fakeRepo struct {
	reservations map[domain.ID]*domain.Reservation
	commands     map[string]*domain.ReservationCommandRequest
	audit        *domain.ReservationAuditEvent
	audits       []domain.ReservationAuditEvent
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{reservations: map[domain.ID]*domain.Reservation{}, commands: map[string]*domain.ReservationCommandRequest{}}
}
func (*fakeRepo) LockCommand(context.Context, domain.ID, string, string) error { return nil }
func (f *fakeRepo) FindCommand(_ context.Context, _ domain.ID, command, key string) (*domain.ReservationCommandRequest, error) {
	return f.commands[command+":"+key], nil
}
func (f *fakeRepo) CreateCommand(_ context.Context, v *domain.ReservationCommandRequest) error {
	copy := *v
	f.commands[v.Command+":"+v.IdempotencyKey] = &copy
	return nil
}
func (*fakeRepo) LockSession(context.Context, domain.ID, domain.ID) error { return nil }
func (f *fakeRepo) Get(_ context.Context, _, _, id domain.ID, _ bool) (*domain.Reservation, error) {
	v := f.reservations[id]
	if v == nil {
		return nil, ErrNotFound
	}
	copy := *v
	return &copy, nil
}
func (f *fakeRepo) FindActive(_ context.Context, _, _, session, person domain.ID) (*domain.Reservation, error) {
	for _, v := range f.reservations {
		if v.SessionID == session && v.PersonID == person && (v.Status == domain.ReservationConfirmed || v.Status == domain.ReservationWaitlisted) {
			copy := *v
			return &copy, nil
		}
	}
	return nil, nil
}
func (f *fakeRepo) ListSession(_ context.Context, _, _, session domain.ID) ([]domain.Reservation, error) {
	out := []domain.Reservation{}
	for _, v := range f.reservations {
		if v.SessionID == session {
			out = append(out, *v)
		}
	}
	SortWaitlist(out)
	return out, nil
}
func (f *fakeRepo) ListPerson(_ context.Context, _, _, person domain.ID, _, _ time.Time) ([]domain.Reservation, error) {
	out := []domain.Reservation{}
	for _, v := range f.reservations {
		if v.PersonID == person {
			out = append(out, *v)
		}
	}
	return out, nil
}
func (f *fakeRepo) CountSession(_ context.Context, _, _, session domain.ID, status domain.ReservationStatus) (int, error) {
	n := 0
	for _, v := range f.reservations {
		if v.SessionID == session && v.Status == status {
			n++
		}
	}
	return n, nil
}
func (f *fakeRepo) CountConsumed(_ context.Context, _, person, _, right domain.ID, _, _ time.Time) (int, error) {
	n := 0
	for _, v := range f.reservations {
		if v.PersonID == person && v.EntitlementID == right && (v.Status == domain.ReservationConfirmed || v.Status == domain.ReservationNoShow) {
			n++
		}
	}
	return n, nil
}
func (f *fakeRepo) Create(_ context.Context, v *domain.Reservation) error {
	copy := *v
	f.reservations[v.ID] = &copy
	return nil
}
func (f *fakeRepo) Update(_ context.Context, v *domain.Reservation, revision int) error {
	stored := f.reservations[v.ID]
	if stored == nil {
		return ErrNotFound
	}
	if stored.Revision != revision {
		return ErrRevisionConflict
	}
	copy := *v
	f.reservations[v.ID] = &copy
	return nil
}
func (f *fakeRepo) FirstWaitlisted(_ context.Context, _, _, session domain.ID) (*domain.Reservation, error) {
	var first *domain.Reservation
	for _, v := range f.reservations {
		if v.SessionID == session && v.Status == domain.ReservationWaitlisted && (first == nil || v.CreatedAt.Before(first.CreatedAt)) {
			copy := *v
			first = &copy
		}
	}
	return first, nil
}
func (f *fakeRepo) GetMakeupSource(_ context.Context, _, _, person, id domain.ID) (*domain.Reservation, error) {
	v := f.reservations[id]
	if v == nil || v.PersonID != person || v.Status != domain.ReservationNoShow {
		return nil, ErrMakeupInvalid
	}
	for _, child := range f.reservations {
		if child.MakeupForReservationID != nil && *child.MakeupForReservationID == id && child.Status != domain.ReservationCancelled {
			return nil, ErrMakeupInvalid
		}
	}
	copy := *v
	return &copy, nil
}
func (f *fakeRepo) CreateAuditEvent(_ context.Context, event *domain.ReservationAuditEvent) error {
	copy := *event
	f.audit = &copy
	f.audits = append(f.audits, copy)
	return nil
}

func TestAthleteBookingUsesLinkedPersonAndAuditsAthlete(t *testing.T) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	repo := newFakeRepo()
	session := domain.ScheduleSession{ID: "session", UnitID: "unit", Status: domain.ScheduleSessionScheduled, TargetKind: domain.ScheduleTargetModality, TargetID: "modality", Capacity: 1, StartsAt: now.Add(time.Hour)}
	service := NewService(repo, fakeSessions{session}, fakeEligibility{}, fakePeople{}, fakeContexts{}, fakeTx{}, WithClock(func() time.Time { return now }))
	scope := domain.AthletePortalScope{MembershipID: "membership", OrganizationID: "org", UnitID: "unit", PersonID: "person"}
	value, fresh, err := service.BookForAthlete(context.Background(), scope, "athlete", "session", "athlete-book-001")
	if err != nil || !fresh || value.PersonID != "person" || value.Status != domain.ReservationConfirmed {
		t.Fatalf("athlete booking: value=%+v fresh=%v err=%v", value, fresh, err)
	}
	if repo.audit == nil || repo.audit.ActorAthleteID != "athlete" || repo.audit.ActorUserID != "" {
		t.Fatalf("audit actor: %+v", repo.audit)
	}
	if _, _, err := service.BookForAthlete(context.Background(), domain.AthletePortalScope{}, "athlete", "session", "athlete-book-002"); err != ErrNotFound {
		t.Fatalf("expected missing scope rejection, got %v", err)
	}
}

func TestAthleteCancellationChecksOwnershipAndPromotesWaitlist(t *testing.T) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	repo := newFakeRepo()
	session := domain.ScheduleSession{ID: "session", UnitID: "unit", Status: domain.ScheduleSessionScheduled, TargetKind: domain.ScheduleTargetModality, TargetID: "modality", Capacity: 1, StartsAt: now.Add(time.Hour)}
	service := NewService(repo, fakeSessions{session}, fakeEligibility{}, fakePeople{}, fakeContexts{}, fakeTx{}, WithClock(func() time.Time { return now }))
	scope := domain.AthletePortalScope{MembershipID: "membership", OrganizationID: "org", UnitID: "unit", PersonID: "person"}
	first, _, err := service.BookForAthlete(context.Background(), scope, "athlete", "session", "athlete-book-001")
	if err != nil {
		t.Fatal(err)
	}
	otherScope := domain.AthletePortalScope{MembershipID: "other", OrganizationID: "org", UnitID: "unit", PersonID: "other"}
	second, _, err := service.BookForAthlete(context.Background(), otherScope, "other-athlete", "session", "athlete-book-002")
	if err != nil || second.Status != domain.ReservationWaitlisted {
		t.Fatalf("waitlist: %+v %v", second, err)
	}
	if _, _, _, err := service.CancelForAthlete(context.Background(), otherScope, "other-athlete", "session", first.ID, "cancel-wrong-person"); err != ErrNotFound {
		t.Fatalf("foreign reservation: %v", err)
	}
	if _, _, _, err := service.CancelForAthlete(context.Background(), scope, "athlete", "other-session", first.ID, "cancel-wrong-session"); err != ErrNotFound {
		t.Fatalf("foreign session: %v", err)
	}
	cancelled, promoted, fresh, err := service.CancelForAthlete(context.Background(), scope, "athlete", "session", first.ID, "cancel-own-reservation")
	if err != nil || !fresh || cancelled.Status != domain.ReservationCancelled || promoted == nil || promoted.ID != second.ID || promoted.Status != domain.ReservationConfirmed {
		t.Fatalf("cancel and promote: %+v %+v %v", cancelled, promoted, err)
	}
	if len(repo.audits) < 4 || repo.audits[len(repo.audits)-2].ActorAthleteID != "athlete" || repo.audits[len(repo.audits)-2].Action != "reservation.cancelled" {
		t.Fatalf("missing athlete audit: %+v", repo.audits)
	}
	if _, _, replayFresh, err := service.CancelForAthlete(context.Background(), scope, "athlete", "session", first.ID, "cancel-own-reservation"); err != nil || replayFresh {
		t.Fatalf("replay: %v fresh=%v", err, replayFresh)
	}
	thirdScope := domain.AthletePortalScope{MembershipID: "third", OrganizationID: "org", UnitID: "unit", PersonID: "third"}
	third, _, err := service.BookForAthlete(context.Background(), thirdScope, "third-athlete", "session", "athlete-book-003")
	if err != nil || third.Status != domain.ReservationWaitlisted {
		t.Fatalf("second waitlist: %+v %v", third, err)
	}
	waitlistCancelled, promoted, _, err := service.CancelForAthlete(context.Background(), thirdScope, "third-athlete", "session", third.ID, "cancel-waitlist")
	if err != nil || waitlistCancelled.Status != domain.ReservationCancelled || promoted != nil {
		t.Fatalf("waitlist cancellation: %+v %+v %v", waitlistCancelled, promoted, err)
	}
	now = session.StartsAt
	if _, _, _, err := service.CancelForAthlete(context.Background(), otherScope, "other-athlete", "session", second.ID, "cancel-after-start"); err != ErrAlreadyFinal {
		t.Fatalf("late cancellation: %v", err)
	}
}

func TestBookingWaitlistPromotionNoShowAndMakeup(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	repo := newFakeRepo()
	session := domain.ScheduleSession{ID: "session", UnitID: "unit", Status: domain.ScheduleSessionScheduled, TargetKind: domain.ScheduleTargetModality, TargetID: "modality", Capacity: 1, StartsAt: now.Add(time.Hour), Timezone: "America/Sao_Paulo"}
	service := NewService(repo, fakeSessions{session}, fakeEligibility{}, fakePeople{}, fakeContexts{}, fakeTx{}, WithClock(func() time.Time { return now }))
	first, fresh, err := service.Book(context.Background(), "actor", "unit", "booking-0001", BookInput{SessionID: "session", PersonID: "one"})
	if err != nil || !fresh || first.Status != domain.ReservationConfirmed {
		t.Fatalf("first booking = %#v, %v", first, err)
	}
	replay, replayFresh, err := service.Book(context.Background(), "actor", "unit", "booking-0001", BookInput{SessionID: "session", PersonID: "one"})
	if err != nil || replayFresh || replay.ID != first.ID {
		t.Fatalf("idempotent replay failed: %#v %v", replay, err)
	}
	second, _, err := service.Book(context.Background(), "actor", "unit", "booking-0002", BookInput{SessionID: "session", PersonID: "two"})
	if err != nil || second.Status != domain.ReservationWaitlisted {
		t.Fatalf("waitlist = %#v, %v", second, err)
	}
	cancelled, promoted, _, err := service.Cancel(context.Background(), "actor", "unit", first.ID, "cancel-0001", CancelInput{ExpectedRevision: 1, Reason: "mudança de plano"})
	if err != nil || cancelled.Status != domain.ReservationCancelled || promoted == nil || promoted.ID != second.ID || promoted.Status != domain.ReservationConfirmed {
		t.Fatalf("promotion failed: %#v %#v %v", cancelled, promoted, err)
	}
	_, replayPromotion, replayFresh, err := service.Cancel(context.Background(), "actor", "unit", first.ID, "cancel-0001", CancelInput{ExpectedRevision: 1, Reason: "mudança de plano"})
	if err != nil || replayFresh || replayPromotion == nil || replayPromotion.ID != second.ID {
		t.Fatalf("cancel replay lost promotion: %#v %v", replayPromotion, err)
	}
	now = session.StartsAt.Add(time.Minute)
	noShow, _, err := service.MarkNoShow(context.Background(), "actor", "unit", second.ID, "noshow-0001", 2)
	if err != nil || noShow.Status != domain.ReservationNoShow {
		t.Fatalf("no show failed: %#v %v", noShow, err)
	}
	now = session.StartsAt.Add(-time.Minute)
	makeup, _, err := service.Book(context.Background(), "actor", "unit", "makeup-0001", BookInput{SessionID: "session", PersonID: "two", MakeupForReservationID: &second.ID})
	if err != nil || makeup.MakeupForReservationID == nil {
		t.Fatalf("makeup failed: %#v %v", makeup, err)
	}
	_, _, err = service.Book(context.Background(), "actor", "unit", "makeup-0002", BookInput{SessionID: "session", PersonID: "two", MakeupForReservationID: &second.ID})
	if err != ErrAlreadyExists && err != ErrMakeupInvalid {
		t.Fatalf("expected reused makeup rejection, got %v", err)
	}
}

func TestBookingReturnsSpecificPrerequisiteErrors(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	session := domain.ScheduleSession{ID: "session", UnitID: "unit", Status: domain.ScheduleSessionScheduled, TargetKind: domain.ScheduleTargetModality, TargetID: "modality", Capacity: 10, StartsAt: now.Add(time.Hour), Timezone: "America/Sao_Paulo"}
	tests := []struct {
		reason domain.MembershipAccessReason
		want   error
	}{
		{domain.MembershipAccessReasonContractPending, ErrContractPending},
		{domain.MembershipAccessReasonHealthFormPending, ErrHealthFormPending},
		{domain.MembershipAccessReasonPaymentRequired, ErrPaymentRequired},
		{domain.MembershipAccessReasonPaymentOverdue, ErrPaymentOverdue},
	}
	for index, test := range tests {
		service := NewService(newFakeRepo(), fakeSessions{session}, fakeEligibility{}, fakePeople{}, fakeContexts{}, fakeTx{}, WithClock(func() time.Time { return now }), WithAccessEvaluator(reservationAccessStub{reason: test.reason}))
		_, _, err := service.Book(context.Background(), "actor", "unit", "prerequisite-000"+string(rune('1'+index)), BookInput{SessionID: "session", PersonID: "person"})
		if err != test.want {
			t.Fatalf("reason=%s err=%v want=%v", test.reason, err, test.want)
		}
	}
}
