package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
)

type TodayOperationsRepository interface {
	ReadToday(context.Context, domain.ID, domain.ID, string, time.Time, time.Time) (*domain.TodayOperations, error)
}
