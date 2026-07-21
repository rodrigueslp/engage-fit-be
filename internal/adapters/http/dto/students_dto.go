package dto

type StudentResponse struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Email                  string `json:"email"`
	Phone                  string `json:"phone"`
	Source                 string `json:"source"`
	ExternalID             string `json:"external_id"`
	RiskStatus             string `json:"risk_status"`
	RiskLastMessageAt      string `json:"risk_last_message_at,omitempty"`
	ContactStatus          string `json:"contact_status"`
	ContactStatusUpdatedAt string `json:"contact_status_updated_at,omitempty"`
	ContactStatusSource    string `json:"contact_status_source,omitempty"`
	AnonymizedAt           string `json:"anonymized_at,omitempty"`
}

type UpdateContactPreferenceRequest struct {
	Status string `json:"status"`
	Source string `json:"source"`
}

type AnonymizeStudentRequest struct {
	Confirmed bool   `json:"confirmed"`
	Reason    string `json:"reason"`
}

type PrivacyCommunicationResponse struct {
	Channel      string `json:"channel"`
	CampaignID   string `json:"campaign_id"`
	Destination  string `json:"destination"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
	SentAt       string `json:"sent_at,omitempty"`
	CreatedAt    string `json:"created_at"`
}

type StudentPrivacyExportResponse struct {
	Student        StudentResponse                `json:"student"`
	Checkins       []CheckinResponse              `json:"checkins"`
	Progress       []CampaignProgressResponse     `json:"campaign_progress"`
	Communications []PrivacyCommunicationResponse `json:"communications"`
	ExportedAt     string                         `json:"exported_at"`
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
