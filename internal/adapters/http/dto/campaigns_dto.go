package dto

type CampaignRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	Active      *bool  `json:"active,omitempty"`
}

type CampaignGoalRequest struct {
	Source         string `json:"source"`
	TargetCheckins int    `json:"target_checkins"`
}

type CampaignResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	Active      bool   `json:"active"`
}

type CampaignGoalResponse struct {
	ID             string `json:"id"`
	CampaignID     string `json:"campaign_id"`
	Source         string `json:"source"`
	TargetCheckins int    `json:"target_checkins"`
}

type CampaignProgressResponse struct {
	ID                 string  `json:"id"`
	CampaignID         string  `json:"campaign_id"`
	StudentID          string  `json:"student_id"`
	StudentName        string  `json:"student_name,omitempty"`
	StudentEmail       string  `json:"student_email,omitempty"`
	StudentPhone       string  `json:"student_phone,omitempty"`
	StudentSource      string  `json:"student_source,omitempty"`
	CurrentCheckins    int     `json:"current_checkins"`
	TargetCheckins     int     `json:"target_checkins"`
	RemainingCheckins  int     `json:"remaining_checkins"`
	ProgressPercentage float64 `json:"progress_percentage"`
	Achieved           bool    `json:"achieved"`
	NearGoal           bool    `json:"near_goal"`
}
