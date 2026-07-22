package repositories

import (
	"context"

	"boxengage/backend/internal/domain"
)

type CampaignRepository interface {
	FindByID(ctx context.Context, boxID, id domain.ID) (*domain.Campaign, error)
	List(ctx context.Context, boxID domain.ID) ([]domain.Campaign, error)
	ListActive(ctx context.Context, boxID domain.ID) ([]domain.Campaign, error)
	Save(ctx context.Context, campaign *domain.Campaign) error
	Update(ctx context.Context, campaign domain.Campaign) error
	Delete(ctx context.Context, boxID, id domain.ID) error

	ListGoals(ctx context.Context, campaignID domain.ID) ([]domain.CampaignGoal, error)
	UpsertGoal(ctx context.Context, goal *domain.CampaignGoal) error
	DeleteGoal(ctx context.Context, campaignID, goalID domain.ID) error

	ListProgress(ctx context.Context, campaignID domain.ID) ([]domain.CampaignProgress, error)
	ListEligibleReportRows(ctx context.Context, boxID domain.ID) ([]domain.EligibleStudentReportRow, error)
	ReplaceProgress(ctx context.Context, campaignID domain.ID, progress []domain.CampaignProgress) error
}
