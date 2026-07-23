package domain

import (
	"testing"
	"time"
)

func TestBillingSubscriptionAllowsAccessDuringGrace(t *testing.T) {
	t.Parallel()
	grace := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)
	subscription := BillingSubscription{Status: BillingStatusPastDue, GraceUntil: &grace}
	if !subscription.AllowsAccess(time.Date(2026, 7, 25, 23, 59, 0, 0, time.UTC)) {
		t.Fatal("access should remain available through the end of the grace date")
	}
	if subscription.AllowsAccess(time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("access should be denied after grace")
	}
}

func TestCanceledBillingSubscriptionDoesNotAllowAccess(t *testing.T) {
	t.Parallel()
	if (BillingSubscription{Status: BillingStatusCanceled}).AllowsAccess(time.Now()) {
		t.Fatal("canceled subscription must not allow access")
	}
}
