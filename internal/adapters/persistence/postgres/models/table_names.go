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

func (EmailSettingsModel) TableName() string {
	return "email_settings"
}

func (EmailTemplateModel) TableName() string {
	return "email_templates"
}

func (EmailCampaignModel) TableName() string {
	return "email_campaigns"
}

func (EmailRecipientModel) TableName() string {
	return "email_recipients"
}

func (AutomationRunModel) TableName() string {
	return "automation_runs"
}

func (AutomationScheduleModel) TableName() string {
	return "automation_schedules"
}

func (WorkoutModel) TableName() string {
	return "workouts"
}

func (WorkoutMessageDraftModel) TableName() string {
	return "workout_message_drafts"
}

func (WorkoutMessageRecipientModel) TableName() string {
	return "workout_message_recipients"
}

func (LLMGenerationLogModel) TableName() string {
	return "llm_generation_logs"
}
