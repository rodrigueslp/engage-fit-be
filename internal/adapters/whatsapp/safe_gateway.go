package whatsapp

import (
	"context"
	"fmt"
	"strings"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/services"
)

type SafeGateway struct {
	next                      services.WhatsappGateway
	appEnv                    string
	allowRealSend             bool
	devRecipientPhone         string
	devAllowedRecipientPhones map[string]struct{}
}

func NewSafeGateway(next services.WhatsappGateway, appEnv string, allowRealSend bool, devRecipientPhone string, devAllowedRecipientPhones string) SafeGateway {
	return SafeGateway{
		next:                      next,
		appEnv:                    appEnv,
		allowRealSend:             allowRealSend,
		devRecipientPhone:         strings.TrimSpace(devRecipientPhone),
		devAllowedRecipientPhones: parseAllowedPhones(devAllowedRecipientPhones),
	}
}

func (g SafeGateway) Test(ctx context.Context, settings domain.WhatsappSettings) error {
	return g.next.Test(ctx, settings)
}

func (g SafeGateway) Send(ctx context.Context, settings domain.WhatsappSettings, message services.WhatsappMessage) error {
	if isMock(settings) || g.appEnv == "production" {
		return g.next.Send(ctx, settings, message)
	}

	if !g.allowRealSend {
		return fmt.Errorf("real WhatsApp send is disabled in %s; set WHATSAPP_ALLOW_REAL_SEND=true and configure WHATSAPP_DEV_ALLOWED_RECIPIENT_PHONES or WHATSAPP_DEV_RECIPIENT_PHONE to test locally", g.appEnv)
	}

	if len(g.devAllowedRecipientPhones) > 0 {
		if _, ok := g.devAllowedRecipientPhones[normalizePhone(message.Phone)]; !ok {
			return fmt.Errorf("recipient %s is not allowed for real WhatsApp send in %s", message.Phone, g.appEnv)
		}
		return g.next.Send(ctx, settings, message)
	}

	if g.devRecipientPhone == "" {
		return fmt.Errorf("WHATSAPP_DEV_ALLOWED_RECIPIENT_PHONES or WHATSAPP_DEV_RECIPIENT_PHONE is required for real WhatsApp send in %s", g.appEnv)
	}
	message.Phone = g.devRecipientPhone
	return g.next.Send(ctx, settings, message)
}

func parseAllowedPhones(value string) map[string]struct{} {
	result := map[string]struct{}{}
	for _, phone := range strings.Split(value, ",") {
		normalized := normalizePhone(phone)
		if normalized != "" {
			result[normalized] = struct{}{}
		}
	}
	return result
}

func normalizePhone(phone string) string {
	phone = strings.TrimSpace(strings.ToLower(phone))
	phone = strings.TrimPrefix(phone, "whatsapp:")
	replacer := strings.NewReplacer("+", "", " ", "", "-", "", "(", "", ")", "")
	return replacer.Replace(phone)
}
