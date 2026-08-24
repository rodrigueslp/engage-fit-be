package repositories

import (
	"context"

	"boxengage/backend/internal/domain"
)

type ContractPrivacyReader interface {
	ListPrivacy(ctx context.Context, organizationID, personID domain.ID) ([]domain.ContractPrivacyExport, error)
}
