package contractlifecycle

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"github.com/google/uuid"
)

var (
	ErrInvalidLifecycleInput = errors.New("invalid contract lifecycle input")
	ErrSignatureUnavailable  = errors.New("contract signature provider unavailable")
	ErrContractNotAccepted   = errors.New("accepted contract required")
)

type Service struct {
	repository  repositories.ContractLifecycleRepository
	contracts   repositories.ContractRepository
	enrollments repositories.ContractEnrollmentSnapshotResolver
	appEnv      string
	providers   map[string]bool
	now         func() time.Time
}

func NewService(repository repositories.ContractLifecycleRepository, contracts repositories.ContractRepository, enrollments repositories.ContractEnrollmentSnapshotResolver, appEnv string, providers []string) *Service {
	allowed := map[string]bool{}
	for _, provider := range providers {
		if value := strings.TrimSpace(strings.ToLower(provider)); value != "" {
			allowed[value] = true
		}
	}
	return &Service{repository: repository, contracts: contracts, enrollments: enrollments, appEnv: appEnv, providers: allowed, now: func() time.Time { return time.Now().UTC() }}
}

type CreateSignatureEnvelopeInput struct {
	Provider          string                             `json:"provider"`
	ProviderReference string                             `json:"provider_reference"`
	Signer            domain.ContractSignerQualification `json:"signer"`
	ExpiresAt         time.Time                          `json:"expires_at"`
}

func (s *Service) CreateSignatureEnvelope(ctx context.Context, organizationID, unitID, contractID, actorID domain.ID, input CreateSignatureEnvelopeInput) (*domain.ContractSignatureEnvelope, error) {
	provider := strings.ToLower(strings.TrimSpace(input.Provider))
	input.ProviderReference = strings.TrimSpace(input.ProviderReference)
	input.Signer.Name = strings.TrimSpace(input.Signer.Name)
	input.Signer.Email = strings.TrimSpace(input.Signer.Email)
	input.Signer.Document = digitsOnly(input.Signer.Document)
	if !validID(organizationID) || !validID(unitID) || !validID(contractID) || !validID(actorID) || !s.providers[provider] || len(input.ProviderReference) < 3 || len(input.ProviderReference) > 160 || len(input.Signer.Name) < 2 || len(input.Signer.Document) < 11 || len(input.Signer.Document) > 14 || input.ExpiresAt.Before(s.now().Add(5*time.Minute)) {
		return nil, ErrInvalidLifecycleInput
	}
	if _, err := mail.ParseAddress(input.Signer.Email); err != nil {
		return nil, ErrInvalidLifecycleInput
	}
	if s.appEnv == "production" && provider == "local_signature_simulator" {
		return nil, ErrSignatureUnavailable
	}
	contract, err := s.contracts.FindInstanceByID(ctx, organizationID, unitID, contractID)
	if err != nil {
		return nil, err
	}
	if contract.Status != domain.ContractInstanceStatusPendingAcceptance {
		return nil, ErrSignatureUnavailable
	}
	now := s.now()
	envelope := &domain.ContractSignatureEnvelope{ID: domain.ID(uuid.NewString()), OrganizationID: organizationID, UnitID: unitID, ContractInstanceID: contractID, Provider: provider, ProviderReference: input.ProviderReference, Status: "pending", ExpiresAt: input.ExpiresAt, CreatedByUserID: actorID, CreatedAt: now, UpdatedAt: now}
	if err := s.repository.CreateSignatureEnvelope(ctx, envelope, input.Signer); err != nil {
		return nil, err
	}
	return envelope, nil
}

func (s *Service) ListSignatureEnvelopes(ctx context.Context, organizationID, unitID, contractID domain.ID) ([]domain.ContractSignatureEnvelope, error) {
	return s.repository.ListSignatureEnvelopes(ctx, organizationID, unitID, contractID)
}

type ReconcileSignatureInput struct {
	OrganizationID    domain.ID      `json:"organization_id"`
	UnitID            domain.ID      `json:"unit_id"`
	ProviderReference string         `json:"provider_reference"`
	EventID           string         `json:"event_id"`
	Status            string         `json:"status"`
	Evidence          map[string]any `json:"evidence"`
}

