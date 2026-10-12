package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
)

type PersonIdentityLinker interface {
	ReconcileAthleteAccount(ctx context.Context, unitID, studentID, athleteAccountID domain.ID, linkedAt time.Time) error
}

// ImportedStudentPersonLinker completes the legacy Student projection without
// merging identities or changing existing person profiles and unit access.
type ImportedStudentPersonLinker interface {
	EnsureImportedStudent(ctx context.Context, unitID, studentID domain.ID) (domain.ID, error)
}

type PeopleRepository interface {
	DuplicateCheck(ctx context.Context, organizationID, unitID domain.ID, cpfLookupHash, phoneE164 string) (*domain.PersonDuplicateCheck, error)
	Create(ctx context.Context, command domain.CreatePersonCommand) (*domain.PersonCreationResult, bool, error)
	List(ctx context.Context, organizationID, unitID domain.ID, criteria domain.PersonSearchCriteria) ([]domain.PersonSummary, error)
	Get(ctx context.Context, organizationID, unitID, personID domain.ID) (*domain.PersonProfile, error)
	Update(ctx context.Context, command domain.UpdatePersonCommand) (*domain.PersonProfile, error)
	LinkUnit(ctx context.Context, command domain.LinkPersonUnitCommand) (*domain.PersonProfile, error)
	LinkExternalIdentity(ctx context.Context, command domain.LinkExternalIdentityCommand) (*domain.PersonProfile, error)
	ListIdentityReviews(ctx context.Context, organizationID, unitID domain.ID) ([]domain.PersonIdentityReview, error)
	ResolveIdentityReview(ctx context.Context, command domain.MergePeopleCommand, resolution string) (*domain.PersonIdentityReview, *domain.PersonProfile, error)
	Merge(ctx context.Context, command domain.MergePeopleCommand) (*domain.PersonProfile, error)
	CountStudentsWithoutPerson(ctx context.Context) (int64, error)
	ResolveContractParty(ctx context.Context, organizationID, personID domain.ID) (*domain.ContractPartyResolution, error)
}
