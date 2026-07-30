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

type ContactConsentEvent struct {
	Action         string
	Source         string
	Phone          string
	ConsentVersion string
	ConsentText    string
	CreatedAt      time.Time
}

type StudentPrivacyExport struct {
	Student                Student
	Checkins               []Checkin
	Progress               []CampaignProgress
	Communications         []PrivacyCommunication
	ContactConsents        []ContactConsentEvent
	RetentionInterventions []RetentionIntervention
	ExportedAt             time.Time
}
