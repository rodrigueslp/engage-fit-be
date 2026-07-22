package repositories

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
)

func (r RewardGormRepository) ListByCampaign(ctx context.Context, boxID, campaignID domain.ID) ([]domain.Reward, error) {
	var modelsList []models.RewardModel
	if err := r.db.WithContext(ctx).
		Model(&models.RewardModel{}).
		Select(`
			rewards.*,
			COALESCE(SUM(CASE WHEN reward_deliveries.delivered = false THEN 1 ELSE 0 END), 0) AS pending_deliveries,
			COALESCE(SUM(CASE WHEN reward_deliveries.delivered = true THEN 1 ELSE 0 END), 0) AS delivered_deliveries
		`).
		Joins("LEFT JOIN reward_deliveries ON reward_deliveries.reward_id = rewards.id").
		Joins("JOIN campaigns ON campaigns.id = rewards.campaign_id").
		Where("campaigns.box_id = ? AND rewards.campaign_id = ?", stringID(boxID), stringID(campaignID)).
		Group("rewards.id").
		Order("rewards.name ASC").
		Find(&modelsList).Error; err != nil {
		return nil, err
	}

	rewards := make([]domain.Reward, 0, len(modelsList))
	for _, model := range modelsList {
		rewards = append(rewards, rewardToDomain(model))
	}
	return rewards, nil
}

func (r RewardGormRepository) FindByID(ctx context.Context, boxID, id domain.ID) (*domain.Reward, error) {
	var model models.RewardModel
	if err := r.db.WithContext(ctx).
		Select("rewards.*").
		Joins("JOIN campaigns ON campaigns.id = rewards.campaign_id").
		Where("campaigns.box_id = ? AND rewards.id = ?", stringID(boxID), stringID(id)).
		First(&model).Error; err != nil {
		return nil, err
	}

	reward := rewardToDomain(model)
	return &reward, nil
}

func (r RewardGormRepository) Save(ctx context.Context, reward *domain.Reward) error {
	if err := ensureID(&reward.ID); err != nil {
		return err
	}

	model := rewardToModel(*reward)
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r RewardGormRepository) Update(ctx context.Context, boxID domain.ID, reward domain.Reward) error {
	model := rewardToModel(reward)
	result := r.db.WithContext(ctx).
		Model(&models.RewardModel{}).
		Where("id = ? AND campaign_id IN (?)", stringID(reward.ID), r.campaignIDsForBox(boxID)).
		Updates(&model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r RewardGormRepository) Delete(ctx context.Context, boxID, id domain.ID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND campaign_id IN (?)", stringID(id), r.campaignIDsForBox(boxID)).
		Delete(&models.RewardModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r RewardGormRepository) campaignIDsForBox(boxID domain.ID) *gorm.DB {
	return r.db.Model(&models.CampaignModel{}).Select("id").Where("box_id = ?", stringID(boxID))
}

func (r RewardGormRepository) ListDeliveries(ctx context.Context, boxID domain.ID) ([]domain.RewardDelivery, error) {
	var modelsList []models.RewardDeliveryModel
	if err := r.deliveryQuery(ctx, boxID).
		Order("reward_deliveries.delivered ASC, reward_deliveries.delivered_at DESC NULLS LAST, students.name ASC").
		Find(&modelsList).Error; err != nil {
		return nil, err
	}

	deliveries := make([]domain.RewardDelivery, 0, len(modelsList))
	for _, model := range modelsList {
		deliveries = append(deliveries, rewardDeliveryToDomain(model))
	}
	return deliveries, nil
}

func (r RewardGormRepository) ListPendingDeliveries(ctx context.Context, boxID domain.ID) ([]domain.RewardDelivery, error) {
	var modelsList []models.RewardDeliveryModel
	if err := r.deliveryQuery(ctx, boxID).
		Where("reward_deliveries.delivered = ?", false).
		Order("reward_deliveries.id ASC").
		Find(&modelsList).Error; err != nil {
		return nil, err
	}

	deliveries := make([]domain.RewardDelivery, 0, len(modelsList))
	for _, model := range modelsList {
		deliveries = append(deliveries, rewardDeliveryToDomain(model))
	}
	return deliveries, nil
}

func (r RewardGormRepository) deliveryQuery(ctx context.Context, boxID domain.ID) *gorm.DB {
	return r.db.WithContext(ctx).
		Model(&models.RewardDeliveryModel{}).
		Select(`
			reward_deliveries.*,
			campaigns.id AS campaign_id,
			campaigns.name AS campaign_name,
			rewards.name AS reward_name,
			students.name AS student_name,
			students.phone AS student_phone
		`).
		Joins("JOIN rewards ON rewards.id = reward_deliveries.reward_id").
		Joins("JOIN campaigns ON campaigns.id = rewards.campaign_id").
		Joins("JOIN students ON students.id = reward_deliveries.student_id").
		Where("campaigns.box_id = ?", stringID(boxID))
}

func (r RewardGormRepository) CountDeliveries(ctx context.Context, boxID domain.ID, delivered bool) (int, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.RewardDeliveryModel{}).
		Joins("JOIN rewards ON rewards.id = reward_deliveries.reward_id").
		Joins("JOIN campaigns ON campaigns.id = rewards.campaign_id").
		Where("campaigns.box_id = ? AND reward_deliveries.delivered = ?", stringID(boxID), delivered).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (r RewardGormRepository) SyncPendingDeliveries(ctx context.Context, rewardID domain.ID, studentIDs []domain.ID) error {
	deliveries := make([]models.RewardDeliveryModel, 0, len(studentIDs))
	storedStudentIDs := make([]string, 0, len(studentIDs))
	for _, studentID := range studentIDs {
		id, err := newID()
		if err != nil {
			return err
		}
		deliveries = append(deliveries, models.RewardDeliveryModel{
			ID:        stringID(id),
			RewardID:  stringID(rewardID),
			StudentID: stringID(studentID),
			Delivered: false,
		})
		storedStudentIDs = append(storedStudentIDs, stringID(studentID))
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		stale := tx.Where("reward_id = ? AND delivered = ?", stringID(rewardID), false)
		if len(storedStudentIDs) > 0 {
			stale = stale.Where("student_id NOT IN ?", storedStudentIDs)
		}
		if err := stale.Delete(&models.RewardDeliveryModel{}).Error; err != nil {
			return err
		}
		if len(deliveries) == 0 {
			return nil
		}

		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "reward_id"}, {Name: "student_id"}},
			DoNothing: true,
		}).Create(&deliveries).Error
	})
}

func (r RewardGormRepository) MarkDelivered(ctx context.Context, boxID domain.ID, deliveryID domain.ID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.RewardDeliveryModel{}).
		Where("id = ? AND reward_id IN (?)",
			stringID(deliveryID),
			r.db.Model(&models.RewardModel{}).
				Select("rewards.id").
				Joins("JOIN campaigns ON campaigns.id = rewards.campaign_id").
				Where("campaigns.box_id = ?", stringID(boxID)),
		).
		Updates(map[string]any{"delivered": true, "delivered_at": now}).Error
}
