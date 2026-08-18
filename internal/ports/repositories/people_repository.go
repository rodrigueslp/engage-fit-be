package repositories

import (
	"context"

	"boxengage/backend/internal/domain"
)

type PeopleRepository interface {
	DuplicateCheck(ctx context.Context, organizationID, unitID domain.ID, cpfLookupHash, phoneE164 string) (*domain.PersonDuplicateCheck, error)
	Create(ctx context.Context, command domain.CreatePersonCommand) (*domain.PersonCreationResult, bool, error)
	CountStudentsWithoutPerson(ctx context.Context) (int64, error)
}
