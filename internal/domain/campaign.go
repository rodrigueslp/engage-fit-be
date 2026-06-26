package domain

import "time"

type Campaign struct {
	ID          ID
	BoxID       ID
	Name        string
	Description string
	StartDate   time.Time
	EndDate     time.Time
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
