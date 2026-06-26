package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
)

type CheckinRepository interface {
	ListByStudent(ctx context.Context, boxID, studentID domain.ID) ([]domain.Checkin, error)
	ListByRange(ctx context.Context, boxID domain.ID, period domain.TimeRange) ([]domain.Checkin, error)
	ListMonthlyFrequency(ctx context.Context, boxID domain.ID, period domain.TimeRange) ([]domain.MonthlyFrequencyReportRow, error)
	CountBySource(ctx context.Context, boxID domain.ID, period domain.TimeRange) (map[domain.Source]int, error)
	LastCheckinDate(ctx context.Context, boxID, studentID domain.ID) (*time.Time, error)
	SaveMany(ctx context.Context, checkins []domain.Checkin) (int, error)
}
