package dto

import "boxengage/backend/internal/app/memberships"

type MembershipPlanResponse struct {
	ID        string                          `json:"id"`
	Name      string                          `json:"name"`
	Status    string                          `json:"status"`
	Published *MembershipPlanVersionResponse  `json:"published_version"`
	Draft     *MembershipPlanVersionResponse  `json:"draft"`
	Versions  []MembershipPlanVersionResponse `json:"versions,omitempty"`
	CreatedAt string                          `json:"created_at"`
	UpdatedAt string                          `json:"updated_at"`
}

type MembershipPlanVersionResponse struct {
	ID            string                              `json:"id"`
	VersionNumber int                                 `json:"version_number"`
	Revision      int                                 `json:"revision"`
	Status        string                              `json:"status"`
	Name          string                              `json:"name"`
	Description   string                              `json:"description"`
	PriceCents    int                                 `json:"price_cents"`
	Currency      string                              `json:"currency"`
	BillingCycle  string                              `json:"billing_cycle"`
	ValidityCount *int                                `json:"validity_count"`
	ValidityUnit  *string                             `json:"validity_unit"`
	PublishedAt   *string                             `json:"published_at"`
	Entitlements  []MembershipPlanEntitlementResponse `json:"entitlements"`
	Addons        []MembershipPlanAddonResponse       `json:"addons"`
	Summary       string                              `json:"summary"`
	Warnings      []string                            `json:"warnings"`
	CreatedAt     string                              `json:"created_at"`
	UpdatedAt     string                              `json:"updated_at"`
}

type MembershipPlanEntitlementResponse struct {
	ID                string  `json:"id"`
	ModalityID        *string `json:"modality_id"`
	ServiceOfferingID *string `json:"service_offering_id"`
	TargetName        string  `json:"target_name"`
	TargetType        string  `json:"target_type"`
	AccessType        string  `json:"access_type"`
	Quantity          *int    `json:"quantity"`
	PeriodUnit        *string `json:"period_unit"`
	DailyLimit        *int    `json:"daily_limit"`
	DeliveryMode      *string `json:"delivery_mode"`
	BookingRequired   bool    `json:"booking_required"`
	Position          int     `json:"position"`
}

type MembershipPlanAddonResponse struct {
	ID              string                                   `json:"id"`
	Name            string                                   `json:"name"`
	Description     string                                   `json:"description"`
	PriceDeltaCents int                                      `json:"price_delta_cents"`
	Position        int                                      `json:"position"`
	Entitlements    []MembershipPlanAddonEntitlementResponse `json:"entitlements"`
	Summary         string                                   `json:"summary"`
	Warnings        []string                                 `json:"warnings"`
}

type MembershipPlanAddonEntitlementResponse struct {
	ID                string  `json:"id"`
	ModalityID        *string `json:"modality_id"`
	ServiceOfferingID *string `json:"service_offering_id"`
	TargetName        string  `json:"target_name"`
	TargetType        string  `json:"target_type"`
	AccessType        string  `json:"access_type"`
	Quantity          *int    `json:"quantity"`
	PeriodUnit        *string `json:"period_unit"`
	DailyLimit        *int    `json:"daily_limit"`
	DeliveryMode      *string `json:"delivery_mode"`
	BookingRequired   bool    `json:"booking_required"`
	Position          int     `json:"position"`
}

type MembershipPlanRequest = memberships.PlanInput

type PublishMembershipPlanRequest struct {
	DraftRevision int `json:"draft_revision"`
}
