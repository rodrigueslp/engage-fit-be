package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
)

type ContactActivationRepository interface {
	FindPublicBox(ctx context.Context, activationCode string) (domain.ID, string, error)
	ActivationCode(ctx context.Context, boxID domain.ID) (string, error)
	FindMatchingStudents(ctx context.Context, boxID domain.ID, source domain.Source, name string, recentCheckinDate time.Time) ([]domain.Student, error)
	CreateActivation(ctx context.Context, activation *domain.ContactActivationRequest) error
	FindActivationByTokenHash(ctx context.Context, tokenHash string) (*domain.ContactActivationRequest, error)
	FindActivationsByPhoneAndSender(ctx context.Context, phone, senderPhone string) ([]domain.ContactActivationRequest, error)
	ConfirmActivation(ctx context.Context, activationID domain.ID, phone string, confirmedAt time.Time) (*domain.ContactActivationRequest, error)
	OptOutActivations(ctx context.Context, activationIDs []domain.ID, phone string, optedOutAt time.Time) error
	ListActivations(ctx context.Context, boxID domain.ID) ([]domain.ContactActivationRequest, error)
	ResolveActivation(ctx context.Context, boxID, activationID, studentID domain.ID, resolvedAt time.Time) (*domain.ContactActivationRequest, error)
	Summary(ctx context.Context, boxID domain.ID) (domain.ContactActivationSummary, error)
}
