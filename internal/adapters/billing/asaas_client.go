package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/services"
)

const maxProviderResponseBytes = 1 << 20
const maxProviderErrorDescriptionBytes = 500

var providerEmailPattern = regexp.MustCompile(`(?i)[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}`)
var providerLongNumberPattern = regexp.MustCompile(`\b[0-9]{8,}\b`)
var providerSecretPattern = regexp.MustCompile(`(?i)\b(access_token|api[_ -]?key|token)\s*[:=]\s*\S+`)

type AsaasClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewAsaasClient(baseURL, apiKey string, timeout time.Duration) *AsaasClient {
	return &AsaasClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *AsaasClient) CreateCustomer(ctx context.Context, input services.CreateBillingCustomerInput) (*services.BillingProviderCustomer, error) {
	payload := map[string]any{
		"name": input.Name, "cpfCnpj": input.CPFCNPJ, "email": input.Email,
		"mobilePhone": input.Phone, "postalCode": input.PostalCode, "address": input.Address,
		"addressNumber": input.AddressNumber, "complement": input.Complement, "province": input.Province,
		"externalReference": input.ExternalReference, "notificationDisabled": input.NotificationDisabled,
	}
	var response struct {
		ID string `json:"id"`
	}
	if err := c.do(ctx, http.MethodPost, "/customers", nil, payload, &response); err != nil {
		return nil, err
	}
	if response.ID == "" {
		return nil, services.ErrBillingProvider
	}
	return &services.BillingProviderCustomer{ID: response.ID}, nil
}

func (c *AsaasClient) FindCustomerByExternalReference(ctx context.Context, externalReference string) (*services.BillingProviderCustomer, error) {
	query := url.Values{}
	query.Set("externalReference", externalReference)
	query.Set("limit", "1")
	var response struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/customers", query, nil, &response); err != nil {
		return nil, err
	}
	if len(response.Data) == 0 || response.Data[0].ID == "" {
		return nil, nil
	}
	return &services.BillingProviderCustomer{ID: response.Data[0].ID}, nil
}

func (c *AsaasClient) UpdateCustomer(ctx context.Context, providerCustomerID string, input services.CreateBillingCustomerInput) error {
	payload := map[string]any{
		"name": input.Name, "cpfCnpj": input.CPFCNPJ, "email": input.Email,
		"mobilePhone": input.Phone, "postalCode": input.PostalCode, "address": input.Address,
		"addressNumber": input.AddressNumber, "complement": input.Complement, "province": input.Province,
		"externalReference": input.ExternalReference, "notificationDisabled": input.NotificationDisabled,
	}
	return c.do(ctx, http.MethodPut, "/customers/"+url.PathEscape(providerCustomerID), nil, payload, nil)
}

func (c *AsaasClient) CreateSubscription(ctx context.Context, input services.CreateBillingSubscriptionInput) (*services.BillingProviderSubscription, error) {
	payload := map[string]any{
		"customer": input.CustomerID, "billingType": input.BillingType,
		"nextDueDate": input.NextDueDate.Format("2006-01-02"), "value": centsToDecimal(input.ValueCents),
		"cycle": "MONTHLY", "description": input.Description, "externalReference": input.ExternalReference,
	}
	if input.EndDate != nil {
		payload["endDate"] = input.EndDate.Format("2006-01-02")
	}
	var response asaasSubscription
	if err := c.do(ctx, http.MethodPost, "/subscriptions", nil, payload, &response); err != nil {
		return nil, err
	}
	nextDueDate, err := parseDate(response.NextDueDate)
	if err != nil || response.ID == "" {
		return nil, services.ErrBillingProvider
	}
	return &services.BillingProviderSubscription{
		ID: response.ID, Status: response.Status, NextDueDate: nextDueDate,
		BillingType: domain.BillingType(response.BillingType),
	}, nil
}

