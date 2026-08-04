package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
)

type ImportHistoryRepository interface {
	FindByID(ctx context.Context, boxID, id domain.ID) (*domain.ImportHistory, error)
	List(ctx context.Context, boxID domain.ID) ([]domain.ImportHistory, error)
	Save(ctx context.Context, importHistory *domain.ImportHistory) error
	MarkCompleted(ctx context.Context, boxID, id domain.ID, studentsCreated, checkinsCreated int, completedAt time.Time) error
	MarkFailed(ctx context.Context, boxID, id domain.ID, errorCode string, completedAt time.Time) error
	SetRetentionBaselineIfEmpty(ctx context.Context, boxID domain.ID, baseline time.Time) error
}
