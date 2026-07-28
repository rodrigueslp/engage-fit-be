package dto

type RetentionSignalResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type RetentionRadarResponse struct {
	StudentID                  string                    `json:"student_id"`
	StudentName                string                    `json:"student_name"`
	StudentPhone               string                    `json:"student_phone"`
	Source                     string                    `json:"source"`
	ContactStatus              string                    `json:"contact_status"`
	Level                      string                    `json:"level"`
	FirstCheckin               *string                   `json:"first_checkin"`
	LastCheckin                *string                   `json:"last_checkin"`
	DaysSinceCheckin           *int                      `json:"days_since_checkin"`
	RecentCheckins             int                       `json:"recent_checkins"`
	PreviousCheckins           int                       `json:"previous_checkins"`
	RecentWeeklyAverage        float64                   `json:"recent_weekly_average"`
	PreviousWeeklyAverage      float64                   `json:"previous_weekly_average"`
	DropPercentage             *float64                  `json:"drop_percentage"`
	Signals                    []RetentionSignalResponse `json:"signals"`
	LastCompletedIntervention  *string                   `json:"last_completed_intervention"`
	FirstReturnAfterAction     *string                   `json:"first_return_after_action"`
	ReturnWithin3Days          bool                      `json:"return_within_3_days"`
	ReturnWithin7Days          bool                      `json:"return_within_7_days"`
	ReturnWithin14Days         bool                      `json:"return_within_14_days"`
	WorkflowStatus             string                    `json:"workflow_status"`
	FollowUpDueAt              *string                   `json:"follow_up_due_at"`
	LastInterventionID         string                    `json:"last_intervention_id"`
	LastInterventionChannel    string                    `json:"last_intervention_channel"`
	LastInterventionStatus     string                    `json:"last_intervention_status"`
	LastInterventionOutcome    string                    `json:"last_intervention_outcome"`
	LastInterventionPlannedFor *string                   `json:"last_intervention_planned_for"`
	LastInterventionCreatedAt  *string                   `json:"last_intervention_created_at"`
}

type RetentionInterventionRequest struct {
	Channel    string  `json:"channel" binding:"required"`
	Status     string  `json:"status"`
	Outcome    string  `json:"outcome"`
	PlannedFor *string `json:"planned_for"`
	Notes      string  `json:"notes"`
}

type RetentionInterventionUpdateRequest struct {
	Status     string  `json:"status" binding:"required"`
	Outcome    string  `json:"outcome"`
	PlannedFor *string `json:"planned_for"`
	Notes      string  `json:"notes"`
}

type RetentionInterventionResponse struct {
	ID              string  `json:"id"`
	StudentID       string  `json:"student_id"`
	CreatedByUserID string  `json:"created_by_user_id"`
	Channel         string  `json:"channel"`
	Status          string  `json:"status"`
	Outcome         string  `json:"outcome"`
	PlannedFor      *string `json:"planned_for"`
	CompletedAt     *string `json:"completed_at"`
	Notes           string  `json:"notes"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}
