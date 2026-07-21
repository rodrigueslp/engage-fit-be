package dto

type MessageTemplateRequest struct {
	Name           string `json:"name"`
	Content        string `json:"content"`
	ContentSID     string `json:"content_sid"`
	ApprovalStatus string `json:"approval_status"`
}

type MessageCampaignRequest struct {
	Name         string `json:"name"`
	CampaignID   string `json:"campaign_id"`
	Audience     string `json:"audience"`
	TemplateID   string `json:"template_id"`
	TemplateType string `json:"template_type"`
}

type MessageTemplateResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Content        string `json:"content"`
	ContentSID     string `json:"content_sid"`
	TemplateType   string `json:"template_type"`
	Provider       string `json:"provider"`
	ApprovalStatus string `json:"approval_status"`
	Language       string `json:"language"`
	Editable       bool   `json:"editable"`
}

type MessageCampaignResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	CampaignID   string `json:"campaign_id"`
	Audience     string `json:"audience"`
	TemplateID   string `json:"template_id"`
	TemplateType string `json:"template_type"`
	SentAt       string `json:"sent_at,omitempty"`
}

type SendMessageCampaignResponse struct {
	Total  int `json:"total"`
	Sent   int `json:"sent"`
	Failed int `json:"failed"`
}

type MessageCampaignPreviewResponse struct {
	Total       int    `json:"total"`
	Body        string `json:"body"`
	StudentID   string `json:"student_id,omitempty"`
	StudentName string `json:"student_name,omitempty"`
	Phone       string `json:"phone,omitempty"`
}

type MessageRecipientResponse struct {
	ID                 string `json:"id"`
	MessageCampaignID  string `json:"message_campaign_id"`
	StudentID          string `json:"student_id"`
	Phone              string `json:"phone"`
	Status             string `json:"status"`
	ErrorMessage       string `json:"error_message,omitempty"`
	ProviderMessageSID string `json:"provider_message_sid,omitempty"`
	ProviderStatus     string `json:"provider_status,omitempty"`
	DispatchID         string `json:"dispatch_id,omitempty"`
	SentAt             string `json:"sent_at,omitempty"`
	CreatedAt          string `json:"created_at"`
}

type OfficialWhatsappTemplatePreviewResponse struct {
	Type               string `json:"type"`
	Label              string `json:"label"`
	Description        string `json:"description"`
	Editable           bool   `json:"editable"`
	ApprovalStatus     string `json:"approvalStatus"`
	ProviderTemplateID string `json:"providerTemplateId"`
	Preview            string `json:"preview"`
}
