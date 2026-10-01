package domain

import "time"

type MembershipAccessPurpose string
type MembershipAccessOutcome string
type MembershipAccessReason string

const (
	MembershipAccessBooking MembershipAccessPurpose = "booking"
	MembershipAccessCheckin MembershipAccessPurpose = "checkin"

	MembershipAccessAllowed MembershipAccessOutcome = "allowed"
	MembershipAccessDenied  MembershipAccessOutcome = "denied"

	MembershipAccessReasonEligible          MembershipAccessReason = "eligible"
	MembershipAccessReasonLegacyTransition  MembershipAccessReason = "legacy_transition"
	MembershipAccessReasonNoEnrollment      MembershipAccessReason = "no_active_enrollment"
	MembershipAccessReasonOutsideValidity   MembershipAccessReason = "outside_validity"
	MembershipAccessReasonTargetNotIncluded MembershipAccessReason = "target_not_included"
	MembershipAccessReasonUnitNotIncluded   MembershipAccessReason = "unit_not_included"
	MembershipAccessReasonLimitReached      MembershipAccessReason = "usage_limit_reached"
	MembershipAccessReasonBookingRequired   MembershipAccessReason = "booking_required"
	MembershipAccessReasonContractPending   MembershipAccessReason = "contract_pending"
	MembershipAccessReasonHealthFormPending MembershipAccessReason = "health_form_pending"
	MembershipAccessReasonPaymentRequired   MembershipAccessReason = "payment_required"
	MembershipAccessReasonPaymentOverdue    MembershipAccessReason = "payment_overdue"
)

type MembershipAccessPrerequisites struct {
	ContractAccepted    bool
	HealthFormSubmitted bool
	InitialPaymentPaid  bool
	PaymentOverdue      bool
}

type MembershipAccessCandidate struct {
	EnrollmentID    ID
	EntitlementID   ID
	HomeUnitID      ID
	TargetKind      ScheduleTargetKind
	TargetID        ID
	AccessType      MembershipAccessType
	Quantity        *int
	PeriodUnit      *MembershipPeriodUnit
	DailyLimit      *int
	BookingRequired bool
	StartsOn        time.Time
	ValidUntil      *time.Time
	Timezone        string
}

type MembershipAccessRequest struct {
	OrganizationID       ID
	UnitID               ID
	PersonID             ID
	TargetKind           ScheduleTargetKind
	TargetID             ID
	OccurredAt           time.Time
	Purpose              MembershipAccessPurpose
	PreferredEnrollment  ID
	PreferredEntitlement ID
	IgnoreUsage          bool
	UsageAlreadyCounted  bool
}

type MembershipAccessDecision struct {
	Outcome        MembershipAccessOutcome `json:"outcome"`
	ReasonCode     MembershipAccessReason  `json:"reason_code"`
	Reason         string                  `json:"reason"`
	EnrollmentID   ID                      `json:"enrollment_id,omitempty"`
	EntitlementID  ID                      `json:"entitlement_id,omitempty"`
	RemainingUsage *int                    `json:"remaining_usage,omitempty"`
	LegacyFallback bool                    `json:"legacy_fallback"`
}

type MembershipAccessUsageQuery struct {
	OrganizationID ID
	PersonID       ID
	EnrollmentID   ID
	EntitlementID  ID
	From           time.Time
	Until          time.Time
}
