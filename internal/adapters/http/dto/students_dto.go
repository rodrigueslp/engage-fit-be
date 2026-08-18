package dto

type StudentResponse struct {
	ID                      string `json:"id"`
	PersonID                string `json:"person_id,omitempty"`
	Name                    string `json:"name"`
	Email                   string `json:"email"`
	Phone                   string `json:"phone"`
	Source                  string `json:"source"`
	ExternalID              string `json:"external_id"`
	RiskStatus              string `json:"risk_status"`
	RiskLastMessageAt       string `json:"risk_last_message_at,omitempty"`
	ContactStatus           string `json:"contact_status"`
	ContactStatusUpdatedAt  string `json:"contact_status_updated_at,omitempty"`
	ContactStatusSource     string `json:"contact_status_source,omitempty"`
	MembershipStartedAt     string `json:"membership_started_at,omitempty"`
	MembershipStartedSource string `json:"membership_started_source,omitempty"`
	AnonymizedAt            string `json:"anonymized_at,omitempty"`
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
	Student                StudentResponse                 `json:"student"`
	Person                 *PersonPrivacyExportResponse    `json:"person,omitempty"`
	Checkins               []CheckinResponse               `json:"checkins"`
	Progress               []CampaignProgressResponse      `json:"campaign_progress"`
	Communications         []PrivacyCommunicationResponse  `json:"communications"`
	ContactConsents        []ContactConsentResponse        `json:"contact_consents"`
	RetentionInterventions []RetentionInterventionResponse `json:"retention_interventions"`
	ExportedAt             string                          `json:"exported_at"`
}

type PersonPrivacyExportResponse struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Phone         string                 `json:"phone"`
	Email         string                 `json:"email"`
	CPF           string                 `json:"cpf"`
	ProfileStatus string                 `json:"profile_status"`
	Address       *PersonAddressResponse `json:"address,omitempty"`
}

type PersonAddressResponse struct {
	PostalCode string `json:"postal_code,omitempty"`
	Street     string `json:"street,omitempty"`
	Number     string `json:"number,omitempty"`
	Complement string `json:"complement,omitempty"`
	District   string `json:"district,omitempty"`
	City       string `json:"city,omitempty"`
	State      string `json:"state,omitempty"`
	Country    string `json:"country"`
}

type ContactConsentResponse struct {
	Action         string `json:"action"`
	Source         string `json:"source"`
	Phone          string `json:"phone"`
	ConsentVersion string `json:"consent_version"`
	ConsentText    string `json:"consent_text"`
	CreatedAt      string `json:"created_at"`
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
	EntryMethod string `json:"entry_method"`
}
