package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
)

type CheckinIngestionRepository interface {
	ListSources(ctx context.Context, boxID domain.ID) ([]domain.CheckinIngestionSource, error)
	FindSource(ctx context.Context, id domain.ID) (*domain.CheckinIngestionSource, error)
	FindSourceForBox(ctx context.Context, boxID, id domain.ID) (*domain.CheckinIngestionSource, error)
	SaveSource(ctx context.Context, source *domain.CheckinIngestionSource) error
	UpdateSource(ctx context.Context, source domain.CheckinIngestionSource) error
	ClaimBatch(ctx context.Context, batch *domain.CheckinIngestionBatch) (bool, *domain.CheckinIngestionBatch, error)
	CompleteBatch(ctx context.Context, id domain.ID, status string, importID domain.ID, totalRecords, students, checkins int, completedAt time.Time) error
}
