package billing

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/services"
)

func TestAsaasClientCreatesMonthlySubscription(t *testing.T) {
	t.Parallel()
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("access_token"); got != "test-key" {
			t.Fatalf("unexpected access token %q", got)
		}
		if r.Method != http.MethodPost || r.URL.Path != "/subscriptions" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sub_123","status":"ACTIVE","billingType":"PIX","nextDueDate":"2026-08-01"}`))
	}))
	defer server.Close()

	client := NewAsaasClient(server.URL, "test-key", time.Second)
	result, err := client.CreateSubscription(context.Background(), services.CreateBillingSubscriptionInput{
		CustomerID: "cus_123", BillingType: domain.BillingTypePix, NextDueDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		ValueCents: 39700, Description: "EngageFit 500", ExternalReference: "subscription:box:key",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ID != "sub_123" {
		t.Fatalf("unexpected subscription id %q", result.ID)
	}
	if requestBody["cycle"] != "MONTHLY" || requestBody["value"] != float64(397) {
		t.Fatalf("unexpected payload %#v", requestBody)
	}
}

func TestAsaasClientMapsPaymentWithoutFloatingPointLoss(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"pay_123","subscription":"sub_123","customer":"cus_123","status":"RECEIVED",
			"billingType":"BOLETO","value":297.99,"netValue":294.12,"dueDate":"2026-07-20",
			"paymentDate":"2026-07-19","invoiceUrl":"https://example.test/invoice"
		}`))
	}))
	defer server.Close()

	client := NewAsaasClient(server.URL, "test-key", time.Second)
	result, err := client.GetPayment(context.Background(), "pay_123")
	if err != nil {
		t.Fatal(err)
	}
	if result.ValueCents != 29799 || result.NetValueCents == nil || *result.NetValueCents != 29412 {
		t.Fatalf("unexpected monetary mapping %#v", result)
	}
	if result.ReceivedAt == nil || result.DueDate.Format("2006-01-02") != "2026-07-20" {
		t.Fatalf("unexpected date mapping %#v", result)
	}
}

func TestAsaasClientRejectsProviderError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors":[{"code":"invalid_value","description":"Valor mínimo inválido para payer@example.com e documento 12345678901.\n token=secret-value"}]}`))
	}))
	defer server.Close()

	client := NewAsaasClient(server.URL, "invalid", time.Second)
	_, err := client.FindCustomerByExternalReference(context.Background(), "box")
	if err == nil {
		t.Fatal("expected provider error")
	}
	var providerError *services.BillingProviderError
	if !errors.As(err, &providerError) {
		t.Fatalf("expected structured provider error, got %T", err)
	}
	if providerError.StatusCode != http.StatusBadRequest || providerError.Operation != "find_customer" || providerError.Code != "invalid_value" {
		t.Fatalf("unexpected provider error %#v", providerError)
	}
	expectedDescription := "Valor mínimo inválido para [redacted-email] e documento [redacted-number]. token=[redacted]"
	if providerError.Description != expectedDescription {
		t.Fatalf("unexpected sanitized description %q", providerError.Description)
	}
	if got := err.Error(); got != "falha no provedor financeiro: provider=asaas operation=find_customer status=400 code=invalid_value" {
		t.Fatalf("provider description must not leak through error string, got %q", got)
	}
}

func TestAsaasClientClassifiesMalformedProviderResponse(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer server.Close()

	client := NewAsaasClient(server.URL, "test-key", time.Second)
	_, err := client.FindCustomerByExternalReference(context.Background(), "box")
	var providerError *services.BillingProviderError
	if !errors.As(err, &providerError) {
		t.Fatalf("expected structured provider error, got %v", err)
	}
	if providerError.StatusCode != http.StatusOK || providerError.Operation != "find_customer" || providerError.Code != "malformed_response" {
		t.Fatalf("unexpected provider error %#v", providerError)
	}
}
