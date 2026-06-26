package models

func (BoxModel) TableName() string {
	return "boxes"
}

func (UserModel) TableName() string {
	return "users"
}

func (StudentModel) TableName() string {
	return "students"
}

func (ImportHistoryModel) TableName() string {
	return "import_histories"
}

func (CheckinModel) TableName() string {
	return "checkins"
}

func (CampaignModel) TableName() string {
	return "campaigns"
}

func (CampaignGoalModel) TableName() string {
	return "campaign_goals"
}

func (CampaignProgressModel) TableName() string {
	return "campaign_progresses"
}

func (RewardModel) TableName() string {
	return "rewards"
}

func (RewardDeliveryModel) TableName() string {
	return "reward_deliveries"
}

func (WhatsappSettingsModel) TableName() string {
	return "whatsapp_settings"
}

func (MessageTemplateModel) TableName() string {
	return "message_templates"
}

func (MessageCampaignModel) TableName() string {
	return "message_campaigns"
}

func (MessageRecipientModel) TableName() string {
	return "message_recipients"
}
