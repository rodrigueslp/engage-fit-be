package repositories

import (
	"context"
	"errors"
	"time"

	"boxengage/backend/internal/domain"
)

var (
	ErrHealthFormNotFound         = errors.New("health form not found")
	ErrHealthFormAlreadySubmitted = errors.New("health form already submitted")
	ErrHealthFormRevisionConflict = errors.New("health form revision conflict")
	ErrHealthFormSessionInvalid   = errors.New("health form session invalid")
)

type HealthFormRepository interface {
	Create(ctx context.Context, form *domain.HealthFormSubmission, answers domain.PARQAnswers) error
	ListForPerson(ctx context.Context, organizationID, unitID, personID domain.ID) ([]domain.HealthFormSubmission, error)
	Find(ctx context.Context, organizationID, unitID, formID domain.ID, includeAnswers bool, actorUserID domain.ID) (*domain.HealthFormSubmission, error)
	RevokeConsent(ctx context.Context, organizationID, unitID, formID, actorUserID domain.ID, at time.Time) (*domain.HealthFormSubmission, error)
	ListPrivacy(ctx context.Context, organizationID, personID domain.ID) ([]domain.HealthFormPrivacyExport, error)
	PurgeExpired(ctx context.Context, now time.Time, apply bool) (int64, error)
	CreateSession(ctx context.Context, session *domain.HealthFormSession) error
	FindSessionByTokenHash(ctx context.Context, tokenHash string) (*domain.HealthFormSession, error)
	CreateFromSession(ctx context.Context, sessionID domain.ID, form *domain.HealthFormSubmission, answers domain.PARQAnswers, at time.Time) error
}
