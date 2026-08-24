package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
)

type ContractEnrollmentSnapshotResolver interface {
	ResolveContractEnrollmentSnapshot(ctx context.Context, organizationID, unitID, enrollmentID domain.ID, asOf time.Time) (*domain.ContractEnrollmentSnapshot, error)
}

type ContractPartyResolver interface {
	ResolveContractParty(ctx context.Context, organizationID, personID domain.ID) (*domain.ContractPartyResolution, error)
}
