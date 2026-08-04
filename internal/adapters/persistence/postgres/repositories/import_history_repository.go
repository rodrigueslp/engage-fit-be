package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
)

func (r ImportHistoryGormRepository) FindByID(ctx context.Context, boxID, id domain.ID) (*domain.ImportHistory, error) {
	var model models.ImportHistoryModel
	if err := databaseForContext(r.db, ctx).Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).First(&model).Error; err != nil {
		return nil, err
	}

	importHistory := importHistoryToDomain(model)
	return &importHistory, nil
}

func (r ImportHistoryGormRepository) SetRetentionBaselineIfEmpty(ctx context.Context, boxID domain.ID, baseline time.Time) error {
	return databaseForContext(r.db, ctx).Table("boxes").
		Where("id = ? AND retention_baseline_at IS NULL", stringID(boxID)).
		Update("retention_baseline_at", baseline.UTC().Format("2006-01-02")).Error
}

func (r ImportHistoryGormRepository) List(ctx context.Context, boxID domain.ID) ([]domain.ImportHistory, error) {
	var modelsList []models.ImportHistoryModel
	if err := databaseForContext(r.db, ctx).Where("box_id = ?", stringID(boxID)).Order("imported_at DESC").Find(&modelsList).Error; err != nil {
		return nil, err
	}

	imports := make([]domain.ImportHistory, 0, len(modelsList))
	for _, model := range modelsList {
		imports = append(imports, importHistoryToDomain(model))
	}
	return imports, nil
}

func (r ImportHistoryGormRepository) Save(ctx context.Context, importHistory *domain.ImportHistory) error {
	if err := ensureID(&importHistory.ID); err != nil {
		return err
	}
	if importHistory.Status == "" {
		importHistory.Status = domain.ImportStatusProcessing
	}

	model := importHistoryToModel(*importHistory)
	return databaseForContext(r.db, ctx).Save(&model).Error
}

func (r ImportHistoryGormRepository) MarkCompleted(ctx context.Context, boxID, id domain.ID, studentsCreated, checkinsCreated int, completedAt time.Time) error {
	return databaseForContext(r.db, ctx).
		Model(&models.ImportHistoryModel{}).
		Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).
		Updates(map[string]any{
			"status":           string(domain.ImportStatusCompleted),
			"students_created": studentsCreated,
			"checkins_created": checkinsCreated,
			"completed_at":     completedAt,
			"error_code":       "",
		}).Error
}

func (r ImportHistoryGormRepository) MarkFailed(ctx context.Context, boxID, id domain.ID, errorCode string, completedAt time.Time) error {
	return databaseForContext(r.db, ctx).
		Model(&models.ImportHistoryModel{}).
		Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).
		Updates(map[string]any{
			"status":       string(domain.ImportStatusFailed),
			"completed_at": completedAt,
			"error_code":   errorCode,
		}).Error
}
