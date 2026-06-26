package dto

type StudentResponse struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Email             string `json:"email"`
	Phone             string `json:"phone"`
	Source            string `json:"source"`
	ExternalID        string `json:"external_id"`
	RiskStatus        string `json:"risk_status"`
	RiskLastMessageAt string `json:"risk_last_message_at,omitempty"`
}

type UpdateStudentRiskStatusRequest struct {
	RiskStatus string `json:"risk_status"`
}

type CheckinResponse struct {
	ID          string `json:"id"`
	StudentID   string `json:"student_id"`
	CheckinDate string `json:"checkin_date"`
	CheckinTime string `json:"checkin_time,omitempty"`
	Source      string `json:"source"`
}
