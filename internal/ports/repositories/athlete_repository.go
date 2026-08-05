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
	ClaimInvitation(ctx context.Context, invitationID domain.ID, account *domain.AthleteAccount, existingAthleteID domain.ID, now time.Time) (*domain.AthleteMembership, error)
	SaveSession(ctx context.Context, session *domain.AthleteSession) error
	FindContextBySessionHash(ctx context.Context, tokenHash string, now time.Time) (*domain.AthleteContext, error)
	RevokeSession(ctx context.Context, tokenHash string, now time.Time) error
	ListPublishedWorkouts(ctx context.Context, athleteID domain.ID) ([]domain.AthleteWorkout, error)
}