func (c *AsaasClient) FindSubscriptionByExternalReference(ctx context.Context, externalReference string) (*services.BillingProviderSubscription, error) {
	query := url.Values{}
	query.Set("externalReference", externalReference)
	query.Set("limit", "1")
	var response struct {
		Data []asaasSubscription `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/subscriptions", query, nil, &response); err != nil {
		return nil, err
	}
	if len(response.Data) == 0 {
		return nil, nil
	}
	row := response.Data[0]
	nextDueDate, err := parseDate(row.NextDueDate)
	if err != nil || row.ID == "" {
		return nil, services.ErrBillingProvider
	}
	return &services.BillingProviderSubscription{
		ID: row.ID, Status: row.Status, NextDueDate: nextDueDate, BillingType: domain.BillingType(row.BillingType),
	}, nil
}

func (c *AsaasClient) CancelSubscription(ctx context.Context, providerSubscriptionID string) error {
	return c.do(ctx, http.MethodDelete, "/subscriptions/"+url.PathEscape(providerSubscriptionID), nil, nil, nil)
}

func (c *AsaasClient) GetPayment(ctx context.Context, providerPaymentID string) (*services.BillingProviderPayment, error) {
	var response asaasPayment
	if err := c.do(ctx, http.MethodGet, "/payments/"+url.PathEscape(providerPaymentID), nil, nil, &response); err != nil {
		return nil, err
	}
	return response.toService()
}

func (c *AsaasClient) ListSubscriptionPayments(ctx context.Context, providerSubscriptionID string) ([]services.BillingProviderPayment, error) {
	result := make([]services.BillingProviderPayment, 0)
	offset := 0
	for {
		query := url.Values{}
		query.Set("subscription", providerSubscriptionID)
		query.Set("limit", "100")
		query.Set("offset", strconv.Itoa(offset))
		var response struct {
			Data       []asaasPayment `json:"data"`
			HasMore    bool           `json:"hasMore"`
			TotalCount int            `json:"totalCount"`
		}
		if err := c.do(ctx, http.MethodGet, "/payments", query, nil, &response); err != nil {
			return nil, err
		}
		for _, payment := range response.Data {
			mapped, err := payment.toService()
			if err != nil {
				return nil, services.ErrBillingProvider
			}
			result = append(result, *mapped)
		}
		if !response.HasMore || len(response.Data) == 0 {
			break
		}
		offset += len(response.Data)
	}
	return result, nil
}

func (c *AsaasClient) do(ctx context.Context, method, path string, query url.Values, payload any, target any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	request.Header.Set("access_token", c.apiKey)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "EngageFit/1.0")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return providerFailure(method, path, 0, "request_failed")
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxProviderResponseBytes))
	if err != nil {
		return providerFailure(method, path, response.StatusCode, "invalid_response")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return newProviderError(method, path, response.StatusCode, responseBody)
	}
	if target == nil || len(responseBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(responseBody, target); err != nil {
		return providerFailure(method, path, response.StatusCode, "malformed_response")
	}
	return nil
}

func providerFailure(method, path string, statusCode int, code string) error {
	return &services.BillingProviderError{
		Provider: "asaas", Operation: providerOperation(method, path),
		StatusCode: statusCode, Code: code,
	}
}

func newProviderError(method, path string, statusCode int, responseBody []byte) error {
	var envelope struct {
		Errors []struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"errors"`
	}
	_ = json.Unmarshal(responseBody, &envelope)

	result := &services.BillingProviderError{
		Provider:   "asaas",
		Operation:  providerOperation(method, path),
		StatusCode: statusCode,
	}
	if len(envelope.Errors) > 0 {
		result.Code = sanitizeProviderText(envelope.Errors[0].Code, 100)
		result.Description = sanitizeProviderText(envelope.Errors[0].Description, maxProviderErrorDescriptionBytes)
	}
	return result
}

