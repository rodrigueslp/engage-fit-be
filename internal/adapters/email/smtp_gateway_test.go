package email

import (
	"context"
	"strings"
	"testing"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/services"
)

func TestSMTPGatewayRequiresExplicitRealSendPermissionInProduction(t *testing.T) {
	gateway := NewSMTPGateway("production", false, "")
	settings := domain.EmailSettings{Provider: "smtp", SMTPHost: "smtp.example.test", SMTPPort: 587, FromEmail: "sender@example.test", Enabled: true}

	err := gateway.Send(context.Background(), settings, services.EmailMessage{ToEmail: "student@example.test", Subject: "Test", Body: "Test"})
	if err == nil || !strings.Contains(err.Error(), "EMAIL_ALLOW_REAL_SEND") {
		t.Fatalf("expected explicit permission error, got %v", err)
	}
}

func TestSMTPGatewayAllowsMockWithoutRealSendPermission(t *testing.T) {
	gateway := NewSMTPGateway("production", false, "")
	settings := domain.EmailSettings{Provider: "mock", Enabled: true}
	if err := gateway.Send(context.Background(), settings, services.EmailMessage{}); err != nil {
		t.Fatal(err)
	}
}
