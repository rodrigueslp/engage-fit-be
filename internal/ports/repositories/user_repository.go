package repositories

import (
	"context"

	"boxengage/backend/internal/domain"
)

type UserRepository interface {
	FindByID(ctx context.Context, id domain.ID) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindOwnerByBoxID(ctx context.Context, boxID domain.ID) (*domain.User, error)
	Save(ctx context.Context, user *domain.User) error
	UpdatePassword(ctx context.Context, id domain.ID, passwordHash string) error
	BumpAuthVersion(ctx context.Context, id domain.ID) error
	UpdatePlatformAdminCredentials(ctx context.Context, id domain.ID, name, passwordHash string) error
}
