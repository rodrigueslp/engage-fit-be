package domain

import "time"

type ReservationStatus string

const (
	ReservationConfirmed  ReservationStatus = "confirmed"
	ReservationWaitlisted ReservationStatus = "waitlisted"
	ReservationCancelled  ReservationStatus = "cancelled"
	ReservationNoShow     ReservationStatus = "no_show"
)

type Reservation struct {
	ID                     ID                `json:"id"`
	OrganizationID         ID                `json:"-"`
	UnitID                 ID                `json:"unit_id"`
	SessionID              ID                `json:"session_id"`
	PersonID               ID                `json:"person_id"`
	PersonName             string            `json:"person_name"`
	EnrollmentID           ID                `json:"enrollment_id"`
	EntitlementID          ID                `json:"entitlement_id"`
	Status                 ReservationStatus `json:"status"`
	WaitlistPosition       *int              `json:"waitlist_position,omitempty"`
	MakeupForReservationID *ID               `json:"makeup_for_reservation_id,omitempty"`
	Revision               int               `json:"revision"`
	CancelledReason        string            `json:"cancelled_reason,omitempty"`
	CreatedAt              time.Time         `json:"created_at"`
	UpdatedAt              time.Time         `json:"updated_at"`
}

type ReservationSummary struct {
	SessionID      ID  `json:"session_id"`
	Capacity       int `json:"capacity"`
	ConfirmedCount int `json:"confirmed_count"`
	WaitlistCount  int `json:"waitlist_count"`
	AvailableCount int `json:"available_count"`
}

type ReservationEntitlement struct {
	EnrollmentID  ID
	EntitlementID ID
	AccessType    MembershipAccessType
	Quantity      *int
	PeriodUnit    *MembershipPeriodUnit
	DailyLimit    *int
}

type ReservationCommandRequest struct {
	ID             ID
	OrganizationID ID
	Command        string
	IdempotencyKey string
	Fingerprint    string
	ReservationID  ID
	ResponseData   map[string]any
	CreatedAt      time.Time
}

type ReservationAuditEvent struct {
	ID             ID
	OrganizationID ID
	UnitID         ID
	SessionID      ID
	ReservationID  ID
	PersonID       ID
	ActorUserID    ID
	ActorAthleteID ID
	Action         string
	Reason         string
	CreatedAt      time.Time
}
