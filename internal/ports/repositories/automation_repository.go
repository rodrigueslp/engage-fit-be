package repositories

import (
	"context"

	"boxengage/backend/internal/domain"
)

type AutomationRepository interface {
	ListRuns(ctx context.Context, boxID domain.ID) ([]domain.AutomationRun, error)
	FindRunByID(ctx context.Context, boxID, id domain.ID) (*domain.AutomationRun, error)
	SaveRun(ctx context.Context, run *domain.AutomationRun) error
	UpdateRun(ctx context.Context, run domain.AutomationRun) error

	ListSchedules(ctx context.Context, boxID domain.ID) ([]domain.AutomationSchedule, error)
	ListEnabledSchedules(ctx context.Context) ([]domain.AutomationSchedule, error)
	FindScheduleByID(ctx context.Context, boxID, id domain.ID) (*domain.AutomationSchedule, error)
	SaveSchedule(ctx context.Context, schedule *domain.AutomationSchedule) error
	UpdateSchedule(ctx context.Context, schedule domain.AutomationSchedule) error
	DeleteSchedule(ctx context.Context, boxID, id domain.ID) error
}
