package dto

type MessageTemplateRequest struct {
	Name       string `json:"name"`
	Content    string `json:"content"`
	ContentSID string `json:"content_sid"`
}

type MessageCampaignRequest struct {
	Name       string `json:"name"`
	Audience   string `json:"audience"`
	TemplateID string `json:"template_id"`
}

type MessageTemplateResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	ContentSID string `json:"content_sid"`
}

type MessageCampaignResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Audience   string `json:"audience"`
	TemplateID string `json:"template_id"`
	SentAt     string `json:"sent_at,omitempty"`
}

type SendMessageCampaignResponse struct {
	Total  int `json:"total"`
	Sent   int `json:"sent"`
	Failed int `json:"failed"`
}

type MessageRecipientResponse struct {
	ID                string `json:"id"`
	MessageCampaignID string `json:"message_campaign_id"`
	StudentID         string `json:"student_id"`
	Phone             string `json:"phone"`
	Status            string `json:"status"`
	ErrorMessage      string `json:"error_message,omitempty"`
	SentAt            string `json:"sent_at,omitempty"`
	CreatedAt         string `json:"created_at"`
}
