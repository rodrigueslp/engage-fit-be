package dto

import "boxengage/backend/internal/app/memberships"

type EnrollmentTermsResponse struct {
	PlanID        string `json:"plan_id"`
	PlanVersionID string `json:"plan_version_id"`
	VersionNumber int    `json:"version_number"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	PriceCents    int    `json:"price_cents"`
	Currency      string `json:"currency"`
	BillingCycle  string `json:"billing_cycle"`
	Summary       string `json:"summary"`
}

type EnrollmentResponse struct {
	ID              string                  `json:"id"`
	PersonID        string                  `json:"person_id"`
	LifecycleStatus string                  `json:"lifecycle_status"`
	EffectiveStatus string                  `json:"effective_status"`
	StartsOn        string                  `json:"starts_on"`
	ValidUntil      *string                 `json:"valid_until"`
	EndedOn         *string                 `json:"ended_on"`
	EndedAt         *string                 `json:"ended_at"`
	EndedReason     string                  `json:"ended_reason,omitempty"`
	Revision        int                     `json:"revision"`
	Terms           EnrollmentTermsResponse `json:"terms"`
	CreatedAt       string                  `json:"created_at"`
	UpdatedAt       string                  `json:"updated_at"`
}

type EnrollmentHistoryResponse struct {
	Current *EnrollmentResponse  `json:"current"`
	History []EnrollmentResponse `json:"history"`
}

type CreateEnrollmentRequest = memberships.CreateEnrollmentInput
type EndEnrollmentRequest = memberships.EndEnrollmentInput

type EnrollmentPrivacyExportResponse struct {
	Enrollment EnrollmentResponse        `json:"enrollment"`
	Events     []EnrollmentEventResponse `json:"events"`
}

type EnrollmentEventResponse struct {
	Action     string         `json:"action"`
	BeforeData map[string]any `json:"before_data"`
	AfterData  map[string]any `json:"after_data"`
	Reason     string         `json:"reason,omitempty"`
	CreatedAt  string         `json:"created_at"`
}
