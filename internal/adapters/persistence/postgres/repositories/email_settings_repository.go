package repositories

import (
	"context"

	"gorm.io/gorm/clause"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
)

func (r EmailSettingsGormRepository) FindByBoxID(ctx context.Context, boxID domain.ID) (*domain.EmailSettings, error) {
	var model models.EmailSettingsModel
	if err := r.db.WithContext(ctx).Where("box_id = ?", stringID(boxID)).First(&model).Error; err != nil {
		return nil, err
	}
	settings := emailSettingsToDomain(model)
	return &settings, nil
}

func (r EmailSettingsGormRepository) Upsert(ctx context.Context, settings *domain.EmailSettings) error {
	if err := ensureID(&settings.ID); err != nil {
		return err
	}
	model := emailSettingsToModel(*settings)
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "box_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"provider", "smtp_host", "smtp_port", "username", "password_encrypted", "from_email", "from_name", "enabled", "updated_at",
		}),
	}).Create(&model).Error
}
