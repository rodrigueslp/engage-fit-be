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
		TotalCheckins: 24, PreviousCheckins: 12, RecentCheckins: 4,
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
		TotalCheckins: 9, PreviousCheckins: 8, RecentCheckins: 1,
	}, today, 7)

	if item.Level != domain.EngagementHistoryInsufficient {
		t.Fatalf("expected history_insufficient, got %s", item.Level)
	}
	if len(item.Signals) != 1 || item.Signals[0].Code != "history_insufficient" {
		t.Fatalf("expected explicit insufficient history signal, got %#v", item.Signals)
	}
}

func TestClassifyDoesNotFlagAnIsolatedVisitAsRetentionRisk(t *testing.T) {
	today := time.Date(2026, time.July, 28, 0, 0, 0, 0, time.UTC)
	first := today.AddDate(0, 0, -100)
	item := classify(domain.RetentionMetrics{
		FirstCheckin: &first, LastCheckin: &first,
		TotalCheckins: 1,
	}, today, 7)

	if item.Level != domain.EngagementHistoryInsufficient {
		t.Fatalf("expected history_insufficient, got %s", item.Level)
	}
	if len(item.Signals) != 1 || item.Signals[0].Code != "routine_insufficient" {
		t.Fatalf("expected explicit routine signal, got %#v", item.Signals)
	}
	if item.WorkflowStatus != domain.RetentionWorkflowNone {
		t.Fatalf("expected no retention action, got %s", item.WorkflowStatus)
	}
	if item.Recommendation.Code != "observe_routine" {
		t.Fatalf("expected routine recommendation, got %#v", item.Recommendation)
	}
}

func TestClassifyMovesLongInactiveStudentToHistoricalReactivation(t *testing.T) {
	today := time.Date(2026, time.July, 31, 0, 0, 0, 0, time.UTC)
	first := today.AddDate(0, 0, -100)
	last := today.AddDate(0, 0, -31)
	item := classify(domain.RetentionMetrics{
		FirstCheckin: &first, LastCheckin: &last,
		TotalCheckins: 12, PreviousCheckins: 6,
	}, today, 7)

	if item.Level != domain.EngagementCritical || item.WorkflowStatus != domain.RetentionWorkflowHistorical {
		t.Fatalf("expected historical critical case, got level=%s workflow=%s", item.Level, item.WorkflowStatus)
	}
	if item.Recommendation.Code != "historical_reactivation" {
		t.Fatalf("expected reactivation recommendation, got %#v", item.Recommendation)
	}
}

func TestClassifyKeepsThirtyDayAbsenceInOperationalQueue(t *testing.T) {
	today := time.Date(2026, time.July, 31, 0, 0, 0, 0, time.UTC)
	first := today.AddDate(0, 0, -100)
	last := today.AddDate(0, 0, -30)
	item := classify(domain.RetentionMetrics{
		FirstCheckin: &first, LastCheckin: &last,
		TotalCheckins: 12, PreviousCheckins: 6,
	}, today, 7)

	if item.WorkflowStatus != domain.RetentionWorkflowNeedsAction {
		t.Fatalf("expected day 30 in operational queue, got %s", item.WorkflowStatus)
	}
}

