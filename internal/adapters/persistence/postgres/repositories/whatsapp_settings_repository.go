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
	decrypted, err := r.cipher.Decrypt(settings.APIKeyEncrypted, whatsappSecretAAD(settings.BoxID))
	if err != nil {
		return nil, err
	}
	settings.APIKeyEncrypted = decrypted
	return &settings, nil
}

func (r WhatsappSettingsGormRepository) Upsert(ctx context.Context, settings *domain.WhatsappSettings) error {
	if err := ensureID(&settings.ID); err != nil {
		return err
	}

	persisted := *settings
	var err error
	persisted.APIKeyEncrypted, err = r.cipher.Encrypt(settings.APIKeyEncrypted, whatsappSecretAAD(settings.BoxID))
	if err != nil {
		return err
	}
	model := whatsappSettingsToModel(persisted)
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "box_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"connection_mode",
			"provider",
			"base_url",
			"instance_name",
			"api_key_encrypted",
			"enabled",
			"updated_at",
		}),
	}).Create(&model).Error
}

func whatsappSecretAAD(boxID domain.ID) string {
	return "whatsapp_settings:" + string(boxID) + ":api_key"
}
