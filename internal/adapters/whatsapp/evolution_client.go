package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/services"
)

type EvolutionClient struct {
	httpClient *http.Client
}

func NewEvolutionClient() EvolutionClient {
	return EvolutionClient{
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c EvolutionClient) Test(ctx context.Context, settings domain.WhatsappSettings) error {
	if isMock(settings) {
		return nil
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(settings, "instance/connectionState/"+settings.InstanceName), nil)
	if err != nil {
		return err
	}
	c.applyHeaders(request, settings)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("evolution api test failed with status %d", response.StatusCode)
	}
	return nil
}

func (c EvolutionClient) Send(ctx context.Context, settings domain.WhatsappSettings, message services.WhatsappMessage) error {
	if isMock(settings) {
		return nil
	}

	payload := map[string]string{
		"number": message.Phone,
		"text":   message.Body,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(settings, "message/sendText/"+settings.InstanceName), bytes.NewReader(body))
	if err != nil {
		return err
	}
	c.applyHeaders(request, settings)
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("evolution api send failed with status %d", response.StatusCode)
	}
	return nil
}

func (c EvolutionClient) url(settings domain.WhatsappSettings, path string) string {
	baseURL := strings.TrimRight(settings.BaseURL, "/")
	return baseURL + "/" + strings.TrimLeft(path, "/")
}

func (c EvolutionClient) applyHeaders(request *http.Request, settings domain.WhatsappSettings) {
	request.Header.Set("apikey", settings.APIKeyEncrypted)
}

func isMock(settings domain.WhatsappSettings) bool {
	return strings.HasPrefix(strings.ToLower(settings.BaseURL), "mock://")
}
