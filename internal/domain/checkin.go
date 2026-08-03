package domain

import "time"

type CheckinEntryMethod string

const (
	CheckinEntryImport      CheckinEntryMethod = "import"
	CheckinEntryManual      CheckinEntryMethod = "manual"
	CheckinEntrySelfService CheckinEntryMethod = "self_service"
)

type Checkin struct {
	ID                   ID
	BoxID                ID
	StudentID            ID
	CheckinDate          time.Time
	CheckinTime          *time.Time
	Source               Source
	ImportHistoryID      ID
	EntryMethod          CheckinEntryMethod
	RecordedByUserID     ID
	SelfCheckinSessionID ID
	CreatedAt            time.Time
}

type SelfCheckinSession struct {
	ID              ID
	BoxID           ID
	CreatedByUserID ID
	TokenHash       string
	ExpiresAt       time.Time
	CreatedAt       time.Time
}
