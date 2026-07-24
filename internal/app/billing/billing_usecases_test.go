package billing

import (
	"context"
	"testing"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
	"gorm.io/gorm"
)

type deletedPaymentRepository struct {
	repositories.BillingRepository
	invoice        *domain.BillingInvoice
	updatedInvoice *domain.BillingInvoice
	eventProcessed bool
	eventFailed    bool
}

func (r *deletedPaymentRepository) SaveWebhookEvent(_ context.Context, event *domain.BillingWebhookEvent) (bool, error) {
	event.ID = "webhook-event-id"
	return true, nil
}

func (r *deletedPaymentRepository) FindInvoiceByProviderPaymentID(_ context.Context, providerPaymentID string) (*domain.BillingInvoice, error) {
	if r.invoice == nil || r.invoice.ProviderPaymentID != providerPaymentID {
		return nil, gorm.ErrRecordNotFound
	}
	invoice := *r.invoice
	return &invoice, nil
}

func (r *deletedPaymentRepository) UpsertInvoice(_ context.Context, invoice *domain.BillingInvoice) error {
	value := *invoice
	r.updatedInvoice = &value
	return nil
}

func (r *deletedPaymentRepository) MarkWebhookEventProcessed(_ context.Context, _ domain.ID, _ time.Time) error {
	r.eventProcessed = true
	return nil
}

func (r *deletedPaymentRepository) MarkWebhookEventFailed(_ context.Context, _ domain.ID, _ string) error {
	r.eventFailed = true
	return nil
}

type deletedPaymentGateway struct {
	services.BillingGateway
	getPaymentCalled bool
}

func (g *deletedPaymentGateway) GetPayment(_ context.Context, _ string) (*services.BillingProviderPayment, error) {
	g.getPaymentCalled = true
	return nil, services.ErrBillingProvider
}

func TestProcessWebhookMarksDeletedPaymentWithoutProviderLookup(t *testing.T) {
	repository := &deletedPaymentRepository{
		invoice: &domain.BillingInvoice{
			ID:                "invoice-id",
			ProviderPaymentID: "pay_deleted",
			Status:            "PENDING",
			InvoiceURL:        "https://example.test/invoice",
			BankSlipURL:       "https://example.test/boleto",
		},
	}
	gateway := &deletedPaymentGateway{}
	service := NewService(repository, gateway, nil, true, "01234567890123456789012345678901")
	service.now = func() time.Time {
		return time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	}

	err := service.ProcessWebhook(
		context.Background(),
		"01234567890123456789012345678901",
		[]byte(`{"id":"evt_deleted","event":"PAYMENT_DELETED","payment":{"id":"pay_deleted","subscription":"sub_123"}}`),
	)
	if err != nil {
		t.Fatalf("ProcessWebhook() error = %v", err)
	}
	if gateway.getPaymentCalled {
		t.Fatal("deleted payment must not be fetched from the provider")
	}
	if repository.updatedInvoice == nil {
		t.Fatal("deleted invoice was not updated")
	}
	if repository.updatedInvoice.Status != "DELETED" {
		t.Fatalf("invoice status = %q, want DELETED", repository.updatedInvoice.Status)
	}
	if repository.updatedInvoice.InvoiceURL != "" || repository.updatedInvoice.BankSlipURL != "" {
		t.Fatalf("deleted invoice retained payment links: %#v", repository.updatedInvoice)
	}
	if !repository.eventProcessed || repository.eventFailed {
		t.Fatalf("webhook state processed=%v failed=%v", repository.eventProcessed, repository.eventFailed)
	}
}
