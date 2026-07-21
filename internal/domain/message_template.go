package domain

import "time"

type MessageTemplate struct {
	ID             ID
	BoxID          ID
	Name           string
	Content        string
	ContentSID     string
	TemplateType   MessageTemplateType
	Provider       string
	ApprovalStatus MessageTemplateApprovalStatus
	Language       string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
