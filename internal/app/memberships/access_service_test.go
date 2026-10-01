package memberships

import (
	"context"
	"testing"
	"time"

	"boxengage/backend/internal/domain"
)

type accessRepoStub struct {
	values                    []domain.MembershipAccessCandidate
	activeHere, anyHere, away bool
}

func (s accessRepoStub) ResolveMembershipAccessCandidates(context.Context, domain.MembershipAccessRequest) ([]domain.MembershipAccessCandidate, error) {
	return s.values, nil
}
func (s accessRepoStub) HasActiveEnrollmentInUnit(context.Context, domain.ID, domain.ID, domain.ID, time.Time) (bool, error) {
	return s.activeHere, nil
}
func (s accessRepoStub) HasEnrollmentInUnit(context.Context, domain.ID, domain.ID, domain.ID) (bool, error) {
	return s.anyHere, nil
}
func (s accessRepoStub) HasEnrollmentOutsideRequestedUnit(context.Context, domain.ID, domain.ID, domain.ID, time.Time) (bool, error) {
	return s.away, nil
}

type accessUsageStub struct{ used int }

func (s accessUsageStub) LockMembershipAccessUsage(context.Context, domain.ID, domain.ID, domain.ID) error {
	return nil
}

func (s accessUsageStub) CountMembershipAccessUsage(context.Context, domain.MembershipAccessUsageQuery) (int, error) {
	return s.used, nil
}

type legacyAccessStub bool

func (s legacyAccessStub) HasLegacyMembership(context.Context, domain.ID, domain.ID, domain.ID) (bool, error) {
	return bool(s), nil
}

type accessPrerequisiteStub struct {
	value domain.MembershipAccessPrerequisites
}

func (s accessPrerequisiteStub) ResolveMembershipAccessPrerequisites(context.Context, domain.ID, domain.ID, domain.ID, domain.ID, time.Time) (domain.MembershipAccessPrerequisites, error) {
	return s.value, nil
}

func TestAccessServiceAllowsAndReturnsRemainingAfterAction(t *testing.T) {
	quantity := 3
	period := domain.MembershipPeriodWeek
	service := NewAccessService(accessRepoStub{values: []domain.MembershipAccessCandidate{{EnrollmentID: "enrollment", EntitlementID: "entitlement", AccessType: domain.MembershipAccessPerPeriod, Quantity: &quantity, PeriodUnit: &period, StartsOn: time.Now().AddDate(0, -1, 0), Timezone: "America/Sao_Paulo"}}}, accessUsageStub{used: 1}, legacyAccessStub(false))
	decision, err := service.Evaluate(context.Background(), domain.MembershipAccessRequest{OrganizationID: "org", UnitID: "unit", PersonID: "person", OccurredAt: time.Now(), Purpose: domain.MembershipAccessBooking})
	if err != nil || decision.Outcome != domain.MembershipAccessAllowed || decision.RemainingUsage == nil || *decision.RemainingUsage != 1 {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
}

func TestAccessServiceDenialReasonsAndLegacyFallback(t *testing.T) {
	cases := []struct {
		name    string
		repo    accessRepoStub
		legacy  legacyAccessStub
		want    domain.MembershipAccessReason
		allowed bool
	}{
		{name: "target", repo: accessRepoStub{activeHere: true}, want: domain.MembershipAccessReasonTargetNotIncluded},
		{name: "validity", repo: accessRepoStub{anyHere: true}, want: domain.MembershipAccessReasonOutsideValidity},
		{name: "unit", repo: accessRepoStub{away: true}, want: domain.MembershipAccessReasonUnitNotIncluded},
		{name: "none", repo: accessRepoStub{}, want: domain.MembershipAccessReasonNoEnrollment},
		{name: "legacy checkin", repo: accessRepoStub{}, legacy: legacyAccessStub(true), want: domain.MembershipAccessReasonLegacyTransition, allowed: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewAccessService(tc.repo, accessUsageStub{}, tc.legacy)
			decision, err := service.Evaluate(context.Background(), domain.MembershipAccessRequest{OrganizationID: "org", UnitID: "unit", PersonID: "person", OccurredAt: time.Now(), Purpose: domain.MembershipAccessCheckin})
			if err != nil || decision.ReasonCode != tc.want || (decision.Outcome == domain.MembershipAccessAllowed) != tc.allowed {
				t.Fatalf("decision=%+v err=%v", decision, err)
			}
		})
	}
}

