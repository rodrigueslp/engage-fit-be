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

type AnnouncementPrivacyView struct {
	AnnouncementID ID         `json:"announcement_id"`
	Title          string     `json:"title"`
	Priority       string     `json:"priority"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
	FirstViewedAt  *time.Time `json:"first_viewed_at,omitempty"`
	LastViewedAt   *time.Time `json:"last_viewed_at,omitempty"`
	ViewCount      int        `json:"view_count"`
}

type StudentPrivacyExport struct {
	Student                Student
	Person                 *PersonPrivacyExport
	Checkins               []Checkin
	Progress               []CampaignProgress
	Communications         []PrivacyCommunication
	ContactConsents        []ContactConsentEvent
	RetentionInterventions []RetentionIntervention
	Enrollments            []EnrollmentPrivacyExport
	Contracts              []ContractPrivacyExport
	Reservations           []Reservation
	MemberBilling          []MemberBillingSnapshot
	Attendance             []AttendanceAdmission
	HealthForms            []HealthFormPrivacyExport
	Announcements          []AnnouncementPrivacyView
	CommerceSales          []CommerceSale
	ExportedAt             time.Time
}
