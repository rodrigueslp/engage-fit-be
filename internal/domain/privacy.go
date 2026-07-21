package domain

import "time"

type PrivacyCommunication struct {
	Channel      string
	CampaignID   ID
	Destination  string
	Status       string
	ErrorMessage string
	SentAt       *time.Time
	CreatedAt    time.Time
}

type StudentPrivacyExport struct {
	Student        Student
	Checkins       []Checkin
	Progress       []CampaignProgress
	Communications []PrivacyCommunication
	ExportedAt     time.Time
}
