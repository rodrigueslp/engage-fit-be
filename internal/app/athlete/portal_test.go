package athlete

import (
	"context"
	"testing"
	"time"

	"boxengage/backend/internal/domain"
)

type portalRepositoryStub struct {
	units []domain.AthletePortalUnit
	scope *domain.AthletePortalScope
	from  time.Time
	until time.Time
}

func (s *portalRepositoryStub) LoadAthletePortal(_ context.Context, _ domain.ID, from, until time.Time) ([]domain.AthletePortalUnit, error) {
	s.from, s.until = from, until
	return s.units, nil
}
func (s *portalRepositoryStub) ResolveAthleteScope(context.Context, domain.ID, domain.ID) (*domain.AthletePortalScope, error) {
	return s.scope, nil
}

type athleteBookerStub struct {
	scope                               domain.AthletePortalScope
	athleteID, sessionID, reservationID domain.ID
}

func (s *athleteBookerStub) CancelForAthlete(_ context.Context, scope domain.AthletePortalScope, athleteID, sessionID, reservationID domain.ID, _ string) (*domain.Reservation, *domain.Reservation, bool, error) {
	s.scope, s.athleteID, s.sessionID, s.reservationID = scope, athleteID, sessionID, reservationID
	return &domain.Reservation{ID: reservationID, PersonID: scope.PersonID, Status: domain.ReservationCancelled}, nil, true, nil
}

func (s *athleteBookerStub) BookForAthlete(_ context.Context, scope domain.AthletePortalScope, athleteID, sessionID domain.ID, _ string) (*domain.Reservation, bool, error) {
	s.scope, s.athleteID, s.sessionID = scope, athleteID, sessionID
	return &domain.Reservation{ID: "reservation", PersonID: scope.PersonID}, true, nil
}

func TestAthleteCancellationRequiresExplicitMembershipScope(t *testing.T) {
	portal := &portalRepositoryStub{}
	booker := &athleteBookerStub{}
	service := NewService(&athleteRepositoryStub{}, studentRepositoryStub{}, passwordServiceStub{}).WithPortalRepository(portal).WithReservationBooker(booker)
	if _, _, err := service.CancelClassReservation(context.Background(), "athlete", "foreign-membership", "session", "reservation", "cancel-key-001"); err != ErrPortalMembershipUnavailable {
		t.Fatalf("foreign membership: %v", err)
	}
	portal.scope = &domain.AthletePortalScope{MembershipID: "membership", OrganizationID: "org", UnitID: "unit", PersonID: "person"}
	value, fresh, err := service.CancelClassReservation(context.Background(), "athlete", "membership", "session", "reservation", "cancel-key-002")
	if err != nil || !fresh || value.Status != domain.ReservationCancelled || booker.reservationID != "reservation" || booker.scope.PersonID != "person" {
		t.Fatalf("scoped cancellation: value=%+v fresh=%v err=%v", value, fresh, err)
	}
}

func TestAthleteBookingRequiresExplicitMembershipScope(t *testing.T) {
	portal := &portalRepositoryStub{}
	booker := &athleteBookerStub{}
	service := NewService(&athleteRepositoryStub{}, studentRepositoryStub{}, passwordServiceStub{}).WithPortalRepository(portal).WithReservationBooker(booker)
	if _, _, err := service.BookClass(context.Background(), "athlete", "foreign-membership", "session", "booking-key-001"); err != ErrPortalMembershipUnavailable {
		t.Fatalf("foreign membership: %v", err)
	}
	portal.scope = &domain.AthletePortalScope{MembershipID: "membership", OrganizationID: "org", UnitID: "unit", PersonID: "person"}
	value, fresh, err := service.BookClass(context.Background(), "athlete", "membership", "session", "booking-key-002")
	if err != nil || !fresh || value.PersonID != "person" || booker.athleteID != "athlete" {
		t.Fatalf("scoped booking: value=%+v fresh=%v err=%v", value, fresh, err)
	}
}

func TestManualPersonalBestRequiresLinkedMembershipAndValidMark(t *testing.T) {
	portal := &portalRepositoryStub{}
	repo := &athleteRepositoryStub{}
	service := NewService(repo, studentRepositoryStub{}, passwordServiceStub{}).WithPortalRepository(portal)
	service.now = func() time.Time { return time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC) }
	input := ManualRecordInput{MembershipID: "membership", MovementName: "Agachamento livre", Metric: "load", BestValue: 90.5, AchievedOn: "2026-09-25"}
	if _, err := service.RecordPersonalBest(context.Background(), "athlete", input); err != ErrPortalMembershipUnavailable {
		t.Fatalf("foreign membership: %v", err)
	}
	portal.scope = &domain.AthletePortalScope{MembershipID: "membership", PersonID: "person"}
	if _, err := service.RecordPersonalBest(context.Background(), "athlete", ManualRecordInput{MembershipID: "membership", MovementName: "Agachamento", Metric: "reps", BestValue: 5.5, AchievedOn: "2026-09-25"}); err != ErrInvalidInput {
		t.Fatalf("fractional reps: %v", err)
	}
	value, err := service.RecordPersonalBest(context.Background(), "athlete", input)
	if err != nil || value.BestValue != 90.5 || repo.manualRecord == nil || repo.manualMembership != "membership" {
		t.Fatalf("manual PR: value=%+v saved=%+v err=%v", value, repo.manualRecord, err)
	}
}

func TestPortalBuildsActionableNoticesWithoutLeakingOtherMemberships(t *testing.T) {
	now := time.Date(2026, 9, 16, 15, 0, 0, 0, time.UTC)
	position := 2
	portal := &portalRepositoryStub{units: []domain.AthletePortalUnit{{
		MembershipID: "membership-1", UnitName: "Box Centro",
		Plan:     &domain.AthletePlanSummary{EnrollmentID: "enrollment-1", Status: "active"},
		Contract: &domain.AthleteContractSummary{ID: "contract-1", Status: "pending_acceptance"},
		Billing:  domain.AthleteBillingSummary{FinancialStatus: "overdue"},
		Agenda:   []domain.AthleteScheduleSession{{ID: "session-1", TargetName: "Cross Training", ReservationStatus: "waitlisted", WaitlistPosition: &position}},
	}}}
	service := NewService(&athleteRepositoryStub{}, studentRepositoryStub{}, passwordServiceStub{}).WithPortalRepository(portal)
	service.now = func() time.Time { return now }

	result, err := service.Portal(context.Background(), "athlete-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Units) != 1 || len(result.Notices) != 3 {
		t.Fatalf("unexpected portal: %+v", result)
	}
	if result.Notices[0].Kind != "contract" || result.Notices[1].Kind != "billing" || result.Notices[2].Kind != "schedule" {
		t.Fatalf("unexpected notices: %+v", result.Notices)
	}
	if !portal.from.Equal(now.Add(-24*time.Hour)) || !portal.until.Equal(now.Add(30*24*time.Hour)) {
		t.Fatalf("unexpected agenda window: %v - %v", portal.from, portal.until)
	}
}

func TestPortalWithoutReaderReturnsStableEmptyPayload(t *testing.T) {
	now := time.Date(2026, 9, 16, 15, 0, 0, 0, time.UTC)
	service := NewService(&athleteRepositoryStub{}, studentRepositoryStub{}, passwordServiceStub{})
	service.now = func() time.Time { return now }

	result, err := service.Portal(context.Background(), "athlete-1")
	if err != nil || len(result.Units) != 0 || len(result.Notices) != 0 || !result.GeneratedAt.Equal(now) {
		t.Fatalf("unexpected empty portal: result=%+v err=%v", result, err)
	}
}
