package repositories

import (
	"context"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
)

func (r WorkoutGormRepository) ListWorkouts(ctx context.Context, boxID domain.ID) ([]domain.Workout, error) {
	var modelsList []models.WorkoutModel
	if err := r.db.WithContext(ctx).Where("box_id = ?", stringID(boxID)).Order("workout_date DESC, created_at DESC").Find(&modelsList).Error; err != nil {
		return nil, err
	}
	workouts := make([]domain.Workout, 0, len(modelsList))
	for _, model := range modelsList {
		workouts = append(workouts, workoutToDomain(model))
	}
	return workouts, nil
}

func (r WorkoutGormRepository) FindWorkoutByID(ctx context.Context, boxID, id domain.ID) (*domain.Workout, error) {
	var model models.WorkoutModel
	if err := r.db.WithContext(ctx).Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).First(&model).Error; err != nil {
		return nil, err
	}
	workout := workoutToDomain(model)
	return &workout, nil
}

func (r WorkoutGormRepository) SaveWorkout(ctx context.Context, workout *domain.Workout) error {
	if err := ensureID(&workout.ID); err != nil {
		return err
	}
	model := workoutToModel(*workout)
	return r.db.WithContext(ctx).Save(&model).Error
}

func (r WorkoutGormRepository) UpdateWorkout(ctx context.Context, workout domain.Workout) error {
	model := workoutToModel(workout)
	return r.db.WithContext(ctx).Where("box_id = ? AND id = ?", stringID(workout.BoxID), stringID(workout.ID)).Save(&model).Error
}

func (r WorkoutGormRepository) DeleteWorkout(ctx context.Context, boxID, id domain.ID) error {
	return r.db.WithContext(ctx).Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).Delete(&models.WorkoutModel{}).Error
}

func (r WorkoutGormRepository) ListDrafts(ctx context.Context, boxID, workoutID domain.ID) ([]domain.WorkoutMessageDraft, error) {
	var modelsList []models.WorkoutMessageDraftModel
	if err := r.db.WithContext(ctx).Where("box_id = ? AND workout_id = ?", stringID(boxID), stringID(workoutID)).Order("generated_at DESC").Find(&modelsList).Error; err != nil {
		return nil, err
	}
	drafts := make([]domain.WorkoutMessageDraft, 0, len(modelsList))
	for _, model := range modelsList {
		drafts = append(drafts, workoutDraftToDomain(model))
	}
	return drafts, nil
}

func (r WorkoutGormRepository) FindDraftByID(ctx context.Context, boxID, id domain.ID) (*domain.WorkoutMessageDraft, error) {
	var model models.WorkoutMessageDraftModel
	if err := r.db.WithContext(ctx).Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).First(&model).Error; err != nil {
		return nil, err
	}
	draft := workoutDraftToDomain(model)
	return &draft, nil
}

func (r WorkoutGormRepository) SaveDraft(ctx context.Context, draft *domain.WorkoutMessageDraft) error {
	if err := ensureID(&draft.ID); err != nil {
		return err
	}
	model := workoutDraftToModel(*draft)
	return r.db.WithContext(ctx).Save(&model).Error
}

func (r WorkoutGormRepository) UpdateDraft(ctx context.Context, draft domain.WorkoutMessageDraft) error {
	model := workoutDraftToModel(draft)
	return r.db.WithContext(ctx).Where("box_id = ? AND id = ?", stringID(draft.BoxID), stringID(draft.ID)).Save(&model).Error
}

func (r WorkoutGormRepository) ListRecipients(ctx context.Context, draftID domain.ID) ([]domain.WorkoutMessageRecipient, error) {
	var modelsList []models.WorkoutMessageRecipientModel
	if err := r.db.WithContext(ctx).Where("workout_message_draft_id = ?", stringID(draftID)).Order("created_at ASC").Find(&modelsList).Error; err != nil {
		return nil, err
	}
	recipients := make([]domain.WorkoutMessageRecipient, 0, len(modelsList))
	for _, model := range modelsList {
		recipients = append(recipients, workoutRecipientToDomain(model))
	}
	return recipients, nil
}

func (r WorkoutGormRepository) SaveRecipients(ctx context.Context, recipients []domain.WorkoutMessageRecipient) error {
	modelsList := make([]models.WorkoutMessageRecipientModel, 0, len(recipients))
	for i := range recipients {
		if err := ensureID(&recipients[i].ID); err != nil {
			return err
		}
		modelsList = append(modelsList, workoutRecipientToModel(recipients[i]))
	}
	if len(modelsList) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&modelsList).Error
}

func (r WorkoutGormRepository) UpdateRecipient(ctx context.Context, recipient domain.WorkoutMessageRecipient) error {
	model := workoutRecipientToModel(recipient)
	return r.db.WithContext(ctx).Save(&model).Error
}

func (r WorkoutGormRepository) SaveGenerationLog(ctx context.Context, log *domain.LLMGenerationLog) error {
	if err := ensureID(&log.ID); err != nil {
		return err
	}
	model := llmGenerationLogToModel(*log)
	return r.db.WithContext(ctx).Save(&model).Error
}
