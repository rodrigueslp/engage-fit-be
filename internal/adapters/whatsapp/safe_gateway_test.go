package whatsapp

import (
	"context"
	"testing"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/services"
)

type recordingWhatsappGateway struct{ calls int }

func (g *recordingWhatsappGateway) Test(context.Context, domain.WhatsappSettings) error { return nil }

func (g *recordingWhatsappGateway) Send(context.Context, domain.WhatsappSettings, services.WhatsappMessage) (*services.WhatsappSendResult, error) {
	g.calls++
	return &services.WhatsappSendResult{ProviderMessageID: "provider-id", InitialStatus: "accepted"}, nil
}

func TestSafeGatewayRequiresExplicitRealSendPermissionInProduction(t *testing.T) {
	next := &recordingWhatsappGateway{}
	gateway := NewSafeGateway(next, "production", false, "", "")
	settings := domain.WhatsappSettings{Provider: "twilio", BaseURL: "https://api.twilio.com", Enabled: true}

	if _, err := gateway.Send(context.Background(), settings, services.WhatsappMessage{Phone: "+5511999999999"}); err == nil {
		t.Fatal("expected production send to be blocked without explicit permission")
	}
	if next.calls != 0 {
		t.Fatalf("provider was called %d times", next.calls)
	}
}

func TestSafeGatewayAllowsMockWithoutRealSendPermission(t *testing.T) {
	next := &recordingWhatsappGateway{}
	gateway := NewSafeGateway(next, "production", false, "", "")
	settings := domain.WhatsappSettings{Provider: "twilio", BaseURL: "mock://whatsapp", Enabled: true}

	if _, err := gateway.Send(context.Background(), settings, services.WhatsappMessage{Phone: "+5511999999999"}); err != nil {
		t.Fatal(err)
	}
	if next.calls != 1 {
		t.Fatalf("mock provider was called %d times", next.calls)
	}
}
