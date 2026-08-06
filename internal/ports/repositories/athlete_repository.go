package repositories

import (
	"context"
	"errors"
	"time"

	"boxengage/backend/internal/domain"
)

var (
	ErrAthleteInvitationUnavailable = errors.New("athlete invitation unavailable")
	ErrAthleteIdentityConflict      = errors.New("athlete identity conflict")
	ErrAthleteAccountNotFound       = errors.New("athlete account not found")
)

type AthleteRepository interface {
	SaveInvitation(ctx context.Context, invitation *domain.AthleteInvitation) error
	FindInvitationByTokenHash(ctx context.Context, tokenHash string) (*domain.AthleteInvitation, error)
	FindAccountByEmail(ctx context.Context, email string) (*domain.AthleteAccount, error)
	FindAccountByID(ctx context.Context, athleteID domain.ID) (*domain.AthleteAccount, error)
	FindFirstActiveBoxID(ctx context.Context, athleteID domain.ID) (domain.ID, error)
	ClaimInvitation(ctx context.Context, invitationID domain.ID, account *domain.AthleteAccount, existingAthleteID domain.ID, now time.Time) (*domain.AthleteMembership, error)
	SaveSession(ctx context.Context, session *domain.AthleteSession) error
	FindContextBySessionHash(ctx context.Context, tokenHash string, now time.Time) (*domain.AthleteContext, error)
	RevokeSession(ctx context.Context, tokenHash string, now time.Time) error
	ListPublishedWorkouts(ctx context.Context, athleteID domain.ID) ([]domain.AthleteWorkout, error)
	FindPublishedWorkout(ctx context.Context, athleteID, workoutID domain.ID) (*domain.AthleteWorkout, error)
	UpsertWorkoutResult(ctx context.Context, result *domain.AthleteWorkoutResult) ([]domain.AthletePersonalRecord, error)
	ListWorkoutResults(ctx context.Context, athleteID domain.ID) ([]domain.AthleteWorkoutResult, error)
	ListPersonalRecords(ctx context.Context, athleteID domain.ID) ([]domain.AthletePersonalRecord, error)
	ConfirmPersonalRecord(ctx context.Context, athleteID, recordID domain.ID, now time.Time) error
	SaveAccountToken(ctx context.Context, token *domain.AthleteAccountToken) error
	ConsumeAccountToken(ctx context.Context, tokenHash, purpose string, now time.Time) (*domain.AthleteAccount, error)
	UpdateAthletePassword(ctx context.Context, athleteID domain.ID, passwordHash string, now time.Time) error
	VerifyAthleteEmail(ctx context.Context, athleteID domain.ID, now time.Time) error
	FindWorkoutInsight(ctx context.Context, athleteID, workoutID domain.ID, inputHash string) (*domain.AthleteWorkoutInsight, error)
	SaveWorkoutInsight(ctx context.Context, insight *domain.AthleteWorkoutInsight) error
}
