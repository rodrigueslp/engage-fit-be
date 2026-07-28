package retention

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

var ErrInvalidIntervention = errors.New("invalid retention intervention")

type Service struct {
	boxes     repositories.BoxRepository
	students  repositories.StudentRepository
	retention repositories.RetentionRepository
	now       func() time.Time
}

func NewService(boxes repositories.BoxRepository, students repositories.StudentRepository, retention repositories.RetentionRepository) Service {
	return Service{boxes: boxes, students: students, retention: retention, now: time.Now}
}

func (s Service) ListRadar(ctx context.Context, boxID domain.ID) ([]domain.RetentionRadarItem, error) {
	box, err := s.boxes.FindByID(ctx, boxID)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	recentStart := today.AddDate(0, 0, -27)
	previousStart := recentStart.AddDate(0, 0, -28)
	metrics, err := s.retention.ListMetrics(ctx, boxID, recentStart, previousStart, today)
	if err != nil {
		return nil, err
	}
	inactiveDays := box.RiskInactiveDays
	if inactiveDays <= 0 {
		inactiveDays = 7
	}
	result := make([]domain.RetentionRadarItem, 0, len(metrics))
	for _, metric := range metrics {
		result = append(result, classify(metric, today, inactiveDays))
	}
	sort.SliceStable(result, func(i, j int) bool {
		left, right := levelPriority(result[i].Level), levelPriority(result[j].Level)
		if left != right {
			return left < right
		}
		return strings.ToLower(result[i].StudentName) < strings.ToLower(result[j].StudentName)
	})
	return result, nil
}

func classify(metric domain.RetentionMetrics, today time.Time, inactiveDays int) domain.RetentionRadarItem {
	item := domain.RetentionRadarItem{RetentionMetrics: metric, Level: domain.EngagementHealthy, Signals: []domain.EngagementSignal{}}
	item.RecentWeeklyAverage = float64(metric.RecentCheckins) / 4
	item.PreviousWeeklyAverage = float64(metric.PreviousCheckins) / 4
	if metric.LastCheckin != nil {
		days := int(today.Sub(time.Date(metric.LastCheckin.Year(), metric.LastCheckin.Month(), metric.LastCheckin.Day(), 0, 0, 0, 0, time.UTC)).Hours() / 24)
		if days < 0 {
			days = 0
		}
		item.DaysSinceCheckin = &days
	}
	if metric.PreviousCheckins >= 4 {
		drop := math.Max(0, float64(metric.PreviousCheckins-metric.RecentCheckins)/float64(metric.PreviousCheckins)*100)
		item.DropPercentage = &drop
	}
	if metric.FirstCheckin == nil || metric.FirstCheckin.After(today.AddDate(0, 0, -56)) {
		item.Level = domain.EngagementHistoryInsufficient
		item.Signals = append(item.Signals, domain.EngagementSignal{Code: "history_insufficient", Message: "Ainda não há 8 semanas de histórico para comparar a frequência."})
	} else {
		days := 0
		if item.DaysSinceCheckin != nil {
			days = *item.DaysSinceCheckin
		}
		drop := 0.0
		if item.DropPercentage != nil {
			drop = *item.DropPercentage
		}
		switch {
		case days >= max(14, inactiveDays*2) || (metric.PreviousCheckins >= 4 && drop >= 75):
			item.Level = domain.EngagementCritical
		case days >= inactiveDays || (metric.PreviousCheckins >= 4 && drop >= 50):
			item.Level = domain.EngagementAtRisk
		case days >= int(math.Ceil(float64(inactiveDays)*0.7)) || (metric.PreviousCheckins >= 4 && drop >= 25):
			item.Level = domain.EngagementAttention
		}
		if days >= inactiveDays {
			item.Signals = append(item.Signals, domain.EngagementSignal{Code: "inactive_days", Message: "Está há vários dias sem registrar presença."})
		}
		if item.DropPercentage != nil && drop >= 25 {
			item.Signals = append(item.Signals, domain.EngagementSignal{Code: "frequency_drop", Message: "A frequência caiu em relação às quatro semanas anteriores."})
		}
	}
	if metric.LastCompletedIntervention != nil && metric.FirstReturnAfterAction != nil {
		actionDate := time.Date(metric.LastCompletedIntervention.Year(), metric.LastCompletedIntervention.Month(), metric.LastCompletedIntervention.Day(), 0, 0, 0, 0, time.UTC)
		returnDate := time.Date(metric.FirstReturnAfterAction.Year(), metric.FirstReturnAfterAction.Month(), metric.FirstReturnAfterAction.Day(), 0, 0, 0, 0, time.UTC)
		days := returnDate.Sub(actionDate).Hours() / 24
		actionAge := today.Sub(actionDate).Hours() / 24
		if days >= 0 {
			item.ReturnWithin3Days = days <= 3
			item.ReturnWithin7Days = days <= 7
			item.ReturnWithin14Days = days <= 14
			if item.ReturnWithin14Days && actionAge <= 30 {
				item.Level = domain.EngagementRecovered
				item.Signals = append(item.Signals, domain.EngagementSignal{Code: "returned_after_action", Message: "Houve uma nova presença em até 14 dias após o acompanhamento."})
			}
		}
	}
	item.WorkflowStatus, item.FollowUpDueAt = workflowStatus(item, today)
	return item
}

