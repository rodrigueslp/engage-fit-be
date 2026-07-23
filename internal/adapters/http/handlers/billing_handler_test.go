package handlers

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"boxengage/backend/internal/adapters/http/apiresponse"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/ports/services"
	"github.com/gin-gonic/gin"
)

func TestRespondBillingProviderErrorExposesSafeRejectionAndLogsCorrelation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var logs bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() {
		slog.SetDefault(originalLogger)
	})

	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Set(middleware.RequestIDKey, "billing-test-request")

	respondBillingError(context, &services.BillingProviderError{
		Provider: "asaas", Operation: "create_subscription", StatusCode: http.StatusBadRequest,
		Code: "invalid_value", Description: "O valor mínimo permitido é R$ 5,00.",
	})

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", response.Code)
	}
	var body apiresponse.ErrorBody
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != "billing_provider_rejected" || body.RequestID != "billing-test-request" || !strings.Contains(body.Message, "valor mínimo") {
		t.Fatalf("unexpected response %#v", body)
	}
	for _, expected := range []string{
		`"msg":"billing_provider_request_failed"`,
		`"request_id":"billing-test-request"`,
		`"operation":"create_subscription"`,
		`"provider_status":400`,
		`"provider_error_code":"invalid_value"`,
		`"provider_error_description":"O valor mínimo permitido é R$ 5,00."`,
	} {
		if !strings.Contains(logs.String(), expected) {
			t.Fatalf("log missing %s: %s", expected, logs.String())
		}
	}
}

func TestRespondBillingProviderErrorHidesAuthenticationDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Set(middleware.RequestIDKey, "billing-auth-request")

	respondBillingError(context, &services.BillingProviderError{
		Provider: "asaas", Operation: "create_subscription", StatusCode: http.StatusUnauthorized,
		Code: "invalid_access_token", Description: "credential rejected",
	})

	if response.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", response.Code)
	}
	if strings.Contains(response.Body.String(), "credential rejected") || !strings.Contains(response.Body.String(), "billing_provider_authentication_failed") {
		t.Fatalf("unexpected authentication response %s", response.Body.String())
	}
}
