package dto

type RetentionSignalResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type RetentionRecommendationResponse struct {
	Code    string `json:"code"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

type RetentionRadarResponse struct {
	StudentID                    string                          `json:"student_id"`
	StudentName                  string                          `json:"student_name"`
	StudentPhone                 string                          `json:"student_phone"`
	Source                       string                          `json:"source"`
	ContactStatus                string                          `json:"contact_status"`
	Level                        string                          `json:"level"`
	FirstCheckin                 *string                         `json:"first_checkin"`
	LastCheckin                  *string                         `json:"last_checkin"`
	DaysSinceCheckin             *int                            `json:"days_since_checkin"`
	TotalCheckins                int                             `json:"total_checkins"`
	RecentCheckins               int                             `json:"recent_checkins"`
	PreviousCheckins             int                             `json:"previous_checkins"`
	RecentWeeklyAverage          float64                         `json:"recent_weekly_average"`
	PreviousWeeklyAverage        float64                         `json:"previous_weekly_average"`
	DropPercentage               *float64                        `json:"drop_percentage"`
	Signals                      []RetentionSignalResponse       `json:"signals"`
	LastCompletedIntervention    *string                         `json:"last_completed_intervention"`
	FirstReturnAfterAction       *string                         `json:"first_return_after_action"`
	ReturnWithin3Days            bool                            `json:"return_within_3_days"`
	ReturnWithin7Days            bool                            `json:"return_within_7_days"`
	ReturnWithin14Days           bool                            `json:"return_within_14_days"`
	WorkflowStatus               string                          `json:"workflow_status"`
	FollowUpDueAt                *string                         `json:"follow_up_due_at"`
	LastInterventionID           string                          `json:"last_intervention_id"`
	LastInterventionChannel      string                          `json:"last_intervention_channel"`
	LastInterventionStatus       string                          `json:"last_intervention_status"`
	LastInterventionOutcome      string                          `json:"last_intervention_outcome"`
	LastInterventionPlannedFor   *string                         `json:"last_intervention_planned_for"`
	LastInterventionCreatedAt    *string                         `json:"last_intervention_created_at"`
	LastInterventionAssigneeID   string                          `json:"last_intervention_assignee_id"`
	LastInterventionAssigneeName string                          `json:"last_intervention_assignee_name"`
	RetentionMonitoringStatus    string                          `json:"retention_monitoring_status"`
	RetentionExclusionReason     string                          `json:"retention_exclusion_reason"`
	RetentionExcludedUntil       *string                         `json:"retention_excluded_until"`
	RetentionExcludedAt          *string                         `json:"retention_excluded_at"`
	Recommendation               RetentionRecommendationResponse `json:"recommendation"`
}

type RetentionRulesResponse struct {
	RecentStart             string  `json:"recent_start"`
	RecentEnd               string  `json:"recent_end"`
	PreviousStart           string  `json:"previous_start"`
	PreviousEnd             string  `json:"previous_end"`
	HistoryRequiredBefore   string  `json:"history_required_before"`
	HistoryDays             int     `json:"history_days"`
	MinimumTotalCheckins    int     `json:"minimum_total_checkins"`
	MinimumPreviousCheckins int     `json:"minimum_previous_checkins"`
	AttentionInactiveDays   int     `json:"attention_inactive_days"`
	AtRiskInactiveDays      int     `json:"at_risk_inactive_days"`
	CriticalInactiveDays    int     `json:"critical_inactive_days"`
	AttentionDropPercentage int     `json:"attention_drop_percentage"`
	AtRiskDropPercentage    int     `json:"at_risk_drop_percentage"`
	CriticalDropPercentage  int     `json:"critical_drop_percentage"`
	OperationalInactiveDays int     `json:"operational_inactive_days"`
	BaselineAt              *string `json:"baseline_at"`
}

type RetentionInterventionRequest struct {
	Channel          string  `json:"channel" binding:"required"`
	Status           string  `json:"status"`
	Outcome          string  `json:"outcome"`
	ReasonCode       string  `json:"reason_code"`
	AssignedToUserID string  `json:"assigned_to_user_id"`
	PlannedFor       *string `json:"planned_for"`
	Notes            string  `json:"notes"`
}

type RetentionInterventionUpdateRequest struct {
	Status           string  `json:"status" binding:"required"`
	Outcome          string  `json:"outcome"`
	ReasonCode       string  `json:"reason_code"`
	AssignedToUserID string  `json:"assigned_to_user_id"`
	PlannedFor       *string `json:"planned_for"`
	Notes            string  `json:"notes"`
}

type RetentionInterventionResponse struct {
	ID                 string  `json:"id"`
	StudentID          string  `json:"student_id"`
	CreatedByUserID    string  `json:"created_by_user_id"`
	AssignedToUserID   string  `json:"assigned_to_user_id"`
	AssignedToUserName string  `json:"assigned_to_user_name"`
	Channel            string  `json:"channel"`
	Status             string  `json:"status"`
	Outcome            string  `json:"outcome"`
	ReasonCode         string  `json:"reason_code"`
	PlannedFor         *string `json:"planned_for"`
	CompletedAt        *string `json:"completed_at"`
	Notes              string  `json:"notes"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

type RetentionBreakdownResponse struct {
	Code  string `json:"code"`
	Count int    `json:"count"`
}

type RetentionSummaryResponse struct {
	PeriodStart            string                       `json:"period_start"`
	PeriodEnd              string                       `json:"period_end"`
	NeedsAction            int                          `json:"needs_action"`
	WaitingReturn          int                          `json:"waiting_return"`
	FollowUpDue            int                          `json:"follow_up_due"`
	Recovered              int                          `json:"recovered"`
	HistoricalInactive     int                          `json:"historical_inactive"`
	Excluded               int                          `json:"excluded"`
	CompletedInterventions int                          `json:"completed_interventions"`
	ReturnWithin3Days      int                          `json:"return_within_3_days"`
	ReturnWithin7Days      int                          `json:"return_within_7_days"`
	ReturnWithin14Days     int                          `json:"return_within_14_days"`
	MedianDaysToReturn     *float64                     `json:"median_days_to_return"`
	Reasons                []RetentionBreakdownResponse `json:"reasons"`
	Channels               []RetentionBreakdownResponse `json:"channels"`
	Outcomes               []RetentionBreakdownResponse `json:"outcomes"`
}

type UpdateMembershipStartRequest struct {
	StartedAt string `json:"started_at" binding:"required"`
}

type UpdateRetentionMonitoringRequest struct {
	Status        string  `json:"status" binding:"required"`
	Reason        string  `json:"reason"`
	ExcludedUntil *string `json:"excluded_until"`
}

type OnboardingJourneyResponse struct {
	StudentID                  string                          `json:"student_id"`
	StudentName                string                          `json:"student_name"`
	StudentPhone               string                          `json:"student_phone"`
	Source                     string                          `json:"source"`
	ContactStatus              string                          `json:"contact_status"`
	MembershipStartedAt        string                          `json:"membership_started_at"`
	MembershipStartedSource    string                          `json:"membership_started_source"`
	MembershipStartConfidence  string                          `json:"membership_start_confidence"`
	ObservationDaysBeforeStart int                             `json:"observation_days_before_start"`
	Day                        int                             `json:"day"`
	FirstCheckin               *string                         `json:"first_checkin"`
	SecondCheckin              *string                         `json:"second_checkin"`
	LastCheckin                *string                         `json:"last_checkin"`
	DaysSinceCheckin           *int                            `json:"days_since_checkin"`
	CheckinsFirst7Days         int                             `json:"checkins_first_7_days"`
	CheckinsFirst14Days        int                             `json:"checkins_first_14_days"`
	CheckinsFirst30Days        int                             `json:"checkins_first_30_days"`
	Status                     string                          `json:"status"`
	StatusMessage              string                          `json:"status_message"`
	Recommendation             RetentionRecommendationResponse `json:"recommendation"`
}
