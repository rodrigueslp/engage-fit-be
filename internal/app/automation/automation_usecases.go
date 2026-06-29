package automation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"boxengage/backend/internal/app/campaigns"
	"boxengage/backend/internal/app/messages"
	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

const (
	ScheduleModeFullDaily   = "full_daily"
	ScheduleModeRecalculate = "recalculate_only"
	ScheduleModeAlmostThere = "send_almost_there"
	ScheduleModeAchieved    = "send_achieved"
	ScheduleModeInactive    = "send_inactive"
	defaultDaysOfWeek       = "0,1,2,3,4,5,6"
	defaultTimezone         = "America/Sao_Paulo"
)

type ListRunsUseCase struct {
	runs repositories.AutomationRepository
}

func NewListRunsUseCase(runs repositories.AutomationRepository) ListRunsUseCase {
	return ListRunsUseCase{runs: runs}
}
func (uc ListRunsUseCase) Execute(ctx context.Context, boxID domain.ID) ([]domain.AutomationRun, error) {
	return uc.runs.ListRuns(ctx, boxID)
}

type GetRunUseCase struct {
	runs repositories.AutomationRepository
}

func NewGetRunUseCase(runs repositories.AutomationRepository) GetRunUseCase {
	return GetRunUseCase{runs: runs}
}
func (uc GetRunUseCase) Execute(ctx context.Context, boxID, id domain.ID) (*domain.AutomationRun, error) {
	return uc.runs.FindRunByID(ctx, boxID, id)
}

type CreateRunUseCase struct {
	runs repositories.AutomationRepository
}

func NewCreateRunUseCase(runs repositories.AutomationRepository) CreateRunUseCase {
	return CreateRunUseCase{runs: runs}
}
func (uc CreateRunUseCase) Execute(ctx context.Context, run *domain.AutomationRun) error {
	if run.Status == "" {
		run.Status = "running"
	}
	if run.StartedAt.IsZero() {
		run.StartedAt = time.Now()
	}
	return uc.runs.SaveRun(ctx, run)
}

type UpdateRunUseCase struct {
	runs repositories.AutomationRepository
}

func NewUpdateRunUseCase(runs repositories.AutomationRepository) UpdateRunUseCase {
	return UpdateRunUseCase{runs: runs}
}
func (uc UpdateRunUseCase) Execute(ctx context.Context, run domain.AutomationRun) error {
	return uc.runs.UpdateRun(ctx, run)
}

type ListSchedulesUseCase struct {
	automation repositories.AutomationRepository
}

func NewListSchedulesUseCase(automation repositories.AutomationRepository) ListSchedulesUseCase {
	return ListSchedulesUseCase{automation: automation}
}
func (uc ListSchedulesUseCase) Execute(ctx context.Context, boxID domain.ID) ([]domain.AutomationSchedule, error) {
	return uc.automation.ListSchedules(ctx, boxID)
}

type GetScheduleUseCase struct {
	automation repositories.AutomationRepository
}

func NewGetScheduleUseCase(automation repositories.AutomationRepository) GetScheduleUseCase {
	return GetScheduleUseCase{automation: automation}
}
func (uc GetScheduleUseCase) Execute(ctx context.Context, boxID, id domain.ID) (*domain.AutomationSchedule, error) {
	return uc.automation.FindScheduleByID(ctx, boxID, id)
}

type CreateScheduleUseCase struct {
	automation repositories.AutomationRepository
}

func NewCreateScheduleUseCase(automation repositories.AutomationRepository) CreateScheduleUseCase {
	return CreateScheduleUseCase{automation: automation}
}
func (uc CreateScheduleUseCase) Execute(ctx context.Context, schedule *domain.AutomationSchedule) error {
	normalizeSchedule(schedule)
	return uc.automation.SaveSchedule(ctx, schedule)
}

type UpdateScheduleUseCase struct {
	automation repositories.AutomationRepository
}

