package domain

import "time"

type HealthFormStatus string

const (
	HealthFormStatusSubmitted HealthFormStatus = "submitted"
	HealthFormStatusRevoked   HealthFormStatus = "consent_revoked"
	HealthFormStatusExpired   HealthFormStatus = "expired"
	HealthFormSchemaVersion                    = 1
)

// PARQAnswers is deliberately small and versioned. Details are encrypted at
// rest by the healthforms repository and never copied into audit events.
type PARQAnswers struct {
	HeartCondition bool   `json:"heart_condition"`
	ChestPain      bool   `json:"chest_pain"`
	Dizziness      bool   `json:"dizziness"`
	BoneJoint      bool   `json:"bone_joint"`
	Medication     bool   `json:"medication"`
	OtherReason    bool   `json:"other_reason"`
	Details        string `json:"details,omitempty"`
}

func (a PARQAnswers) RequiresReview() bool {
	return a.HeartCondition || a.ChestPain || a.Dizziness || a.BoneJoint || a.Medication || a.OtherReason
}

type HealthFormSubmission struct {
	ID                ID               `json:"id"`
	OrganizationID    ID               `json:"-"`
	UnitID            ID               `json:"unit_id"`
	PersonID          ID               `json:"person_id"`
	EnrollmentID      ID               `json:"enrollment_id"`
	SchemaVersion     int              `json:"schema_version"`
	Status            HealthFormStatus `json:"status"`
	ConsentVersion    string           `json:"consent_version"`
	ConsentText       string           `json:"consent_text"`
	ConsentAcceptedAt time.Time        `json:"consent_accepted_at"`
	ConsentRevokedAt  *time.Time       `json:"consent_revoked_at,omitempty"`
	RequiresReview    bool             `json:"requires_review"`
	Answers           *PARQAnswers     `json:"answers,omitempty"`
	RetentionDueAt    time.Time        `json:"retention_due_at"`
	CreatedByUserID   ID               `json:"-"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

type HealthFormPrivacyExport struct {
	Submission HealthFormSubmission `json:"submission"`
	Answers    PARQAnswers          `json:"answers"`
}

type HealthFormSession struct {
	ID              ID         `json:"id"`
	OrganizationID  ID         `json:"-"`
	UnitID          ID         `json:"-"`
	PersonID        ID         `json:"-"`
	EnrollmentID    ID         `json:"-"`
	SchemaVersion   int        `json:"schema_version"`
	TokenHash       string     `json:"-"`
	Status          string     `json:"status"`
	ExpiresAt       time.Time  `json:"expires_at"`
	SubmittedAt     *time.Time `json:"submitted_at,omitempty"`
	CreatedByUserID ID         `json:"-"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type HealthFormSessionLaunch struct {
	LaunchToken string    `json:"launch_token"`
	LaunchPath  string    `json:"launch_path"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type PublicHealthFormSession struct {
	SchemaVersion  int       `json:"schema_version"`
	ConsentVersion string    `json:"consent_version"`
	ConsentText    string    `json:"consent_text"`
	ExpiresAt      time.Time `json:"expires_at"`
}

func HealthFormAnswersAAD(organizationID, submissionID ID) string {
	return "healthforms:" + string(organizationID) + ":" + string(submissionID) + ":answers:v1"
}
