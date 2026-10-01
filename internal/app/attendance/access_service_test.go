package attendance

import (
	"context"
	"errors"
	"testing"
	"time"

	"boxengage/backend/internal/domain"
)

type accessRepositoryStub struct {
	subject     domain.AttendanceSubject
	reservation *domain.Reservation
	decisions   []domain.AttendanceDecision
	records     []domain.AttendanceRecord
	commands    map[string]domain.AttendanceCommandRequest
}

func (s *accessRepositoryStub) LockAttendanceCommand(context.Context, domain.ID, string) error {
	return nil
}
func (s *accessRepositoryStub) FindAttendanceCommand(_ context.Context, _ domain.ID, key string) (*domain.AttendanceCommandRequest, error) {
	value, ok := s.commands[key]
	if !ok {
		return nil, nil
	}
	return &value, nil
}
func (s *accessRepositoryStub) ResolveAttendanceSubject(context.Context, domain.ID, domain.ID, domain.ID, time.Time) (*domain.AttendanceSubject, error) {
	value := s.subject
	return &value, nil
}
func (s *accessRepositoryStub) FindAttendanceReservation(context.Context, domain.ID, domain.ID, domain.ID, domain.ID) (*domain.Reservation, error) {
	return s.reservation, nil
}
func (s *accessRepositoryStub) CreateAttendanceDecision(_ context.Context, value *domain.AttendanceDecision) error {
	s.decisions = append(s.decisions, *value)
	return nil
}
func (s *accessRepositoryStub) CreateAttendanceRecord(_ context.Context, value *domain.AttendanceRecord, _ domain.Source, _ domain.ID) error {
	s.records = append(s.records, *value)
	return nil
}
func (s *accessRepositoryStub) CreateAttendanceCommand(_ context.Context, value *domain.AttendanceCommandRequest) error {
	if s.commands == nil {
		s.commands = map[string]domain.AttendanceCommandRequest{}
	}
	s.commands[value.IdempotencyKey] = *value
	return nil
}
func (s *accessRepositoryStub) GetAttendanceAdmission(_ context.Context, _ domain.ID, decisionID domain.ID) (*domain.AttendanceAdmission, error) {
	for _, d := range s.decisions {
		if d.ID == decisionID {
			return &domain.AttendanceAdmission{Decision: d}, nil
		}
	}
	return nil, nil
}
func (s *accessRepositoryStub) ListAttendanceForDate(context.Context, domain.ID, domain.ID, string) ([]domain.AttendanceAdmission, error) {
	return nil, nil
}

type accessContextStub struct{}

func (accessContextStub) ResolveSchedulingUnitContext(context.Context, domain.ID, domain.ID) (*domain.SchedulingUnitContext, error) {
	return &domain.SchedulingUnitContext{OrganizationID: "org", UnitID: "unit", Role: "OWNER", Authorized: true, Timezone: "America/Sao_Paulo"}, nil
}

type accessSessionsStub struct{ session *domain.ScheduleSession }

func (s accessSessionsStub) GetPublishedSession(context.Context, domain.ID, domain.ID, domain.ID) (*domain.ScheduleSession, error) {
	return s.session, nil
}
func (accessSessionsStub) ListPublishedSessions(context.Context, domain.ID, domain.ID, time.Time, time.Time) ([]domain.ScheduleSession, error) {
	return nil, nil
}
func (accessSessionsStub) ListSessionEvents(context.Context, domain.ID, domain.ID, int64, int) ([]domain.SessionEventV1, error) {
	return nil, nil
}

type accessTxStub struct{}

func (accessTxStub) WithinTransaction(ctx context.Context, operation func(context.Context) error) error {
	return operation(ctx)
}

type accessControlUnitStub struct{ mode domain.AccessControlMode }

func (s accessControlUnitStub) FindByID(context.Context, domain.ID) (*domain.Box, error) {
	return &domain.Box{AccessControlMode: s.mode}, nil
}

