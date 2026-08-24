package domain

import "time"

type EnrollmentLifecycleStatus string
type EnrollmentEffectiveStatus string

const (
	EnrollmentLifecycleActive EnrollmentLifecycleStatus = "active"
	EnrollmentLifecycleEnded  EnrollmentLifecycleStatus = "ended"

	EnrollmentEffectiveScheduled EnrollmentEffectiveStatus = "scheduled"
	EnrollmentEffectiveActive    EnrollmentEffectiveStatus = "active"
	EnrollmentEffectiveExpired   EnrollmentEffectiveStatus = "expired"
	EnrollmentEffectiveEnded     EnrollmentEffectiveStatus = "ended"
)

type EnrollmentSubject struct {
	PersonID         ID
	DisplayName      string
	Eligible         bool
	IneligibleReason string
}

type EnrollmentAddon struct {
	ID                 ID
	OrganizationID     ID
	EnrollmentID       ID
	PlanVersionAddonID ID
	Name               string
	PriceDeltaCents    int
	Summary            string
	CreatedAt          time.Time
}

type EnrollmentTerms struct {
	PlanID           ID
	PlanVersionID    ID
	VersionNumber    int
	Name             string
	Description      string
	PriceCents       int
	BasePriceCents   int
	AddonsPriceCents int
	TotalPriceCents  int
	Currency         string
	BillingCycle     MembershipBillingCycle
	Summary          string
	BaseSummary      string
}

type Enrollment struct {
	ID               ID
	OrganizationID   ID
	PersonID         ID
	HomeUnitID       ID
	PlanVersionID    ID
	LifecycleStatus  EnrollmentLifecycleStatus
	EffectiveStatus  EnrollmentEffectiveStatus
	StartsOn         time.Time
	ValidUntil       *time.Time
	EndedOn          *time.Time
	EndedAt          *time.Time
	EndedReason      string
	Revision         int
	BasePriceCents   int
	AddonsPriceCents int
	TotalPriceCents  int
	Addons           []EnrollmentAddon
	CreatedByUserID  ID
	EndedByUserID    ID
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Terms            EnrollmentTerms
}

type EnrollmentHistory struct {
	Current *Enrollment
	History []Enrollment
}

type EnrollmentCommandRequest struct {
	ID             ID
	OrganizationID ID
	Command        string
	IdempotencyKey string
	Fingerprint    string
	EnrollmentID   ID
	ResponseData   map[string]any
	CreatedAt      time.Time
}

type EnrollmentEvent struct {
	ID             ID
	OrganizationID ID
	UnitID         ID
	PersonID       ID
	EnrollmentID   ID
	ActorUserID    ID
	Action         string
	BeforeData     map[string]any
	AfterData      map[string]any
	Reason         string
	CreatedAt      time.Time
}

type EnrollmentPrivacyExport struct {
	Enrollment Enrollment
	Events     []EnrollmentEvent
}

func EnrollmentStatus(value Enrollment, asOf time.Time) EnrollmentEffectiveStatus {
	if value.LifecycleStatus == EnrollmentLifecycleEnded {
		return EnrollmentEffectiveEnded
	}
	if enrollmentDateKey(asOf) < enrollmentDateKey(value.StartsOn) {
		return EnrollmentEffectiveScheduled
	}
	if value.ValidUntil != nil && enrollmentDateKey(asOf) > enrollmentDateKey(*value.ValidUntil) {
		return EnrollmentEffectiveExpired
	}
	return EnrollmentEffectiveActive
}

func enrollmentDateKey(value time.Time) int {
	return value.Year()*10000 + int(value.Month())*100 + value.Day()
}
