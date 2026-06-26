package repositories

import (
	"context"

	"gorm.io/gorm/clause"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
)

func (r CampaignGormRepository) FindByID(ctx context.Context, boxID, id domain.ID) (*domain.Campaign, error) {
	var model models.CampaignModel
	if err := r.db.WithContext(ctx).Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).First(&model).Error; err != nil {
		return nil, err
	}

	campaign := campaignToDomain(model)
	return &campaign, nil
}

func (r CampaignGormRepository) List(ctx context.Context, boxID domain.ID) ([]domain.Campaign, error) {
	var modelsList []models.CampaignModel
	if err := r.db.WithContext(ctx).Where("box_id = ?", stringID(boxID)).Order("created_at DESC").Find(&modelsList).Error; err != nil {
		return nil, err
	}

	return campaignsToDomain(modelsList), nil
}

func (r CampaignGormRepository) ListActive(ctx context.Context, boxID domain.ID) ([]domain.Campaign, error) {
	var modelsList []models.CampaignModel
	if err := r.db.WithContext(ctx).Where("box_id = ? AND active = ?", stringID(boxID), true).Order("end_date ASC").Find(&modelsList).Error; err != nil {
		return nil, err
	}

	return campaignsToDomain(modelsList), nil
}

func (r CampaignGormRepository) Save(ctx context.Context, campaign *domain.Campaign) error {
	if err := ensureID(&campaign.ID); err != nil {
		return err
	}

	model := campaignToModel(*campaign)
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r CampaignGormRepository) Update(ctx context.Context, campaign domain.Campaign) error {
	model := campaignToModel(campaign)
	return r.db.WithContext(ctx).Save(&model).Error
}

func (r CampaignGormRepository) Delete(ctx context.Context, boxID, id domain.ID) error {
	return r.db.WithContext(ctx).Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).Delete(&models.CampaignModel{}).Error
}

func (r CampaignGormRepository) ListGoals(ctx context.Context, campaignID domain.ID) ([]domain.CampaignGoal, error) {
	var modelsList []models.CampaignGoalModel
	if err := r.db.WithContext(ctx).Where("campaign_id = ?", stringID(campaignID)).Order("source ASC").Find(&modelsList).Error; err != nil {
		return nil, err
	}

	goals := make([]domain.CampaignGoal, 0, len(modelsList))
	for _, model := range modelsList {
		goals = append(goals, campaignGoalToDomain(model))
	}
	return goals, nil
}

func (r CampaignGormRepository) UpsertGoal(ctx context.Context, goal *domain.CampaignGoal) error {
	if err := ensureID(&goal.ID); err != nil {
		return err
	}

	model := campaignGoalToModel(*goal)
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "campaign_id"}, {Name: "source"}},
		DoUpdates: clause.AssignmentColumns([]string{"target_checkins"}),
	}).Create(&model).Error
}

func (r CampaignGormRepository) DeleteGoal(ctx context.Context, campaignID, goalID domain.ID) error {
	return r.db.WithContext(ctx).
		Where("campaign_id = ? AND id = ?", stringID(campaignID), stringID(goalID)).
		Delete(&models.CampaignGoalModel{}).Error
}

func (r CampaignGormRepository) ListProgress(ctx context.Context, campaignID domain.ID) ([]domain.CampaignProgress, error) {
	var modelsList []models.CampaignProgressModel
	if err := r.db.WithContext(ctx).Where("campaign_id = ?", stringID(campaignID)).Order("progress_percentage DESC").Find(&modelsList).Error; err != nil {
		return nil, err
	}

	progress := make([]domain.CampaignProgress, 0, len(modelsList))
	for _, model := range modelsList {
		progress = append(progress, campaignProgressToDomain(model))
	}
	return progress, nil
}

func (r CampaignGormRepository) ListEligibleReportRows(ctx context.Context, boxID domain.ID) ([]domain.EligibleStudentReportRow, error) {
	type eligibleReportRow struct {
		CampaignID         string
		CampaignName       string
		StudentID          string
		StudentName        string
		StudentPhone       string
		Source             string
		CurrentCheckins    int
		TargetCheckins     int
		RemainingCheckins  int
		ProgressPercentage float64
		RewardName         string
	}

	var rows []eligibleReportRow
	if err := r.db.WithContext(ctx).
		Model(&models.CampaignProgressModel{}).
		Select(`
			campaigns.id AS campaign_id,
			campaigns.name AS campaign_name,
			students.id AS student_id,
			students.name AS student_name,
			students.phone AS student_phone,
			students.source AS source,
			campaign_progresses.current_checkins AS current_checkins,
			campaign_progresses.target_checkins AS target_checkins,
			GREATEST(campaign_progresses.target_checkins - campaign_progresses.current_checkins, 0) AS remaining_checkins,
			campaign_progresses.progress_percentage AS progress_percentage,
			COALESCE(STRING_AGG(DISTINCT rewards.name, ', '), '') AS reward_name
		`).
		Joins("JOIN campaigns ON campaigns.id = campaign_progresses.campaign_id").
		Joins("JOIN students ON students.id = campaign_progresses.student_id").
		Joins("LEFT JOIN rewards ON rewards.campaign_id = campaigns.id").
		Where("campaigns.box_id = ? AND campaign_progresses.achieved = ?", stringID(boxID), true).
		Group("campaigns.id, campaigns.name, students.id, students.name, students.phone, students.source, campaign_progresses.current_checkins, campaign_progresses.target_checkins, campaign_progresses.progress_percentage").
		Order("campaigns.end_date DESC, students.name ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]domain.EligibleStudentReportRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.EligibleStudentReportRow{
			CampaignID:         domainID(row.CampaignID),
			CampaignName:       row.CampaignName,
			StudentID:          domainID(row.StudentID),
			StudentName:        row.StudentName,
			StudentPhone:       row.StudentPhone,
			Source:             domain.Source(row.Source),
			CurrentCheckins:    row.CurrentCheckins,
			TargetCheckins:     row.TargetCheckins,
			RemainingCheckins:  row.RemainingCheckins,
			ProgressPercentage: row.ProgressPercentage,
			RewardName:         row.RewardName,
		})
	}
	return result, nil
}

func (r CampaignGormRepository) SaveProgressMany(ctx context.Context, progress []domain.CampaignProgress) error {
	modelsList := make([]models.CampaignProgressModel, 0, len(progress))
	for i := range progress {
		if err := ensureID(&progress[i].ID); err != nil {
			return err
		}
		modelsList = append(modelsList, campaignProgressToModel(progress[i]))
	}

	if len(modelsList) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "campaign_id"}, {Name: "student_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"current_checkins",
			"target_checkins",
			"progress_percentage",
			"achieved",
			"near_goal",
			"updated_at",
		}),
	}).Create(&modelsList).Error
}

func campaignsToDomain(modelsList []models.CampaignModel) []domain.Campaign {
	campaigns := make([]domain.Campaign, 0, len(modelsList))
	for _, model := range modelsList {
		campaigns = append(campaigns, campaignToDomain(model))
	}
	return campaigns
}
