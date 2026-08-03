package domain

import "time"

type ContactActivationStatus string

const (
	ContactActivationAwaitingMessage ContactActivationStatus = "awaiting_message"
	ContactActivationConfirmed       ContactActivationStatus = "confirmed"
	ContactActivationPendingSync     ContactActivationStatus = "pending_sync"
	ContactActivationNeedsReview     ContactActivationStatus = "needs_review"
	ContactActivationExpired         ContactActivationStatus = "expired"
	ContactActivationCancelled       ContactActivationStatus = "cancelled"
)

type ContactActivationRequest struct {
	ID                ID
	BoxID             ID
	StudentID         ID
	StudentName       string
	ClaimedName       string
	Source            Source
	RecentCheckinDate *time.Time
	IsNewStudent      bool
	SenderPhone       string
	Phone             string
	TokenHash         string
	MatchStrategy     string
	Status            ContactActivationStatus
	ConsentVersion    string
	ConsentText       string
	ConsentedAt       *time.Time
	ExpiresAt         time.Time
	ResolvedAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ContactActivationConfig struct {
	BoxID          ID
	BoxName        string
	ActivationCode string
	SenderPhone    string
	ConsentVersion string
	ConsentText    string
}

type ContactActivationSummary struct {
	TotalStudents   int64
	WithPhone       int64
	OptedIn         int64
	OptedOut        int64
	PendingReview   int64
	PendingSync     int64
	AwaitingMessage int64
	ActivationCode  string
	SenderPhone     string
	WhatsappReady   bool
}

type ContactActivationCandidate struct {
	Student          Student
	HasRecentCheckin bool
}

type ContactActivationMatchData struct {
	Candidates        []ContactActivationCandidate
	LatestCheckinDate *time.Time
}
