package domain

import "time"

type ImportHistory struct {
	ID           ID
	BoxID        ID
	Filename     string
	Source       Source
	TotalRecords int
	ImportedAt   time.Time
}
