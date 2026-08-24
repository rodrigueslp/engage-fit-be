package contracts

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"boxengage/backend/internal/domain"
	"github.com/google/uuid"
)

var (
	ErrSimulatorDisabledInProduction = errors.New("contract acceptance simulator is strictly forbidden in production")
	ErrSimulatorNotEnabled           = errors.New("contract acceptance simulator is not enabled")
)

type LocalSimulatorAdapter struct {
	appEnv           string
	simulatorEnabled bool
	now              func() time.Time
}

func NewLocalSimulatorAdapter(appEnv string, simulatorEnabled bool) *LocalSimulatorAdapter {
	return &LocalSimulatorAdapter{
		appEnv:           appEnv,
		simulatorEnabled: simulatorEnabled,
		now:              func() time.Time { return time.Now().UTC() },
	}
}

func (a *LocalSimulatorAdapter) Prepare(ctx context.Context, instanceID domain.ID, documentSHA256 string, expiresAt time.Time) (*domain.AcceptanceSessionLaunch, error) {
	if a.appEnv == "production" {
		return nil, ErrSimulatorDisabledInProduction
	}
	if !a.simulatorEnabled {
		return nil, ErrSimulatorNotEnabled
	}

	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("generate random launch token: %w", err)
	}
	launchToken := base64.RawURLEncoding.EncodeToString(randomBytes)

	adapterRef := fmt.Sprintf("sim_%s", uuid.NewString())
	launchPath := fmt.Sprintf("/#/public/contract-acceptance/%s", launchToken)

	return &domain.AcceptanceSessionLaunch{
		AdapterReference: adapterRef,
		LaunchToken:      launchToken,
		LaunchPath:       launchPath,
		ExpiresAt:        expiresAt,
	}, nil
}

func (a *LocalSimulatorAdapter) Verify(ctx context.Context, session *domain.ContractAcceptanceSession, submission domain.AcceptanceSubmission) (*domain.AcceptanceVerificationResult, error) {
	if a.appEnv == "production" {
		return nil, ErrSimulatorDisabledInProduction
	}
	if !a.simulatorEnabled {
		return nil, ErrSimulatorNotEnabled
	}

	if !submission.Accepted {
		return &domain.AcceptanceVerificationResult{Verified: false}, nil
	}

	acceptedAt := a.now()
	evidence := map[string]any{
		"adapter":                      domain.AdapterLocalSimulator,
		"adapter_reference":            session.AdapterReference,
		"document_sha256":              submission.DocumentSHA256,
		"acceptance_statement_version": submission.AcceptanceStatementVersion,
		"accepted_at":                  acceptedAt.Format(time.RFC3339),
		"simulator_notice":             "Simulador local de homologação — sem validade jurídica de assinatura eletrônica",
	}

	return &domain.AcceptanceVerificationResult{
		Verified:           true,
		AcceptedAt:         acceptedAt,
		NormalizedEvidence: evidence,
	}, nil
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
