package retention

import (
	"testing"
	"time"

	"boxengage/backend/internal/domain"
)

func TestClassifyUsesExplainableFrequencySignals(t *testing.T) {
	today := time.Date(2026, time.July, 28, 0, 0, 0, 0, time.UTC)
	first := today.AddDate(0, 0, -100)
	last := today.AddDate(0, 0, -9)
	item := classify(domain.RetentionMetrics{
		FirstCheckin: &first, LastCheckin: &last,
		PreviousCheckins: 12, RecentCheckins: 4,
	}, today, 7)

	if item.Level != domain.EngagementAtRisk {
		t.Fatalf("expected at_risk, got %s", item.Level)
	}
	if item.DropPercentage == nil || *item.DropPercentage < 66 || *item.DropPercentage > 67 {
		t.Fatalf("expected a 66.67 percent drop, got %v", item.DropPercentage)
	}
	if len(item.Signals) != 2 {
		t.Fatalf("expected inactivity and frequency signals, got %#v", item.Signals)
	}
}

func TestClassifyDoesNotPretendToKnowWithShortHistory(t *testing.T) {
	today := time.Date(2026, time.July, 28, 0, 0, 0, 0, time.UTC)
	first := today.AddDate(0, 0, -20)
	last := today.AddDate(0, 0, -10)
	item := classify(domain.RetentionMetrics{
		FirstCheckin: &first, LastCheckin: &last,
		PreviousCheckins: 8, RecentCheckins: 1,
	}, today, 7)

	if item.Level != domain.EngagementHistoryInsufficient {
		t.Fatalf("expected history_insufficient, got %s", item.Level)
	}
	if len(item.Signals) != 1 || item.Signals[0].Code != "history_insufficient" {
		t.Fatalf("expected explicit insufficient history signal, got %#v", item.Signals)
	}
}

func TestClassifyObservesReturnWithoutClaimingCausality(t *testing.T) {
	today := time.Date(2026, time.July, 28, 0, 0, 0, 0, time.UTC)
	first := today.AddDate(0, 0, -100)
	last := today.AddDate(0, 0, -1)
	action := today.AddDate(0, 0, -6)
	returned := today.AddDate(0, 0, -2)
	item := classify(domain.RetentionMetrics{
		FirstCheckin: &first, LastCheckin: &last,
		PreviousCheckins: 8, RecentCheckins: 2,
		LastCompletedIntervention: &action, FirstReturnAfterAction: &returned,
		LastInterventionID: "action-1", LastInterventionStatus: "completed",
	}, today, 7)

	if item.Level != domain.EngagementRecovered || !item.ReturnWithin7Days || !item.ReturnWithin14Days {
		t.Fatalf("expected observed return within 7 and 14 days, got %#v", item)
	}
	if item.ReturnWithin3Days {
		t.Fatal("did not expect return within 3 days")
	}
	if item.WorkflowStatus != domain.RetentionWorkflowRecovered {
		t.Fatalf("expected recovered workflow, got %s", item.WorkflowStatus)
	}
}

func TestWorkflowMovesContactOutOfDailyActionQueue(t *testing.T) {
	today := time.Date(2026, time.July, 28, 0, 0, 0, 0, time.UTC)
	action := today.AddDate(0, 0, -1)
	review := today.AddDate(0, 0, 6)
	status, due := workflowStatus(domain.RetentionRadarItem{
		Level: domain.EngagementCritical,
		RetentionMetrics: domain.RetentionMetrics{
			LastInterventionID: "action-1", LastInterventionStatus: "completed",
			LastInterventionOutcome: "contacted", LastCompletedIntervention: &action,
			LastInterventionPlannedFor: &review,
		},
	}, today)
	if status != domain.RetentionWorkflowWaitingReturn || due == nil || !due.Equal(review) {
		t.Fatalf("expected waiting return until review date, got %s %v", status, due)
	}
}

func TestWorkflowReopensWhenReviewIsDue(t *testing.T) {
	today := time.Date(2026, time.July, 28, 0, 0, 0, 0, time.UTC)
	action := today.AddDate(0, 0, -8)
	status, _ := workflowStatus(domain.RetentionRadarItem{
		Level: domain.EngagementCritical,
		RetentionMetrics: domain.RetentionMetrics{
			LastInterventionID: "action-1", LastInterventionStatus: "completed",
			LastInterventionOutcome: "no_response", LastCompletedIntervention: &action,
		},
	}, today)
	if status != domain.RetentionWorkflowFollowUpDue {
		t.Fatalf("expected follow up due, got %s", status)
	}
}

func TestWorkflowCanPauseOrCloseACase(t *testing.T) {
	today := time.Date(2026, time.July, 28, 0, 0, 0, 0, time.UTC)
	future := today.AddDate(0, 0, 14)
	base := domain.RetentionRadarItem{Level: domain.EngagementCritical, RetentionMetrics: domain.RetentionMetrics{LastInterventionID: "action-1", LastInterventionStatus: "completed"}}
	paused := base
	paused.LastInterventionOutcome, paused.LastInterventionPlannedFor = "paused", &future
	if status, _ := workflowStatus(paused, today); status != domain.RetentionWorkflowPaused {
		t.Fatalf("expected paused, got %s", status)
	}
	closed := base
	closed.LastInterventionOutcome = "not_interested"
	if status, _ := workflowStatus(closed, today); status != domain.RetentionWorkflowClosed {
		t.Fatalf("expected closed, got %s", status)
	}
}
