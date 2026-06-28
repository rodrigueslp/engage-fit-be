package whatsapp

import (
	"context"
	"fmt"
	"strings"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/services"
)

const (
	ProviderMetaCloud = "meta_cloud"
	ProviderTwilio    = "twilio"
)

type ProviderGateway struct {
	metaCloud services.WhatsappGateway
	twilio    services.WhatsappGateway
}

func NewProviderGateway(metaCloud services.WhatsappGateway, twilio services.WhatsappGateway) ProviderGateway {
	return ProviderGateway{metaCloud: metaCloud, twilio: twilio}
}

func (g ProviderGateway) Test(ctx context.Context, settings domain.WhatsappSettings) error {
	gateway, err := g.gateway(settings)
	if err != nil {
		return err
	}
	return gateway.Test(ctx, settings)
}

func (g ProviderGateway) Send(ctx context.Context, settings domain.WhatsappSettings, message services.WhatsappMessage) error {
	gateway, err := g.gateway(settings)
	if err != nil {
		return err
	}
	return gateway.Send(ctx, settings, message)
}

func (g ProviderGateway) gateway(settings domain.WhatsappSettings) (services.WhatsappGateway, error) {
	provider := strings.TrimSpace(strings.ToLower(settings.Provider))
	if provider == "" {
		provider = ProviderTwilio
	}

	switch provider {
	case ProviderMetaCloud:
		return g.metaCloud, nil
	case ProviderTwilio:
		return g.twilio, nil
	default:
		return nil, fmt.Errorf("unsupported WhatsApp provider %q", settings.Provider)
	}
}
