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
	Recommendation               RetentionRecommendationResponse `json:"recommendation"`
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

type OnboardingJourneyResponse struct {
	StudentID               string                          `json:"student_id"`
	StudentName             string                          `json:"student_name"`
	StudentPhone            string                          `json:"student_phone"`
	Source                  string                          `json:"source"`
	ContactStatus           string                          `json:"contact_status"`
	MembershipStartedAt     string                          `json:"membership_started_at"`
	MembershipStartedSource string                          `json:"membership_started_source"`
	Day                     int                             `json:"day"`
	FirstCheckin            *string                         `json:"first_checkin"`
	SecondCheckin           *string                         `json:"second_checkin"`
	LastCheckin             *string                         `json:"last_checkin"`
	DaysSinceCheckin        *int                            `json:"days_since_checkin"`
	CheckinsFirst7Days      int                             `json:"checkins_first_7_days"`
	CheckinsFirst14Days     int                             `json:"checkins_first_14_days"`
	CheckinsFirst30Days     int                             `json:"checkins_first_30_days"`
	Status                  string                          `json:"status"`
	StatusMessage           string                          `json:"status_message"`
	Recommendation          RetentionRecommendationResponse `json:"recommendation"`
}
