package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
)

type RetentionRepository interface {
	ListMetrics(ctx context.Context, boxID domain.ID, recentStart, previousStart, end time.Time) ([]domain.RetentionMetrics, error)
	ListInterventions(ctx context.Context, boxID, studentID domain.ID) ([]domain.RetentionIntervention, error)
	FindIntervention(ctx context.Context, boxID, id domain.ID) (*domain.RetentionIntervention, error)
	SaveIntervention(ctx context.Context, intervention *domain.RetentionIntervention) error
	UpdateIntervention(ctx context.Context, intervention domain.RetentionIntervention) error
}
