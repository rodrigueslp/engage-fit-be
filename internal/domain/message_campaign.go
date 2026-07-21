package domain

import "time"

type MessageCampaign struct {
	ID           ID
	BoxID        ID
	CampaignID   ID
	Name         string
	Audience     MessageAudience
	TemplateID   ID
	TemplateType MessageTemplateType
	SentAt       *time.Time
	CreatedAt    time.Time
}
