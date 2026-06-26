package repositories

import (
	"context"

	"gorm.io/gorm/clause"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
)

func (r WhatsappSettingsGormRepository) FindByBoxID(ctx context.Context, boxID domain.ID) (*domain.WhatsappSettings, error) {
	var model models.WhatsappSettingsModel
	if err := r.db.WithContext(ctx).Where("box_id = ?", stringID(boxID)).First(&model).Error; err != nil {
		return nil, err
	}

	settings := whatsappSettingsToDomain(model)
	return &settings, nil
}

func (r WhatsappSettingsGormRepository) Upsert(ctx context.Context, settings *domain.WhatsappSettings) error {
	if err := ensureID(&settings.ID); err != nil {
		return err
	}

	model := whatsappSettingsToModel(*settings)
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "box_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"provider",
			"base_url",
			"instance_name",
			"api_key_encrypted",
			"enabled",
			"updated_at",
		}),
	}).Create(&model).Error
}
