package domain

import "time"

// AthletePortal is the privacy-scoped read model used by the member PWA. Every
// item is resolved from an explicit athlete membership/student/person link.
type AthletePortal struct {
	GeneratedAt time.Time           `json:"generated_at"`
	Units       []AthletePortalUnit `json:"units"`
	Notices     []AthleteNotice     `json:"notices"`
}

type AthletePortalScope struct {
	MembershipID   ID
	OrganizationID ID
	UnitID         ID
	PersonID       ID
}

type AthletePortalUnit struct {
	MembershipID ID                       `json:"membership_id"`
	UnitID       ID                       `json:"unit_id"`
	UnitName     string                   `json:"unit_name"`
	CheckinCount int                      `json:"checkin_count"`
	Plan         *AthletePlanSummary      `json:"plan,omitempty"`
	Contract     *AthleteContractSummary  `json:"contract,omitempty"`
	Billing      AthleteBillingSummary    `json:"billing"`
	Agenda       []AthleteScheduleSession `json:"agenda"`
}

type AthletePlanSummary struct {
	EnrollmentID ID                     `json:"enrollment_id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Status       string                 `json:"status"`
	StartsOn     string                 `json:"starts_on"`
	ValidUntil   *string                `json:"valid_until,omitempty"`
	BillingCycle MembershipBillingCycle `json:"billing_cycle"`
	PriceCents   int                    `json:"price_cents"`
	Currency     string                 `json:"currency"`
}

type AthleteContractSummary struct {
	ID                  ID         `json:"id"`
	EnrollmentID        ID         `json:"enrollment_id"`
	Title               string     `json:"title"`
	Status              string     `json:"status"`
	BodyMarkdown        string     `json:"body_markdown"`
	AcceptanceStatement string     `json:"acceptance_statement"`
	IssuedAt            *time.Time `json:"issued_at,omitempty"`
	AcceptedAt          *time.Time `json:"accepted_at,omitempty"`
}

type AthleteBillingSummary struct {
	FinancialStatus string                 `json:"financial_status"`
	OpenAmountCents int                    `json:"open_amount_cents"`
	Charges         []AthleteChargeSummary `json:"charges"`
}

type AthleteChargeSummary struct {
	ID                  ID             `json:"id"`
	EnrollmentID        ID             `json:"enrollment_id"`
	DueOn               string         `json:"due_on"`
	AmountCents         int            `json:"amount_cents"`
	PaidCents           int            `json:"paid_cents"`
	RefundedCents       int            `json:"refunded_cents"`
	Currency            string         `json:"currency"`
	Status              string         `json:"status"`
	PaymentMethod       string         `json:"payment_method"`
	PaymentInstructions map[string]any `json:"payment_instructions,omitempty"`
	PaidAt              *time.Time     `json:"paid_at,omitempty"`
}

type AthleteScheduleSession struct {
	ID                ID         `json:"id"`
	TargetName        string     `json:"target_name"`
	ProfessionalName  string     `json:"professional_name"`
	SpaceName         string     `json:"space_name,omitempty"`
	StartsAt          time.Time  `json:"starts_at"`
	EndsAt            time.Time  `json:"ends_at"`
	Timezone          string     `json:"timezone"`
	Capacity          int        `json:"capacity"`
	ConfirmedCount    int        `json:"confirmed_count"`
	AvailableCount    int        `json:"available_count"`
	ReservationID     *ID        `json:"reservation_id,omitempty"`
	ReservationStatus string     `json:"reservation_status,omitempty"`
	WaitlistPosition  *int       `json:"waitlist_position,omitempty"`
	AttendedAt        *time.Time `json:"attended_at,omitempty"`
}

type AthleteNotice struct {
	ID           string `json:"id"`
	MembershipID ID     `json:"membership_id"`
	Kind         string `json:"kind"`
	Severity     string `json:"severity"`
	Title        string `json:"title"`
	Message      string `json:"message"`
	Action       string `json:"action,omitempty"`
}
