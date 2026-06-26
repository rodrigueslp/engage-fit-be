package repositories

import (
	"context"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
)

func (r BoxGormRepository) FindByID(ctx context.Context, id domain.ID) (*domain.Box, error) {
	var model models.BoxModel
	if err := r.db.WithContext(ctx).Where("id = ?", stringID(id)).First(&model).Error; err != nil {
		return nil, err
	}

	box := boxToDomain(model)
	return &box, nil
}

func (r BoxGormRepository) Save(ctx context.Context, box *domain.Box) error {
	if err := ensureID(&box.ID); err != nil {
		return err
	}

	model := boxToModel(*box)
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r BoxGormRepository) Update(ctx context.Context, box domain.Box) error {
	model := boxToModel(box)
	return r.db.WithContext(ctx).Save(&model).Error
}
