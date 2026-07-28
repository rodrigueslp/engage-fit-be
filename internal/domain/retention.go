package domain

import "time"

type EngagementLevel string

const (
	EngagementHistoryInsufficient EngagementLevel = "history_insufficient"
	EngagementHealthy             EngagementLevel = "healthy"
	EngagementAttention           EngagementLevel = "attention"
	EngagementAtRisk              EngagementLevel = "at_risk"
	EngagementCritical            EngagementLevel = "critical"
	EngagementRecovered           EngagementLevel = "recovered"
)

type EngagementSignal struct {
	Code    string
	Message string
}

type RetentionWorkflowStatus string

const (
	RetentionWorkflowNone          RetentionWorkflowStatus = "none"
	RetentionWorkflowNeedsAction   RetentionWorkflowStatus = "needs_action"
	RetentionWorkflowWaitingReturn RetentionWorkflowStatus = "waiting_return"
	RetentionWorkflowFollowUpDue   RetentionWorkflowStatus = "follow_up_due"
	RetentionWorkflowPaused        RetentionWorkflowStatus = "paused"
	RetentionWorkflowClosed        RetentionWorkflowStatus = "closed"
	RetentionWorkflowRecovered     RetentionWorkflowStatus = "recovered"
)

type RetentionMetrics struct {
	StudentID                  ID
	StudentName                string
	StudentPhone               string
	Source                     Source
	ContactStatus              ContactStatus
	FirstCheckin               *time.Time
	LastCheckin                *time.Time
	RecentCheckins             int
	PreviousCheckins           int
	LastCompletedIntervention  *time.Time
	FirstReturnAfterAction     *time.Time
	LastInterventionID         ID
	LastInterventionChannel    string
	LastInterventionStatus     string
	LastInterventionOutcome    string
	LastInterventionPlannedFor *time.Time
	LastInterventionCreatedAt  *time.Time
}

type RetentionRadarItem struct {
	RetentionMetrics
	Level                 EngagementLevel
	DaysSinceCheckin      *int
	RecentWeeklyAverage   float64
	PreviousWeeklyAverage float64
	DropPercentage        *float64
	Signals               []EngagementSignal
	ReturnWithin3Days     bool
	ReturnWithin7Days     bool
	ReturnWithin14Days    bool
	WorkflowStatus        RetentionWorkflowStatus
	FollowUpDueAt         *time.Time
}

type RetentionIntervention struct {
	ID              ID
	BoxID           ID
	StudentID       ID
	CreatedByUserID ID
	Channel         string
	Status          string
	Outcome         string
	PlannedFor      *time.Time
	CompletedAt     *time.Time
	Notes           string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