func workflowStatus(item domain.RetentionRadarItem, today time.Time) (domain.RetentionWorkflowStatus, *time.Time) {
	if item.LastInterventionID == "" || item.LastInterventionStatus == "cancelled" {
		if needsAttention(item.Level) {
			return domain.RetentionWorkflowNeedsAction, nil
		}
		return domain.RetentionWorkflowNone, nil
	}
	if item.LastInterventionOutcome == "not_interested" {
		return domain.RetentionWorkflowClosed, nil
	}
	if item.Level == domain.EngagementRecovered {
		return domain.RetentionWorkflowRecovered, nil
	}
	if item.LastInterventionStatus == "planned" {
		if item.LastInterventionPlannedFor != nil && dueReached(*item.LastInterventionPlannedFor, today) {
			return domain.RetentionWorkflowFollowUpDue, item.LastInterventionPlannedFor
		}
		return domain.RetentionWorkflowWaitingReturn, item.LastInterventionPlannedFor
	}
	if item.LastInterventionOutcome == "paused" {
		if item.LastInterventionPlannedFor == nil || !dueReached(*item.LastInterventionPlannedFor, today) {
			return domain.RetentionWorkflowPaused, item.LastInterventionPlannedFor
		}
		return domain.RetentionWorkflowFollowUpDue, item.LastInterventionPlannedFor
	}
	dueAt := item.LastInterventionPlannedFor
	if dueAt == nil && item.LastCompletedIntervention != nil {
		value := item.LastCompletedIntervention.AddDate(0, 0, 7)
		dueAt = &value
	}
	if dueAt != nil && dueReached(*dueAt, today) {
		return domain.RetentionWorkflowFollowUpDue, dueAt
	}
	return domain.RetentionWorkflowWaitingReturn, dueAt
}

func dueReached(dueAt, today time.Time) bool {
	dueDate := time.Date(dueAt.Year(), dueAt.Month(), dueAt.Day(), 0, 0, 0, 0, time.UTC)
	return !dueDate.After(today)
}

func needsAttention(level domain.EngagementLevel) bool {
	return level == domain.EngagementAttention || level == domain.EngagementAtRisk || level == domain.EngagementCritical
}

func levelPriority(level domain.EngagementLevel) int {
	switch level {
	case domain.EngagementCritical:
		return 0
	case domain.EngagementAtRisk:
		return 1
	case domain.EngagementAttention:
		return 2
	case domain.EngagementHistoryInsufficient:
		return 3
	case domain.EngagementRecovered:
		return 4
	default:
		return 5
	}
}

func (s Service) ListInterventions(ctx context.Context, boxID, studentID domain.ID) ([]domain.RetentionIntervention, error) {
	if _, err := s.students.FindByID(ctx, boxID, studentID); err != nil {
		return nil, err
	}
	return s.retention.ListInterventions(ctx, boxID, studentID)
}

func (s Service) CreateIntervention(ctx context.Context, item *domain.RetentionIntervention) error {
	if _, err := s.students.FindByID(ctx, item.BoxID, item.StudentID); err != nil {
		return err
	}
	now := s.now()
	item.Channel = strings.TrimSpace(item.Channel)
	item.Status = strings.TrimSpace(item.Status)
	item.Outcome = strings.TrimSpace(item.Outcome)
	item.Notes = strings.TrimSpace(item.Notes)
	if item.Status == "" {
		item.Status = "planned"
	}
	if item.Status == "completed" && item.CompletedAt == nil {
		item.CompletedAt = &now
	}
	if !validIntervention(*item) {
		return ErrInvalidIntervention
	}
	item.CreatedAt, item.UpdatedAt = now, now
	return s.retention.SaveIntervention(ctx, item)
}

func (s Service) UpdateIntervention(ctx context.Context, boxID, id domain.ID, status, outcome, notes string, plannedFor *time.Time) (*domain.RetentionIntervention, error) {
	item, err := s.retention.FindIntervention(ctx, boxID, id)
	if err != nil {
		return nil, err
	}
	item.Status, item.Outcome, item.Notes, item.PlannedFor = strings.TrimSpace(status), strings.TrimSpace(outcome), strings.TrimSpace(notes), plannedFor
	now := s.now()
	if item.Status == "completed" && item.CompletedAt == nil {
		item.CompletedAt = &now
	}
	if item.Status != "completed" {
		item.CompletedAt = nil
	}
	if !validIntervention(*item) {
		return nil, ErrInvalidIntervention
	}
	item.UpdatedAt = now
	if err := s.retention.UpdateIntervention(ctx, *item); err != nil {
		return nil, err
	}
	return item, nil
}

func validIntervention(item domain.RetentionIntervention) bool {
	if len(item.Notes) > 500 {
		return false
	}
	if item.Channel != "whatsapp" && item.Channel != "phone" && item.Channel != "in_person" && item.Channel != "other" {
		return false
	}
	if item.Status != "planned" && item.Status != "completed" && item.Status != "cancelled" {
		return false
	}
	if item.Outcome != "" && item.Outcome != "contacted" && item.Outcome != "no_response" && item.Outcome != "follow_up" && item.Outcome != "paused" && item.Outcome != "not_interested" && item.Outcome != "other" {
		return false
	}
	return item.Status != "completed" || item.CompletedAt != nil
}
