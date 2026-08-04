package domain

import "time"

type ImportStatus string

const (
	ImportStatusProcessing ImportStatus = "processing"
	ImportStatusCompleted  ImportStatus = "completed"
	ImportStatusFailed     ImportStatus = "failed"
)

type ImportHistory struct {
	ID              ID
	BoxID           ID
	Filename        string
	Source          Source
	Status          ImportStatus
	TotalRecords    int
	StudentsCreated *int
	CheckinsCreated *int
	ImportedAt      time.Time
	CompletedAt     *time.Time
	ErrorCode       string
}
