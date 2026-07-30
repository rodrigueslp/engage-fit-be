package domain

import "time"

type CheckinIngestionSource struct {
	ID              ID
	BoxID           ID
	CreatedByUserID ID
	Name            string
	Source          Source
	TokenHash       string
	Enabled         bool
	LastIngestedAt  *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CheckinIngestionBatch struct {
	ID              ID
	SourceID        ID
	BoxID           ID
	IdempotencyKey  string
	Status          string
	ImportHistoryID ID
	TotalRecords    int
	StudentsCreated int
	CheckinsCreated int
	CreatedAt       time.Time
	CompletedAt     *time.Time
}