func NewUpdateScheduleUseCase(automation repositories.AutomationRepository) UpdateScheduleUseCase {
	return UpdateScheduleUseCase{automation: automation}
}
func (uc UpdateScheduleUseCase) Execute(ctx context.Context, schedule domain.AutomationSchedule) error {
	normalizeSchedule(&schedule)
	return uc.automation.UpdateSchedule(ctx, schedule)
}

type DeleteScheduleUseCase struct {
	automation repositories.AutomationRepository
}

func NewDeleteScheduleUseCase(automation repositories.AutomationRepository) DeleteScheduleUseCase {
	return DeleteScheduleUseCase{automation: automation}
}
func (uc DeleteScheduleUseCase) Execute(ctx context.Context, boxID, id domain.ID) error {
	return uc.automation.DeleteSchedule(ctx, boxID, id)
}

type ExecuteScheduleUseCase struct {
	automation  repositories.AutomationRepository
	campaigns   repositories.CampaignRepository
	messages    repositories.MessageRepository
	recalculate campaigns.RecalculateCampaignProgressUseCase
	sendMessage messages.SendMessageCampaignUseCase
}

func NewExecuteScheduleUseCase(automation repositories.AutomationRepository, campaignsRepo repositories.CampaignRepository, messageRepo repositories.MessageRepository, recalculate campaigns.RecalculateCampaignProgressUseCase, sendMessage messages.SendMessageCampaignUseCase) ExecuteScheduleUseCase {
	return ExecuteScheduleUseCase{automation: automation, campaigns: campaignsRepo, messages: messageRepo, recalculate: recalculate, sendMessage: sendMessage}
}

func (uc ExecuteScheduleUseCase) Execute(ctx context.Context, boxID, scheduleID domain.ID) (*domain.AutomationRun, error) {
	schedule, err := uc.automation.FindScheduleByID(ctx, boxID, scheduleID)
	if err != nil {
		return nil, err
	}
	return uc.ExecuteSchedule(ctx, *schedule)
}

func (uc ExecuteScheduleUseCase) ExecuteSchedule(ctx context.Context, schedule domain.AutomationSchedule) (*domain.AutomationRun, error) {
	now := time.Now()
	run := domain.AutomationRun{BoxID: schedule.BoxID, Status: "running", Source: "schedule", Filename: schedule.Name, StartedAt: now}
	if err := uc.automation.SaveRun(ctx, &run); err != nil {
		return nil, err
	}

	var errors []string
	activeCampaigns, err := uc.campaigns.ListActive(ctx, schedule.BoxID)
	if err != nil {
		errors = append(errors, err.Error())
	} else if shouldRecalculate(schedule.Mode) {
		for _, campaign := range activeCampaigns {
			if err := uc.recalculate.Execute(ctx, schedule.BoxID, campaign.ID); err != nil {
				errors = append(errors, err.Error())
				continue
			}
			run.RecalculatedCampaigns++
		}
	}

	if shouldSendMessages(schedule.Mode) && len(activeCampaigns) > 0 {
		activeIDs := map[domain.ID]bool{}
		for _, campaign := range activeCampaigns {
			activeIDs[campaign.ID] = true
		}
		messageCampaigns, err := uc.messages.ListCampaigns(ctx, schedule.BoxID)
		if err != nil {
			errors = append(errors, err.Error())
		} else {
			for _, messageCampaign := range messageCampaigns {
				if !activeIDs[messageCampaign.CampaignID] || !audienceMatchesMode(schedule.Mode, messageCampaign.Audience) || (messageCampaign.SentAt != nil && !schedule.AllowResend) {
					run.SkippedMessageCampaigns++
					continue
				}
				result, err := uc.sendMessage.Execute(ctx, schedule.BoxID, messageCampaign.ID)
				if err != nil {
					run.FailedMessages++
					errors = append(errors, err.Error())
					continue
				}
				run.SentMessages += result.Sent
				run.FailedMessages += result.Failed
			}
		}
	}

	finishedAt := time.Now()
	run.FinishedAt = &finishedAt
	run.Status = "success"
	if len(errors) > 0 {
		run.Status = "failed"
		run.ErrorMessage = strings.Join(errors, " | ")
	}
	if err := uc.automation.UpdateRun(ctx, run); err != nil {
		return nil, err
	}

	schedule.LastRunAt = &finishedAt
	_ = uc.automation.UpdateSchedule(ctx, schedule)
	return &run, nil
}

