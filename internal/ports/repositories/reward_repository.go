package repositories

import (
	"context"

	"boxengage/backend/internal/domain"
)

type RewardRepository interface {
	ListByCampaign(ctx context.Context, campaignID domain.ID) ([]domain.Reward, error)
	FindByID(ctx context.Context, id domain.ID) (*domain.Reward, error)
	Save(ctx context.Context, reward *domain.Reward) error
	Update(ctx context.Context, reward domain.Reward) error
	Delete(ctx context.Context, id domain.ID) error

	ListDeliveries(ctx context.Context, boxID domain.ID) ([]domain.RewardDelivery, error)
	ListPendingDeliveries(ctx context.Context, boxID domain.ID) ([]domain.RewardDelivery, error)
	CountDeliveries(ctx context.Context, boxID domain.ID, delivered bool) (int, error)
	CreatePendingDeliveries(ctx context.Context, rewardID domain.ID, studentIDs []domain.ID) error
	MarkDelivered(ctx context.Context, boxID domain.ID, deliveryID domain.ID) error
}
