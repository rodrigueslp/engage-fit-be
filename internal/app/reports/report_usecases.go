package reports

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

type EligibleStudentsReportUseCase struct {
	campaigns repositories.CampaignRepository
}

func NewEligibleStudentsReportUseCase(campaigns repositories.CampaignRepository) EligibleStudentsReportUseCase {
	return EligibleStudentsReportUseCase{campaigns: campaigns}
}

func (uc EligibleStudentsReportUseCase) Execute(ctx context.Context, boxID domain.ID) ([]domain.EligibleStudentReportRow, error) {
	return uc.campaigns.ListEligibleReportRows(ctx, boxID)
}

type PendingRewardsReportUseCase struct {
	rewards repositories.RewardRepository
}

func NewPendingRewardsReportUseCase(rewards repositories.RewardRepository) PendingRewardsReportUseCase {
	return PendingRewardsReportUseCase{rewards: rewards}
}

func (uc PendingRewardsReportUseCase) Execute(ctx context.Context, boxID domain.ID) ([]domain.RewardDelivery, error) {
	return uc.rewards.ListPendingDeliveries(ctx, boxID)
}

type MonthlyFrequencyReportUseCase struct {
	checkins repositories.CheckinRepository
}

func NewMonthlyFrequencyReportUseCase(checkins repositories.CheckinRepository) MonthlyFrequencyReportUseCase {
	return MonthlyFrequencyReportUseCase{checkins: checkins}
}

func (uc MonthlyFrequencyReportUseCase) Execute(ctx context.Context, boxID domain.ID, period domain.TimeRange) ([]domain.MonthlyFrequencyReportRow, error) {
	return uc.checkins.ListMonthlyFrequency(ctx, boxID, period)
}

func EligibleStudentsCSV(rows []domain.EligibleStudentReportRow) ([]string, [][]string) {
	headers := []string{"campanha", "aluno", "telefone", "plataforma", "checkins", "meta", "faltam", "progresso", "brinde"}
	csvRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		csvRows = append(csvRows, []string{
			row.CampaignName,
			row.StudentName,
			row.StudentPhone,
			string(row.Source),
			strconv.Itoa(row.CurrentCheckins),
			strconv.Itoa(row.TargetCheckins),
			strconv.Itoa(row.RemainingCheckins),
			fmt.Sprintf("%.0f%%", row.ProgressPercentage),
			row.RewardName,
		})
	}
	return headers, csvRows
}

func PendingRewardsCSV(rows []domain.RewardDelivery) ([]string, [][]string) {
	headers := []string{"campanha", "aluno", "telefone", "brinde", "status"}
	csvRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		csvRows = append(csvRows, []string{
			row.CampaignName,
			row.StudentName,
			row.StudentPhone,
			row.RewardName,
			"pendente",
		})
	}
	return headers, csvRows
}

func MonthlyFrequencyCSV(rows []domain.MonthlyFrequencyReportRow) ([]string, [][]string) {
	headers := []string{"aluno", "telefone", "plataforma", "checkins", "primeiro_checkin", "ultimo_checkin"}
	csvRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		csvRows = append(csvRows, []string{
			row.StudentName,
			row.StudentPhone,
			string(row.Source),
			strconv.Itoa(row.Checkins),
			formatDate(row.FirstCheckin),
			formatDate(row.LastCheckin),
		})
	}
	return headers, csvRows
}

func formatDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02")
}
