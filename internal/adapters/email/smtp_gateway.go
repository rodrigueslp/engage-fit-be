package email

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/observability"
	"boxengage/backend/internal/ports/services"
)

type SMTPGateway struct {
	appEnv            string
	allowRealSend     bool
	devRecipientEmail string
}

func NewSMTPGateway(appEnv string, allowRealSend bool, devRecipientEmail string) SMTPGateway {
	return SMTPGateway{appEnv: appEnv, allowRealSend: allowRealSend, devRecipientEmail: devRecipientEmail}
}

func (g SMTPGateway) Test(ctx context.Context, settings domain.EmailSettings) error {
	_ = ctx
	provider := normalizeProvider(settings.Provider)
	if provider == "mock" {
		return nil
	}
	if strings.TrimSpace(settings.SMTPHost) == "" {
		return fmt.Errorf("smtp host is required")
	}
	if settings.SMTPPort <= 0 {
		return fmt.Errorf("smtp port is required")
	}
	if strings.TrimSpace(settings.FromEmail) == "" {
		return fmt.Errorf("from email is required")
	}
	return nil
}

func (g SMTPGateway) Send(ctx context.Context, settings domain.EmailSettings, message services.EmailMessage) (resultErr error) {
	startedAt := time.Now()
	defer func() {
		status := "success"
		if resultErr != nil {
			status = "error"
		}
		observability.RecordGateway(ctx, "smtp", "send", status, time.Since(startedAt))
	}()
	if !settings.Enabled {
		return fmt.Errorf("email provider is disabled")
	}
	provider := normalizeProvider(settings.Provider)
	if provider == "mock" || strings.HasPrefix(strings.ToLower(settings.SMTPHost), "mock://") {
		return nil
	}
	if g.appEnv == "development" && !g.allowRealSend {
		return fmt.Errorf("real email sending is blocked in development; set EMAIL_ALLOW_REAL_SEND=true or use provider mock")
	}
	toEmail := strings.TrimSpace(message.ToEmail)
	if g.appEnv == "development" && strings.TrimSpace(g.devRecipientEmail) != "" {
		toEmail = strings.TrimSpace(g.devRecipientEmail)
	}
	if err := g.Test(ctx, settings); err != nil {
		return err
	}
	if toEmail == "" {
		return fmt.Errorf("recipient email is required")
	}

	addr := fmt.Sprintf("%s:%d", settings.SMTPHost, settings.SMTPPort)
	auth := smtp.Auth(nil)
	if strings.TrimSpace(settings.Username) != "" || strings.TrimSpace(settings.PasswordEncrypted) != "" {
		auth = smtp.PlainAuth("", settings.Username, settings.PasswordEncrypted, settings.SMTPHost)
	}
	fromName := strings.TrimSpace(settings.FromName)
	from := settings.FromEmail
	fromHeader := from
	if fromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", fromName, from)
	}
	raw := strings.Join([]string{
		"From: " + fromHeader,
		"To: " + toEmail,
		"Subject: " + message.Subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		message.Body,
	}, "\r\n")
	return smtp.SendMail(addr, auth, from, []string{toEmail}, []byte(raw))
}

func normalizeProvider(provider string) string {
	provider = strings.TrimSpace(strings.ToLower(provider))
	if provider == "" {
		return "smtp"
	}
	return provider
}
