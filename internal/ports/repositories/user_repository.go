package repositories

import (
	"context"

	"boxengage/backend/internal/domain"
)

type UserRepository interface {
	FindByID(ctx context.Context, id domain.ID) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	Save(ctx context.Context, user *domain.User) error
}
