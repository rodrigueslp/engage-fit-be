package repositories

import (
	"context"
	"errors"
	"time"

	"boxengage/backend/internal/domain"
)

var (
	ErrScheduleSeriesNotFound          = errors.New("schedule series not found")
	ErrScheduleSessionNotFound         = errors.New("schedule session not found")
	ErrScheduleSeriesInvalid           = errors.New("schedule series invalid")
	ErrScheduleSeriesArchived          = errors.New("schedule series archived")
	ErrScheduleSeriesAlreadyPublished  = errors.New("schedule series already published")
	ErrScheduleTargetUnavailable       = errors.New("schedule target unavailable")
	ErrScheduleSpaceUnavailable        = errors.New("schedule space unavailable")
	ErrScheduleProfessionalUnavailable = errors.New("schedule professional unavailable")
	ErrScheduleConflict                = errors.New("schedule conflict")
	ErrScheduleInvalidLocalTime        = errors.New("invalid local time")
	ErrSchedulePreviewRangeInvalid     = errors.New("schedule preview range invalid")
	ErrSchedulePreviewTooLarge         = errors.New("schedule preview too large")
	ErrScheduleSessionAlreadyStarted   = errors.New("schedule session already started")
	ErrScheduleSessionCancelled        = errors.New("schedule session cancelled")
	ErrScheduleMaterializerUnavailable = errors.New("schedule materializer unavailable")
)

type SchedulingGridRepository interface {
	LockCommand(ctx context.Context, organizationID domain.ID, command, idempotencyKey string) error
	FindCommand(ctx context.Context, organizationID domain.ID, command, idempotencyKey string) (*domain.SchedulingCommandRequest, error)
	CreateCommand(ctx context.Context, request *domain.SchedulingCommandRequest) error
	CreateAuditEvent(ctx context.Context, event *domain.SchedulingAuditEvent) error
	LockSeries(ctx context.Context, organizationID, seriesID domain.ID) error
	LockResources(ctx context.Context, organizationID domain.ID, professionalIDs, spaceIDs []domain.ID) error
	ListSeries(ctx context.Context, organizationID, unitID domain.ID, includeArchived bool) ([]domain.ScheduleSeries, error)
	ListSeriesAcrossOrganization(ctx context.Context, organizationID domain.ID) ([]domain.ScheduleSeries, error)
	ListPublishedSeries(ctx context.Context) ([]domain.ScheduleSeries, error)
	GetByID(ctx context.Context, organizationID, unitID, professionalID domain.ID) (*domain.Professional, error)
	GetSeries(ctx context.Context, organizationID, unitID, seriesID domain.ID, forUpdate bool) (*domain.ScheduleSeries, error)
	CreateSeries(ctx context.Context, series *domain.ScheduleSeries) error
	UpdateSeries(ctx context.Context, series *domain.ScheduleSeries) error
	CountActiveSeriesForProfessional(ctx context.Context, organizationID, professionalID domain.ID) (int64, error)
	ListSessions(ctx context.Context, organizationID, unitID domain.ID, from, until time.Time, includeCancelled bool) ([]domain.ScheduleSession, error)
	ListSessionsAcrossOrganization(ctx context.Context, organizationID domain.ID, from, until time.Time) ([]domain.ScheduleSession, error)
	GetSession(ctx context.Context, organizationID, unitID, sessionID domain.ID, forUpdate bool) (*domain.ScheduleSession, error)
	CreateSession(ctx context.Context, session *domain.ScheduleSession) error
	UpdateSession(ctx context.Context, session *domain.ScheduleSession) error
	CreateSessionEvent(ctx context.Context, event *domain.SessionEventV1) error
	ListSessionEvents(ctx context.Context, organizationID, unitID domain.ID, afterSequence int64, limit int) ([]domain.SessionEventV1, error)
	ClaimMaterializationRun(ctx context.Context, executionKey, windowFrom, windowUntil string, startedAt time.Time) (domain.ID, bool, error)
	CompleteMaterializationRun(ctx context.Context, runID domain.ID, status string, seriesRead, sessionsCreated, sessionsKept, conflicts int, errorCode string, finishedAt time.Time) error
}

type SchedulingCatalogResolver interface {
	ResolveSchedulingReferences(ctx context.Context, organizationID, unitID domain.ID, targetKind domain.ScheduleTargetKind, targetID domain.ID, spaceID *domain.ID) (*domain.ScheduleResolvedReferences, error)
}

type PublishedSessionReader interface {
	GetPublishedSession(ctx context.Context, organizationID, unitID, sessionID domain.ID) (*domain.ScheduleSession, error)
	ListPublishedSessions(ctx context.Context, organizationID, unitID domain.ID, from, until time.Time) ([]domain.ScheduleSession, error)
	ListSessionEvents(ctx context.Context, organizationID, unitID domain.ID, afterSequence int64, limit int) ([]domain.SessionEventV1, error)
}

type SessionReservationReconciler interface {
	CancelSessionReservations(context.Context, domain.ID, domain.ID, domain.ID, domain.ID, string) error
}
