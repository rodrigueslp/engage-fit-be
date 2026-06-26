package imports

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"gorm.io/gorm"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

type ImportCheckinsInput struct {
	BoxID    domain.ID
	Source   domain.Source
	Filename string
	File     io.Reader
}

type ImportCheckinsOutput struct {
	ImportID     domain.ID
	TotalRecords int
	Students     int
	Checkins     int
}

type ImportCheckinsUseCase struct {
	parser    services.CheckinFileParser
	imports   repositories.ImportHistoryRepository
	students  repositories.StudentRepository
	checkins  repositories.CheckinRepository
	campaigns repositories.CampaignRepository
	rewards   repositories.RewardRepository
}

func NewImportCheckinsUseCase(parser services.CheckinFileParser, imports repositories.ImportHistoryRepository, students repositories.StudentRepository, checkins repositories.CheckinRepository, campaigns repositories.CampaignRepository, rewards repositories.RewardRepository) ImportCheckinsUseCase {
	return ImportCheckinsUseCase{parser: parser, imports: imports, students: students, checkins: checkins, campaigns: campaigns, rewards: rewards}
}

func (uc ImportCheckinsUseCase) Execute(ctx context.Context, input ImportCheckinsInput) (*ImportCheckinsOutput, error) {
	parsed, err := uc.parser.Parse(ctx, input.File, input.Source, input.Filename)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	importHistory := domain.ImportHistory{
		BoxID:        input.BoxID,
		Filename:     input.Filename,
		Source:       input.Source,
		TotalRecords: len(parsed),
		ImportedAt:   now,
	}
	if err := uc.imports.Save(ctx, &importHistory); err != nil {
		return nil, err
	}

	studentsCreated := 0
	checkins := make([]domain.Checkin, 0, len(parsed))

	for _, parsedCheckin := range parsed {
		identity := studentIdentity(parsedCheckin)
		student, err := uc.students.FindByExternalID(ctx, input.BoxID, input.Source, identity)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}

			student = &domain.Student{
				BoxID:      input.BoxID,
				Name:       parsedCheckin.StudentName,
				Email:      parsedCheckin.StudentEmail,
				Phone:      parsedCheckin.StudentPhone,
				Source:     input.Source,
				ExternalID: identity,
				CreatedAt:  now,
				UpdatedAt:  now,
			}
			if err := uc.students.Save(ctx, student); err != nil {
				return nil, err
			}
			studentsCreated++
		}

		checkins = append(checkins, domain.Checkin{
			BoxID:           input.BoxID,
			StudentID:       student.ID,
			CheckinDate:     parsedCheckin.CheckinDate,
			CheckinTime:     parsedCheckin.CheckinTime,
			Source:          input.Source,
			ImportHistoryID: importHistory.ID,
			CreatedAt:       now,
		})
	}

	insertedCheckins, err := uc.checkins.SaveMany(ctx, checkins)
	if err != nil {
		return nil, err
	}
	if err := uc.recalculateActiveCampaigns(ctx, input.BoxID); err != nil {
		return nil, err
	}

	return &ImportCheckinsOutput{
		ImportID:     importHistory.ID,
		TotalRecords: len(parsed),
		Students:     studentsCreated,
		Checkins:     insertedCheckins,
	}, nil
}

func (uc ImportCheckinsUseCase) recalculateActiveCampaigns(ctx context.Context, boxID domain.ID) error {
	activeCampaigns, err := uc.campaigns.ListActive(ctx, boxID)
	if err != nil {
		return err
	}

	allStudents, err := uc.students.List(ctx, boxID, repositories.StudentFilters{})
	if err != nil {
		return err
	}

	for _, campaign := range activeCampaigns {
		goals, err := uc.campaigns.ListGoals(ctx, campaign.ID)
		if err != nil {
			return err
		}
		goalsBySource := map[domain.Source]int{}
		for _, goal := range goals {
			goalsBySource[goal.Source] = goal.TargetCheckins
		}

		campaignCheckins, err := uc.checkins.ListByRange(ctx, boxID, domain.TimeRange{Start: campaign.StartDate, End: campaign.EndDate})
		if err != nil {
			return err
		}

		countsByStudent := map[domain.ID]int{}
		for _, checkin := range campaignCheckins {
			countsByStudent[checkin.StudentID]++
		}

		progress := make([]domain.CampaignProgress, 0, len(allStudents))
		for _, student := range allStudents {
			target, ok := goalsBySource[student.Source]
			if !ok {
				continue
			}
			item := domain.NewCampaignProgress(campaign.ID, student.ID, countsByStudent[student.ID], target)
			progress = append(progress, item)
		}

		if err := uc.campaigns.SaveProgressMany(ctx, progress); err != nil {
			return err
		}

		eligibleStudentIDs := []domain.ID{}
		for _, item := range progress {
			if item.Achieved {
				eligibleStudentIDs = append(eligibleStudentIDs, item.StudentID)
			}
		}

		rewards, err := uc.rewards.ListByCampaign(ctx, campaign.ID)
		if err != nil {
			return err
		}
		for _, reward := range rewards {
			if err := uc.rewards.CreatePendingDeliveries(ctx, reward.ID, eligibleStudentIDs); err != nil {
				return err
			}
		}
	}

	return nil
}

func studentIdentity(parsed services.ParsedCheckin) string {
	if parsed.StudentExternalID != "" {
		return strings.TrimSpace(strings.ToLower(parsed.StudentExternalID))
	}
	if parsed.StudentEmail != "" {
		return strings.TrimSpace(strings.ToLower(parsed.StudentEmail))
	}
	if parsed.StudentPhone != "" {
		return strings.TrimSpace(strings.ToLower(parsed.StudentPhone))
	}
	return strings.TrimSpace(strings.ToLower(parsed.StudentName))
}
