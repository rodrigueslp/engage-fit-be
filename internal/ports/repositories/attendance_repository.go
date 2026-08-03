package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
)

type AttendanceRepository interface {
	SaveSelfCheckinSession(ctx context.Context, session *domain.SelfCheckinSession) error
	FindValidSelfCheckinSession(ctx context.Context, tokenHash string, now time.Time) (*domain.SelfCheckinSession, string, error)
	FindActiveBoxMembersByPhone(ctx context.Context, boxID domain.ID, phone string) ([]domain.Student, error)
	SaveBoxMemberCheckin(ctx context.Context, checkin *domain.Checkin) (created bool, error error)
}
