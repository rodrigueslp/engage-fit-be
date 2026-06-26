package repositories

import "gorm.io/gorm"

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
	db *gorm.DB
}

func NewWhatsappSettingsGormRepository(db *gorm.DB) WhatsappSettingsGormRepository {
	return WhatsappSettingsGormRepository{db: db}
}

type MessageGormRepository struct {
	db *gorm.DB
}

func NewMessageGormRepository(db *gorm.DB) MessageGormRepository {
	return MessageGormRepository{db: db}
}
