package rewards

import (
	"context"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

type CreateRewardUseCase struct {
	rewards repositories.RewardRepository
}

type UpdateRewardUseCase struct {
	rewards repositories.RewardRepository
}

func NewUpdateRewardUseCase(rewards repositories.RewardRepository) UpdateRewardUseCase {
	return UpdateRewardUseCase{rewards: rewards}
}

func (uc UpdateRewardUseCase) Execute(ctx context.Context, reward domain.Reward) error {
	return uc.rewards.Update(ctx, reward)
}

type DeleteRewardUseCase struct {
	rewards repositories.RewardRepository
}

func NewDeleteRewardUseCase(rewards repositories.RewardRepository) DeleteRewardUseCase {
	return DeleteRewardUseCase{rewards: rewards}
}

func (uc DeleteRewardUseCase) Execute(ctx context.Context, rewardID domain.ID) error {
	return uc.rewards.Delete(ctx, rewardID)
}

func NewCreateRewardUseCase(rewards repositories.RewardRepository) CreateRewardUseCase {
	return CreateRewardUseCase{rewards: rewards}
}

func (uc CreateRewardUseCase) Execute(ctx context.Context, reward *domain.Reward) error {
	return uc.rewards.Save(ctx, reward)
}

type ListRewardsUseCase struct {
	rewards repositories.RewardRepository
}

type GetRewardUseCase struct {
	rewards repositories.RewardRepository
}

func NewGetRewardUseCase(rewards repositories.RewardRepository) GetRewardUseCase {
	return GetRewardUseCase{rewards: rewards}
}

func (uc GetRewardUseCase) Execute(ctx context.Context, rewardID domain.ID) (*domain.Reward, error) {
	return uc.rewards.FindByID(ctx, rewardID)
}

func NewListRewardsUseCase(rewards repositories.RewardRepository) ListRewardsUseCase {
	return ListRewardsUseCase{rewards: rewards}
}

func (uc ListRewardsUseCase) Execute(ctx context.Context, campaignID domain.ID) ([]domain.Reward, error) {
	return uc.rewards.ListByCampaign(ctx, campaignID)
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
