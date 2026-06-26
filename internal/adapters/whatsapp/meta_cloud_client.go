package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/services"
)

const DefaultMetaCloudBaseURL = "https://graph.facebook.com/v20.0"

type MetaCloudClient struct {
	httpClient *http.Client
}

func NewMetaCloudClient() MetaCloudClient {
	return MetaCloudClient{
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c MetaCloudClient) Test(ctx context.Context, settings domain.WhatsappSettings) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(settings, settings.InstanceName), nil)
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
		return fmt.Errorf("meta cloud api test failed with status %d: %s", response.StatusCode, readErrorBody(response.Body))
	}
	return nil
}

func (c MetaCloudClient) Send(ctx context.Context, settings domain.WhatsappSettings, message services.WhatsappMessage) error {
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                message.Phone,
		"type":              "text",
		"text": map[string]any{
			"preview_url": false,
			"body":        message.Body,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(settings, settings.InstanceName+"/messages"), bytes.NewReader(body))
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
		return fmt.Errorf("meta cloud api send failed with status %d: %s", response.StatusCode, readErrorBody(response.Body))
	}
	return nil
}

func (c MetaCloudClient) url(settings domain.WhatsappSettings, path string) string {
	baseURL := strings.TrimRight(settings.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultMetaCloudBaseURL
	}
	return baseURL + "/" + strings.TrimLeft(path, "/")
}

func (c MetaCloudClient) applyHeaders(request *http.Request, settings domain.WhatsappSettings) {
	request.Header.Set("Authorization", "Bearer "+settings.APIKeyEncrypted)
}

func readErrorBody(body io.Reader) string {
	content, err := io.ReadAll(io.LimitReader(body, 2048))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}
