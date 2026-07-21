package repositories

import (
	"context"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
)

func (r MessageGormRepository) ListTemplates(ctx context.Context, boxID domain.ID) ([]domain.MessageTemplate, error) {
	var modelsList []models.MessageTemplateModel
	if err := r.db.WithContext(ctx).Where("box_id = ?", stringID(boxID)).Order("name ASC").Find(&modelsList).Error; err != nil {
		return nil, err
	}

	templates := make([]domain.MessageTemplate, 0, len(modelsList))
	for _, model := range modelsList {
		templates = append(templates, messageTemplateToDomain(model))
	}
	return templates, nil
}

func (r MessageGormRepository) FindTemplateByID(ctx context.Context, boxID, id domain.ID) (*domain.MessageTemplate, error) {
	var model models.MessageTemplateModel
	if err := r.db.WithContext(ctx).Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).First(&model).Error; err != nil {
		return nil, err
	}

	template := messageTemplateToDomain(model)
	return &template, nil
}

func (r MessageGormRepository) FindTemplateByType(ctx context.Context, boxID domain.ID, templateType domain.MessageTemplateType) (*domain.MessageTemplate, error) {
	var model models.MessageTemplateModel
	if err := r.db.WithContext(ctx).Where("box_id = ? AND template_type = ?", stringID(boxID), string(templateType)).First(&model).Error; err != nil {
		return nil, err
	}

	template := messageTemplateToDomain(model)
	return &template, nil
}

func (r MessageGormRepository) SaveTemplate(ctx context.Context, template *domain.MessageTemplate) error {
	if err := ensureID(&template.ID); err != nil {
		return err
	}

	model := messageTemplateToModel(*template)
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r MessageGormRepository) UpdateTemplate(ctx context.Context, template domain.MessageTemplate) error {
	model := messageTemplateToModel(template)
	return r.db.WithContext(ctx).Save(&model).Error
}

func (r MessageGormRepository) DeleteTemplate(ctx context.Context, boxID, id domain.ID) error {
	return r.db.WithContext(ctx).Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).Delete(&models.MessageTemplateModel{}).Error
}

func (r MessageGormRepository) ListCampaigns(ctx context.Context, boxID domain.ID) ([]domain.MessageCampaign, error) {
	var modelsList []models.MessageCampaignModel
	if err := r.db.WithContext(ctx).Where("box_id = ?", stringID(boxID)).Order("created_at DESC").Find(&modelsList).Error; err != nil {
		return nil, err
	}

	campaigns := make([]domain.MessageCampaign, 0, len(modelsList))
	for _, model := range modelsList {
		campaigns = append(campaigns, messageCampaignToDomain(model))
	}
	return campaigns, nil
}

func (r MessageGormRepository) FindCampaignByID(ctx context.Context, boxID, id domain.ID) (*domain.MessageCampaign, error) {
	var model models.MessageCampaignModel
	if err := r.db.WithContext(ctx).Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).First(&model).Error; err != nil {
		return nil, err
	}

	campaign := messageCampaignToDomain(model)
	return &campaign, nil
}

func (r MessageGormRepository) SaveCampaign(ctx context.Context, campaign *domain.MessageCampaign) error {
	if err := ensureID(&campaign.ID); err != nil {
		return err
	}

	model := messageCampaignToModel(*campaign)
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r MessageGormRepository) UpdateCampaign(ctx context.Context, campaign domain.MessageCampaign) error {
	model := messageCampaignToModel(campaign)
	return r.db.WithContext(ctx).Save(&model).Error
}

func (r MessageGormRepository) ListRecipients(ctx context.Context, messageCampaignID domain.ID) ([]domain.MessageRecipient, error) {
	var modelsList []models.MessageRecipientModel
	if err := r.db.WithContext(ctx).Where("message_campaign_id = ?", stringID(messageCampaignID)).Order("created_at ASC").Find(&modelsList).Error; err != nil {
		return nil, err
	}

	recipients := make([]domain.MessageRecipient, 0, len(modelsList))
	for _, model := range modelsList {
		recipients = append(recipients, messageRecipientToDomain(model))
	}
	return recipients, nil
}

func (r MessageGormRepository) SaveRecipients(ctx context.Context, recipients []domain.MessageRecipient) error {
	modelsList := make([]models.MessageRecipientModel, 0, len(recipients))
	for i := range recipients {
		if err := ensureID(&recipients[i].ID); err != nil {
			return err
		}
		modelsList = append(modelsList, messageRecipientToModel(recipients[i]))
	}

	if len(modelsList) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Create(&modelsList).Error
}

func (r MessageGormRepository) UpdateRecipient(ctx context.Context, recipient domain.MessageRecipient) error {
	model := messageRecipientToModel(recipient)
	return r.db.WithContext(ctx).Save(&model).Error
}
