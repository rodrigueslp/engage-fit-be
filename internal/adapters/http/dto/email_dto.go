package dto

type EmailSettingsRequest struct {
	Provider  string `json:"provider"`
	SMTPHost  string `json:"smtp_host"`
	SMTPPort  int    `json:"smtp_port"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	FromEmail string `json:"from_email"`
	FromName  string `json:"from_name"`
	Enabled   bool   `json:"enabled"`
}

type EmailSettingsResponse struct {
	ID          string `json:"id"`
	BoxID       string `json:"box_id"`
	Provider    string `json:"provider"`
	SMTPHost    string `json:"smtp_host"`
	SMTPPort    int    `json:"smtp_port"`
	Username    string `json:"username"`
	FromEmail   string `json:"from_email"`
	FromName    string `json:"from_name"`
	HasPassword bool   `json:"has_password"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	Enabled     bool   `json:"enabled"`
}

type EmailTemplateRequest struct {
	Name    string `json:"name"`
	Subject string `json:"subject"`
	Content string `json:"content"`
}

type EmailCampaignRequest struct {
	Name       string `json:"name"`
	CampaignID string `json:"campaign_id"`
	Audience   string `json:"audience"`
	TemplateID string `json:"template_id"`
}

type EmailTemplateResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Subject string `json:"subject"`
	Content string `json:"content"`
}

type EmailCampaignResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	CampaignID string `json:"campaign_id"`
	Audience   string `json:"audience"`
	TemplateID string `json:"template_id"`
	SentAt     string `json:"sent_at,omitempty"`
}

type SendEmailCampaignResponse struct {
	Total  int `json:"total"`
	Sent   int `json:"sent"`
	Failed int `json:"failed"`
}

type EmailCampaignPreviewResponse struct {
	Total       int    `json:"total"`
	Subject     string `json:"subject"`
	Body        string `json:"body"`
	StudentID   string `json:"student_id,omitempty"`
	StudentName string `json:"student_name,omitempty"`
	Email       string `json:"email,omitempty"`
}

type EmailRecipientResponse struct {
	ID              string `json:"id"`
	EmailCampaignID string `json:"email_campaign_id"`
	StudentID       string `json:"student_id"`
	Email           string `json:"email"`
	Status          string `json:"status"`
	ErrorMessage    string `json:"error_message,omitempty"`
	SentAt          string `json:"sent_at,omitempty"`
	CreatedAt       string `json:"created_at"`
}
