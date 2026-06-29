package repositories

import (
	"context"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
)

func (r EmailGormRepository) ListTemplates(ctx context.Context, boxID domain.ID) ([]domain.EmailTemplate, error) {
	var modelsList []models.EmailTemplateModel
	if err := r.db.WithContext(ctx).Where("box_id = ?", stringID(boxID)).Order("name ASC").Find(&modelsList).Error; err != nil {
		return nil, err
	}
	templates := make([]domain.EmailTemplate, 0, len(modelsList))
	for _, model := range modelsList {
		templates = append(templates, emailTemplateToDomain(model))
	}
	return templates, nil
}

func (r EmailGormRepository) FindTemplateByID(ctx context.Context, boxID, id domain.ID) (*domain.EmailTemplate, error) {
	var model models.EmailTemplateModel
	if err := r.db.WithContext(ctx).Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).First(&model).Error; err != nil {
		return nil, err
	}
	template := emailTemplateToDomain(model)
	return &template, nil
}

func (r EmailGormRepository) SaveTemplate(ctx context.Context, template *domain.EmailTemplate) error {
	if err := ensureID(&template.ID); err != nil {
		return err
	}
	model := emailTemplateToModel(*template)
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r EmailGormRepository) UpdateTemplate(ctx context.Context, template domain.EmailTemplate) error {
	model := emailTemplateToModel(template)
	return r.db.WithContext(ctx).Save(&model).Error
}

func (r EmailGormRepository) DeleteTemplate(ctx context.Context, boxID, id domain.ID) error {
	return r.db.WithContext(ctx).Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).Delete(&models.EmailTemplateModel{}).Error
}

func (r EmailGormRepository) ListCampaigns(ctx context.Context, boxID domain.ID) ([]domain.EmailCampaign, error) {
	var modelsList []models.EmailCampaignModel
	if err := r.db.WithContext(ctx).Where("box_id = ?", stringID(boxID)).Order("created_at DESC").Find(&modelsList).Error; err != nil {
		return nil, err
	}
	campaigns := make([]domain.EmailCampaign, 0, len(modelsList))
	for _, model := range modelsList {
		campaigns = append(campaigns, emailCampaignToDomain(model))
	}
	return campaigns, nil
}

func (r EmailGormRepository) FindCampaignByID(ctx context.Context, boxID, id domain.ID) (*domain.EmailCampaign, error) {
	var model models.EmailCampaignModel
	if err := r.db.WithContext(ctx).Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).First(&model).Error; err != nil {
		return nil, err
	}
	campaign := emailCampaignToDomain(model)
	return &campaign, nil
}

func (r EmailGormRepository) SaveCampaign(ctx context.Context, campaign *domain.EmailCampaign) error {
	if err := ensureID(&campaign.ID); err != nil {
		return err
	}
	model := emailCampaignToModel(*campaign)
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r EmailGormRepository) UpdateCampaign(ctx context.Context, campaign domain.EmailCampaign) error {
	model := emailCampaignToModel(campaign)
	return r.db.WithContext(ctx).Save(&model).Error
}

func (r EmailGormRepository) ListRecipients(ctx context.Context, emailCampaignID domain.ID) ([]domain.EmailRecipient, error) {
	var modelsList []models.EmailRecipientModel
	if err := r.db.WithContext(ctx).Where("email_campaign_id = ?", stringID(emailCampaignID)).Order("created_at ASC").Find(&modelsList).Error; err != nil {
		return nil, err
	}
	recipients := make([]domain.EmailRecipient, 0, len(modelsList))
	for _, model := range modelsList {
		recipients = append(recipients, emailRecipientToDomain(model))
	}
	return recipients, nil
}

func (r EmailGormRepository) SaveRecipients(ctx context.Context, recipients []domain.EmailRecipient) error {
	modelsList := make([]models.EmailRecipientModel, 0, len(recipients))
	for i := range recipients {
		if err := ensureID(&recipients[i].ID); err != nil {
			return err
		}
		modelsList = append(modelsList, emailRecipientToModel(recipients[i]))
	}
	if len(modelsList) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&modelsList).Error
}

func (r EmailGormRepository) UpdateRecipient(ctx context.Context, recipient domain.EmailRecipient) error {
	model := emailRecipientToModel(recipient)
	return r.db.WithContext(ctx).Save(&model).Error
}
