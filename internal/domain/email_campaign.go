package domain

import "time"

type EmailCampaign struct {
	ID         ID
	BoxID      ID
	CampaignID ID
	Name       string
	Audience   MessageAudience
	TemplateID ID
	SentAt     *time.Time
	CreatedAt  time.Time
}
