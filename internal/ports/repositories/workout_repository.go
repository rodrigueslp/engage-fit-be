package repositories

import (
	"context"

	"boxengage/backend/internal/domain"
)

type WorkoutRepository interface {
	ListWorkouts(ctx context.Context, boxID domain.ID) ([]domain.Workout, error)
	FindWorkoutByID(ctx context.Context, boxID, id domain.ID) (*domain.Workout, error)
	SaveWorkout(ctx context.Context, workout *domain.Workout) error
	UpdateWorkout(ctx context.Context, workout domain.Workout) error
	DeleteWorkout(ctx context.Context, boxID, id domain.ID) error

	ListDrafts(ctx context.Context, boxID, workoutID domain.ID) ([]domain.WorkoutMessageDraft, error)
	FindDraftByID(ctx context.Context, boxID, id domain.ID) (*domain.WorkoutMessageDraft, error)
	SaveDraft(ctx context.Context, draft *domain.WorkoutMessageDraft) error
	UpdateDraft(ctx context.Context, draft domain.WorkoutMessageDraft) error

	ListRecipients(ctx context.Context, draftID domain.ID) ([]domain.WorkoutMessageRecipient, error)
	SaveRecipients(ctx context.Context, recipients []domain.WorkoutMessageRecipient) error
	UpdateRecipient(ctx context.Context, recipient domain.WorkoutMessageRecipient) error

	SaveGenerationLog(ctx context.Context, log *domain.LLMGenerationLog) error
}
