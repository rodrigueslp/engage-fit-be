package rewards

import (
	"context"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

type CreateRewardUseCase struct {
	rewards   repositories.RewardRepository
	campaigns repositories.CampaignRepository
}

type UpdateRewardUseCase struct {
	rewards repositories.RewardRepository
}

func NewUpdateRewardUseCase(rewards repositories.RewardRepository) UpdateRewardUseCase {
	return UpdateRewardUseCase{rewards: rewards}
}

func (uc UpdateRewardUseCase) Execute(ctx context.Context, boxID domain.ID, reward domain.Reward) error {
	return uc.rewards.Update(ctx, boxID, reward)
}

type DeleteRewardUseCase struct {
	rewards repositories.RewardRepository
}

func NewDeleteRewardUseCase(rewards repositories.RewardRepository) DeleteRewardUseCase {
	return DeleteRewardUseCase{rewards: rewards}
}

func (uc DeleteRewardUseCase) Execute(ctx context.Context, boxID, rewardID domain.ID) error {
	return uc.rewards.Delete(ctx, boxID, rewardID)
}

func NewCreateRewardUseCase(rewards repositories.RewardRepository, campaigns repositories.CampaignRepository) CreateRewardUseCase {
	return CreateRewardUseCase{rewards: rewards, campaigns: campaigns}
}

func (uc CreateRewardUseCase) Execute(ctx context.Context, boxID domain.ID, reward *domain.Reward) error {
	if _, err := uc.campaigns.FindByID(ctx, boxID, reward.CampaignID); err != nil {
		return err
	}
	return uc.rewards.Save(ctx, reward)
}

type ListRewardsUseCase struct {
	rewards   repositories.RewardRepository
	campaigns repositories.CampaignRepository
}

type GetRewardUseCase struct {
	rewards repositories.RewardRepository
}

func NewGetRewardUseCase(rewards repositories.RewardRepository) GetRewardUseCase {
	return GetRewardUseCase{rewards: rewards}
}

func (uc GetRewardUseCase) Execute(ctx context.Context, boxID, rewardID domain.ID) (*domain.Reward, error) {
	return uc.rewards.FindByID(ctx, boxID, rewardID)
}

func NewListRewardsUseCase(rewards repositories.RewardRepository, campaigns repositories.CampaignRepository) ListRewardsUseCase {
	return ListRewardsUseCase{rewards: rewards, campaigns: campaigns}
}

func (uc ListRewardsUseCase) Execute(ctx context.Context, boxID, campaignID domain.ID) ([]domain.Reward, error) {
	if _, err := uc.campaigns.FindByID(ctx, boxID, campaignID); err != nil {
		return nil, err
	}
	return uc.rewards.ListByCampaign(ctx, boxID, campaignID)
}

type ListPendingRewardDeliveriesUseCase struct {
	rewards repositories.RewardRepository
}

type ListRewardDeliveriesUseCase struct {
	rewards repositories.RewardRepository
}

func NewListRewardDeliveriesUseCase(rewards repositories.RewardRepository) ListRewardDeliveriesUseCase {
	return ListRewardDeliveriesUseCase{rewards: rewards}
}

func (uc ListRewardDeliveriesUseCase) Execute(ctx context.Context, boxID domain.ID) ([]domain.RewardDelivery, error) {
	return uc.rewards.ListDeliveries(ctx, boxID)
}

func NewListPendingRewardDeliveriesUseCase(rewards repositories.RewardRepository) ListPendingRewardDeliveriesUseCase {
	return ListPendingRewardDeliveriesUseCase{rewards: rewards}
}

func (uc ListPendingRewardDeliveriesUseCase) Execute(ctx context.Context, boxID domain.ID) ([]domain.RewardDelivery, error) {
	return uc.rewards.ListPendingDeliveries(ctx, boxID)
}

type MarkRewardDeliveredUseCase struct {
	rewards repositories.RewardRepository
}

func NewMarkRewardDeliveredUseCase(rewards repositories.RewardRepository) MarkRewardDeliveredUseCase {
	return MarkRewardDeliveredUseCase{rewards: rewards}
}

func (uc MarkRewardDeliveredUseCase) Execute(ctx context.Context, boxID domain.ID, deliveryID domain.ID) error {
	return uc.rewards.MarkDelivered(ctx, boxID, deliveryID)
}
