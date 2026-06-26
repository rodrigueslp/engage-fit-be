package dashboard

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

type Summary struct {
	TotalStudents      int
	TotalCheckins      int
	EligibleStudents   int
	NearGoalStudents   int
	AtRiskStudents     int
	PendingRewards     int
	DeliveredRewards   int
	CheckinsByPlatform map[domain.Source]int
}

type GetDashboardSummaryUseCase struct {
	boxes     repositories.BoxRepository
	students  repositories.StudentRepository
	checkins  repositories.CheckinRepository
	campaigns repositories.CampaignRepository
	rewards   repositories.RewardRepository
}

func NewGetDashboardSummaryUseCase(boxes repositories.BoxRepository, students repositories.StudentRepository, checkins repositories.CheckinRepository, campaigns repositories.CampaignRepository, rewards repositories.RewardRepository) GetDashboardSummaryUseCase {
	return GetDashboardSummaryUseCase{boxes: boxes, students: students, checkins: checkins, campaigns: campaigns, rewards: rewards}
}

func (uc GetDashboardSummaryUseCase) Execute(ctx context.Context, boxID domain.ID) (*Summary, error) {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	students, err := uc.students.List(ctx, boxID, repositories.StudentFilters{})
	if err != nil {
		return nil, err
	}

	monthlyCheckins, err := uc.checkins.ListByRange(ctx, boxID, domain.TimeRange{Start: monthStart, End: monthEnd})
	if err != nil {
		return nil, err
	}

	checkinsByPlatform, err := uc.checkins.CountBySource(ctx, boxID, domain.TimeRange{Start: monthStart, End: monthEnd})
	if err != nil {
		return nil, err
	}

	activeCampaigns, err := uc.campaigns.ListActive(ctx, boxID)
	if err != nil {
		return nil, err
	}

	eligible := map[domain.ID]bool{}
	nearGoal := map[domain.ID]bool{}
	for _, campaign := range activeCampaigns {
		progress, err := uc.campaigns.ListProgress(ctx, campaign.ID)
		if err != nil {
			return nil, err
		}
		for _, item := range progress {
			if item.Achieved {
				eligible[item.StudentID] = true
			}
			if item.NearGoal {
				nearGoal[item.StudentID] = true
			}
		}
	}

	atRisk, err := uc.atRiskStudents(ctx, boxID, students)
	if err != nil {
		return nil, err
	}

	pendingRewards, err := uc.rewards.CountDeliveries(ctx, boxID, false)
	if err != nil {
		return nil, err
	}
	deliveredRewards, err := uc.rewards.CountDeliveries(ctx, boxID, true)
	if err != nil {
		return nil, err
	}

	return &Summary{
		TotalStudents:      len(students),
		TotalCheckins:      len(monthlyCheckins),
		EligibleStudents:   len(eligible),
		NearGoalStudents:   len(nearGoal),
		AtRiskStudents:     len(atRisk),
		PendingRewards:     pendingRewards,
		DeliveredRewards:   deliveredRewards,
		CheckinsByPlatform: checkinsByPlatform,
	}, nil
}

type ListActiveCampaignsUseCase struct {
	campaigns repositories.CampaignRepository
}

func NewListActiveCampaignsUseCase(campaigns repositories.CampaignRepository) ListActiveCampaignsUseCase {
	return ListActiveCampaignsUseCase{campaigns: campaigns}
}

func (uc ListActiveCampaignsUseCase) Execute(ctx context.Context, boxID domain.ID) ([]domain.Campaign, error) {
	return uc.campaigns.ListActive(ctx, boxID)
}

type ListNearGoalStudentsUseCase struct {
	students  repositories.StudentRepository
	campaigns repositories.CampaignRepository
}

func NewListNearGoalStudentsUseCase(students repositories.StudentRepository, campaigns repositories.CampaignRepository) ListNearGoalStudentsUseCase {
	return ListNearGoalStudentsUseCase{students: students, campaigns: campaigns}
}

func (uc ListNearGoalStudentsUseCase) Execute(ctx context.Context, boxID domain.ID) ([]domain.Student, error) {
	activeCampaigns, err := uc.campaigns.ListActive(ctx, boxID)
	if err != nil {
		return nil, err
	}

	unique := map[domain.ID]domain.Student{}
	for _, campaign := range activeCampaigns {
		nearGoal := true
		students, err := uc.students.List(ctx, boxID, repositories.StudentFilters{CampaignID: &campaign.ID, NearGoal: &nearGoal})
		if err != nil {
			return nil, err
		}
		for _, student := range students {
			unique[student.ID] = student
		}
	}

	result := make([]domain.Student, 0, len(unique))
	for _, student := range unique {
		result = append(result, student)
	}
	return result, nil
}

type ListAtRiskStudentsUseCase struct {
	boxes    repositories.BoxRepository
	students repositories.StudentRepository
	checkins repositories.CheckinRepository
}

func NewListAtRiskStudentsUseCase(boxes repositories.BoxRepository, students repositories.StudentRepository, checkins repositories.CheckinRepository) ListAtRiskStudentsUseCase {
	return ListAtRiskStudentsUseCase{boxes: boxes, students: students, checkins: checkins}
}

func (uc ListAtRiskStudentsUseCase) Execute(ctx context.Context, boxID domain.ID) ([]domain.Student, error) {
	students, err := uc.students.List(ctx, boxID, repositories.StudentFilters{})
	if err != nil {
		return nil, err
	}
	return uc.atRiskStudents(ctx, boxID, students)
}

func (uc GetDashboardSummaryUseCase) atRiskStudents(ctx context.Context, boxID domain.ID, students []domain.Student) ([]domain.Student, error) {
	box, err := uc.boxes.FindByID(ctx, boxID)
	if err != nil {
		return nil, err
	}
	return atRiskStudents(ctx, boxID, students, uc.checkins, box.RiskInactiveDays)
}

func (uc ListAtRiskStudentsUseCase) atRiskStudents(ctx context.Context, boxID domain.ID, students []domain.Student) ([]domain.Student, error) {
	box, err := uc.boxes.FindByID(ctx, boxID)
	if err != nil {
		return nil, err
	}
	return atRiskStudents(ctx, boxID, students, uc.checkins, box.RiskInactiveDays)
}

func atRiskStudents(ctx context.Context, boxID domain.ID, students []domain.Student, checkins repositories.CheckinRepository, inactiveDays int) ([]domain.Student, error) {
	if inactiveDays <= 0 {
		inactiveDays = 7
	}
	threshold := time.Now().AddDate(0, 0, -inactiveDays)
	result := []domain.Student{}
	for _, student := range students {
		lastCheckin, err := checkins.LastCheckinDate(ctx, boxID, student.ID)
		if err != nil {
			result = append(result, student)
			continue
		}
		if lastCheckin.Before(threshold) || lastCheckin.Equal(threshold) {
			result = append(result, student)
		}
	}
	return result, nil
}
