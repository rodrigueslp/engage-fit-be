package domain

import "time"

// TodayOperations is a read-only operational projection. Source modules keep
// ownership of their records; this projection only composes the current unit's
// agenda and actionable work for the selected local day.
type TodayOperations struct {
	Date              string              `json:"date"`
	Timezone          string              `json:"timezone"`
	GeneratedAt       time.Time           `json:"generated_at"`
	Sessions          []TodaySession      `json:"sessions"`
	Charges           []TodayCharge       `json:"charges"`
	PeopleActions     []TodayPeopleAction `json:"people_actions"`
	AttendanceTotal   int                 `json:"attendance_total"`
	DeniedAccessTotal int                 `json:"denied_access_total"`
}

type TodaySession struct {
	SessionID        ID        `json:"session_id"`
	TargetName       string    `json:"target_name"`
	ProfessionalName string    `json:"professional_name"`
	SpaceName        string    `json:"space_name,omitempty"`
	StartsAt         time.Time `json:"starts_at"`
	EndsAt           time.Time `json:"ends_at"`
	Capacity         int       `json:"capacity"`
	ConfirmedCount   int       `json:"confirmed_count"`
	WaitlistCount    int       `json:"waitlist_count"`
	AttendanceCount  int       `json:"attendance_count"`
}

type TodayCharge struct {
	ChargeID        ID     `json:"charge_id"`
	PersonID        ID     `json:"person_id"`
	EnrollmentID    ID     `json:"enrollment_id"`
	PersonName      string `json:"person_name"`
	DueOn           string `json:"due_on"`
	OpenAmountCents int    `json:"open_amount_cents"`
	Status          string `json:"status"`
}

type TodayPeopleAction struct {
	Kind         string `json:"kind"`
	PersonID     ID     `json:"person_id"`
	EnrollmentID ID     `json:"enrollment_id"`
	PersonName   string `json:"person_name"`
	ReferenceID  ID     `json:"reference_id"`
	Label        string `json:"label"`
}
