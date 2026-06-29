package domain

import "time"

type EmailRecipient struct {
	ID              ID
	EmailCampaignID ID
	StudentID       ID
	Email           string
	Status          MessageRecipientStatus
	ErrorMessage    string
	SentAt          *time.Time
	CreatedAt       time.Time
}
