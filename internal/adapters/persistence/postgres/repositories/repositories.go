package repositories

import (
	"boxengage/backend/internal/ports/services"
	"gorm.io/gorm"
)

type BoxGormRepository struct {
	db *gorm.DB
}

func NewBoxGormRepository(db *gorm.DB) BoxGormRepository {
	return BoxGormRepository{db: db}
}

type UserGormRepository struct {
	db *gorm.DB
}

func NewUserGormRepository(db *gorm.DB) UserGormRepository {
	return UserGormRepository{db: db}
}

type StudentGormRepository struct {
	db *gorm.DB
}

type PrivacyGormRepository struct {
	db *gorm.DB
}

func NewPrivacyGormRepository(db *gorm.DB) PrivacyGormRepository {
	return PrivacyGormRepository{db: db}
}

func NewStudentGormRepository(db *gorm.DB) StudentGormRepository {
	return StudentGormRepository{db: db}
}

type CheckinGormRepository struct {
	db *gorm.DB
}

func NewCheckinGormRepository(db *gorm.DB) CheckinGormRepository {
	return CheckinGormRepository{db: db}
}

type ImportHistoryGormRepository struct {
	db *gorm.DB
}

func NewImportHistoryGormRepository(db *gorm.DB) ImportHistoryGormRepository {
	return ImportHistoryGormRepository{db: db}
}

type CampaignGormRepository struct {
	db *gorm.DB
}

func NewCampaignGormRepository(db *gorm.DB) CampaignGormRepository {
	return CampaignGormRepository{db: db}
}

type RewardGormRepository struct {
	db *gorm.DB
}

func NewRewardGormRepository(db *gorm.DB) RewardGormRepository {
	return RewardGormRepository{db: db}
}

type WhatsappSettingsGormRepository struct {
	db     *gorm.DB
	cipher services.SecretCipher
}

func NewWhatsappSettingsGormRepository(db *gorm.DB, cipher services.SecretCipher) WhatsappSettingsGormRepository {
	return WhatsappSettingsGormRepository{db: db, cipher: cipher}
}

type MessageGormRepository struct {
	db *gorm.DB
}

func NewMessageGormRepository(db *gorm.DB) MessageGormRepository {
	return MessageGormRepository{db: db}
}

type EmailSettingsGormRepository struct {
	db     *gorm.DB
	cipher services.SecretCipher
}

func NewEmailSettingsGormRepository(db *gorm.DB, cipher services.SecretCipher) EmailSettingsGormRepository {
	return EmailSettingsGormRepository{db: db, cipher: cipher}
}

type EmailGormRepository struct {
	db *gorm.DB
}

func NewEmailGormRepository(db *gorm.DB) EmailGormRepository {
	return EmailGormRepository{db: db}
}

type AutomationGormRepository struct {
	db *gorm.DB
}

func NewAutomationGormRepository(db *gorm.DB) AutomationGormRepository {
	return AutomationGormRepository{db: db}
}

type WorkoutGormRepository struct {
	db *gorm.DB
}

type MessagingGovernanceGormRepository struct {
	db *gorm.DB
}

func NewMessagingGovernanceGormRepository(db *gorm.DB) MessagingGovernanceGormRepository {
	return MessagingGovernanceGormRepository{db: db}
}

func NewWorkoutGormRepository(db *gorm.DB) WorkoutGormRepository {
	return WorkoutGormRepository{db: db}
}