func TestAccessServiceRequiresManualAccessControlMode(t *testing.T) {
	repo := &accessRepositoryStub{subject: domain.AttendanceSubject{PersonID: "person", StudentID: "student", Name: "Ana", Source: domain.SourceBoxMember, RelationshipActive: true, ActiveEnrollmentID: "enrollment"}}
	for _, test := range []struct {
		name    string
		mode    domain.AccessControlMode
		allowed bool
	}{
		{name: "disabled", mode: domain.AccessControlDisabled},
		{name: "integrated reserved", mode: domain.AccessControlIntegrated},
		{name: "manual", mode: domain.AccessControlManual, allowed: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := NewAccessService(repo, accessContextStub{}, accessSessionsStub{}, accessTxStub{}).WithUnitAccessControl(accessControlUnitStub{mode: test.mode})
			_, err := service.Admit(context.Background(), "actor", "unit", "attendance.mode-test", AdmitInput{PersonID: "person"})
			if test.allowed && err != nil {
				t.Fatalf("manual mode error: %v", err)
			}
			if !test.allowed && !errors.Is(err, ErrAccessUnavailable) {
				t.Fatalf("mode %s error = %v", test.mode, err)
			}
		})
	}
}

func TestAccessServiceGeneralAdmissionRules(t *testing.T) {
	now := time.Date(2026, 9, 15, 15, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		subject  domain.AttendanceSubject
		outcome  domain.AttendanceDecisionOutcome
		reason   domain.AttendanceReasonCode
		presence bool
	}{
		{"active enrollment", domain.AttendanceSubject{PersonID: "person", StudentID: "student", Name: "Ana", Source: domain.SourceBoxMember, RelationshipActive: true, ActiveEnrollmentID: "enrollment"}, domain.AttendanceAllowed, domain.AttendanceReasonActiveEnrollment, true},
		{"member without enrollment", domain.AttendanceSubject{PersonID: "person", StudentID: "student", Name: "Ana", Source: domain.SourceBoxMember, RelationshipActive: true}, domain.AttendanceDenied, domain.AttendanceReasonNoActiveEnrollment, false},
		{"corporate relationship", domain.AttendanceSubject{PersonID: "person", StudentID: "student", Name: "Ana", Source: domain.SourceWellhub, RelationshipActive: true}, domain.AttendanceAllowed, domain.AttendanceReasonCorporateRelationship, true},
		{"inactive relationship", domain.AttendanceSubject{PersonID: "person", StudentID: "student", Name: "Ana", Source: domain.SourceTotalPass}, domain.AttendanceDenied, domain.AttendanceReasonInactiveRelationship, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &accessRepositoryStub{subject: test.subject}
			service := NewAccessService(repo, accessContextStub{}, accessSessionsStub{}, accessTxStub{})
			service.now = func() time.Time { return now }
			result, err := service.Admit(context.Background(), "actor", "unit", "attendance.test-key", AdmitInput{PersonID: "person"})
			if err != nil {
				t.Fatal(err)
			}
			if result.Decision.Outcome != test.outcome || result.Decision.ReasonCode != test.reason {
				t.Fatalf("unexpected decision: %+v", result.Decision)
			}
			if (result.Attendance != nil) != test.presence {
				t.Fatalf("presence mismatch: %+v", result.Attendance)
			}
		})
	}
}

