package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
)

type BillingRepository interface {
	ListPlans(ctx context.Context, includeInactive bool) ([]domain.BillingPlan, error)
	FindPlanByID(ctx context.Context, id domain.ID) (*domain.BillingPlan, error)
	FindPlanByCode(ctx context.Context, code string) (*domain.BillingPlan, error)
	SavePlan(ctx context.Context, plan *domain.BillingPlan) error
	UpdatePlan(ctx context.Context, plan domain.BillingPlan) error
	PlanHasSubscriptions(ctx context.Context, id domain.ID) (bool, error)

	FindCustomerByBoxID(ctx context.Context, boxID domain.ID) (*domain.BillingCustomer, error)
	SaveCustomer(ctx context.Context, customer *domain.BillingCustomer) error
	UpdateCustomer(ctx context.Context, customer domain.BillingCustomer) error

	FindCurrentSubscriptionByBoxID(ctx context.Context, boxID domain.ID) (*domain.BillingSubscription, error)
	FindSubscriptionByProviderID(ctx context.Context, providerID string) (*domain.BillingSubscription, error)
	ListSubscriptions(ctx context.Context) ([]domain.BillingSubscription, error)
	ListSubscriptionsDueForEnforcement(ctx context.Context, now time.Time) ([]domain.BillingSubscription, error)
	SaveSubscription(ctx context.Context, subscription *domain.BillingSubscription) error
	UpdateSubscription(ctx context.Context, subscription domain.BillingSubscription) error

	UpsertInvoice(ctx context.Context, invoice *domain.BillingInvoice) error
	FindInvoiceByProviderPaymentID(ctx context.Context, providerPaymentID string) (*domain.BillingInvoice, error)
	ListInvoicesByBoxID(ctx context.Context, boxID domain.ID) ([]domain.BillingInvoice, error)

	SaveWebhookEvent(ctx context.Context, event *domain.BillingWebhookEvent) (bool, error)
	MarkWebhookEventProcessed(ctx context.Context, id domain.ID, processedAt time.Time) error
	MarkWebhookEventFailed(ctx context.Context, id domain.ID, message string) error

	SetBillingAccess(ctx context.Context, boxID domain.ID, blocked bool, reason string, changedAt time.Time) error
	ApplyPlanMessagingPolicy(ctx context.Context, boxID domain.ID, plan domain.BillingPlan) error
	ListOverviews(ctx context.Context) ([]domain.BillingOverview, error)
	Summary(ctx context.Context, now time.Time) (*domain.BillingSummary, error)
}
