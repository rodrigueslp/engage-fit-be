package automation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"boxengage/backend/internal/app/campaigns"
	"boxengage/backend/internal/app/messages"
	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/observability"
	"boxengage/backend/internal/ports/repositories"
	"gorm.io/gorm"
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

var (
	ErrInvalidSchedule       = errors.New("invalid automation schedule")
	ErrInvalidIdempotencyKey = errors.New("invalid idempotency key")
	validIdempotencyKey      = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
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
func (uc CreateRunUseCase) Execute(ctx context.Context, run *domain.AutomationRun) (*domain.AutomationRun, bool, error) {
	if run.ExecutionKey != "" && !validIdempotencyKey.MatchString(run.ExecutionKey) {
		return nil, false, fmt.Errorf("%w: use 1-128 safe characters", ErrInvalidIdempotencyKey)
	}
	if run.Status == "" {
		run.Status = "running"
	}
	if run.StartedAt.IsZero() {
		run.StartedAt = time.Now()
	}
	return startAutomationRun(ctx, uc.runs, run)
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
	if err := normalizeAndValidateSchedule(schedule); err != nil {
		return err
	}
	return uc.automation.SaveSchedule(ctx, schedule)
}

type UpdateScheduleUseCase struct {
	automation repositories.AutomationRepository
}

func NewUpdateScheduleUseCase(automation repositories.AutomationRepository) UpdateScheduleUseCase {
	return UpdateScheduleUseCase{automation: automation}
}
func (uc UpdateScheduleUseCase) Execute(ctx context.Context, schedule domain.AutomationSchedule) error {
	if err := normalizeAndValidateSchedule(&schedule); err != nil {
		return err
	}
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
	automation    repositories.AutomationRepository
	campaigns     repositories.CampaignRepository
	messages      repositories.MessageRepository
	recalculate   campaigns.RecalculateCampaignProgressUseCase
	sendMessage   messages.SendMessageCampaignUseCase
	staleAfter    time.Duration
	catchupWindow time.Duration
}

func NewExecuteScheduleUseCase(automation repositories.AutomationRepository, campaignsRepo repositories.CampaignRepository, messageRepo repositories.MessageRepository, recalculate campaigns.RecalculateCampaignProgressUseCase, sendMessage messages.SendMessageCampaignUseCase, timings ...time.Duration) ExecuteScheduleUseCase {
	timeout := 2 * time.Hour
	if len(timings) > 0 && timings[0] > 0 {
		timeout = timings[0]
	}
	catchup := 15 * time.Minute
	if len(timings) > 1 && timings[1] > 0 {
		catchup = timings[1]
	}
	return ExecuteScheduleUseCase{automation: automation, campaigns: campaignsRepo, messages: messageRepo, recalculate: recalculate, sendMessage: sendMessage, staleAfter: timeout, catchupWindow: catchup}
}

func (uc ExecuteScheduleUseCase) Execute(ctx context.Context, boxID, scheduleID domain.ID) (*domain.AutomationRun, error) {
	return uc.ExecuteWithKey(ctx, boxID, scheduleID, "")
}

func (uc ExecuteScheduleUseCase) ExecuteWithKey(ctx context.Context, boxID, scheduleID domain.ID, executionKey string) (*domain.AutomationRun, error) {
	if executionKey != "" && !validIdempotencyKey.MatchString(executionKey) {
		return nil, fmt.Errorf("%w: use 1-128 safe characters", ErrInvalidIdempotencyKey)
	}
	schedule, err := uc.automation.FindScheduleByID(ctx, boxID, scheduleID)
	if err != nil {
		return nil, err
	}
	return uc.executeSchedule(ctx, *schedule, executionKey, nil, true)
}

func (uc ExecuteScheduleUseCase) ExecuteSchedule(ctx context.Context, schedule domain.AutomationSchedule) (*domain.AutomationRun, error) {
	return uc.executeSchedule(ctx, schedule, "", nil, true)
}

func (uc ExecuteScheduleUseCase) executeSchedule(ctx context.Context, schedule domain.AutomationSchedule, executionKey string, scheduledFor *time.Time, updateLastRun bool) (*domain.AutomationRun, error) {
	now := time.Now()
	run := domain.AutomationRun{BoxID: schedule.BoxID, ScheduleID: schedule.ID, ExecutionKey: executionKey, ScheduledFor: scheduledFor, Status: "running", Source: "schedule", Mode: schedule.Mode, Filename: schedule.Name, StartedAt: now}
	startedRun, existing, err := startAutomationRun(ctx, uc.automation, &run)
	if err != nil {
		return nil, err
	}
	if existing {
		return startedRun, nil
	}
	run = *startedRun
	run.Mode = schedule.Mode

	var failures []string
	activeCampaigns, err := uc.campaigns.ListActive(ctx, schedule.BoxID)
	if err != nil {
		failures = append(failures, err.Error())
	} else if shouldRecalculate(schedule.Mode) {
		for _, campaign := range activeCampaigns {
			if err := uc.recalculate.Execute(ctx, schedule.BoxID, campaign.ID); err != nil {
				failures = append(failures, err.Error())
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
			failures = append(failures, err.Error())
		} else {
			for _, messageCampaign := range messageCampaigns {
				if !activeIDs[messageCampaign.CampaignID] || !audienceMatchesMode(schedule.Mode, messageCampaign.Audience) || (messageCampaign.SentAt != nil && !schedule.AllowResend) {
					run.SkippedMessageCampaigns++
					continue
				}
				result, err := uc.sendMessage.Execute(ctx, schedule.BoxID, messageCampaign.ID)
				if err != nil {
					run.FailedMessages++
					failures = append(failures, err.Error())
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
	if len(failures) > 0 {
		run.Status = "failed"
		run.ErrorMessage = strings.Join(failures, " | ")
	}
	if err := uc.automation.UpdateRun(ctx, run); err != nil {
		return nil, err
	}

	if updateLastRun {
		if err := uc.automation.MarkScheduleRun(ctx, schedule.BoxID, schedule.ID, finishedAt); err != nil {
			return &run, err
		}
	}
	return &run, nil
}

func (uc ExecuteScheduleUseCase) ExecuteDue(ctx context.Context, now time.Time) ([]domain.AutomationRun, error) {
	var executionErrors []error
	staleRuns, err := uc.automation.FailStaleRuns(ctx, now.Add(-uc.staleAfter))
	if err != nil {
		executionErrors = append(executionErrors, fmt.Errorf("mark stale automation runs: %w", err))
	} else if staleRuns > 0 {
		observability.RecordStaleAutomationRuns(ctx, staleRuns)
		slog.WarnContext(ctx, "automation_stale_runs_failed", "count", staleRuns, "timeout_minutes", uc.staleAfter.Minutes())
	}
	schedules, err := uc.automation.ListEnabledSchedules(ctx)
	if err != nil {
		return nil, errors.Join(append(executionErrors, err)...)
	}
	runs := []domain.AutomationRun{}
	for _, schedule := range schedules {
		if !IsScheduleDueWithin(schedule, now, uc.catchupWindow) {
			continue
		}
		scheduledFor := scheduledTime(schedule, now).UTC()
		claimed, err := uc.automation.ClaimSchedule(ctx, schedule.BoxID, schedule.ID, scheduledFor)
		if err != nil {
			executionErrors = append(executionErrors, fmt.Errorf("claim schedule %s: %w", schedule.ID, err))
			continue
		}
		if !claimed {
			continue
		}
		schedule.LastRunAt = &scheduledFor
		executionKey := fmt.Sprintf("schedule:%s:%s", schedule.ID, scheduledFor.Format("20060102T1504Z"))
		run, err := uc.executeSchedule(ctx, schedule, executionKey, &scheduledFor, false)
		if err != nil {
			executionErrors = append(executionErrors, fmt.Errorf("execute schedule %s: %w", schedule.ID, err))
			continue
		}
		runs = append(runs, *run)
	}
	return runs, errors.Join(executionErrors...)
}

func IsScheduleDue(schedule domain.AutomationSchedule, now time.Time) bool {
	return IsScheduleDueWithin(schedule, now, time.Minute)
}

func IsScheduleDueWithin(schedule domain.AutomationSchedule, now time.Time, catchupWindow time.Duration) bool {
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
	scheduledFor := scheduledTime(schedule, now)
	if localNow.Before(scheduledFor) || !localNow.Before(scheduledFor.Add(catchupWindow)) {
		return false
	}
	if schedule.LastRunAt != nil {
		lastRun := schedule.LastRunAt.In(location)
		if !lastRun.Before(scheduledFor) && lastRun.Format("2006-01-02") == scheduledFor.Format("2006-01-02") {
			return false
		}
	}
	return true
}

func scheduledTime(schedule domain.AutomationSchedule, now time.Time) time.Time {
	location, err := time.LoadLocation(scheduleTimezone(schedule.Timezone))
	if err != nil {
		location = time.FixedZone("BRT", -3*60*60)
	}
	localNow := now.In(location)
	parsed, err := time.Parse("15:04", schedule.RunTime)
	if err != nil {
		return localNow
	}
	return time.Date(localNow.Year(), localNow.Month(), localNow.Day(), parsed.Hour(), parsed.Minute(), 0, 0, location)
}

func normalizeAndValidateSchedule(schedule *domain.AutomationSchedule) error {
	schedule.Name = strings.TrimSpace(schedule.Name)
	if schedule.Name == "" || len(schedule.Name) > 255 {
		return fmt.Errorf("%w: name is required and must have at most 255 characters", ErrInvalidSchedule)
	}
	schedule.Mode = strings.TrimSpace(schedule.Mode)
	if schedule.Mode == "" {
		schedule.Mode = ScheduleModeFullDaily
	}
	if !validScheduleMode(schedule.Mode) {
		return fmt.Errorf("%w: mode is invalid", ErrInvalidSchedule)
	}
	if strings.TrimSpace(schedule.RunTime) == "" {
		schedule.RunTime = "08:00"
	}
	parsedRunTime, err := time.Parse("15:04", schedule.RunTime)
	if err != nil || parsedRunTime.Format("15:04") != schedule.RunTime {
		return fmt.Errorf("%w: run_time must use HH:MM", ErrInvalidSchedule)
	}
	schedule.Timezone = scheduleTimezone(schedule.Timezone)
	if _, err := time.LoadLocation(schedule.Timezone); err != nil {
		return fmt.Errorf("%w: timezone is invalid", ErrInvalidSchedule)
	}
	normalizedDays, err := normalizeDaysOfWeek(schedule.DaysOfWeek)
	if err != nil {
		return err
	}
	schedule.DaysOfWeek = normalizedDays
	if schedule.CreatedAt.IsZero() {
		schedule.CreatedAt = time.Now()
	}
	schedule.UpdatedAt = time.Now()
	return nil
}

func validScheduleMode(mode string) bool {
	switch mode {
	case ScheduleModeFullDaily, ScheduleModeRecalculate, ScheduleModeAlmostThere, ScheduleModeAchieved, ScheduleModeInactive:
		return true
	default:
		return false
	}
}

func normalizeDaysOfWeek(days string) (string, error) {
	if strings.TrimSpace(days) == "" {
		return defaultDaysOfWeek, nil
	}
	seen := make(map[int]bool)
	for _, part := range strings.Split(days, ",") {
		day, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || day < 0 || day > 6 {
			return "", fmt.Errorf("%w: days_of_week must contain values from 0 to 6", ErrInvalidSchedule)
		}
		seen[day] = true
	}
	values := make([]int, 0, len(seen))
	for day := range seen {
		values = append(values, day)
	}
	sort.Ints(values)
	parts := make([]string, 0, len(values))
	for _, day := range values {
		parts = append(parts, fmt.Sprintf("%d", day))
	}
	return strings.Join(parts, ","), nil
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

func startAutomationRun(ctx context.Context, repository repositories.AutomationRepository, run *domain.AutomationRun) (*domain.AutomationRun, bool, error) {
	if run.ExecutionKey != "" {
		existing, err := repository.FindRunByExecutionKey(ctx, run.BoxID, run.ExecutionKey)
		if err == nil {
			return existing, true, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, err
		}
	}
	if err := repository.SaveRun(ctx, run); err != nil {
		if run.ExecutionKey != "" {
			if existing, findErr := repository.FindRunByExecutionKey(ctx, run.BoxID, run.ExecutionKey); findErr == nil {
				return existing, true, nil
			}
		}
		return nil, false, err
	}
	return run, false, nil
}
