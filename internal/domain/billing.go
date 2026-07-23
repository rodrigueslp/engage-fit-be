package domain

import "time"

type BillingSubscriptionStatus string

const (
	BillingStatusTrialing  BillingSubscriptionStatus = "trialing"
	BillingStatusPending   BillingSubscriptionStatus = "pending"
	BillingStatusActive    BillingSubscriptionStatus = "active"
	BillingStatusPastDue   BillingSubscriptionStatus = "past_due"
	BillingStatusSuspended BillingSubscriptionStatus = "suspended"
	BillingStatusCanceled  BillingSubscriptionStatus = "canceled"
)

type BillingType string

const (
	BillingTypeUndefined  BillingType = "UNDEFINED"
	BillingTypeBoleto     BillingType = "BOLETO"
	BillingTypeCreditCard BillingType = "CREDIT_CARD"
	BillingTypePix        BillingType = "PIX"
)

type BillingPlan struct {
	ID                  ID
	Code                string
	Version             int
	Name                string
	Description         string
	MonthlyPriceCents   int64
	Currency            string
	MonthlyMessageLimit int
	DailyMessageLimit   int
	PerDispatchLimit    int
	WarningPercent      int
	GracePeriodDays     int
	Active              bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type BillingCustomer struct {
	ID                   ID
	BoxID                ID
	Provider             string
	ProviderCustomerID   string
	LegalName            string
	CPFCNPJ              string
	Email                string
	Phone                string
	PostalCode           string
	Address              string
	AddressNumber        string
	Complement           string
	Province             string
	City                 string
	State                string
	NotificationDisabled bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type BillingSubscription struct {
	ID                     ID
	BoxID                  ID
	BillingCustomerID      ID
	PlanID                 ID
	Provider               string
	ProviderSubscriptionID string
	Status                 BillingSubscriptionStatus
	BillingType            BillingType
	NextDueDate            time.Time
	CurrentPeriodStart     *time.Time
	CurrentPeriodEnd       *time.Time
	GraceUntil             *time.Time
	StartedAt              time.Time
	CanceledAt             *time.Time
	CancelAtPeriodEnd      bool
	ExternalReference      string
	LastReconciledAt       *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type BillingInvoice struct {
	ID                ID
	BoxID             ID
	SubscriptionID    ID
	Provider          string
	ProviderPaymentID string
	Status            string
	BillingType       BillingType
	ValueCents        int64
	NetValueCents     *int64
	DueDate           time.Time
	OriginalDueDate   *time.Time
	ConfirmedAt       *time.Time
	ReceivedAt        *time.Time
	InvoiceURL        string
	BankSlipURL       string
	ExternalReference string
	Description       string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type BillingWebhookEvent struct {
	ID                ID
	Provider          string
	ProviderEventID   string
	EventType         string
	ProviderPaymentID string
	Payload           []byte
	Status            string
	Attempts          int
	ErrorMessage      string
	ReceivedAt        time.Time
	ProcessedAt       *time.Time
}

type BillingOverview struct {
	Box           Box
	Customer      *BillingCustomer
	Subscription  *BillingSubscription
	Plan          *BillingPlan
	LatestInvoice *BillingInvoice
}

type BillingSummary struct {
	MonthlyRecurringRevenueCents int64
	ActiveSubscriptions          int
	PastDueSubscriptions         int
	SuspendedSubscriptions       int
	CanceledSubscriptions        int
	PendingAmountCents           int64
	ReceivedThisMonthCents       int64
}

func (s BillingSubscription) AllowsAccess(now time.Time) bool {
	switch s.Status {
	case BillingStatusTrialing, BillingStatusActive:
		return true
	case BillingStatusPastDue:
		return s.GraceUntil != nil && !now.After(endOfDay(*s.GraceUntil))
	default:
		return false
	}
}

func endOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), value.Location())
}
