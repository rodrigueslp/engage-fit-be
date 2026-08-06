package athlete

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

type AccountMailer struct {
	athletes     repositories.AthleteRepository
	settings     repositories.EmailSettingsRepository
	gateway      services.EmailGateway
	publicWebURL string
}

func NewAccountMailer(athletes repositories.AthleteRepository, settings repositories.EmailSettingsRepository, gateway services.EmailGateway, publicWebURL string) AccountMailer {
	return AccountMailer{athletes: athletes, settings: settings, gateway: gateway, publicWebURL: strings.TrimRight(publicWebURL, "/")}
}

func (m AccountMailer) SendAthleteAccountLink(ctx context.Context, athleteID domain.ID, email, name, purpose, token string) error {
	boxID, err := m.athletes.FindFirstActiveBoxID(ctx, athleteID)
	if err != nil {
		return err
	}
	settings, err := m.settings.FindByBoxID(ctx, boxID)
	if err != nil {
		return err
	}
	path, subject, action := "athlete/verify-email", "Confirme seu e-mail no EngageFit", "Confirmar e-mail"
	if purpose == "reset_password" {
		path, subject, action = "athlete/reset-password", "Redefina sua senha do EngageFit", "Redefinir senha"
	}
	link := fmt.Sprintf("%s/#/%s/%s", m.publicWebURL, path, url.PathEscape(token))
	body := fmt.Sprintf("Olá, %s!\n\nUse o link abaixo para %s:\n%s\n\nSe você não solicitou esta ação, ignore esta mensagem.", strings.Fields(name)[0], strings.ToLower(action), link)
	return m.gateway.Send(ctx, *settings, services.EmailMessage{ToEmail: email, Subject: subject, Body: body})
}
