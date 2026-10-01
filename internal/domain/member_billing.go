package domain

import "time"

// MemberBillingContract is the financial contract owned by memberbilling. It
// snapshots enrollment terms; changing the plan never rewrites prior charges.
type MemberBillingContract struct {
	ID              ID                     `json:"id"`
	OrganizationID  ID                     `json:"-"`
	UnitID          ID                     `json:"unit_id"`
	PersonID        ID                     `json:"person_id"`
	EnrollmentID    ID                     `json:"enrollment_id"`
	BillingCycle    MembershipBillingCycle `json:"billing_cycle"`
	AmountCents     int                    `json:"amount_cents"`
	Currency        string                 `json:"currency"`
	Status          string                 `json:"status"`
	PaymentMethod   string                 `json:"payment_method"`
	StartedOn       time.Time              `json:"started_on"`
	NextDueOn       *time.Time             `json:"next_due_on"`
	CreatedByUserID ID                     `json:"-"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type MemberCharge struct {
	ID                  ID              `json:"id"`
	OrganizationID      ID              `json:"-"`
	UnitID              ID              `json:"unit_id"`
	PersonID            ID              `json:"person_id"`
	EnrollmentID        ID              `json:"enrollment_id"`
	ContractID          ID              `json:"contract_id"`
	Sequence            int             `json:"sequence"`
	DueOn               time.Time       `json:"due_on"`
	AmountCents         int             `json:"amount_cents"`
	PaidCents           int             `json:"paid_cents"`
	RefundedCents       int             `json:"refunded_cents"`
	Currency            string          `json:"currency"`
	Status              string          `json:"status"`
	PaymentMethod       string          `json:"payment_method"`
	Provider            string          `json:"provider"`
	ProviderReference   string          `json:"provider_reference,omitempty"`
	PaymentInstructions map[string]any  `json:"payment_instructions,omitempty"`
	PaidAt              *time.Time      `json:"paid_at"`
	CanceledAt          *time.Time      `json:"canceled_at"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	Payments            []MemberPayment `json:"payments"`
}

type MemberPayment struct {
	ID              ID        `json:"id"`
	OrganizationID  ID        `json:"-"`
	ChargeID        ID        `json:"charge_id"`
	Kind            string    `json:"kind"`
	Method          string    `json:"method"`
	AmountCents     int       `json:"amount_cents"`
	OccurredAt      time.Time `json:"occurred_at"`
	ActorUserID     *ID       `json:"actor_user_id,omitempty"`
	ProviderEventID string    `json:"provider_event_id,omitempty"`
	Reason          string    `json:"reason,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type MemberBillingSnapshot struct {
	Contract            *MemberBillingContract `json:"contract"`
	Charges             []MemberCharge         `json:"charges"`
	OpenAmountCents     int                    `json:"open_amount_cents"`
	PaidAmountCents     int                    `json:"paid_amount_cents"`
	RefundedAmountCents int                    `json:"refunded_amount_cents"`
	FinancialStatus     string                 `json:"financial_status"`
}

type MemberBillingCommand struct {
	OrganizationID                       ID
	Command, IdempotencyKey, Fingerprint string
	ResultID                             ID
	CreatedAt                            time.Time
}