func (s *Service) ReconcileSignature(ctx context.Context, provider string, input ReconcileSignatureInput) (*domain.ContractSignatureEnvelope, bool, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	input.ProviderReference = strings.TrimSpace(input.ProviderReference)
	input.EventID = strings.TrimSpace(input.EventID)
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))
	allowedStatus := map[string]bool{"pending": true, "accepted": true, "declined": true, "expired": true, "failed": true}
	if !s.providers[provider] || !validID(input.OrganizationID) || !validID(input.UnitID) || len(input.ProviderReference) < 3 || len(input.ProviderReference) > 160 || len(input.EventID) < 3 || len(input.EventID) > 160 || !allowedStatus[input.Status] {
		return nil, false, ErrInvalidLifecycleInput
	}
	envelope, err := s.repository.FindSignatureEnvelopeByReference(ctx, input.OrganizationID, input.UnitID, provider, input.ProviderReference)
	if err != nil {
		return nil, false, err
	}
	if input.Status == "accepted" {
		contract, findErr := s.contracts.FindInstanceByID(ctx, input.OrganizationID, input.UnitID, envelope.ContractInstanceID)
		if findErr != nil {
			return nil, false, findErr
		}
		snapshot, resolveErr := s.enrollments.ResolveContractEnrollmentSnapshot(ctx, input.OrganizationID, input.UnitID, contract.EnrollmentID, s.now())
		if resolveErr != nil {
			return nil, false, resolveErr
		}
		if snapshot.EffectiveStatus != domain.EnrollmentEffectiveScheduled && snapshot.EffectiveStatus != domain.EnrollmentEffectiveActive {
			return nil, false, ErrSignatureUnavailable
		}
	}
	if input.Evidence == nil {
		input.Evidence = map[string]any{}
	}
	return s.repository.ReconcileSignatureEnvelope(ctx, input.OrganizationID, input.UnitID, provider, input.ProviderReference, input.EventID, input.Status, input.Evidence, s.now())
}

type CreateLegalActionInput struct {
	SourceContractInstanceID domain.ID      `json:"source_contract_instance_id"`
	ActionType               string         `json:"action_type"`
	EffectiveOn              string         `json:"effective_on"`
	EndsOn                   *string        `json:"ends_on"`
	Reason                   string         `json:"reason"`
	Terms                    map[string]any `json:"terms"`
	FinancialInstruction     map[string]any `json:"financial_instruction"`
}

func (s *Service) CreateLegalAction(ctx context.Context, organizationID, unitID, enrollmentID, actorID domain.ID, input CreateLegalActionInput) (*domain.ContractLegalAction, error) {
	input.ActionType = strings.ToLower(strings.TrimSpace(input.ActionType))
	input.Reason = strings.TrimSpace(input.Reason)
	allowed := map[string]bool{"amendment": true, "renewal": true, "pause": true, "resume": true, "cancellation": true}
	effective, err := time.Parse("2006-01-02", input.EffectiveOn)
	if err != nil || !allowed[input.ActionType] || len(input.Reason) < 3 || len(input.Reason) > 500 || !validID(enrollmentID) || !validID(input.SourceContractInstanceID) {
		return nil, ErrInvalidLifecycleInput
	}
	if input.EndsOn != nil {
		ends, parseErr := time.Parse("2006-01-02", *input.EndsOn)
		if parseErr != nil || ends.Before(effective) {
			return nil, ErrInvalidLifecycleInput
		}
	}
	contract, err := s.contracts.FindInstanceByID(ctx, organizationID, unitID, input.SourceContractInstanceID)
	if err != nil {
		return nil, err
	}
	if contract.EnrollmentID != enrollmentID || contract.Status != domain.ContractInstanceStatusAccepted {
		return nil, ErrContractNotAccepted
	}
	if input.Terms == nil {
		input.Terms = map[string]any{}
	}
	if input.FinancialInstruction == nil {
		input.FinancialInstruction = map[string]any{"mode": "manual_review"}
	}
	now := s.now()
	action := &domain.ContractLegalAction{ID: domain.ID(uuid.NewString()), OrganizationID: organizationID, UnitID: unitID, PersonID: contract.PersonID, EnrollmentID: enrollmentID, SourceContractInstanceID: contract.ID, ActionType: input.ActionType, Status: "recorded", EffectiveOn: input.EffectiveOn, EndsOn: input.EndsOn, Reason: input.Reason, TermsSnapshot: input.Terms, FinancialInstruction: input.FinancialInstruction, Revision: 1, CreatedByUserID: actorID, CreatedAt: now, UpdatedAt: now}
	if err := s.repository.CreateLegalAction(ctx, action); err != nil {
		return nil, err
	}
	return action, nil
}

func (s *Service) ListLegalActions(ctx context.Context, organizationID, unitID, enrollmentID domain.ID) ([]domain.ContractLegalAction, error) {
	return s.repository.ListLegalActions(ctx, organizationID, unitID, enrollmentID)
}

func validID(value domain.ID) bool { _, err := uuid.Parse(string(value)); return err == nil }
func digitsOnly(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
