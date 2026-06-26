package domain

import "time"

type MessageCampaign struct {
	ID         ID
	BoxID      ID
	Name       string
	Audience   MessageAudience
	TemplateID ID
	SentAt     *time.Time
	CreatedAt  time.Time
}
