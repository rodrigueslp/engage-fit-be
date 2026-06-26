package repositories

import portrepo "boxengage/backend/internal/ports/repositories"

var (
	_ portrepo.BoxRepository              = BoxGormRepository{}
	_ portrepo.UserRepository             = UserGormRepository{}
	_ portrepo.StudentRepository          = StudentGormRepository{}
	_ portrepo.CheckinRepository          = CheckinGormRepository{}
	_ portrepo.ImportHistoryRepository    = ImportHistoryGormRepository{}
	_ portrepo.CampaignRepository         = CampaignGormRepository{}
	_ portrepo.RewardRepository           = RewardGormRepository{}
	_ portrepo.WhatsappSettingsRepository = WhatsappSettingsGormRepository{}
	_ portrepo.MessageRepository          = MessageGormRepository{}
)
