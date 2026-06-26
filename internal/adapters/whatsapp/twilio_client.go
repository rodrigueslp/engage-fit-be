package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/services"
)

const DefaultTwilioBaseURL = "https://api.twilio.com"

type TwilioClient struct {
	httpClient *http.Client
}

func NewTwilioClient() TwilioClient {
	return TwilioClient{
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c TwilioClient) Test(ctx context.Context, settings domain.WhatsappSettings) error {
	accountSID, authToken, err := twilioCredentials(settings)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(settings, "2010-04-01/Accounts/"+accountSID+".json"), nil)
	if err != nil {
		return err
	}
	request.SetBasicAuth(accountSID, authToken)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("twilio api test failed with status %d: %s", response.StatusCode, readErrorBody(response.Body))
	}
	return nil
}

func (c TwilioClient) Send(ctx context.Context, settings domain.WhatsappSettings, message services.WhatsappMessage) error {
	accountSID, authToken, err := twilioCredentials(settings)
	if err != nil {
		return err
	}

	form := url.Values{}
	form.Set("To", twilioWhatsappAddress(message.Phone))
	if strings.HasPrefix(settings.InstanceName, "MG") {
		form.Set("MessagingServiceSid", settings.InstanceName)
	} else {
		form.Set("From", twilioWhatsappAddress(settings.InstanceName))
	}

	if strings.TrimSpace(message.ContentSID) != "" {
		form.Set("ContentSid", strings.TrimSpace(message.ContentSID))
		if len(message.ContentVariables) > 0 {
			contentVariables, err := json.Marshal(message.ContentVariables)
			if err != nil {
				return err
			}
			form.Set("ContentVariables", string(contentVariables))
		}
	} else {
		form.Set("Body", message.Body)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(settings, "2010-04-01/Accounts/"+accountSID+"/Messages.json"), strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	request.SetBasicAuth(accountSID, authToken)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("twilio api send failed with status %d: %s", response.StatusCode, readErrorBody(response.Body))
	}
	return nil
}

func (c TwilioClient) url(settings domain.WhatsappSettings, path string) string {
	baseURL := strings.TrimRight(settings.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultTwilioBaseURL
	}
	return baseURL + "/" + strings.TrimLeft(path, "/")
}

func twilioCredentials(settings domain.WhatsappSettings) (string, string, error) {
	accountSID, authToken, ok := strings.Cut(settings.APIKeyEncrypted, ":")
	accountSID = strings.TrimSpace(accountSID)
	authToken = strings.TrimSpace(authToken)
	if !ok || accountSID == "" || authToken == "" {
		return "", "", fmt.Errorf("twilio credentials must be configured as Account SID:Auth Token")
	}
	return accountSID, authToken, nil
}

func twilioWhatsappAddress(phone string) string {
	phone = strings.TrimSpace(phone)
	if strings.HasPrefix(phone, "whatsapp:") {
		return phone
	}
	return "whatsapp:+" + strings.TrimPrefix(phone, "+")
}
