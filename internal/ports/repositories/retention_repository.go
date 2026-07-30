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
	SummarizeInterventions(ctx context.Context, boxID domain.ID, start, end time.Time) (*domain.RetentionInterventionSummary, error)
	ListOnboardingMetrics(ctx context.Context, boxID domain.ID, today time.Time) ([]domain.OnboardingMetrics, error)
	UpdateMembershipStart(ctx context.Context, boxID, studentID domain.ID, startedAt time.Time, source string, updatedAt time.Time) error
}
