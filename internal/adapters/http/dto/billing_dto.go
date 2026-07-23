package dto

type BillingPlanRequest struct {
	Code                string `json:"code"`
	Version             int    `json:"version"`
	Name                string `json:"name" binding:"required"`
	Description         string `json:"description"`
	MonthlyPriceCents   int64  `json:"monthly_price_cents"`
	MonthlyMessageLimit int    `json:"monthly_message_limit"`
	DailyMessageLimit   int    `json:"daily_message_limit"`
	PerDispatchLimit    int    `json:"per_dispatch_limit"`
	WarningPercent      int    `json:"warning_percent"`
	GracePeriodDays     int    `json:"grace_period_days"`
	Active              bool   `json:"active"`
	Reason              string `json:"reason" binding:"required"`
}

type BillingPlanResponse struct {
	ID                  string `json:"id"`
	Code                string `json:"code"`
	Version             int    `json:"version"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	MonthlyPriceCents   int64  `json:"monthly_price_cents"`
	Currency            string `json:"currency"`
	MonthlyMessageLimit int    `json:"monthly_message_limit"`
	DailyMessageLimit   int    `json:"daily_message_limit"`
	PerDispatchLimit    int    `json:"per_dispatch_limit"`
	WarningPercent      int    `json:"warning_percent"`
	GracePeriodDays     int    `json:"grace_period_days"`
	Active              bool   `json:"active"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
}

type BillingCustomerRequest struct {
	LegalName            string `json:"legal_name" binding:"required"`
	CPFCNPJ              string `json:"cpf_cnpj"`
	Email                string `json:"email" binding:"required"`
	Phone                string `json:"phone"`
	PostalCode           string `json:"postal_code"`
	Address              string `json:"address"`
	AddressNumber        string `json:"address_number"`
	Complement           string `json:"complement"`
	Province             string `json:"province"`
	City                 string `json:"city"`
	State                string `json:"state"`
	NotificationDisabled bool   `json:"notification_disabled"`
	Reason               string `json:"reason" binding:"required"`
}

type BillingCustomerResponse struct {
	ID                   string `json:"id"`
	BoxID                string `json:"box_id"`
	Provider             string `json:"provider"`
	ProviderCustomerID   string `json:"provider_customer_id"`
	LegalName            string `json:"legal_name"`
	CPFCNPJ              string `json:"cpf_cnpj"`
	Email                string `json:"email"`
	Phone                string `json:"phone"`
	PostalCode           string `json:"postal_code"`
	Address              string `json:"address"`
	AddressNumber        string `json:"address_number"`
	Complement           string `json:"complement"`
	Province             string `json:"province"`
	City                 string `json:"city"`
	State                string `json:"state"`
	NotificationDisabled bool   `json:"notification_disabled"`
}

type CreateBillingSubscriptionRequest struct {
	PlanID      string `json:"plan_id" binding:"required"`
	BillingType string `json:"billing_type" binding:"required"`
	NextDueDate string `json:"next_due_date" binding:"required"`
	EndDate     string `json:"end_date"`
	Reason      string `json:"reason" binding:"required"`
}

type BillingActionRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type GrantBillingGraceRequest struct {
	Until  string `json:"until" binding:"required"`
	Reason string `json:"reason" binding:"required"`
}

type BillingSubscriptionResponse struct {
	ID                     string `json:"id"`
	BoxID                  string `json:"box_id"`
	PlanID                 string `json:"plan_id"`
	Provider               string `json:"provider"`
	ProviderSubscriptionID string `json:"provider_subscription_id,omitempty"`
	Status                 string `json:"status"`
	BillingType            string `json:"billing_type"`
	NextDueDate            string `json:"next_due_date"`
	CurrentPeriodStart     string `json:"current_period_start,omitempty"`
	CurrentPeriodEnd       string `json:"current_period_end,omitempty"`
	GraceUntil             string `json:"grace_until,omitempty"`
	StartedAt              string `json:"started_at"`
	CanceledAt             string `json:"canceled_at,omitempty"`
	CancelAtPeriodEnd      bool   `json:"cancel_at_period_end"`
	LastReconciledAt       string `json:"last_reconciled_at,omitempty"`
}

type BillingInvoiceResponse struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	BillingType       string `json:"billing_type"`
	ValueCents        int64  `json:"value_cents"`
	NetValueCents     *int64 `json:"net_value_cents,omitempty"`
	DueDate           string `json:"due_date"`
	ConfirmedAt       string `json:"confirmed_at,omitempty"`
	ReceivedAt        string `json:"received_at,omitempty"`
	InvoiceURL        string `json:"invoice_url,omitempty"`
	BankSlipURL       string `json:"bank_slip_url,omitempty"`
	Description       string `json:"description"`
	ProviderPaymentID string `json:"provider_payment_id,omitempty"`
}

type BillingOverviewResponse struct {
	BoxID                string                       `json:"box_id"`
	BoxName              string                       `json:"box_name"`
	BoxStatus            string                       `json:"box_status"`
	BillingAccessBlocked bool                         `json:"billing_access_blocked"`
	BillingAccessReason  string                       `json:"billing_access_reason"`
	Customer             *BillingCustomerResponse     `json:"customer,omitempty"`
	Subscription         *BillingSubscriptionResponse `json:"subscription,omitempty"`
	Plan                 *BillingPlanResponse         `json:"plan,omitempty"`
	LatestInvoice        *BillingInvoiceResponse      `json:"latest_invoice,omitempty"`
	Invoices             []BillingInvoiceResponse     `json:"invoices,omitempty"`
}

type BillingSummaryResponse struct {
	MonthlyRecurringRevenueCents int64 `json:"monthly_recurring_revenue_cents"`
	ActiveSubscriptions          int   `json:"active_subscriptions"`
	PastDueSubscriptions         int   `json:"past_due_subscriptions"`
	SuspendedSubscriptions       int   `json:"suspended_subscriptions"`
	CanceledSubscriptions        int   `json:"canceled_subscriptions"`
	PendingAmountCents           int64 `json:"pending_amount_cents"`
	ReceivedThisMonthCents       int64 `json:"received_this_month_cents"`
}
