package contactactivation

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
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
		matchData: domain.ContactActivationMatchData{Candidates: []domain.ContactActivationCandidate{{
			Student:          domain.Student{ID: domain.ID("student-1"), BoxID: domain.ID("box-1"), Name: "Adriana Segatelli", Source: domain.SourceTotalPass},
			HasRecentCheckin: true,
		}}},
	}
	service := NewService(repository, studentRepositoryStub{}, settingsResolverStub{}, "")
	service.now = func() time.Time { return now }
	checkinDate := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)

	started, err := service.Start(context.Background(), StartInput{
		ActivationCode: "activation-code", Name: "  Adriana Segatelli ", Source: domain.SourceTotalPass,
		RecentCheckinDate: &checkinDate, ConsentAccepted: true,
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

func TestDecideActivationMatchAcceptsCompatibleNameWithExactCheckin(t *testing.T) {
	checkinDate := time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)
	decision := decideActivationMatch("Vítor Lima", checkinDate, domain.ContactActivationMatchData{
		Candidates: []domain.ContactActivationCandidate{{
			Student: domain.Student{ID: "student-1", Name: "Vitor Lima de Oliveira"}, HasRecentCheckin: true,
		}},
		LatestCheckinDate: &checkinDate,
	})
	if decision.studentID != "student-1" || decision.strategy != "compatible_name_checkin" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestDecideActivationMatchAcceptsExactUniqueNameWhenSourceIsDelayed(t *testing.T) {
	latestDate := time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)
	claimedDate := latestDate.AddDate(0, 0, 1)
	decision := decideActivationMatch("João Paulo da Rocha", claimedDate, domain.ContactActivationMatchData{
		Candidates:        []domain.ContactActivationCandidate{{Student: domain.Student{ID: "student-1", Name: "Joao Paulo da Rocha"}}},
		LatestCheckinDate: &latestDate,
	})
	if decision.studentID != "student-1" || decision.strategy != "exact_unique_source_lag" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestDecideActivationMatchWaitsForImportWhenPartialNameHasNoImportedCheckin(t *testing.T) {
	latestDate := time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)
	claimedDate := latestDate.AddDate(0, 0, 1)
	decision := decideActivationMatch("Vitor Lima", claimedDate, domain.ContactActivationMatchData{
		Candidates: []domain.ContactActivationCandidate{
			{Student: domain.Student{ID: "student-1", Name: "Vitor Lima de Oliveira"}},
			{Student: domain.Student{ID: "student-2", Name: "Vitor Lima Santos"}},
		},
		LatestCheckinDate: &latestDate,
	})
	if !decision.pending || decision.studentID != "" || decision.strategy != "awaiting_source_sync" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestDecideActivationMatchKeepsAmbiguousImportedNamesForReview(t *testing.T) {
	checkinDate := time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)
	decision := decideActivationMatch("Vitor Lima", checkinDate, domain.ContactActivationMatchData{
		Candidates: []domain.ContactActivationCandidate{
			{Student: domain.Student{ID: "student-1", Name: "Vitor Lima de Oliveira"}, HasRecentCheckin: true},
			{Student: domain.Student{ID: "student-2", Name: "Vitor Lima Santos"}, HasRecentCheckin: true},
		},
		LatestCheckinDate: &checkinDate,
	})
	if decision.pending || decision.studentID != "" || decision.strategy != "ambiguous_name" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestReprocessPendingResolvesAfterImportedCheckin(t *testing.T) {
	checkinDate := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	repository := &activationRepositoryStub{
		pendingItems: []domain.ContactActivationRequest{{
			ID: "activation-1", BoxID: "box-1", ClaimedName: "Vitor Lima", Source: domain.SourceTotalPass,
			RecentCheckinDate: &checkinDate, Status: domain.ContactActivationPendingSync,
		}},
		matchData: domain.ContactActivationMatchData{
			Candidates: []domain.ContactActivationCandidate{{
				Student: domain.Student{ID: "student-1", Name: "Vitor Lima de Oliveira"}, HasRecentCheckin: true,
			}},
			LatestCheckinDate: &checkinDate,
		},
	}
	service := NewService(repository, studentRepositoryStub{}, settingsResolverStub{}, "")
	service.now = func() time.Time { return checkinDate.Add(12 * time.Hour) }

	if err := service.ReprocessPending(context.Background(), "box-1", domain.SourceTotalPass); err != nil {
		t.Fatal(err)
	}
	if repository.resolvedStudentID != "student-1" || repository.resolvedStrategy != "after_import_compatible_name_checkin" {
		t.Fatalf("unexpected resolution: student=%s strategy=%s", repository.resolvedStudentID, repository.resolvedStrategy)
	}
}

func TestStartNewStudentDoesNotRequirePreviousCheckin(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	repository := &activationRepositoryStub{
		boxID: domain.ID("box-1"), boxName: "CrossFit Alados", activationCode: "activation-code",
	}
	service := NewService(repository, studentRepositoryStub{}, settingsResolverStub{}, "")
	service.now = func() time.Time { return now }

	started, err := service.Start(context.Background(), StartInput{
		ActivationCode: "activation-code", Name: "Pessoa Nova", Source: domain.SourceWellhub,
		IsNewStudent: true, ConsentAccepted: true,
	})
	if err != nil {
		t.Fatalf("start new student activation: %v", err)
	}
	if started.WhatsappURL == "" || repository.created == nil {
		t.Fatalf("expected activation request, got %+v", started)
	}
	if !repository.created.IsNewStudent || repository.created.RecentCheckinDate != nil || repository.created.StudentID != "" {
		t.Fatalf("unexpected new student activation: %+v", repository.created)
	}
	if repository.matchCalls != 0 {
		t.Fatalf("new student must not be matched against check-in history, got %d calls", repository.matchCalls)
	}
}

func TestCreateStudentFromReviewAndCancel(t *testing.T) {
	now := time.Date(2026, 8, 3, 18, 0, 0, 0, time.UTC)
	repository := &activationRepositoryStub{}
	service := NewService(repository, studentRepositoryStub{}, settingsResolverStub{}, "")
	service.now = func() time.Time { return now }

	created, err := service.CreateStudentFromReview(context.Background(), "box-1", "activation-1")
	if err != nil {
		t.Fatalf("create student from review: %v", err)
	}
	if !repository.createFromReviewCalled || created.Status != domain.ContactActivationConfirmed {
		t.Fatalf("unexpected create result: %+v", created)
	}
	cancelled, err := service.CancelReview(context.Background(), "box-1", "activation-2")
	if err != nil {
		t.Fatalf("cancel review: %v", err)
	}
	if !repository.cancelReviewCalled || cancelled.Status != domain.ContactActivationCancelled {
		t.Fatalf("unexpected cancel result: %+v", cancelled)
	}
}

func TestCreateStudentFromReviewMapsConflict(t *testing.T) {
	repository := &activationRepositoryStub{createFromReviewErr: repositories.ErrContactActivationConflict}
	service := NewService(repository, studentRepositoryStub{}, settingsResolverStub{}, "")
	_, err := service.CreateStudentFromReview(context.Background(), "box-1", "activation-1")
	if !errors.Is(err, ErrReviewConflict) {
		t.Fatalf("expected review conflict, got %v", err)
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
	boxID                  domain.ID
	boxName                string
	activationCode         string
	matchData              domain.ContactActivationMatchData
	created                *domain.ContactActivationRequest
	confirmedPhone         string
	matchCalls             int
	pendingItems           []domain.ContactActivationRequest
	resolvedStudentID      domain.ID
	resolvedStrategy       string
	createFromReviewCalled bool
	cancelReviewCalled     bool
	createFromReviewErr    error
}

func (r *activationRepositoryStub) FindPublicBox(context.Context, string) (domain.ID, string, error) {
	return r.boxID, r.boxName, nil
}
func (r *activationRepositoryStub) ActivationCode(context.Context, domain.ID) (string, error) {
	return r.activationCode, nil
}
func (r *activationRepositoryStub) FindActivationMatchData(context.Context, domain.ID, domain.Source, time.Time) (domain.ContactActivationMatchData, error) {
	r.matchCalls++
	return r.matchData, nil
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
func (r *activationRepositoryStub) ListPendingSyncActivations(context.Context, domain.ID, domain.Source) ([]domain.ContactActivationRequest, error) {
	return r.pendingItems, nil
}
func (r *activationRepositoryStub) ResolveActivation(_ context.Context, _ domain.ID, _ domain.ID, studentID domain.ID, strategy string, _ time.Time) (*domain.ContactActivationRequest, error) {
	r.resolvedStudentID = studentID
	r.resolvedStrategy = strategy
	return &domain.ContactActivationRequest{StudentID: studentID, MatchStrategy: strategy}, nil
}
func (r *activationRepositoryStub) CreateStudentFromReview(context.Context, domain.ID, domain.ID, time.Time) (*domain.ContactActivationRequest, error) {
	r.createFromReviewCalled = true
	if r.createFromReviewErr != nil {
		return nil, r.createFromReviewErr
	}
	return &domain.ContactActivationRequest{Status: domain.ContactActivationConfirmed}, nil
}
func (r *activationRepositoryStub) CancelReview(context.Context, domain.ID, domain.ID, time.Time) (*domain.ContactActivationRequest, error) {
	r.cancelReviewCalled = true
	return &domain.ContactActivationRequest{Status: domain.ContactActivationCancelled}, nil
}
func (r *activationRepositoryStub) MarkActivationNeedsReview(context.Context, domain.ID, domain.ID, string, time.Time) error {
	return nil
}
func (r *activationRepositoryStub) Summary(context.Context, domain.ID) (domain.ContactActivationSummary, error) {
	return domain.ContactActivationSummary{}, nil
}
