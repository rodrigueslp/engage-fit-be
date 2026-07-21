package repositories

import (
	"strings"
	"testing"
	"time"

	"boxengage/backend/internal/domain"
)

func TestValidateReservation(t *testing.T) {
	base := policyRow{Scope: string(domain.MessagingPolicyScopeBox), DailyMessageLimit: 100, MonthlyMessageLimit: 1000, PerDispatchLimit: 50, DailyCostLimitMicros: 10_000_000, MonthlyCostLimitMicros: 100_000_000}
	tests := []struct {
		name       string
		policy     policyRow
		daily      usageRow
		monthly    usageRow
		recipients int
		cost       int64
		contains   string
	}{
		{name: "allows available quota", policy: base, recipients: 20, cost: 2_000_000},
		{name: "blocks administratively", policy: withBlocked(base), recipients: 1, cost: 1, contains: "bloqueados"},
		{name: "blocks dispatch size", policy: base, recipients: 51, cost: 1, contains: "por disparo"},
		{name: "counts accepted and reserved daily", policy: base, daily: usageRow{AcceptedCount: 80, ReservedCount: 10}, recipients: 11, cost: 1, contains: "limite diário"},
		{name: "blocks monthly estimated cost", policy: base, monthly: usageRow{EstimatedCostMicros: 99_000_000}, recipients: 10, cost: 2_000_000, contains: "orçamento mensal"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reason := validateReservation(test.policy, test.daily, test.monthly, test.recipients, test.cost)
			if test.contains == "" && reason != "" {
				t.Fatalf("expected reservation to be allowed, got %q", reason)
			}
			if test.contains != "" && !strings.Contains(reason, test.contains) {
				t.Fatalf("expected reason containing %q, got %q", test.contains, reason)
			}
		})
	}
}

func TestPeriodStartsUsesPolicyTimezone(t *testing.T) {
	at := time.Date(2026, time.July, 17, 1, 30, 0, 0, time.UTC)
	daily, monthly := periodStarts(at, "America/Sao_Paulo")
	if daily.Day() != 16 || daily.Month() != time.July {
		t.Fatalf("expected local daily period to start on July 16, got %s", daily)
	}
	if monthly.Day() != 1 || monthly.Month() != time.July {
		t.Fatalf("unexpected monthly period start: %s", monthly)
	}
}

func withBlocked(policy policyRow) policyRow {
	policy.Blocked = true
	return policy
}
