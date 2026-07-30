package contactactivation

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

func TestStartAndConfirmActivation(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	repository := &activationRepositoryStub{
		boxID: domain.ID("box-1"), boxName: "CrossFit Alados", activationCode: "activation-code",
		matches: []domain.Student{{ID: domain.ID("student-1"), BoxID: domain.ID("box-1"), Name: "Adriana Segatelli", Source: domain.SourceTotalPass}},
	}
	service := NewService(repository, studentRepositoryStub{}, settingsResolverStub{}, "")
	service.now = func() time.Time { return now }
	checkinDate := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)

	started, err := service.Start(context.Background(), StartInput{
		ActivationCode: "activation-code", Name: "  Adriana Segatelli ", Source: domain.SourceTotalPass,
		RecentCheckinDate: checkinDate, ConsentAccepted: true,
	})
	if err != nil {
		t.Fatalf("start activation: %v", err)
	}
	if !strings.HasPrefix(started.WhatsappURL, "https://wa.me/5511999999999?text=") {
		t.Fatalf("unexpected activation result: %+v", started)
	}
	if repository.created == nil || repository.created.StudentID != "student-1" {
		t.Fatalf("expected matched activation, got %+v", repository.created)
	}

	messageURL, err := url.Parse(started.WhatsappURL)
	if err != nil {
		t.Fatal(err)
	}
	body := messageURL.Query().Get("text")
	values := url.Values{
		"Body": {body}, "From": {"whatsapp:+5511988887777"}, "To": {"whatsapp:+5511999999999"},
	}
	webhookURL := "https://www.engagefit.com.br/api/v1/webhooks/twilio/whatsapp"
	result, err := service.HandleInbound(context.Background(), webhookURL, twilioSignature(webhookURL, values, "test-secret"), values)
	if err != nil {
		t.Fatalf("confirm activation: %v", err)
	}
	if repository.confirmedPhone != "5511988887777" {
		t.Fatalf("expected inbound phone to be persisted, got %q", repository.confirmedPhone)
	}
	if !strings.Contains(result.Message, "Adriana") || !strings.Contains(result.Message, "SAIR") {
		t.Fatalf("unexpected confirmation: %q", result.Message)
	}
}

func TestInboundRejectsInvalidSignature(t *testing.T) {
	repository := &activationRepositoryStub{
		boxID: domain.ID("box-1"),
		created: &domain.ContactActivationRequest{
			ID: domain.ID("activation-1"), BoxID: domain.ID("box-1"), StudentID: domain.ID("student-1"),
			StudentName: "Adriana", SenderPhone: "5511999999999", TokenHash: tokenHash("valid-token-value-123"),
			Status: domain.ContactActivationAwaitingMessage, ExpiresAt: time.Now().Add(time.Hour),
		},
	}
	service := NewService(repository, studentRepositoryStub{}, settingsResolverStub{}, "")
	values := url.Values{
		"Body": {"Código: EF-valid-token-value-123"},
		"From": {"whatsapp:+5511988887777"}, "To": {"whatsapp:+5511999999999"},
	}
	_, err := service.HandleInbound(context.Background(), "https://example.com/webhook", "invalid", values)
	if err != ErrInvalidSignature {
		t.Fatalf("expected invalid signature, got %v", err)
	}
	if repository.confirmedPhone != "" {
		t.Fatal("invalid webhook must not persist the phone")
	}
}

func twilioSignature(webhookURL string, values url.Values, secret string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	payload := webhookURL
	for _, key := range keys {
		for _, value := range values[key] {
			payload += key + value
		}
	}
	mac := hmac.New(sha1.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

type settingsResolverStub struct{}

func (settingsResolverStub) Resolve(context.Context, domain.ID) (*domain.WhatsappSettings, error) {
	return &domain.WhatsappSettings{
		ConnectionMode: domain.WhatsappConnectionDedicated, Provider: "twilio",
		InstanceName: "whatsapp:+5511999999999", APIKeyEncrypted: "ACtest:test-secret", Enabled: true,
	}, nil
}

type studentRepositoryStub struct{}

func (studentRepositoryStub) FindByID(context.Context, domain.ID, domain.ID) (*domain.Student, error) {
	return &domain.Student{}, nil
}
func (studentRepositoryStub) FindByExternalID(context.Context, domain.ID, domain.Source, string) (*domain.Student, error) {
	return nil, nil
}
func (studentRepositoryStub) List(context.Context, domain.ID, repositories.StudentFilters) ([]domain.Student, error) {
	return nil, nil
}
func (studentRepositoryStub) Save(context.Context, *domain.Student) error { return nil }
func (studentRepositoryStub) UpdateRiskStatus(context.Context, domain.ID, domain.ID, domain.StudentRiskStatus) error {
	return nil
}
func (studentRepositoryStub) MarkRiskMessageSent(context.Context, domain.ID, domain.ID, time.Time) error {
	return nil
}
func (studentRepositoryStub) UpdateContactPreference(context.Context, domain.ID, domain.ID, domain.ContactStatus, string, time.Time) error {
	return nil
}

type activationRepositoryStub struct {
	boxID          domain.ID
	boxName        string
	activationCode string
	matches        []domain.Student
	created        *domain.ContactActivationRequest
	confirmedPhone string
}

func (r *activationRepositoryStub) FindPublicBox(context.Context, string) (domain.ID, string, error) {
	return r.boxID, r.boxName, nil
}
func (r *activationRepositoryStub) ActivationCode(context.Context, domain.ID) (string, error) {
	return r.activationCode, nil
}
func (r *activationRepositoryStub) FindMatchingStudents(context.Context, domain.ID, domain.Source, string, time.Time) ([]domain.Student, error) {
	return r.matches, nil
}
func (r *activationRepositoryStub) CreateActivation(_ context.Context, activation *domain.ContactActivationRequest) error {
	copy := *activation
	copy.ID = domain.ID("activation-1")
	r.created = &copy
	return nil
}
func (r *activationRepositoryStub) FindActivationByTokenHash(_ context.Context, hash string) (*domain.ContactActivationRequest, error) {
	if r.created != nil && r.created.TokenHash == hash {
		copy := *r.created
		return &copy, nil
	}
	return nil, ErrInvalidInput
}
func (r *activationRepositoryStub) FindActivationsByPhoneAndSender(context.Context, string, string) ([]domain.ContactActivationRequest, error) {
	return nil, nil
}
func (r *activationRepositoryStub) ConfirmActivation(_ context.Context, _ domain.ID, phone string, confirmedAt time.Time) (*domain.ContactActivationRequest, error) {
	r.confirmedPhone = phone
	copy := *r.created
	copy.Phone = phone
	copy.StudentName = "Adriana Segatelli"
	copy.Status = domain.ContactActivationConfirmed
	copy.ConsentedAt = &confirmedAt
	return &copy, nil
}
func (r *activationRepositoryStub) OptOutActivations(context.Context, []domain.ID, string, time.Time) error {
	return nil
}
func (r *activationRepositoryStub) ListActivations(context.Context, domain.ID) ([]domain.ContactActivationRequest, error) {
	return nil, nil
}
func (r *activationRepositoryStub) ResolveActivation(context.Context, domain.ID, domain.ID, domain.ID, time.Time) (*domain.ContactActivationRequest, error) {
	return nil, nil
}
func (r *activationRepositoryStub) Summary(context.Context, domain.ID) (domain.ContactActivationSummary, error) {
	return domain.ContactActivationSummary{}, nil
}
