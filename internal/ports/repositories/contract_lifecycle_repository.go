package repositories

import (
	"context"
	"errors"
	"time"

	"boxengage/backend/internal/domain"
)

var (
	ErrSignatureEnvelopeNotFound = errors.New("signature envelope not found")
	ErrSignatureEnvelopeExists   = errors.New("signature envelope exists")
	ErrContractLegalConflict     = errors.New("contract legal action conflict")
)

type ContractLifecycleRepository interface {
	CreateSignatureEnvelope(ctx context.Context, envelope *domain.ContractSignatureEnvelope, signer domain.ContractSignerQualification) error
	ListSignatureEnvelopes(ctx context.Context, organizationID, unitID, contractID domain.ID) ([]domain.ContractSignatureEnvelope, error)
	FindSignatureEnvelopeByReference(ctx context.Context, organizationID, unitID domain.ID, provider, providerReference string) (*domain.ContractSignatureEnvelope, error)
	ReconcileSignatureEnvelope(ctx context.Context, organizationID, unitID domain.ID, provider, providerReference, eventID, status string, evidence map[string]any, at time.Time) (*domain.ContractSignatureEnvelope, bool, error)
	CreateLegalAction(ctx context.Context, action *domain.ContractLegalAction) error
	ListLegalActions(ctx context.Context, organizationID, unitID, enrollmentID domain.ID) ([]domain.ContractLegalAction, error)
}
