package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
)

type AttendanceAccessRepository interface {
	LockAttendanceCommand(context.Context, domain.ID, string) error
	FindAttendanceCommand(context.Context, domain.ID, string) (*domain.AttendanceCommandRequest, error)
	ResolveAttendanceSubject(context.Context, domain.ID, domain.ID, domain.ID, time.Time) (*domain.AttendanceSubject, error)
	FindAttendanceReservation(context.Context, domain.ID, domain.ID, domain.ID, domain.ID) (*domain.Reservation, error)
	CreateAttendanceDecision(context.Context, *domain.AttendanceDecision) error
	CreateAttendanceRecord(context.Context, *domain.AttendanceRecord, domain.Source, domain.ID) error
	CreateAttendanceCommand(context.Context, *domain.AttendanceCommandRequest) error
	GetAttendanceAdmission(context.Context, domain.ID, domain.ID) (*domain.AttendanceAdmission, error)
	ListAttendanceForDate(context.Context, domain.ID, domain.ID, string) ([]domain.AttendanceAdmission, error)
}