func TestAccessServiceEnforcesDailyAndBookingLimits(t *testing.T) {
	daily := 1
	base := domain.MembershipAccessCandidate{EnrollmentID: "enrollment", EntitlementID: "entitlement", AccessType: domain.MembershipAccessOpen, DailyLimit: &daily, StartsOn: time.Now().AddDate(0, -1, 0), Timezone: "UTC"}
	service := NewAccessService(accessRepoStub{values: []domain.MembershipAccessCandidate{base}}, accessUsageStub{used: 1}, legacyAccessStub(false))
	decision, _ := service.Evaluate(context.Background(), domain.MembershipAccessRequest{OccurredAt: time.Now(), Purpose: domain.MembershipAccessCheckin})
	if decision.ReasonCode != domain.MembershipAccessReasonLimitReached {
		t.Fatalf("decision=%+v", decision)
	}
	base.DailyLimit, base.BookingRequired = nil, true
	service = NewAccessService(accessRepoStub{values: []domain.MembershipAccessCandidate{base}}, accessUsageStub{}, legacyAccessStub(false))
	decision, _ = service.Evaluate(context.Background(), domain.MembershipAccessRequest{OccurredAt: time.Now(), Purpose: domain.MembershipAccessCheckin})
	if decision.ReasonCode != domain.MembershipAccessReasonBookingRequired {
		t.Fatalf("decision=%+v", decision)
	}
	decision, _ = service.Evaluate(context.Background(), domain.MembershipAccessRequest{OccurredAt: time.Now(), Purpose: domain.MembershipAccessCheckin, PreferredEnrollment: base.EnrollmentID, PreferredEntitlement: base.EntitlementID, UsageAlreadyCounted: true})
	if decision.Outcome != domain.MembershipAccessAllowed {
		t.Fatalf("confirmed reservation decision=%+v", decision)
	}
}

func TestAccessServiceRequiresContractHealthFormAndPayment(t *testing.T) {
	candidate := domain.MembershipAccessCandidate{EnrollmentID: "enrollment", EntitlementID: "entitlement", AccessType: domain.MembershipAccessOpen, StartsOn: time.Now().AddDate(0, -1, 0), Timezone: "UTC"}
	tests := []struct {
		name   string
		value  domain.MembershipAccessPrerequisites
		reason domain.MembershipAccessReason
	}{
		{name: "contract", reason: domain.MembershipAccessReasonContractPending},
		{name: "health form", value: domain.MembershipAccessPrerequisites{ContractAccepted: true}, reason: domain.MembershipAccessReasonHealthFormPending},
		{name: "initial payment", value: domain.MembershipAccessPrerequisites{ContractAccepted: true, HealthFormSubmitted: true}, reason: domain.MembershipAccessReasonPaymentRequired},
		{name: "overdue payment", value: domain.MembershipAccessPrerequisites{ContractAccepted: true, HealthFormSubmitted: true, InitialPaymentPaid: true, PaymentOverdue: true}, reason: domain.MembershipAccessReasonPaymentOverdue},
		{name: "eligible", value: domain.MembershipAccessPrerequisites{ContractAccepted: true, HealthFormSubmitted: true, InitialPaymentPaid: true}, reason: domain.MembershipAccessReasonEligible},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewAccessService(accessRepoStub{values: []domain.MembershipAccessCandidate{candidate}}, accessUsageStub{}, legacyAccessStub(false)).WithPrerequisiteReader(accessPrerequisiteStub{value: test.value})
			decision, err := service.Evaluate(context.Background(), domain.MembershipAccessRequest{OrganizationID: "org", UnitID: "unit", PersonID: "person", OccurredAt: time.Now(), Purpose: domain.MembershipAccessBooking})
			if err != nil || decision.ReasonCode != test.reason {
				t.Fatalf("decision=%+v err=%v", decision, err)
			}
			if (decision.Outcome == domain.MembershipAccessAllowed) != (test.reason == domain.MembershipAccessReasonEligible) {
				t.Fatalf("unexpected outcome: %+v", decision)
			}
		})
	}
}
