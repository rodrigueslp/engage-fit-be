package campaigns

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

type CreateCampaignInput struct {
	BoxID       domain.ID
	Name        string
	Description string
	StartDate   time.Time
	EndDate     time.Time
}

type CreateCampaignUseCase struct {
	campaigns repositories.CampaignRepository
}

func NewCreateCampaignUseCase(campaigns repositories.CampaignRepository) CreateCampaignUseCase {
	return CreateCampaignUseCase{campaigns: campaigns}
}

func (uc CreateCampaignUseCase) Execute(ctx context.Context, input CreateCampaignInput) (*domain.Campaign, error) {
	now := time.Now()
	campaign := domain.Campaign{
		BoxID:       input.BoxID,
		Name:        input.Name,
		Description: input.Description,
		StartDate:   input.StartDate,
		EndDate:     input.EndDate,
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := uc.campaigns.Save(ctx, &campaign); err != nil {
		return nil, err
	}
	return &campaign, nil
}

type ListCampaignsUseCase struct {
	campaigns repositories.CampaignRepository
}

func NewListCampaignsUseCase(campaigns repositories.CampaignRepository) ListCampaignsUseCase {
	return ListCampaignsUseCase{campaigns: campaigns}
}

func (uc ListCampaignsUseCase) Execute(ctx context.Context, boxID domain.ID) ([]domain.Campaign, error) {
	return uc.campaigns.List(ctx, boxID)
}

type GetCampaignUseCase struct {
	campaigns repositories.CampaignRepository
}

func NewGetCampaignUseCase(campaigns repositories.CampaignRepository) GetCampaignUseCase {
	return GetCampaignUseCase{campaigns: campaigns}
}

func (uc GetCampaignUseCase) Execute(ctx context.Context, boxID, campaignID domain.ID) (*domain.Campaign, error) {
	return uc.campaigns.FindByID(ctx, boxID, campaignID)
}

type CloseCampaignUseCase struct {
	campaigns repositories.CampaignRepository
}

type UpdateCampaignInput struct {
	BoxID       domain.ID
	CampaignID  domain.ID
	Name        string
	Description string
	StartDate   time.Time
	EndDate     time.Time
	Active      bool
}

type UpdateCampaignUseCase struct {
	campaigns repositories.CampaignRepository
}

func NewUpdateCampaignUseCase(campaigns repositories.CampaignRepository) UpdateCampaignUseCase {
	return UpdateCampaignUseCase{campaigns: campaigns}
}

func (uc UpdateCampaignUseCase) Execute(ctx context.Context, input UpdateCampaignInput) (*domain.Campaign, error) {
	campaign, err := uc.campaigns.FindByID(ctx, input.BoxID, input.CampaignID)
	if err != nil {
		return nil, err
	}
	campaign.Name = input.Name
	campaign.Description = input.Description
	campaign.StartDate = input.StartDate
	campaign.EndDate = input.EndDate
	campaign.Active = input.Active
	campaign.UpdatedAt = time.Now()
	if err := uc.campaigns.Update(ctx, *campaign); err != nil {
		return nil, err
	}
	return campaign, nil
}

type DeleteCampaignUseCase struct {
	campaigns repositories.CampaignRepository
}

func NewDeleteCampaignUseCase(campaigns repositories.CampaignRepository) DeleteCampaignUseCase {
	return DeleteCampaignUseCase{campaigns: campaigns}
}

func (uc DeleteCampaignUseCase) Execute(ctx context.Context, boxID, campaignID domain.ID) error {
	return uc.campaigns.Delete(ctx, boxID, campaignID)
}

func NewCloseCampaignUseCase(campaigns repositories.CampaignRepository) CloseCampaignUseCase {
	return CloseCampaignUseCase{campaigns: campaigns}
}

func (uc CloseCampaignUseCase) Execute(ctx context.Context, boxID, campaignID domain.ID) error {
	campaign, err := uc.campaigns.FindByID(ctx, boxID, campaignID)
	if err != nil {
		return err
	}
	campaign.Active = false
	return uc.campaigns.Update(ctx, *campaign)
}

type UpsertCampaignGoalUseCase struct {
	campaigns repositories.CampaignRepository
}

func NewUpsertCampaignGoalUseCase(campaigns repositories.CampaignRepository) UpsertCampaignGoalUseCase {
	return UpsertCampaignGoalUseCase{campaigns: campaigns}
}

func (uc UpsertCampaignGoalUseCase) Execute(ctx context.Context, boxID domain.ID, goal *domain.CampaignGoal) error {
	if _, err := uc.campaigns.FindByID(ctx, boxID, goal.CampaignID); err != nil {
		return err
	}
	return uc.campaigns.UpsertGoal(ctx, goal)
}

type DeleteCampaignGoalUseCase struct {
	campaigns repositories.CampaignRepository
}

func NewDeleteCampaignGoalUseCase(campaigns repositories.CampaignRepository) DeleteCampaignGoalUseCase {
	return DeleteCampaignGoalUseCase{campaigns: campaigns}
}

func (uc DeleteCampaignGoalUseCase) Execute(ctx context.Context, boxID, campaignID, goalID domain.ID) error {
	if _, err := uc.campaigns.FindByID(ctx, boxID, campaignID); err != nil {
		return err
	}
	return uc.campaigns.DeleteGoal(ctx, campaignID, goalID)
}

type ListCampaignGoalsUseCase struct {
	campaigns repositories.CampaignRepository
}

func NewListCampaignGoalsUseCase(campaigns repositories.CampaignRepository) ListCampaignGoalsUseCase {
	return ListCampaignGoalsUseCase{campaigns: campaigns}
}

func (uc ListCampaignGoalsUseCase) Execute(ctx context.Context, boxID, campaignID domain.ID) ([]domain.CampaignGoal, error) {
	if _, err := uc.campaigns.FindByID(ctx, boxID, campaignID); err != nil {
		return nil, err
	}
	return uc.campaigns.ListGoals(ctx, campaignID)
}

type ListCampaignProgressUseCase struct {
	campaigns repositories.CampaignRepository
}

func NewListCampaignProgressUseCase(campaigns repositories.CampaignRepository) ListCampaignProgressUseCase {
	return ListCampaignProgressUseCase{campaigns: campaigns}
}

func (uc ListCampaignProgressUseCase) Execute(ctx context.Context, boxID, campaignID domain.ID) ([]domain.CampaignProgress, error) {
	if _, err := uc.campaigns.FindByID(ctx, boxID, campaignID); err != nil {
		return nil, err
	}
	return uc.campaigns.ListProgress(ctx, campaignID)
}

type RecalculateCampaignProgressUseCase struct {
	campaigns repositories.CampaignRepository
	students  repositories.StudentRepository
	checkins  repositories.CheckinRepository
	rewards   repositories.RewardRepository
}

func NewRecalculateCampaignProgressUseCase(campaigns repositories.CampaignRepository, students repositories.StudentRepository, checkins repositories.CheckinRepository, rewards repositories.RewardRepository) RecalculateCampaignProgressUseCase {
	return RecalculateCampaignProgressUseCase{campaigns: campaigns, students: students, checkins: checkins, rewards: rewards}
}

func (uc RecalculateCampaignProgressUseCase) Execute(ctx context.Context, boxID, campaignID domain.ID) error {
	campaign, err := uc.campaigns.FindByID(ctx, boxID, campaignID)
	if err != nil {
		return err
	}

	goals, err := uc.campaigns.ListGoals(ctx, campaignID)
	if err != nil {
		return err
	}
	allStudents, err := uc.students.List(ctx, boxID, repositories.StudentFilters{})
	if err != nil {
		return err
	}
	checkins, err := uc.checkins.ListByRange(ctx, boxID, domain.TimeRange{Start: campaign.StartDate, End: campaign.EndDate})
	if err != nil {
		return err
	}

	progress := domain.BuildCampaignProgress(campaignID, allStudents, checkins, goals)

	if err := uc.campaigns.ReplaceProgress(ctx, campaignID, progress); err != nil {
		return err
	}

	eligibleStudentIDs := []domain.ID{}
	for _, item := range progress {
		if item.Achieved {
			eligibleStudentIDs = append(eligibleStudentIDs, item.StudentID)
		}
	}

	rewards, err := uc.rewards.ListByCampaign(ctx, boxID, campaignID)
	if err != nil {
		return err
	}
	for _, reward := range rewards {
		if err := uc.rewards.SyncPendingDeliveries(ctx, reward.ID, eligibleStudentIDs); err != nil {
			return err
		}
	}

	return nil
}
