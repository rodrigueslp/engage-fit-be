package repositories

import (
	"context"

	"boxengage/backend/internal/domain"
)

type ImportHistoryRepository interface {
	FindByID(ctx context.Context, boxID, id domain.ID) (*domain.ImportHistory, error)
	List(ctx context.Context, boxID domain.ID) ([]domain.ImportHistory, error)
	Save(ctx context.Context, importHistory *domain.ImportHistory) error
}
