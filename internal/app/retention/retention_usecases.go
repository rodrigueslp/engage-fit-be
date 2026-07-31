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

const (
	retentionHistoryDays     = 56
	minimumRetentionCheckins = 4
	minimumPreviousCheckins  = 4
	attentionDropPercentage  = 25
	atRiskDropPercentage     = 50
	criticalDropPercentage   = 75
	operationalInactiveDays  = 30
)

type Service struct {
	boxes     repositories.BoxRepository
	students  repositories.StudentRepository
	retention repositories.RetentionRepository
	team      repositories.TeamRepository
	now       func() time.Time
}

func NewService(boxes repositories.BoxRepository, students repositories.StudentRepository, retention repositories.RetentionRepository, team repositories.TeamRepository) Service {
	return Service{boxes: boxes, students: students, retention: retention, team: team, now: time.Now}
}

func (s Service) ListRadar(ctx context.Context, boxID domain.ID) ([]domain.RetentionRadarItem, error) {
	box, err := s.boxes.FindByID(ctx, boxID)
	if err != nil {
		return nil, err
	}
	rules := retentionRules(*box, s.now())
	metrics, err := s.retention.ListMetrics(ctx, boxID, rules.RecentStart, rules.PreviousStart, rules.RecentEnd)
	if err != nil {
		return nil, err
	}
	result := make([]domain.RetentionRadarItem, 0, len(metrics))
	for _, metric := range metrics {
		result = append(result, classify(metric, rules.RecentEnd, rules.AtRiskInactiveDays))
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

func (s Service) Rules(ctx context.Context, boxID domain.ID) (*domain.RetentionRules, error) {
	box, err := s.boxes.FindByID(ctx, boxID)
	if err != nil {
		return nil, err
	}
	rules := retentionRules(*box, s.now())
	return &rules, nil
}

func retentionRules(box domain.Box, now time.Time) domain.RetentionRules {
	now = now.UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	inactiveDays := box.RiskInactiveDays
	if inactiveDays <= 0 {
		inactiveDays = 7
	}
	recentStart := today.AddDate(0, 0, -27)
	previousStart := recentStart.AddDate(0, 0, -28)
	return domain.RetentionRules{
		RecentStart: recentStart, RecentEnd: today,
		PreviousStart: previousStart, PreviousEnd: recentStart.AddDate(0, 0, -1),
		HistoryRequiredBefore: today.AddDate(0, 0, -retentionHistoryDays),
		HistoryDays:           retentionHistoryDays, MinimumTotalCheckins: minimumRetentionCheckins,
		MinimumPreviousCheckins: minimumPreviousCheckins,
		AttentionInactiveDays:   int(math.Ceil(float64(inactiveDays) * 0.7)),
		AtRiskInactiveDays:      inactiveDays, CriticalInactiveDays: max(14, inactiveDays*2),
		AttentionDropPercentage: attentionDropPercentage,
		AtRiskDropPercentage:    atRiskDropPercentage, CriticalDropPercentage: criticalDropPercentage,
		OperationalInactiveDays: operationalInactiveDays, BaselineAt: box.RetentionBaselineAt,
	}
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
	if metric.PreviousCheckins >= minimumPreviousCheckins {
		drop := math.Max(0, float64(metric.PreviousCheckins-metric.RecentCheckins)/float64(metric.PreviousCheckins)*100)
		item.DropPercentage = &drop
	}
	if metric.TotalCheckins < minimumRetentionCheckins {
		item.Level = domain.EngagementHistoryInsufficient
		item.Signals = append(item.Signals, domain.EngagementSignal{Code: "routine_insufficient", Message: "Ainda não há 4 presenças para caracterizar uma rotina de frequência."})
	} else if metric.FirstCheckin == nil || metric.FirstCheckin.After(today.AddDate(0, 0, -retentionHistoryDays)) {
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
		case days >= max(14, inactiveDays*2) || (metric.PreviousCheckins >= minimumPreviousCheckins && drop >= criticalDropPercentage):
			item.Level = domain.EngagementCritical
		case days >= inactiveDays || (metric.PreviousCheckins >= minimumPreviousCheckins && drop >= atRiskDropPercentage):
			item.Level = domain.EngagementAtRisk
		case days >= int(math.Ceil(float64(inactiveDays)*0.7)) || (metric.PreviousCheckins >= minimumPreviousCheckins && drop >= attentionDropPercentage):
			item.Level = domain.EngagementAttention
		}
		if days >= inactiveDays {
			item.Signals = append(item.Signals, domain.EngagementSignal{Code: "inactive_days", Message: "Está há vários dias sem registrar presença."})
		}
		if item.DropPercentage != nil && drop >= attentionDropPercentage {
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
	item.Recommendation = recommendation(item)
	return item
}

func recommendation(item domain.RetentionRadarItem) domain.RetentionRecommendation {
	switch item.WorkflowStatus {
	case domain.RetentionWorkflowExcluded:
		return domain.RetentionRecommendation{Code: "retention_excluded", Title: "Fora do monitoramento", Message: "O box decidiu não acompanhar este aluno no radar de retenção. Os check-ins continuam preservados."}
	case domain.RetentionWorkflowHistorical:
		return domain.RetentionRecommendation{Code: "historical_reactivation", Title: "Tratar como reativação", Message: "A última presença ocorreu há mais de 30 dias. Revise o vínculo antes de realizar uma abordagem de reativação."}
	case domain.RetentionWorkflowRecovered:
		return domain.RetentionRecommendation{Code: "acknowledge_return", Title: "Reconhecer o retorno", Message: "Houve uma nova presença após o acompanhamento. Considere reconhecer a retomada sem atribuir causalidade ao contato."}
	case domain.RetentionWorkflowFollowUpDue:
		return domain.RetentionRecommendation{Code: "review_follow_up", Title: "Revisar acompanhamento", Message: "A data combinada para revisão chegou e ainda não foi observado retorno."}
	case domain.RetentionWorkflowWaitingReturn:
		return domain.RetentionRecommendation{Code: "wait_for_review", Title: "Aguardar até a revisão", Message: "Já existe acompanhamento em andamento. Evite uma nova abordagem antes da data informada."}
	case domain.RetentionWorkflowPaused:
		return domain.RetentionRecommendation{Code: "respect_pause", Title: "Respeitar a pausa", Message: "O acompanhamento está pausado até a data registrada."}
	case domain.RetentionWorkflowClosed:
		return domain.RetentionRecommendation{Code: "do_not_contact", Title: "Não realizar nova abordagem", Message: "O caso foi encerrado conforme o resultado registrado."}
	}
	if item.WorkflowStatus == domain.RetentionWorkflowNeedsAction {
		if item.ContactStatus == domain.ContactStatusOptedOut {
			return domain.RetentionRecommendation{Code: "talk_in_person", Title: "Priorizar conversa presencial", Message: "Há um sinal relevante, mas o aluno não autorizou contato eletrônico."}
		}
		if item.Level == domain.EngagementCritical {
			return domain.RetentionRecommendation{Code: "contact_today", Title: "Entrar em contato hoje", Message: "A ausência ou queda de frequência atingiu o nível mais alto das regras atuais."}
		}
		return domain.RetentionRecommendation{Code: "check_context", Title: "Entender o contexto", Message: "Confirme com o aluno se existe alguma mudança de rotina antes que o afastamento aumente."}
	}
	if item.Level == domain.EngagementHistoryInsufficient {
		for _, signal := range item.Signals {
			if signal.Code == "routine_insufficient" {
				return domain.RetentionRecommendation{Code: "observe_routine", Title: "Aguardar formação de rotina", Message: "São necessárias pelo menos quatro presenças antes de interpretar ausência como sinal de retenção."}
			}
		}
		return domain.RetentionRecommendation{Code: "observe_history", Title: "Observar a formação do histórico", Message: "Ainda não há dados suficientes para comparar oito semanas de frequência."}
	}
	return domain.RetentionRecommendation{Code: "no_action", Title: "Nenhuma ação necessária", Message: "As regras atuais não identificaram mudança relevante de frequência."}
}

func workflowStatus(item domain.RetentionRadarItem, today time.Time) (domain.RetentionWorkflowStatus, *time.Time) {
	if item.RetentionMonitoringStatus == domain.RetentionMonitoringExcluded &&
		(item.RetentionExcludedUntil == nil || !dateOnly(*item.RetentionExcludedUntil).Before(today)) {
		return domain.RetentionWorkflowExcluded, item.RetentionExcludedUntil
	}
	if item.LastInterventionID == "" || item.LastInterventionStatus == "cancelled" {
		if needsAttention(item.Level) {
			if item.DaysSinceCheckin != nil && *item.DaysSinceCheckin > operationalInactiveDays {
				return domain.RetentionWorkflowHistorical, nil
			}
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

func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
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

func (s Service) Summary(ctx context.Context, boxID domain.ID, start, end time.Time) (*domain.RetentionSummary, error) {
	if start.IsZero() || end.IsZero() || end.Before(start) || end.Sub(start) > 365*24*time.Hour {
		return nil, ErrInvalidIntervention
	}
	radar, err := s.ListRadar(ctx, boxID)
	if err != nil {
		return nil, err
	}
	interventions, err := s.retention.SummarizeInterventions(ctx, boxID, start, end.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}
	result := &domain.RetentionSummary{
		PeriodStart: start, PeriodEnd: end,
		CompletedInterventions: interventions.CompletedInterventions,
		ReturnWithin3Days:      interventions.ReturnWithin3Days, ReturnWithin7Days: interventions.ReturnWithin7Days,
		ReturnWithin14Days: interventions.ReturnWithin14Days, MedianDaysToReturn: interventions.MedianDaysToReturn,
		Reasons: interventions.Reasons, Channels: interventions.Channels, Outcomes: interventions.Outcomes,
	}
	for _, item := range radar {
		switch item.WorkflowStatus {
		case domain.RetentionWorkflowNeedsAction:
			result.NeedsAction++
		case domain.RetentionWorkflowWaitingReturn:
			result.WaitingReturn++
		case domain.RetentionWorkflowFollowUpDue:
			result.FollowUpDue++
		case domain.RetentionWorkflowRecovered:
			result.Recovered++
		case domain.RetentionWorkflowHistorical:
			result.HistoricalInactive++
		case domain.RetentionWorkflowExcluded:
			result.Excluded++
		}
	}
	return result, nil
}

func (s Service) ListOnboardingJourney(ctx context.Context, boxID domain.ID) ([]domain.OnboardingJourneyItem, error) {
	box, err := s.boxes.FindByID(ctx, boxID)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	metrics, err := s.retention.ListOnboardingMetrics(ctx, boxID, today)
	if err != nil {
		return nil, err
	}
	inactiveDays := box.RiskInactiveDays
	if inactiveDays <= 0 {
		inactiveDays = 7
	}
	result := make([]domain.OnboardingJourneyItem, 0, len(metrics))
	for _, metric := range metrics {
		confidence, eligible := onboardingStartConfidence(metric)
		if !eligible {
			continue
		}
		item := domain.OnboardingJourneyItem{OnboardingMetrics: metric, MembershipStartConfidence: confidence}
		startedAt := time.Date(metric.MembershipStartedAt.Year(), metric.MembershipStartedAt.Month(), metric.MembershipStartedAt.Day(), 0, 0, 0, 0, time.UTC)
		item.Day = int(today.Sub(startedAt).Hours()/24) + 1
		if metric.LastCheckin != nil {
			last := time.Date(metric.LastCheckin.Year(), metric.LastCheckin.Month(), metric.LastCheckin.Day(), 0, 0, 0, 0, time.UTC)
			days := int(today.Sub(last).Hours() / 24)
			item.DaysSinceCheckin = &days
		}
		switch {
		case metric.FirstCheckin == nil:
			item.Status = "no_first_visit"
			item.StatusMessage = "Ainda não há presença registrada desde o início informado."
			item.Recommendation = domain.RetentionRecommendation{Code: "confirm_first_visit", Title: "Confirmar a primeira presença", Message: "Verifique se o início está correto e se o aluno precisa de apoio para realizar a primeira aula."}
		case metric.SecondCheckin == nil && item.Day >= 3:
			item.Status = "needs_second_visit"
			item.StatusMessage = "A primeira presença ocorreu, mas ainda não houve uma segunda visita."
			if metric.ContactStatus == domain.ContactStatusOptedOut {
				item.Recommendation = domain.RetentionRecommendation{Code: "second_visit_in_person", Title: "Conversar presencialmente", Message: "O aluno ainda não formou a segunda presença e não autorizou contato eletrônico."}
			} else {
				item.Recommendation = domain.RetentionRecommendation{Code: "encourage_second_visit", Title: "Incentivar a segunda presença", Message: "Uma segunda visita cedo ajuda a equipe a acompanhar a formação da rotina."}
			}
		case item.DaysSinceCheckin != nil && *item.DaysSinceCheckin >= inactiveDays:
			item.Status = "interrupted"
			item.StatusMessage = "A rotina inicial foi interrompida pelo limite de inatividade atual."
			item.Recommendation = domain.RetentionRecommendation{Code: "understand_early_interruption", Title: "Entender a interrupção", Message: "Confirme o contexto antes que o afastamento se prolongue."}
		case metric.SecondCheckin == nil:
			item.Status = "building_habit"
			item.StatusMessage = "O aluno iniciou recentemente e ainda está formando a rotina."
			item.Recommendation = domain.RetentionRecommendation{Code: "observe_second_visit", Title: "Acompanhar a próxima presença", Message: "Ainda é cedo para sinalizar atraso; observe se ocorrerá uma segunda visita."}
		default:
			item.Status = "on_track"
			item.StatusMessage = "Há pelo menos duas presenças e nenhuma interrupção relevante."
			item.Recommendation = domain.RetentionRecommendation{Code: "reinforce_consistency", Title: "Reforçar a consistência", Message: "Reconheça a continuidade e acompanhe os marcos de 7, 14 e 30 dias."}
		}
		result = append(result, item)
	}
	sort.SliceStable(result, func(i, j int) bool {
		left, right := onboardingPriority(result[i].Status), onboardingPriority(result[j].Status)
		if left != right {
			return left < right
		}
		return result[i].Day > result[j].Day
	})
	return result, nil
}

func onboardingStartConfidence(metric domain.OnboardingMetrics) (domain.MembershipStartConfidence, bool) {
	if metric.MembershipStartedSource == "manual" || metric.MembershipStartedSource == "integration" {
		return domain.MembershipStartConfirmed, true
	}
	if metric.MembershipStartedSource == "first_checkin_inferred" && metric.ObservationDaysBeforeStart >= retentionHistoryDays {
		return domain.MembershipStartProbable, true
	}
	return "", false
}

func onboardingPriority(status string) int {
	switch status {
	case "no_first_visit":
		return 0
	case "interrupted":
		return 1
	case "needs_second_visit":
		return 2
	case "building_habit":
		return 3
	default:
		return 4
	}
}

func (s Service) UpdateMembershipStart(ctx context.Context, boxID, studentID domain.ID, startedAt time.Time) error {
	if _, err := s.students.FindByID(ctx, boxID, studentID); err != nil {
		return err
	}
	now := s.now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	startedAt = time.Date(startedAt.Year(), startedAt.Month(), startedAt.Day(), 0, 0, 0, 0, time.UTC)
	if startedAt.After(today) || startedAt.Before(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)) {
		return ErrInvalidIntervention
	}
	return s.retention.UpdateMembershipStart(ctx, boxID, studentID, startedAt, "manual", now)
}

func (s Service) UpdateMonitoring(ctx context.Context, boxID, studentID, actorUserID domain.ID, status domain.RetentionMonitoringStatus, reason string, excludedUntil *time.Time) error {
	if _, err := s.students.FindByID(ctx, boxID, studentID); err != nil {
		return err
	}
	reason = strings.TrimSpace(reason)
	if status != domain.RetentionMonitoringMonitored && status != domain.RetentionMonitoringExcluded {
		return ErrInvalidIntervention
	}
	if status == domain.RetentionMonitoringExcluded && !validMonitoringReason(reason) {
		return ErrInvalidIntervention
	}
	now := s.now().UTC()
	today := dateOnly(now)
	if excludedUntil != nil {
		value := dateOnly(*excludedUntil)
		if value.Before(today) {
			return ErrInvalidIntervention
		}
		excludedUntil = &value
	}
	if status == domain.RetentionMonitoringMonitored {
		reason, excludedUntil = "", nil
	}
	return s.retention.UpdateMonitoring(ctx, boxID, studentID, actorUserID, status, reason, excludedUntil, now)
}

func validMonitoringReason(value string) bool {
	switch value {
	case "visitor", "former_member", "long_pause", "outside_retention", "other":
		return true
	default:
		return false
	}
}

func (s Service) CreateIntervention(ctx context.Context, item *domain.RetentionIntervention) error {
	if _, err := s.students.FindByID(ctx, item.BoxID, item.StudentID); err != nil {
		return err
	}
	if err := s.validateAssignee(ctx, item.BoxID, item.AssignedToUserID); err != nil {
		return err
	}
	now := s.now()
	item.Channel = strings.TrimSpace(item.Channel)
	item.Status = strings.TrimSpace(item.Status)
	item.Outcome = strings.TrimSpace(item.Outcome)
	item.ReasonCode = strings.TrimSpace(item.ReasonCode)
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

func (s Service) UpdateIntervention(ctx context.Context, boxID, id domain.ID, status, outcome, reasonCode, notes string, plannedFor *time.Time, assignedToUserID domain.ID) (*domain.RetentionIntervention, error) {
	item, err := s.retention.FindIntervention(ctx, boxID, id)
	if err != nil {
		return nil, err
	}
	if err := s.validateAssignee(ctx, boxID, assignedToUserID); err != nil {
		return nil, err
	}
	item.Status, item.Outcome, item.ReasonCode, item.Notes, item.PlannedFor, item.AssignedToUserID = strings.TrimSpace(status), strings.TrimSpace(outcome), strings.TrimSpace(reasonCode), strings.TrimSpace(notes), plannedFor, assignedToUserID
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

func (s Service) validateAssignee(ctx context.Context, boxID, userID domain.ID) error {
	if userID == "" {
		return nil
	}
	if s.team == nil {
		return ErrInvalidIntervention
	}
	user, err := s.team.FindMember(ctx, boxID, userID)
	if err != nil {
		return ErrInvalidIntervention
	}
	if user.Role == domain.UserRoleCoach && !user.Active {
		return ErrInvalidIntervention
	}
	return nil
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
	if item.ReasonCode != "" && item.ReasonCode != "travel" && item.ReasonCode != "schedule" && item.ReasonCode != "financial" && item.ReasonCode != "motivation" && item.ReasonCode != "service" && item.ReasonCode != "health" && item.ReasonCode != "moved" && item.ReasonCode != "unknown" && item.ReasonCode != "other" {
		return false
	}
	return item.Status != "completed" || item.CompletedAt != nil
}