func providerOperation(method, path string) string {
	switch {
	case method == http.MethodPost && path == "/customers":
		return "create_customer"
	case method == http.MethodGet && path == "/customers":
		return "find_customer"
	case method == http.MethodPut && strings.HasPrefix(path, "/customers/"):
		return "update_customer"
	case method == http.MethodPost && path == "/subscriptions":
		return "create_subscription"
	case method == http.MethodGet && path == "/subscriptions":
		return "find_subscription"
	case method == http.MethodDelete && strings.HasPrefix(path, "/subscriptions/"):
		return "cancel_subscription"
	case method == http.MethodGet && path == "/payments":
		return "list_subscription_payments"
	case method == http.MethodGet && strings.HasPrefix(path, "/payments/"):
		return "get_payment"
	default:
		return "request"
	}
}

func sanitizeProviderText(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	value = providerEmailPattern.ReplaceAllString(value, "[redacted-email]")
	value = providerLongNumberPattern.ReplaceAllString(value, "[redacted-number]")
	value = providerSecretPattern.ReplaceAllString(value, "$1=[redacted]")
	runes := []rune(value)
	if len(runes) > limit {
		value = string(runes[:limit])
	}
	return value
}

type asaasSubscription struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	BillingType string `json:"billingType"`
	NextDueDate string `json:"nextDueDate"`
}

type asaasPayment struct {
	ID                string   `json:"id"`
	Subscription      string   `json:"subscription"`
	Customer          string   `json:"customer"`
	Status            string   `json:"status"`
	BillingType       string   `json:"billingType"`
	Value             float64  `json:"value"`
	NetValue          *float64 `json:"netValue"`
	DueDate           string   `json:"dueDate"`
	OriginalDueDate   string   `json:"originalDueDate"`
	ConfirmedDate     string   `json:"confirmedDate"`
	ClientPaymentDate string   `json:"clientPaymentDate"`
	PaymentDate       string   `json:"paymentDate"`
	InvoiceURL        string   `json:"invoiceUrl"`
	BankSlipURL       string   `json:"bankSlipUrl"`
	ExternalReference string   `json:"externalReference"`
	Description       string   `json:"description"`
}

func (p asaasPayment) toService() (*services.BillingProviderPayment, error) {
	dueDate, err := parseDate(p.DueDate)
	if err != nil || p.ID == "" {
		return nil, services.ErrBillingProvider
	}
	originalDueDate := optionalDate(p.OriginalDueDate)
	confirmedAt := optionalDateTime(p.ConfirmedDate)
	receivedAt := optionalDateTime(p.PaymentDate)
	if receivedAt == nil {
		receivedAt = optionalDateTime(p.ClientPaymentDate)
	}
	var netValueCents *int64
	if p.NetValue != nil {
		value := decimalToCents(*p.NetValue)
		netValueCents = &value
	}
	return &services.BillingProviderPayment{
		ID: p.ID, SubscriptionID: p.Subscription, CustomerID: p.Customer, Status: p.Status,
		BillingType: domain.BillingType(p.BillingType), ValueCents: decimalToCents(p.Value),
		NetValueCents: netValueCents, DueDate: dueDate, OriginalDueDate: originalDueDate,
		ConfirmedAt: confirmedAt, ReceivedAt: receivedAt, InvoiceURL: p.InvoiceURL,
		BankSlipURL: p.BankSlipURL, ExternalReference: p.ExternalReference, Description: p.Description,
	}, nil
}

func parseDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", value)
}

func optionalDate(value string) *time.Time {
	if value == "" {
		return nil
	}
	parsed, err := parseDate(value)
	if err != nil {
		return nil
	}
	return &parsed
}

func optionalDateTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02", "2006-01-02 15:04:05"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return &parsed
		}
	}
	return nil
}

func decimalToCents(value float64) int64 {
	return int64(math.Round(value * 100))
}

func centsToDecimal(value int64) float64 {
	return float64(value) / 100
}

var _ services.BillingGateway = (*AsaasClient)(nil)
