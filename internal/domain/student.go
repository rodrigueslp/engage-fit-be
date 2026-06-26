package domain

import "time"

type Student struct {
	ID                ID
	BoxID             ID
	Name              string
	Email             string
	Phone             string
	Source            Source
	ExternalID        string
	RiskStatus        StudentRiskStatus
	RiskLastMessageAt *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
