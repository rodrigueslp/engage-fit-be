package domain

import "time"

type AttendanceDecisionOutcome string
type AttendanceReasonCode string

const (
	AttendanceAllowed AttendanceDecisionOutcome = "allowed"
	AttendanceDenied  AttendanceDecisionOutcome = "denied"

	AttendanceReasonActiveEnrollment           AttendanceReasonCode = "active_enrollment"
	AttendanceReasonCorporateRelationship      AttendanceReasonCode = "corporate_relationship"
	AttendanceReasonConfirmedReservation       AttendanceReasonCode = "confirmed_reservation"
	AttendanceReasonEligibleWalkIn             AttendanceReasonCode = "eligible_walk_in"
	AttendanceReasonPersonUnavailable          AttendanceReasonCode = "person_unavailable"
	AttendanceReasonInactiveRelationship       AttendanceReasonCode = "inactive_relationship"
	AttendanceReasonNoActiveEnrollment         AttendanceReasonCode = "no_active_enrollment"
	AttendanceReasonSessionUnavailable         AttendanceReasonCode = "session_unavailable"
	AttendanceReasonOutsideAttendanceWindow    AttendanceReasonCode = "outside_attendance_window"
	AttendanceReasonWaitlisted                 AttendanceReasonCode = "waitlisted"
	AttendanceReasonSessionRequiresReservation AttendanceReasonCode = "session_requires_reservation"
	AttendanceReasonNoSessionEntitlement       AttendanceReasonCode = "no_session_entitlement"
	AttendanceReasonTargetNotIncluded          AttendanceReasonCode = "target_not_included"
	AttendanceReasonUsageLimitReached          AttendanceReasonCode = "usage_limit_reached"
	AttendanceReasonBookingRequired            AttendanceReasonCode = "booking_required"
	AttendanceReasonLegacyTransition           AttendanceReasonCode = "legacy_transition"
	AttendanceReasonContractPending            AttendanceReasonCode = "contract_pending"
	AttendanceReasonHealthFormPending          AttendanceReasonCode = "health_form_pending"
	AttendanceReasonPaymentRequired            AttendanceReasonCode = "payment_required"
	AttendanceReasonPaymentOverdue             AttendanceReasonCode = "payment_overdue"
)

type AttendanceSubject struct {
	PersonID           ID
	StudentID          ID
	Name               string
	Source             Source
	RelationshipActive bool
	ActiveEnrollmentID ID
}

type AttendanceDecision struct {
	ID               ID                        `json:"decision_id"`
	OrganizationID   ID                        `json:"-"`
	UnitID           ID                        `json:"unit_id"`
	PersonID         ID                        `json:"person_id"`
	StudentID        ID                        `json:"student_id"`
	PersonName       string                    `json:"person_name"`
	Source           Source                    `json:"source"`
	SessionID        *ID                       `json:"session_id,omitempty"`
	EnrollmentID     *ID                       `json:"enrollment_id,omitempty"`
	EntitlementID    *ID                       `json:"entitlement_id,omitempty"`
	ReservationID    *ID                       `json:"reservation_id,omitempty"`
	RemainingUsage   *int                      `json:"remaining_usage,omitempty"`
	Outcome          AttendanceDecisionOutcome `json:"outcome"`
	ReasonCode       AttendanceReasonCode      `json:"reason_code"`
	Reason           string                    `json:"reason"`
	OccurredAt       time.Time                 `json:"occurred_at"`
	ActorUserID      ID                        `json:"-"`
	IdempotentReplay bool                      `json:"idempotent_replay"`
}

type AttendanceRecord struct {
	ID              ID        `json:"attendance_id"`
	OrganizationID  ID        `json:"-"`
	UnitID          ID        `json:"unit_id"`
	DecisionID      ID        `json:"decision_id"`
	PersonID        ID        `json:"person_id"`
	StudentID       ID        `json:"student_id"`
	SessionID       *ID       `json:"session_id,omitempty"`
	CheckinID       *ID       `json:"checkin_id,omitempty"`
	LocalDate       string    `json:"local_date"`
	OccurredAt      time.Time `json:"occurred_at"`
	AlreadyRecorded bool      `json:"already_recorded"`
}

type AttendanceAdmission struct {
	Decision   AttendanceDecision `json:"decision"`
	Attendance *AttendanceRecord  `json:"attendance,omitempty"`
}

type AttendanceCommandRequest struct {
	OrganizationID ID
	IdempotencyKey string
	Fingerprint    string
	DecisionID     ID
}