func (uc ExecuteScheduleUseCase) ExecuteDue(ctx context.Context, now time.Time) ([]domain.AutomationRun, error) {
	schedules, err := uc.automation.ListEnabledSchedules(ctx)
	if err != nil {
		return nil, err
	}
	runs := []domain.AutomationRun{}
	for _, schedule := range schedules {
		if !IsScheduleDue(schedule, now) {
			continue
		}
		run, err := uc.ExecuteSchedule(ctx, schedule)
		if err != nil {
			return runs, err
		}
		runs = append(runs, *run)
	}
	return runs, nil
}

func IsScheduleDue(schedule domain.AutomationSchedule, now time.Time) bool {
	if !schedule.Enabled {
		return false
	}
	location, err := time.LoadLocation(scheduleTimezone(schedule.Timezone))
	if err != nil {
		location = time.FixedZone("BRT", -3*60*60)
	}
	localNow := now.In(location)
	if !dayAllowed(schedule.DaysOfWeek, int(localNow.Weekday())) {
		return false
	}
	if localNow.Format("15:04") != schedule.RunTime {
		return false
	}
	if schedule.LastRunAt != nil && schedule.LastRunAt.In(location).Format("2006-01-02 15:04") == localNow.Format("2006-01-02 15:04") {
		return false
	}
	return true
}

func normalizeSchedule(schedule *domain.AutomationSchedule) {
	schedule.Mode = normalizeMode(schedule.Mode)
	if strings.TrimSpace(schedule.RunTime) == "" {
		schedule.RunTime = "08:00"
	}
	schedule.Timezone = scheduleTimezone(schedule.Timezone)
	if strings.TrimSpace(schedule.DaysOfWeek) == "" {
		schedule.DaysOfWeek = defaultDaysOfWeek
	}
	if schedule.CreatedAt.IsZero() {
		schedule.CreatedAt = time.Now()
	}
	schedule.UpdatedAt = time.Now()
}

func normalizeMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case ScheduleModeFullDaily, ScheduleModeRecalculate, ScheduleModeAlmostThere, ScheduleModeAchieved, ScheduleModeInactive:
		return strings.TrimSpace(mode)
	default:
		return ScheduleModeFullDaily
	}
}

func scheduleTimezone(timezone string) string {
	if strings.TrimSpace(timezone) == "" {
		return defaultTimezone
	}
	return strings.TrimSpace(timezone)
}

func shouldRecalculate(mode string) bool {
	return mode == ScheduleModeFullDaily || mode == ScheduleModeRecalculate || shouldSendMessages(mode)
}
func shouldSendMessages(mode string) bool {
	return mode == ScheduleModeFullDaily || mode == ScheduleModeAlmostThere || mode == ScheduleModeAchieved || mode == ScheduleModeInactive
}

func audienceMatchesMode(mode string, audience domain.MessageAudience) bool {
	switch mode {
	case ScheduleModeFullDaily:
		return true
	case ScheduleModeAlmostThere:
		return audience == domain.MessageAudienceAlmostThere || audience == domain.MessageAudienceNearGoal
	case ScheduleModeAchieved:
		return audience == domain.MessageAudienceAchieved
	case ScheduleModeInactive:
		return audience == domain.MessageAudienceInactive
	default:
		return false
	}
}

func dayAllowed(days string, weekday int) bool {
	if strings.TrimSpace(days) == "" {
		return true
	}
	wanted := fmt.Sprintf("%d", weekday)
	for _, part := range strings.Split(days, ",") {
		if strings.TrimSpace(part) == wanted {
			return true
		}
	}
	return false
}
