package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"boxengage/backend/internal/domain"
)

var ErrBillingProvider = errors.New("falha no provedor financeiro")

type BillingProviderError struct {
	Provider    string
	Operation   string
	StatusCode  int
	Code        string
	Description string
}

func (e *BillingProviderError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s: provider=%s operation=%s status=%d code=%s", ErrBillingProvider, e.Provider, e.Operation, e.StatusCode, e.Code)
	}
	return fmt.Sprintf("%s: provider=%s operation=%s status=%d", ErrBillingProvider, e.Provider, e.Operation, e.StatusCode)
}

func (e *BillingProviderError) Unwrap() error {
	return ErrBillingProvider
}

type BillingProviderCustomer struct {
	ID string
}

type CreateBillingCustomerInput struct {
	Name                 string
	CPFCNPJ              string
	Email                string
	Phone                string
	PostalCode           string
	Address              string
	AddressNumber        string
	Complement           string
	Province             string
	ExternalReference    string
	NotificationDisabled bool
}

type BillingProviderSubscription struct {
	ID          string
	Status      string
	NextDueDate time.Time
	BillingType domain.BillingType
	InvoiceURL  string
}

type CreateBillingSubscriptionInput struct {
	CustomerID        string
	BillingType       domain.BillingType
	NextDueDate       time.Time
	ValueCents        int64
	Description       string
	ExternalReference string
	EndDate           *time.Time
}

type BillingProviderPayment struct {
	ID                string
	SubscriptionID    string
	CustomerID        string
	Status            string
	BillingType       domain.BillingType
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
}

type BillingGateway interface {
	CreateCustomer(ctx context.Context, input CreateBillingCustomerInput) (*BillingProviderCustomer, error)
	FindCustomerByExternalReference(ctx context.Context, externalReference string) (*BillingProviderCustomer, error)
	UpdateCustomer(ctx context.Context, providerCustomerID string, input CreateBillingCustomerInput) error
	CreateSubscription(ctx context.Context, input CreateBillingSubscriptionInput) (*BillingProviderSubscription, error)
	FindSubscriptionByExternalReference(ctx context.Context, externalReference string) (*BillingProviderSubscription, error)
	CancelSubscription(ctx context.Context, providerSubscriptionID string) error
	GetPayment(ctx context.Context, providerPaymentID string) (*BillingProviderPayment, error)
	ListSubscriptionPayments(ctx context.Context, providerSubscriptionID string) ([]BillingProviderPayment, error)
}
