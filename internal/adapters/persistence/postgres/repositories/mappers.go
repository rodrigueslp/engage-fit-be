package repositories

import (
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
)

func boxToDomain(model models.BoxModel) domain.Box {
	riskInactiveDays := model.RiskInactiveDays
	if riskInactiveDays <= 0 {
		riskInactiveDays = 7
	}
	riskMessageCooldownDays := model.RiskMessageCooldownDays
	if riskMessageCooldownDays <= 0 {
		riskMessageCooldownDays = 14
	}
	return domain.Box{
		ID:                      domainID(model.ID),
		Name:                    model.Name,
		RiskInactiveDays:        riskInactiveDays,
		RiskMessageCooldownDays: riskMessageCooldownDays,
		CreatedAt:               model.CreatedAt,
		UpdatedAt:               model.UpdatedAt,
	}
}

func boxToModel(box domain.Box) models.BoxModel {
	riskInactiveDays := box.RiskInactiveDays
	if riskInactiveDays <= 0 {
		riskInactiveDays = 7
	}
	riskMessageCooldownDays := box.RiskMessageCooldownDays
	if riskMessageCooldownDays <= 0 {
		riskMessageCooldownDays = 14
	}
	return models.BoxModel{
		ID:                      stringID(box.ID),
		Name:                    box.Name,
		RiskInactiveDays:        riskInactiveDays,
		RiskMessageCooldownDays: riskMessageCooldownDays,
		CreatedAt:               box.CreatedAt,
		UpdatedAt:               box.UpdatedAt,
	}
}

func userToDomain(model models.UserModel) domain.User {
	return domain.User{
		ID:           domainID(model.ID),
		BoxID:        domainID(model.BoxID),
		Name:         model.Name,
		Email:        model.Email,
		PasswordHash: model.PasswordHash,
		Role:         domain.UserRole(model.Role),
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}

func studentToDomain(model models.StudentModel) domain.Student {
	riskStatus := domain.StudentRiskStatus(model.RiskStatus)
	if riskStatus == "" {
		riskStatus = domain.StudentRiskStatusActive
	}
	return domain.Student{
		ID:                domainID(model.ID),
		BoxID:             domainID(model.BoxID),
		Name:              model.Name,
		Email:             model.Email,
		Phone:             model.Phone,
		Source:            domain.Source(model.Source),
		ExternalID:        model.ExternalID,
		RiskStatus:        riskStatus,
		RiskLastMessageAt: model.RiskLastMessageAt,
		CreatedAt:         model.CreatedAt,
		UpdatedAt:         model.UpdatedAt,
	}
}

func studentToModel(student domain.Student) models.StudentModel {
	riskStatus := student.RiskStatus
	if riskStatus == "" {
		riskStatus = domain.StudentRiskStatusActive
	}
	return models.StudentModel{
		ID:                stringID(student.ID),
		BoxID:             stringID(student.BoxID),
		Name:              student.Name,
		Email:             student.Email,
		Phone:             student.Phone,
		Source:            string(student.Source),
		ExternalID:        student.ExternalID,
		RiskStatus:        string(riskStatus),
		RiskLastMessageAt: student.RiskLastMessageAt,
		CreatedAt:         student.CreatedAt,
		UpdatedAt:         student.UpdatedAt,
	}
}

func checkinToDomain(model models.CheckinModel) domain.Checkin {
	return domain.Checkin{
		ID:              domainID(model.ID),
		BoxID:           domainID(model.BoxID),
		StudentID:       domainID(model.StudentID),
		CheckinDate:     model.CheckinDate,
		CheckinTime:     parseCheckinTime(model.CheckinTime),
		Source:          domain.Source(model.Source),
		ImportHistoryID: domainID(model.ImportHistoryID),
		CreatedAt:       model.CreatedAt,
	}
}

func checkinToModel(checkin domain.Checkin) models.CheckinModel {
	return models.CheckinModel{
		ID:              stringID(checkin.ID),
		BoxID:           stringID(checkin.BoxID),
		StudentID:       stringID(checkin.StudentID),
		CheckinDate:     checkin.CheckinDate,
		CheckinTime:     formatCheckinTime(checkin.CheckinTime),
		Source:          string(checkin.Source),
		ImportHistoryID: stringID(checkin.ImportHistoryID),
		CreatedAt:       checkin.CreatedAt,
	}
}

func parseCheckinTime(value *string) *time.Time {
	if value == nil || *value == "" {
		return nil
	}
	parsed, err := time.Parse("15:04:05", *value)
	if err != nil {
		return nil
	}
	normalized := time.Date(1970, 1, 1, parsed.Hour(), parsed.Minute(), parsed.Second(), 0, time.UTC)
	return &normalized
}

func formatCheckinTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format("15:04:05")
	return &formatted
}

