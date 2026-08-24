package services

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
)

type ContractAcceptancePort interface {
	Prepare(ctx context.Context, instanceID domain.ID, documentSHA256 string, expiresAt time.Time) (*domain.AcceptanceSessionLaunch, error)
	Verify(ctx context.Context, session *domain.ContractAcceptanceSession, submission domain.AcceptanceSubmission) (*domain.AcceptanceVerificationResult, error)
}
