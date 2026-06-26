package repositories

import (
	"context"

	"boxengage/backend/internal/domain"
)

type BoxRepository interface {
	FindByID(ctx context.Context, id domain.ID) (*domain.Box, error)
	Save(ctx context.Context, box *domain.Box) error
	Update(ctx context.Context, box domain.Box) error
}
