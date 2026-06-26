package domain

import "time"

type MessageRecipient struct {
	ID                ID
	MessageCampaignID ID
	StudentID         ID
	Phone             string
	Status            MessageRecipientStatus
	ErrorMessage      string
	SentAt            *time.Time
	CreatedAt         time.Time
}
