package repositories

import (
	"context"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
)

func (r ImportHistoryGormRepository) FindByID(ctx context.Context, boxID, id domain.ID) (*domain.ImportHistory, error) {
	var model models.ImportHistoryModel
	if err := r.db.WithContext(ctx).Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).First(&model).Error; err != nil {
		return nil, err
	}

	importHistory := importHistoryToDomain(model)
	return &importHistory, nil
}

func (r ImportHistoryGormRepository) List(ctx context.Context, boxID domain.ID) ([]domain.ImportHistory, error) {
	var modelsList []models.ImportHistoryModel
	if err := r.db.WithContext(ctx).Where("box_id = ?", stringID(boxID)).Order("imported_at DESC").Find(&modelsList).Error; err != nil {
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

	model := importHistoryToModel(*importHistory)
	return r.db.WithContext(ctx).Save(&model).Error
}
