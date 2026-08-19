package memberships

import (
	"testing"
	"time"

	"boxengage/backend/internal/domain"
)

func TestCalculateValidUntilUsesInclusiveCalendarArithmetic(t *testing.T) {
	location := time.FixedZone("America/Sao_Paulo", -3*60*60)
	day, week, month, year := domain.MembershipValidityDay, domain.MembershipValidityWeek, domain.MembershipValidityMonth, domain.MembershipValidityYear
	tests := []struct {
		name     string
		start    string
		count    int
		unit     domain.MembershipValidityUnit
		expected string
	}{
		{"one day", "2026-08-19", 1, day, "2026-08-19"},
		{"two weeks", "2026-08-19", 2, week, "2026-09-01"},
		{"one month", "2026-08-19", 1, month, "2026-09-18"},
		{"one year", "2026-08-19", 1, year, "2027-08-18"},
		{"month end clamps", "2026-01-31", 1, month, "2026-02-27"},
		{"leap day year clamps", "2024-02-29", 1, year, "2025-02-27"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			start, err := time.ParseInLocation("2006-01-02", test.start, location)
			if err != nil {
				t.Fatal(err)
			}
			version := domain.MembershipPlanVersion{BillingCycle: domain.MembershipBillingOneTime, ValidityCount: &test.count, ValidityUnit: &test.unit}
			got := calculateValidUntil(start, version)
			if got == nil || got.Format("2006-01-02") != test.expected {
				t.Fatalf("calculateValidUntil() = %v, want %s", got, test.expected)
			}
		})
	}

	start, _ := time.ParseInLocation("2006-01-02", "2026-08-19", location)
	if got := calculateValidUntil(start, domain.MembershipPlanVersion{BillingCycle: domain.MembershipBillingMonthly}); got != nil {
		t.Fatalf("recurring plan valid_until = %v, want nil", got)
	}
}

func TestEnrollmentEffectiveStatusPrecedence(t *testing.T) {
	location := time.FixedZone("America/Sao_Paulo", -3*60*60)
	date := func(value string) time.Time {
		parsed, err := time.ParseInLocation("2006-01-02", value, location)
		if err != nil {
			t.Fatal(err)
		}
		return parsed
	}
	today := date("2026-08-19")
	validUntil := date("2026-08-18")
	tests := []struct {
		name string
		item domain.Enrollment
		want domain.EnrollmentEffectiveStatus
	}{
		{"scheduled", domain.Enrollment{LifecycleStatus: domain.EnrollmentLifecycleActive, StartsOn: date("2026-08-20")}, domain.EnrollmentEffectiveScheduled},
		{"active recurring", domain.Enrollment{LifecycleStatus: domain.EnrollmentLifecycleActive, StartsOn: date("2026-08-19")}, domain.EnrollmentEffectiveActive},
		{"expired", domain.Enrollment{LifecycleStatus: domain.EnrollmentLifecycleActive, StartsOn: date("2026-07-01"), ValidUntil: &validUntil}, domain.EnrollmentEffectiveExpired},
		{"ended wins", domain.Enrollment{LifecycleStatus: domain.EnrollmentLifecycleEnded, StartsOn: date("2026-08-20")}, domain.EnrollmentEffectiveEnded},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := domain.EnrollmentStatus(test.item, today); got != test.want {
				t.Fatalf("EnrollmentStatus() = %s, want %s", got, test.want)
			}
		})
	}
}

func TestEnrollmentDateAndEndValidation(t *testing.T) {
	location := time.FixedZone("America/Sao_Paulo", -3*60*60)
	now := time.Date(2026, 8, 19, 15, 0, 0, 0, location)
	service := &EnrollmentService{now: func() time.Time { return now }, location: location}

	for _, value := range []string{"2016-08-19", "2026-08-19", "2027-08-19"} {
		if _, err := service.parseStart(value); err != nil {
			t.Errorf("parseStart(%s) unexpected error: %v", value, err)
		}
	}
	for _, value := range []string{"2016-08-18", "2027-08-20", "19/08/2026"} {
		if _, err := service.parseStart(value); err != ErrInvalidStartDate {
			t.Errorf("parseStart(%s) error = %v, want invalid_start_date", value, err)
		}
	}

	future := domain.Enrollment{StartsOn: time.Date(2026, 8, 20, 0, 0, 0, 0, location)}
	if err := service.validateEndDate(future, service.today()); err != nil {
		t.Fatalf("future enrollment ended today: %v", err)
	}
	if err := service.validateEndDate(future, future.StartsOn); err != ErrInvalidEnd {
		t.Fatalf("future enrollment ended after today error = %v", err)
	}

	started := domain.Enrollment{StartsOn: time.Date(2026, 8, 1, 0, 0, 0, 0, location)}
	if err := service.validateEndDate(started, time.Date(2026, 8, 10, 0, 0, 0, 0, location)); err != nil {
		t.Fatalf("valid end date: %v", err)
	}
	if err := service.validateEndDate(started, time.Date(2026, 7, 31, 0, 0, 0, 0, location)); err != ErrInvalidEnd {
		t.Fatalf("end before start error = %v", err)
	}
}