func TestAccessServiceSessionRequiresConfirmedReservationInWindow(t *testing.T) {
	now := time.Date(2026, 9, 15, 15, 0, 0, 0, time.UTC)
	sessionID := domain.ID("session")
	session := &domain.ScheduleSession{ID: sessionID, Status: domain.ScheduleSessionScheduled, StartsAt: now.Add(30 * time.Minute), EndsAt: now.Add(90 * time.Minute)}
	subject := domain.AttendanceSubject{PersonID: "person", StudentID: "student", Name: "Ana", Source: domain.SourceBoxMember, RelationshipActive: true, ActiveEnrollmentID: "enrollment"}
	for _, test := range []struct {
		name        string
		reservation *domain.Reservation
		outcome     domain.AttendanceDecisionOutcome
		reason      domain.AttendanceReasonCode
	}{
		{"confirmed", &domain.Reservation{ID: "reservation", Status: domain.ReservationConfirmed}, domain.AttendanceAllowed, domain.AttendanceReasonConfirmedReservation},
		{"waitlisted", &domain.Reservation{ID: "reservation", Status: domain.ReservationWaitlisted}, domain.AttendanceDenied, domain.AttendanceReasonWaitlisted},
		{"missing", nil, domain.AttendanceDenied, domain.AttendanceReasonSessionRequiresReservation},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := &accessRepositoryStub{subject: subject, reservation: test.reservation}
			service := NewAccessService(repo, accessContextStub{}, accessSessionsStub{session}, accessTxStub{})
			service.now = func() time.Time { return now }
			result, err := service.Admit(context.Background(), "actor", "unit", "attendance.session-key", AdmitInput{PersonID: "person", SessionID: &sessionID})
			if err != nil {
				t.Fatal(err)
			}
			if result.Decision.Outcome != test.outcome || result.Decision.ReasonCode != test.reason {
				t.Fatalf("unexpected: %+v", result.Decision)
			}
		})
	}
}

func TestAccessServiceIsIdempotentAndRejectsKeyReuse(t *testing.T) {
	now := time.Date(2026, 9, 15, 15, 0, 0, 0, time.UTC)
	repo := &accessRepositoryStub{subject: domain.AttendanceSubject{PersonID: "person", StudentID: "student", Name: "Ana", Source: domain.SourceBoxMember, RelationshipActive: true, ActiveEnrollmentID: "enrollment"}}
	service := NewAccessService(repo, accessContextStub{}, accessSessionsStub{}, accessTxStub{})
	service.now = func() time.Time { return now }
	first, err := service.Admit(context.Background(), "actor", "unit", "attendance.same-key", AdmitInput{PersonID: "person"})
	if err != nil {
		t.Fatal(err)
	}
	replay, err := service.Admit(context.Background(), "actor", "unit", "attendance.same-key", AdmitInput{PersonID: "person"})
	if err != nil || !replay.Decision.IdempotentReplay || replay.Decision.ID != first.Decision.ID || len(repo.decisions) != 1 {
		t.Fatalf("replay=%+v decisions=%d err=%v", replay, len(repo.decisions), err)
	}
	_, err = service.Admit(context.Background(), "actor", "unit", "attendance.same-key", AdmitInput{PersonID: "another-person"})
	if !errors.Is(err, ErrIdempotencyReused) {
		t.Fatalf("expected key reuse, got %v", err)
	}
}

func TestMembershipPrerequisiteDenialsKeepSpecificAttendanceReasons(t *testing.T) {
	tests := []struct {
		access domain.MembershipAccessReason
		want   domain.AttendanceReasonCode
	}{
		{domain.MembershipAccessReasonContractPending, domain.AttendanceReasonContractPending},
		{domain.MembershipAccessReasonHealthFormPending, domain.AttendanceReasonHealthFormPending},
		{domain.MembershipAccessReasonPaymentRequired, domain.AttendanceReasonPaymentRequired},
		{domain.MembershipAccessReasonPaymentOverdue, domain.AttendanceReasonPaymentOverdue},
	}
	for _, test := range tests {
		decision := &domain.AttendanceDecision{ReasonCode: domain.AttendanceReasonNoActiveEnrollment}
		access := &domain.MembershipAccessDecision{ReasonCode: test.access, Reason: "motivo específico"}
		applyMembershipPrerequisiteDenial(decision, access)
		if decision.ReasonCode != test.want || decision.Reason != access.Reason {
			t.Fatalf("access=%s decision=%+v", test.access, decision)
		}
	}
}
