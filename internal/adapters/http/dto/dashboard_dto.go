package dto

type DashboardSummaryResponse struct {
	TotalStudents      int            `json:"total_students"`
	TotalCheckins      int            `json:"total_checkins"`
	EligibleStudents   int            `json:"eligible_students"`
	NearGoalStudents   int            `json:"near_goal_students"`
	AtRiskStudents     int            `json:"at_risk_students"`
	PendingRewards     int            `json:"pending_rewards"`
	DeliveredRewards   int            `json:"delivered_rewards"`
	CheckinsByPlatform map[string]int `json:"checkins_by_platform"`
}
