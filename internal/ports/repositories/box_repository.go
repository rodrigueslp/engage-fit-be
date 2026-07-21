package repositories

import (
	"context"

	"boxengage/backend/internal/domain"
)

type BoxRepository interface {
	FindByID(ctx context.Context, id domain.ID) (*domain.Box, error)
	ListAll(ctx context.Context) ([]domain.Box, error)
	Save(ctx context.Context, box *domain.Box) error
	SaveWithOwner(ctx context.Context, box *domain.Box, owner *domain.User) error
	Update(ctx context.Context, box domain.Box) error
}
