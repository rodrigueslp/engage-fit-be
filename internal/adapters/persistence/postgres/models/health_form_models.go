package models

import "time"

type HealthFormSubmissionModel struct {
	ID                string `gorm:"primaryKey"`
	OrganizationID    string `gorm:"index"`
	UnitID            string `gorm:"index"`
	PersonID          string `gorm:"index"`
	EnrollmentID      string `gorm:"index"`
	SchemaVersion     int
	Status            string
	ConsentVersion    string
	ConsentText       string
	ConsentAcceptedAt time.Time
	ConsentRevokedAt  *time.Time
	RequiresReview    bool
	AnswersEncrypted  string
	RetentionDueAt    time.Time
	CreatedByUserID   string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (HealthFormSubmissionModel) TableName() string { return "health_form_submissions" }

type HealthFormAccessEventModel struct {
	ID             string `gorm:"primaryKey"`
	OrganizationID string
	UnitID         string
	SubmissionID   string
	ActorUserID    string
	Action         string
	CreatedAt      time.Time
}

func (HealthFormAccessEventModel) TableName() string { return "health_form_access_events" }

type HealthFormSessionModel struct {
	ID              string `gorm:"primaryKey"`
	OrganizationID  string
	UnitID          string
	PersonID        string
	EnrollmentID    string
	SchemaVersion   int
	TokenHash       string `gorm:"uniqueIndex"`
	Status          string
	ExpiresAt       time.Time
	SubmittedAt     *time.Time
	CreatedByUserID string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (HealthFormSessionModel) TableName() string { return "health_form_sessions" }
