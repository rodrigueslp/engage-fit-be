package domain

import "time"

type WorkoutStatus string

const (
	WorkoutStatusDraft     WorkoutStatus = "draft"
	WorkoutStatusPublished WorkoutStatus = "published"
)

type WorkoutMessageDraftStatus string

const (
	WorkoutMessageDraftStatusDraft    WorkoutMessageDraftStatus = "draft"
	WorkoutMessageDraftStatusApproved WorkoutMessageDraftStatus = "approved"
	WorkoutMessageDraftStatusSent     WorkoutMessageDraftStatus = "sent"
)

type Workout struct {
	ID          ID
	BoxID       ID
	WorkoutDate time.Time
	Title       string
	Goal        string
	Movements   string
	CoachNotes  string
	Status      WorkoutStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type WorkoutMessageDraft struct {
	ID               ID
	BoxID            ID
	WorkoutID        ID
	CampaignID       ID
	Audience         MessageAudience
	GeneratedBody    string
	ApprovedBody     string
	Status           WorkoutMessageDraftStatus
	TotalRecipients  int
	SentRecipients   int
	FailedRecipients int
	GeneratedAt      time.Time
	ApprovedAt       *time.Time
	SentAt           *time.Time
}

type WorkoutMessageRecipient struct {
	ID                    ID
	WorkoutMessageDraftID ID
	StudentID             ID
	Phone                 string
	Status                MessageRecipientStatus
	ErrorMessage          string
	SentAt                *time.Time
	CreatedAt             time.Time
}

type LLMGenerationLog struct {
	ID            ID
	BoxID         ID
	WorkoutID     ID
	DraftID       ID
	Provider      string
	Model         string
	PromptSummary string
	Status        string
	ErrorMessage  string
	CreatedAt     time.Time
}