func TestRetentionExclusionIsReversibleAndCanExpire(t *testing.T) {
	today := time.Date(2026, time.July, 31, 0, 0, 0, 0, time.UTC)
	first := today.AddDate(0, 0, -100)
	last := today.AddDate(0, 0, -10)
	future := today.AddDate(0, 0, 10)
	metric := domain.RetentionMetrics{
		FirstCheckin: &first, LastCheckin: &last, TotalCheckins: 12, PreviousCheckins: 6,
		RetentionMonitoringStatus: domain.RetentionMonitoringExcluded,
		RetentionExclusionReason: "visitor", RetentionExcludedUntil: &future,
	}
	item := classify(metric, today, 7)
	if item.WorkflowStatus != domain.RetentionWorkflowExcluded {
		t.Fatalf("expected active exclusion, got %s", item.WorkflowStatus)
	}

	past := today.AddDate(0, 0, -1)
	metric.RetentionExcludedUntil = &past
	item = classify(metric, today, 7)
	if item.WorkflowStatus != domain.RetentionWorkflowNeedsAction {
		t.Fatalf("expected expired exclusion to resume monitoring, got %s", item.WorkflowStatus)
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
		TotalCheckins: 16, PreviousCheckins: 8, RecentCheckins: 2,
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

func TestRetentionRulesDescribeTheSameWindowsAndThresholdsUsedByRadar(t *testing.T) {
	now := time.Date(2026, time.July, 30, 15, 0, 0, 0, time.UTC)
	rules := retentionRules(domain.Box{RiskInactiveDays: 7}, now)

	if got := rules.RecentStart.Format("2006-01-02"); got != "2026-07-03" {
		t.Fatalf("unexpected recent start %s", got)
	}
	if got := rules.PreviousStart.Format("2006-01-02"); got != "2026-06-05" {
		t.Fatalf("unexpected previous start %s", got)
	}
	if got := rules.HistoryRequiredBefore.Format("2006-01-02"); got != "2026-06-04" {
		t.Fatalf("unexpected history cutoff %s", got)
	}
	if rules.MinimumTotalCheckins != 4 || rules.MinimumPreviousCheckins != 4 {
		t.Fatalf("unexpected minimum check-ins %#v", rules)
	}
	if rules.AttentionInactiveDays != 5 || rules.AtRiskInactiveDays != 7 || rules.CriticalInactiveDays != 14 {
		t.Fatalf("unexpected inactivity thresholds %#v", rules)
	}
}

func TestOnboardingStartConfidenceRequiresPriorCoverageForInferredDates(t *testing.T) {
	probable, eligible := onboardingStartConfidence(domain.OnboardingMetrics{
		MembershipStartedSource:    "first_checkin_inferred",
		ObservationDaysBeforeStart: 96,
	})
	if !eligible || probable != domain.MembershipStartProbable {
		t.Fatalf("expected probable start, got %q eligible=%v", probable, eligible)
	}

	unknown, eligible := onboardingStartConfidence(domain.OnboardingMetrics{
		MembershipStartedSource:    "first_checkin_inferred",
		ObservationDaysBeforeStart: 28,
	})
	if eligible || unknown != "" {
		t.Fatalf("expected unknown start to stay out of onboarding, got %q eligible=%v", unknown, eligible)
	}

	confirmed, eligible := onboardingStartConfidence(domain.OnboardingMetrics{
		MembershipStartedSource: "manual",
	})
	if !eligible || confirmed != domain.MembershipStartConfirmed {
		t.Fatalf("expected confirmed start, got %q eligible=%v", confirmed, eligible)
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

func TestRecommendationRespectsContactPreferenceAndWorkflow(t *testing.T) {
	item := domain.RetentionRadarItem{
		Level:            domain.EngagementCritical,
		RetentionMetrics: domain.RetentionMetrics{ContactStatus: domain.ContactStatusOptedOut},
		WorkflowStatus:   domain.RetentionWorkflowNeedsAction,
	}
	result := recommendation(item)
	if result.Code != "talk_in_person" {
		t.Fatalf("expected in-person recommendation, got %#v", result)
	}
	item.WorkflowStatus = domain.RetentionWorkflowWaitingReturn
	result = recommendation(item)
	if result.Code != "wait_for_review" {
		t.Fatalf("expected recommendation to avoid duplicate contact, got %#v", result)
	}
}

func TestValidInterventionAcceptsStructuredReason(t *testing.T) {
	now := time.Now()
	item := domain.RetentionIntervention{
		Channel: "in_person", Status: "completed", Outcome: "contacted",
		ReasonCode: "schedule", CompletedAt: &now,
	}
	if !validIntervention(item) {
		t.Fatal("expected structured reason to be valid")
	}
	item.ReasonCode = "medical_diagnosis"
	if validIntervention(item) {
		t.Fatal("unexpected free-form sensitive reason code accepted")
	}
}
