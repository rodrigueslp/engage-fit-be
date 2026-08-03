package repositories

import (
	"context"
	"errors"
	"time"

	"boxengage/backend/internal/domain"
)

var ErrContactActivationConflict = errors.New("contact activation conflicts with an existing student")

type ContactActivationRepository interface {
	FindPublicBox(ctx context.Context, activationCode string) (domain.ID, string, error)
	ActivationCode(ctx context.Context, boxID domain.ID) (string, error)
	FindActivationMatchData(ctx context.Context, boxID domain.ID, source domain.Source, recentCheckinDate time.Time) (domain.ContactActivationMatchData, error)
	CreateActivation(ctx context.Context, activation *domain.ContactActivationRequest) error
	FindActivationByTokenHash(ctx context.Context, tokenHash string) (*domain.ContactActivationRequest, error)
	FindActivationsByPhoneAndSender(ctx context.Context, phone, senderPhone string) ([]domain.ContactActivationRequest, error)
	ConfirmActivation(ctx context.Context, activationID domain.ID, phone string, confirmedAt time.Time) (*domain.ContactActivationRequest, error)
	OptOutActivations(ctx context.Context, activationIDs []domain.ID, phone string, optedOutAt time.Time) error
	ListActivations(ctx context.Context, boxID domain.ID) ([]domain.ContactActivationRequest, error)
	ListPendingSyncActivations(ctx context.Context, boxID domain.ID, source domain.Source) ([]domain.ContactActivationRequest, error)
	ResolveActivation(ctx context.Context, boxID, activationID, studentID domain.ID, matchStrategy string, resolvedAt time.Time) (*domain.ContactActivationRequest, error)
	CreateStudentFromReview(ctx context.Context, boxID, activationID domain.ID, resolvedAt time.Time) (*domain.ContactActivationRequest, error)
	CancelReview(ctx context.Context, boxID, activationID domain.ID, resolvedAt time.Time) (*domain.ContactActivationRequest, error)
	MarkActivationNeedsReview(ctx context.Context, boxID, activationID domain.ID, matchStrategy string, updatedAt time.Time) error
	Summary(ctx context.Context, boxID domain.ID) (domain.ContactActivationSummary, error)
}
