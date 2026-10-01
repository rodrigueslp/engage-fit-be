package memberbilling

import (
	"testing"
	"time"

	"boxengage/backend/internal/domain"
)

func TestNextDuePreservesSupportedCycles(t *testing.T) {
	start := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		cycle domain.MembershipBillingCycle
		want  string
	}{
		{domain.MembershipBillingMonthly, "2026-03-10"},
		{domain.MembershipBillingQuarterly, "2026-05-10"},
		{domain.MembershipBillingSemiannual, "2026-08-10"},
		{domain.MembershipBillingAnnual, "2027-02-10"},
	}
	for _, test := range tests {
		got := nextDue(start, test.cycle)
		if got == nil || got.Format("2006-01-02") != test.want {
			t.Fatalf("%s: got %v, want %s", test.cycle, got, test.want)
		}
	}
	if got := nextDue(start, domain.MembershipBillingOneTime); got != nil {
		t.Fatalf("one-time should not recur: %v", got)
	}
	monthEnd := nextDue(time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), domain.MembershipBillingMonthly)
	if monthEnd.Format("2006-01-02") != "2026-02-28" {
		t.Fatalf("month end drifted: %s", monthEnd.Format("2006-01-02"))
	}
}

func TestSummarizeProjectsOverdueWithoutMutatingPersistence(t *testing.T) {
	contract := &domain.MemberBillingContract{ID: "contract"}
	charges := []domain.MemberCharge{{ID: "late", AmountCents: 10000, Status: "pending", DueOn: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}, {ID: "paid", AmountCents: 5000, PaidCents: 5000, RefundedCents: 1000, Status: "paid", DueOn: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)}}
	result := summarize(contract, charges, time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC))
	if result.FinancialStatus != "overdue" || result.OpenAmountCents != 10000 || result.PaidAmountCents != 5000 || result.RefundedAmountCents != 1000 {
		t.Fatalf("unexpected summary: %+v", result)
	}
	if charges[0].Status != "pending" {
		t.Fatal("summary must not mutate repository input")
	}
}

func TestManualAndCheckoutMethodsRemainSeparate(t *testing.T) {
	if !validMethod("card", false) || validMethod("cash", false) {
		t.Fatal("checkout methods are invalid")
	}
	if !validMethod("cash", true) || !validMethod("card_machine", true) || validMethod("card", true) {
		t.Fatal("manual methods are invalid")
	}
}

func TestIdempotencyKeyValidation(t *testing.T) {
	if !validKey("billing.12345678") || validKey("short") || validKey("billing key invalid") {
		t.Fatal("unexpected key validation")
	}
}

func TestPaymentProjectionProtectsBalanceAndRefunds(t *testing.T) {
	charge := &domain.MemberCharge{AmountCents: 10000, Status: "pending"}
	occurred := time.Now()
	if err := applyPaymentProjection(charge, "manual", 4000, occurred); err != nil || charge.Status != "partially_paid" {
		t.Fatalf("partial: %+v %v", charge, err)
	}
	if err := applyPaymentProjection(charge, "manual", 6001, occurred); err != ErrInvalidTransition {
		t.Fatalf("overpayment err=%v", err)
	}
	if err := applyPaymentProjection(charge, "manual", 6000, occurred); err != nil || charge.Status != "paid" {
		t.Fatalf("paid: %+v %v", charge, err)
	}
	if err := applyPaymentProjection(charge, "refund", 2500, occurred); err != nil || charge.Status != "paid" {
		t.Fatalf("partial refund: %+v %v", charge, err)
	}
	if err := applyPaymentProjection(charge, "refund", 7500, occurred); err != nil || charge.Status != "refunded" {
		t.Fatalf("full refund: %+v %v", charge, err)
	}
	if err := applyPaymentProjection(charge, "refund", 1, occurred); err != ErrInvalidTransition {
		t.Fatalf("excess refund err=%v", err)
	}
}