func importHistoryToDomain(model models.ImportHistoryModel) domain.ImportHistory {
	return domain.ImportHistory{
		ID:           domainID(model.ID),
		BoxID:        domainID(model.BoxID),
		Filename:     model.Filename,
		Source:       domain.Source(model.Source),
		TotalRecords: model.TotalRecords,
		ImportedAt:   model.ImportedAt,
	}
}

func importHistoryToModel(importHistory domain.ImportHistory) models.ImportHistoryModel {
	return models.ImportHistoryModel{
		ID:           stringID(importHistory.ID),
		BoxID:        stringID(importHistory.BoxID),
		Filename:     importHistory.Filename,
		Source:       string(importHistory.Source),
		TotalRecords: importHistory.TotalRecords,
		ImportedAt:   importHistory.ImportedAt,
	}
}

func campaignToDomain(model models.CampaignModel) domain.Campaign {
	return domain.Campaign{
		ID:          domainID(model.ID),
		BoxID:       domainID(model.BoxID),
		Name:        model.Name,
		Description: model.Description,
		StartDate:   model.StartDate,
		EndDate:     model.EndDate,
		Active:      model.Active,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}

func campaignToModel(campaign domain.Campaign) models.CampaignModel {
	return models.CampaignModel{
		ID:          stringID(campaign.ID),
		BoxID:       stringID(campaign.BoxID),
		Name:        campaign.Name,
		Description: campaign.Description,
		StartDate:   campaign.StartDate,
		EndDate:     campaign.EndDate,
		Active:      campaign.Active,
		CreatedAt:   campaign.CreatedAt,
		UpdatedAt:   campaign.UpdatedAt,
	}
}

func campaignGoalToDomain(model models.CampaignGoalModel) domain.CampaignGoal {
	return domain.CampaignGoal{ID: domainID(model.ID), CampaignID: domainID(model.CampaignID), Source: domain.Source(model.Source), TargetCheckins: model.TargetCheckins}
}

func campaignGoalToModel(goal domain.CampaignGoal) models.CampaignGoalModel {
	return models.CampaignGoalModel{ID: stringID(goal.ID), CampaignID: stringID(goal.CampaignID), Source: string(goal.Source), TargetCheckins: goal.TargetCheckins}
}

func campaignProgressToDomain(model models.CampaignProgressModel) domain.CampaignProgress {
	return domain.CampaignProgress{
		ID:                 domainID(model.ID),
		CampaignID:         domainID(model.CampaignID),
		StudentID:          domainID(model.StudentID),
		CurrentCheckins:    model.CurrentCheckins,
		TargetCheckins:     model.TargetCheckins,
		ProgressPercentage: model.ProgressPercentage,
		Achieved:           model.Achieved,
		NearGoal:           model.NearGoal,
		UpdatedAt:          model.UpdatedAt,
	}
}

func campaignProgressToModel(progress domain.CampaignProgress) models.CampaignProgressModel {
	return models.CampaignProgressModel{
		ID:                 stringID(progress.ID),
		CampaignID:         stringID(progress.CampaignID),
		StudentID:          stringID(progress.StudentID),
		CurrentCheckins:    progress.CurrentCheckins,
		TargetCheckins:     progress.TargetCheckins,
		ProgressPercentage: progress.ProgressPercentage,
		Achieved:           progress.Achieved,
		NearGoal:           progress.NearGoal,
		UpdatedAt:          progress.UpdatedAt,
	}
}

func rewardToDomain(model models.RewardModel) domain.Reward {
	return domain.Reward{
		ID:                  domainID(model.ID),
		CampaignID:          domainID(model.CampaignID),
		Name:                model.Name,
		Description:         model.Description,
		Quantity:            model.Quantity,
		PendingDeliveries:   model.PendingDeliveries,
		DeliveredDeliveries: model.DeliveredDeliveries,
	}
}

func rewardToModel(reward domain.Reward) models.RewardModel {
	return models.RewardModel{ID: stringID(reward.ID), CampaignID: stringID(reward.CampaignID), Name: reward.Name, Description: reward.Description, Quantity: reward.Quantity}
}

func rewardDeliveryToDomain(model models.RewardDeliveryModel) domain.RewardDelivery {
	return domain.RewardDelivery{
		ID:           domainID(model.ID),
		CampaignID:   domainID(model.CampaignID),
		CampaignName: model.CampaignName,
		RewardID:     domainID(model.RewardID),
		RewardName:   model.RewardName,
		StudentID:    domainID(model.StudentID),
		StudentName:  model.StudentName,
		StudentPhone: model.StudentPhone,
		Delivered:    model.Delivered,
		DeliveredAt:  model.DeliveredAt,
	}
}

func whatsappSettingsToDomain(model models.WhatsappSettingsModel) domain.WhatsappSettings {
	return domain.WhatsappSettings{
		ID:              domainID(model.ID),
		BoxID:           domainID(model.BoxID),
		Provider:        model.Provider,
		BaseURL:         model.BaseURL,
		InstanceName:    model.InstanceName,
		APIKeyEncrypted: model.APIKeyEncrypted,
		Enabled:         model.Enabled,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}
}

func whatsappSettingsToModel(settings domain.WhatsappSettings) models.WhatsappSettingsModel {
	return models.WhatsappSettingsModel{
		ID:              stringID(settings.ID),
		BoxID:           stringID(settings.BoxID),
		Provider:        settings.Provider,
		BaseURL:         settings.BaseURL,
		InstanceName:    settings.InstanceName,
		APIKeyEncrypted: settings.APIKeyEncrypted,
		Enabled:         settings.Enabled,
		CreatedAt:       settings.CreatedAt,
		UpdatedAt:       settings.UpdatedAt,
	}
}

func messageTemplateToDomain(model models.MessageTemplateModel) domain.MessageTemplate {
	return domain.MessageTemplate{ID: domainID(model.ID), BoxID: domainID(model.BoxID), Name: model.Name, Content: model.Content, ContentSID: model.ContentSID, CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt}
}

func messageTemplateToModel(template domain.MessageTemplate) models.MessageTemplateModel {
	return models.MessageTemplateModel{ID: stringID(template.ID), BoxID: stringID(template.BoxID), Name: template.Name, Content: template.Content, ContentSID: template.ContentSID, CreatedAt: template.CreatedAt, UpdatedAt: template.UpdatedAt}
}

func messageCampaignToDomain(model models.MessageCampaignModel) domain.MessageCampaign {
	return domain.MessageCampaign{ID: domainID(model.ID), BoxID: domainID(model.BoxID), CampaignID: domainID(model.CampaignID), Name: model.Name, Audience: domain.MessageAudience(model.Audience), TemplateID: domainID(model.TemplateID), SentAt: model.SentAt, CreatedAt: model.CreatedAt}
}

func messageCampaignToModel(campaign domain.MessageCampaign) models.MessageCampaignModel {
	return models.MessageCampaignModel{ID: stringID(campaign.ID), BoxID: stringID(campaign.BoxID), CampaignID: stringID(campaign.CampaignID), Name: campaign.Name, Audience: string(campaign.Audience), TemplateID: stringID(campaign.TemplateID), SentAt: campaign.SentAt, CreatedAt: campaign.CreatedAt}
}

func messageRecipientToDomain(model models.MessageRecipientModel) domain.MessageRecipient {
	return domain.MessageRecipient{
		ID:                domainID(model.ID),
		MessageCampaignID: domainID(model.MessageCampaignID),
		StudentID:         domainID(model.StudentID),
		Phone:             model.Phone,
		Status:            domain.MessageRecipientStatus(model.Status),
		ErrorMessage:      model.ErrorMessage,
		SentAt:            model.SentAt,
		CreatedAt:         model.CreatedAt,
	}
}

func messageRecipientToModel(recipient domain.MessageRecipient) models.MessageRecipientModel {
	return models.MessageRecipientModel{
		ID:                stringID(recipient.ID),
		MessageCampaignID: stringID(recipient.MessageCampaignID),
		StudentID:         stringID(recipient.StudentID),
		Phone:             recipient.Phone,
		Status:            string(recipient.Status),
		ErrorMessage:      recipient.ErrorMessage,
		SentAt:            recipient.SentAt,
		CreatedAt:         recipient.CreatedAt,
	}
}

func emailSettingsToDomain(model models.EmailSettingsModel) domain.EmailSettings {
	return domain.EmailSettings{
		ID:                domainID(model.ID),
		BoxID:             domainID(model.BoxID),
		Provider:          model.Provider,
		SMTPHost:          model.SMTPHost,
		SMTPPort:          model.SMTPPort,
		Username:          model.Username,
		PasswordEncrypted: model.PasswordEncrypted,
		FromEmail:         model.FromEmail,
		FromName:          model.FromName,
		Enabled:           model.Enabled,
		CreatedAt:         model.CreatedAt,
		UpdatedAt:         model.UpdatedAt,
	}
}

func emailSettingsToModel(settings domain.EmailSettings) models.EmailSettingsModel {
	return models.EmailSettingsModel{
		ID:                stringID(settings.ID),
		BoxID:             stringID(settings.BoxID),
		Provider:          settings.Provider,
		SMTPHost:          settings.SMTPHost,
		SMTPPort:          settings.SMTPPort,
		Username:          settings.Username,
		PasswordEncrypted: settings.PasswordEncrypted,
		FromEmail:         settings.FromEmail,
		FromName:          settings.FromName,
		Enabled:           settings.Enabled,
		CreatedAt:         settings.CreatedAt,
		UpdatedAt:         settings.UpdatedAt,
	}
}

func emailTemplateToDomain(model models.EmailTemplateModel) domain.EmailTemplate {
	return domain.EmailTemplate{ID: domainID(model.ID), BoxID: domainID(model.BoxID), Name: model.Name, Subject: model.Subject, Content: model.Content, CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt}
}

func emailTemplateToModel(template domain.EmailTemplate) models.EmailTemplateModel {
	return models.EmailTemplateModel{ID: stringID(template.ID), BoxID: stringID(template.BoxID), Name: template.Name, Subject: template.Subject, Content: template.Content, CreatedAt: template.CreatedAt, UpdatedAt: template.UpdatedAt}
}

func emailCampaignToDomain(model models.EmailCampaignModel) domain.EmailCampaign {
	return domain.EmailCampaign{ID: domainID(model.ID), BoxID: domainID(model.BoxID), CampaignID: domainID(model.CampaignID), Name: model.Name, Audience: domain.MessageAudience(model.Audience), TemplateID: domainID(model.TemplateID), SentAt: model.SentAt, CreatedAt: model.CreatedAt}
}

func emailCampaignToModel(campaign domain.EmailCampaign) models.EmailCampaignModel {
	return models.EmailCampaignModel{ID: stringID(campaign.ID), BoxID: stringID(campaign.BoxID), CampaignID: stringID(campaign.CampaignID), Name: campaign.Name, Audience: string(campaign.Audience), TemplateID: stringID(campaign.TemplateID), SentAt: campaign.SentAt, CreatedAt: campaign.CreatedAt}
}

func emailRecipientToDomain(model models.EmailRecipientModel) domain.EmailRecipient {
	return domain.EmailRecipient{
		ID:              domainID(model.ID),
		EmailCampaignID: domainID(model.EmailCampaignID),
		StudentID:       domainID(model.StudentID),
		Email:           model.Email,
		Status:          domain.MessageRecipientStatus(model.Status),
		ErrorMessage:    model.ErrorMessage,
		SentAt:          model.SentAt,
		CreatedAt:       model.CreatedAt,
	}
}

func emailRecipientToModel(recipient domain.EmailRecipient) models.EmailRecipientModel {
	return models.EmailRecipientModel{
		ID:              stringID(recipient.ID),
		EmailCampaignID: stringID(recipient.EmailCampaignID),
		StudentID:       stringID(recipient.StudentID),
		Email:           recipient.Email,
		Status:          string(recipient.Status),
		ErrorMessage:    recipient.ErrorMessage,
		SentAt:          recipient.SentAt,
		CreatedAt:       recipient.CreatedAt,
	}
}

func automationRunToDomain(model models.AutomationRunModel) domain.AutomationRun {
	return domain.AutomationRun{
		ID:                      domainID(model.ID),
		BoxID:                   domainID(model.BoxID),
		Status:                  model.Status,
		Source:                  model.Source,
		Filename:                model.Filename,
		Imported:                model.Imported,
		RecalculatedCampaigns:   model.RecalculatedCampaigns,
		SkippedMessageCampaigns: model.SkippedMessageCampaigns,
		SentMessages:            model.SentMessages,
		FailedMessages:          model.FailedMessages,
		ErrorMessage:            model.ErrorMessage,
		StartedAt:               model.StartedAt,
		FinishedAt:              model.FinishedAt,
	}
}

func automationRunToModel(run domain.AutomationRun) models.AutomationRunModel {
	return models.AutomationRunModel{
		ID:                      stringID(run.ID),
		BoxID:                   stringID(run.BoxID),
		Status:                  run.Status,
		Source:                  run.Source,
		Filename:                run.Filename,
		Imported:                run.Imported,
		RecalculatedCampaigns:   run.RecalculatedCampaigns,
		SkippedMessageCampaigns: run.SkippedMessageCampaigns,
		SentMessages:            run.SentMessages,
		FailedMessages:          run.FailedMessages,
		ErrorMessage:            run.ErrorMessage,
		StartedAt:               run.StartedAt,
		FinishedAt:              run.FinishedAt,
	}
}

func automationScheduleToDomain(model models.AutomationScheduleModel) domain.AutomationSchedule {
	return domain.AutomationSchedule{
		ID:          domainID(model.ID),
		BoxID:       domainID(model.BoxID),
		Name:        model.Name,
		Mode:        model.Mode,
		RunTime:     model.RunTime,
		Timezone:    model.Timezone,
		DaysOfWeek:  model.DaysOfWeek,
		AllowResend: model.AllowResend,
		Enabled:     model.Enabled,
		LastRunAt:   model.LastRunAt,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}

func automationScheduleToModel(schedule domain.AutomationSchedule) models.AutomationScheduleModel {
	return models.AutomationScheduleModel{
		ID:          stringID(schedule.ID),
		BoxID:       stringID(schedule.BoxID),
		Name:        schedule.Name,
		Mode:        schedule.Mode,
		RunTime:     schedule.RunTime,
		Timezone:    schedule.Timezone,
		DaysOfWeek:  schedule.DaysOfWeek,
		AllowResend: schedule.AllowResend,
		Enabled:     schedule.Enabled,
		LastRunAt:   schedule.LastRunAt,
		CreatedAt:   schedule.CreatedAt,
		UpdatedAt:   schedule.UpdatedAt,
	}
}
