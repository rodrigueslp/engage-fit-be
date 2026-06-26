package domain

import "time"

type Checkin struct {
	ID              ID
	BoxID           ID
	StudentID       ID
	CheckinDate     time.Time
	CheckinTime     *time.Time
	Source          Source
	ImportHistoryID ID
	CreatedAt       time.Time
}
