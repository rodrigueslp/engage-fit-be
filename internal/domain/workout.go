package domain

import "time"

type WorkoutStatus string

const (
	WorkoutStatusDraft     WorkoutStatus = "draft"
	WorkoutStatusPublished WorkoutStatus = "published"
)

type WorkoutSectionType string

const (
	WorkoutSectionWarmup    WorkoutSectionType = "warmup"
	WorkoutSectionSkill     WorkoutSectionType = "skill"
	WorkoutSectionStrength  WorkoutSectionType = "strength"
	WorkoutSectionWOD       WorkoutSectionType = "wod"
	WorkoutSectionAccessory WorkoutSectionType = "accessory"
	WorkoutSectionCooldown  WorkoutSectionType = "cooldown"
	WorkoutSectionOther     WorkoutSectionType = "other"
)

type WorkoutSection struct {
	Type    WorkoutSectionType `json:"type"`
	Title   string             `json:"title"`
	Content string             `json:"content"`
}

type WorkoutClassification struct {
	Version          string           `json:"version"`
	GeneratedBy      string           `json:"generated_by"`
	SuggestedTitle   string           `json:"suggested_title"`
	Sections         []WorkoutSection `json:"sections"`
	Formats          []string         `json:"formats"`
	DurationSeconds  int              `json:"duration_seconds,omitempty"`
	MovementMentions []string         `json:"movement_mentions"`
}

type WorkoutMessageDraftStatus string

const (
	WorkoutMessageDraftStatusDraft    WorkoutMessageDraftStatus = "draft"
	WorkoutMessageDraftStatusApproved WorkoutMessageDraftStatus = "approved"
	WorkoutMessageDraftStatusSent     WorkoutMessageDraftStatus = "sent"
)

type Workout struct {
	ID             ID
	BoxID          ID
	WorkoutDate    time.Time
	Title          string
	Goal           string
	Movements      string
	CoachNotes     string
	RawText        string
	Classification WorkoutClassification
	ClassifiedAt   *time.Time
	Status         WorkoutStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
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
	ProviderMessageSID    string
	ProviderStatus        string
	DispatchID            ID
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
