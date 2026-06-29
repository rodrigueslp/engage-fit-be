package repositories

import (
	"context"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
)

func (r AutomationGormRepository) ListRuns(ctx context.Context, boxID domain.ID) ([]domain.AutomationRun, error) {
	var modelsList []models.AutomationRunModel
	if err := r.db.WithContext(ctx).Where("box_id = ?", stringID(boxID)).Order("started_at DESC").Limit(100).Find(&modelsList).Error; err != nil {
		return nil, err
	}
	runs := make([]domain.AutomationRun, 0, len(modelsList))
	for _, model := range modelsList {
		runs = append(runs, automationRunToDomain(model))
	}
	return runs, nil
}

func (r AutomationGormRepository) FindRunByID(ctx context.Context, boxID, id domain.ID) (*domain.AutomationRun, error) {
	var model models.AutomationRunModel
	if err := r.db.WithContext(ctx).Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).First(&model).Error; err != nil {
		return nil, err
	}
	run := automationRunToDomain(model)
	return &run, nil
}

func (r AutomationGormRepository) SaveRun(ctx context.Context, run *domain.AutomationRun) error {
	if err := ensureID(&run.ID); err != nil {
		return err
	}
	model := automationRunToModel(*run)
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r AutomationGormRepository) UpdateRun(ctx context.Context, run domain.AutomationRun) error {
	model := automationRunToModel(run)
	return r.db.WithContext(ctx).Save(&model).Error
}

func (r AutomationGormRepository) ListSchedules(ctx context.Context, boxID domain.ID) ([]domain.AutomationSchedule, error) {
	var modelsList []models.AutomationScheduleModel
	if err := r.db.WithContext(ctx).Where("box_id = ?", stringID(boxID)).Order("run_time ASC, name ASC").Find(&modelsList).Error; err != nil {
		return nil, err
	}
	schedules := make([]domain.AutomationSchedule, 0, len(modelsList))
	for _, model := range modelsList {
		schedules = append(schedules, automationScheduleToDomain(model))
	}
	return schedules, nil
}

func (r AutomationGormRepository) ListEnabledSchedules(ctx context.Context) ([]domain.AutomationSchedule, error) {
	var modelsList []models.AutomationScheduleModel
	if err := r.db.WithContext(ctx).Where("enabled = ?", true).Order("run_time ASC, name ASC").Find(&modelsList).Error; err != nil {
		return nil, err
	}
	schedules := make([]domain.AutomationSchedule, 0, len(modelsList))
	for _, model := range modelsList {
		schedules = append(schedules, automationScheduleToDomain(model))
	}
	return schedules, nil
}

func (r AutomationGormRepository) FindScheduleByID(ctx context.Context, boxID, id domain.ID) (*domain.AutomationSchedule, error) {
	var model models.AutomationScheduleModel
	if err := r.db.WithContext(ctx).Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).First(&model).Error; err != nil {
		return nil, err
	}
	schedule := automationScheduleToDomain(model)
	return &schedule, nil
}

func (r AutomationGormRepository) SaveSchedule(ctx context.Context, schedule *domain.AutomationSchedule) error {
	if err := ensureID(&schedule.ID); err != nil {
		return err
	}
	model := automationScheduleToModel(*schedule)
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r AutomationGormRepository) UpdateSchedule(ctx context.Context, schedule domain.AutomationSchedule) error {
	model := automationScheduleToModel(schedule)
	return r.db.WithContext(ctx).Save(&model).Error
}

func (r AutomationGormRepository) DeleteSchedule(ctx context.Context, boxID, id domain.ID) error {
	return r.db.WithContext(ctx).Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).Delete(&models.AutomationScheduleModel{}).Error
}
