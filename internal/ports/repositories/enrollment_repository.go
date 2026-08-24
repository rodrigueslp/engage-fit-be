package repositories

import (
	"context"
	"errors"
	"time"

	"boxengage/backend/internal/domain"
)

var (
	ErrEnrollmentNotFound          = errors.New("enrollment not found")
	ErrActiveEnrollmentExists      = errors.New("active enrollment exists")
	ErrPlanVersionChanged          = errors.New("plan version changed")
	ErrEnrollmentRevisionConflict  = errors.New("enrollment revision conflict")
	ErrEnrollmentAlreadyEnded      = errors.New("enrollment already ended")
	ErrEnrollmentIdempotencyReused = errors.New("enrollment idempotency reused")
	ErrEnrollmentAddonNotFound     = errors.New("enrollment addon not found")
	ErrEnrollmentAddonDuplicate    = errors.New("enrollment addon duplicate")
)

type EnrollmentRepository interface {
	LockCommand(ctx context.Context, organizationID domain.ID, command, idempotencyKey string) error
	FindCommand(ctx context.Context, organizationID domain.ID, command, idempotencyKey string) (*domain.EnrollmentCommandRequest, error)
	CreateCommand(ctx context.Context, request *domain.EnrollmentCommandRequest) error
	LockSubject(ctx context.Context, organizationID, personID, unitID domain.ID) error
	ListForPerson(ctx context.Context, organizationID, unitID, personID domain.ID) (*domain.EnrollmentHistory, error)
	GetByID(ctx context.Context, organizationID, unitID, enrollmentID domain.ID) (*domain.Enrollment, error)
	GetOpenForUpdate(ctx context.Context, organizationID, unitID, personID domain.ID) (*domain.Enrollment, error)
	GetPlanVersionForEnrollment(ctx context.Context, organizationID, unitID, planID, versionID domain.ID) (*domain.MembershipPlan, *domain.MembershipPlanVersion, error)
	CreateEnrollment(ctx context.Context, enrollment *domain.Enrollment) error
	EndEnrollment(ctx context.Context, enrollmentID, organizationID, unitID domain.ID, expectedRevision int, endedOn time.Time, endedAt time.Time, reason string, actorID domain.ID) (*domain.Enrollment, error)
	CreateEvent(ctx context.Context, event *domain.EnrollmentEvent) error
	ListPrivacy(ctx context.Context, unitID, personID domain.ID) ([]domain.EnrollmentPrivacyExport, error)
}

type EnrollmentSubjectResolver interface {
	ResolveEnrollmentSubject(ctx context.Context, organizationID, unitID, studentID domain.ID) (*domain.EnrollmentSubject, error)
}

type EnrollmentPrivacyReader interface {
	ListPrivacy(ctx context.Context, unitID, personID domain.ID) ([]domain.EnrollmentPrivacyExport, error)
}
