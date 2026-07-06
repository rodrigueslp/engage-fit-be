package workouts

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

type ListWorkoutsUseCase struct {
	workouts repositories.WorkoutRepository
}

func NewListWorkoutsUseCase(workouts repositories.WorkoutRepository) ListWorkoutsUseCase {
	return ListWorkoutsUseCase{workouts: workouts}
}
func (uc ListWorkoutsUseCase) Execute(ctx context.Context, boxID domain.ID) ([]domain.Workout, error) {
	return uc.workouts.ListWorkouts(ctx, boxID)
}

type GetWorkoutUseCase struct {
	workouts repositories.WorkoutRepository
}

func NewGetWorkoutUseCase(workouts repositories.WorkoutRepository) GetWorkoutUseCase {
	return GetWorkoutUseCase{workouts: workouts}
}
func (uc GetWorkoutUseCase) Execute(ctx context.Context, boxID, id domain.ID) (*domain.Workout, error) {
	return uc.workouts.FindWorkoutByID(ctx, boxID, id)
}

type CreateWorkoutUseCase struct {
	workouts repositories.WorkoutRepository
}

func NewCreateWorkoutUseCase(workouts repositories.WorkoutRepository) CreateWorkoutUseCase {
	return CreateWorkoutUseCase{workouts: workouts}
}
func (uc CreateWorkoutUseCase) Execute(ctx context.Context, workout *domain.Workout) error {
	if workout.Status == "" {
		workout.Status = domain.WorkoutStatusDraft
	}
	now := time.Now()
	if workout.CreatedAt.IsZero() {
		workout.CreatedAt = now
	}
	workout.UpdatedAt = now
	return uc.workouts.SaveWorkout(ctx, workout)
}

type UpdateWorkoutUseCase struct {
	workouts repositories.WorkoutRepository
}

func NewUpdateWorkoutUseCase(workouts repositories.WorkoutRepository) UpdateWorkoutUseCase {
	return UpdateWorkoutUseCase{workouts: workouts}
}
func (uc UpdateWorkoutUseCase) Execute(ctx context.Context, workout domain.Workout) error {
	workout.UpdatedAt = time.Now()
	return uc.workouts.UpdateWorkout(ctx, workout)
}

type DeleteWorkoutUseCase struct {
	workouts repositories.WorkoutRepository
}

func NewDeleteWorkoutUseCase(workouts repositories.WorkoutRepository) DeleteWorkoutUseCase {
	return DeleteWorkoutUseCase{workouts: workouts}
}
func (uc DeleteWorkoutUseCase) Execute(ctx context.Context, boxID, id domain.ID) error {
	return uc.workouts.DeleteWorkout(ctx, boxID, id)
}

type ListWorkoutDraftsUseCase struct {
	workouts repositories.WorkoutRepository
}

func NewListWorkoutDraftsUseCase(workouts repositories.WorkoutRepository) ListWorkoutDraftsUseCase {
	return ListWorkoutDraftsUseCase{workouts: workouts}
}
func (uc ListWorkoutDraftsUseCase) Execute(ctx context.Context, boxID, workoutID domain.ID) ([]domain.WorkoutMessageDraft, error) {
	return uc.workouts.ListDrafts(ctx, boxID, workoutID)
}

type GetWorkoutDraftUseCase struct {
	workouts repositories.WorkoutRepository
}

func NewGetWorkoutDraftUseCase(workouts repositories.WorkoutRepository) GetWorkoutDraftUseCase {
	return GetWorkoutDraftUseCase{workouts: workouts}
}
func (uc GetWorkoutDraftUseCase) Execute(ctx context.Context, boxID, id domain.ID) (*domain.WorkoutMessageDraft, error) {
	return uc.workouts.FindDraftByID(ctx, boxID, id)
}

type UpdateWorkoutDraftUseCase struct {
	workouts repositories.WorkoutRepository
}

func NewUpdateWorkoutDraftUseCase(workouts repositories.WorkoutRepository) UpdateWorkoutDraftUseCase {
	return UpdateWorkoutDraftUseCase{workouts: workouts}
}
func (uc UpdateWorkoutDraftUseCase) Execute(ctx context.Context, boxID, id domain.ID, body string) (*domain.WorkoutMessageDraft, error) {
	draft, err := uc.workouts.FindDraftByID(ctx, boxID, id)
	if err != nil {
		return nil, err
	}
	if draft.Status == domain.WorkoutMessageDraftStatusSent {
		return nil, errors.New("sent drafts cannot be edited")
	}
	draft.ApprovedBody = strings.TrimSpace(body)
	if draft.Status == domain.WorkoutMessageDraftStatusApproved && draft.ApprovedBody == "" {
		draft.Status = domain.WorkoutMessageDraftStatusDraft
		draft.ApprovedAt = nil
	}
	if err := uc.workouts.UpdateDraft(ctx, *draft); err != nil {
		return nil, err
	}
	return draft, nil
}

type ApproveWorkoutDraftUseCase struct {
	workouts repositories.WorkoutRepository
}

func NewApproveWorkoutDraftUseCase(workouts repositories.WorkoutRepository) ApproveWorkoutDraftUseCase {
	return ApproveWorkoutDraftUseCase{workouts: workouts}
}
func (uc ApproveWorkoutDraftUseCase) Execute(ctx context.Context, boxID, id domain.ID, body string) (*domain.WorkoutMessageDraft, error) {
	draft, err := uc.workouts.FindDraftByID(ctx, boxID, id)
	if err != nil {
		return nil, err
	}
	approvedBody := strings.TrimSpace(body)
	if approvedBody == "" {
		approvedBody = strings.TrimSpace(draft.ApprovedBody)
	}
	if approvedBody == "" {
		approvedBody = strings.TrimSpace(draft.GeneratedBody)
	}
	if approvedBody == "" {
		return nil, errors.New("approved body is required")
	}
	now := time.Now()
	draft.ApprovedBody = approvedBody
	draft.Status = domain.WorkoutMessageDraftStatusApproved
	draft.ApprovedAt = &now
	if err := uc.workouts.UpdateDraft(ctx, *draft); err != nil {
		return nil, err
	}
	return draft, nil
}

type GenerateWorkoutDraftUseCase struct {
	workouts  repositories.WorkoutRepository
	boxes     repositories.BoxRepository
	students  repositories.StudentRepository
	checkins  repositories.CheckinRepository
	campaigns repositories.CampaignRepository
	generator services.LLMGenerator
}

func NewGenerateWorkoutDraftUseCase(workouts repositories.WorkoutRepository, boxes repositories.BoxRepository, students repositories.StudentRepository, checkins repositories.CheckinRepository, campaigns repositories.CampaignRepository, generator services.LLMGenerator) GenerateWorkoutDraftUseCase {
	return GenerateWorkoutDraftUseCase{workouts: workouts, boxes: boxes, students: students, checkins: checkins, campaigns: campaigns, generator: generator}
}

func (uc GenerateWorkoutDraftUseCase) Execute(ctx context.Context, boxID, workoutID domain.ID, audience domain.MessageAudience, campaignID domain.ID, studentIDs []domain.ID) (*domain.WorkoutMessageDraft, error) {
	workout, err := uc.workouts.FindWorkoutByID(ctx, boxID, workoutID)
	if err != nil {
		return nil, err
	}
	if requiresCampaign(audience) && campaignID == "" {
		return nil, errors.New("campaign_id is required for this audience")
	}
	box, err := uc.boxes.FindByID(ctx, boxID)
	if err != nil {
		return nil, err
	}
	students, err := uc.resolveRecipients(ctx, boxID, audience, campaignID, studentIDs)
	if err != nil {
		return nil, err
	}
	output, err := uc.generator.GenerateWorkoutMessage(ctx, services.WorkoutMessageGenerationInput{
		BoxName:    box.Name,
		Date:       workout.WorkoutDate.Format("2006-01-02"),
		Title:      workout.Title,
		Goal:       workout.Goal,
		Movements:  workout.Movements,
		CoachNotes: workout.CoachNotes,
		RawText:    workout.Movements,
		Audience:   string(audience),
	})
	now := time.Now()
	log := domain.LLMGenerationLog{BoxID: boxID, WorkoutID: workoutID, Provider: "openai", PromptSummary: workout.Title + " / " + string(audience), CreatedAt: now}
	if err != nil {
		log.Status = "failed"
		log.ErrorMessage = err.Error()
		_ = uc.workouts.SaveGenerationLog(ctx, &log)
		return nil, err
	}
	draft := domain.WorkoutMessageDraft{
		BoxID:           boxID,
		WorkoutID:       workoutID,
		CampaignID:      campaignID,
		Audience:        audience,
		GeneratedBody:   output.Body,
		Status:          domain.WorkoutMessageDraftStatusDraft,
		TotalRecipients: countStudentsWithPhone(students),
		GeneratedAt:     now,
	}
	if err := uc.workouts.SaveDraft(ctx, &draft); err != nil {
		return nil, err
	}
	if len(studentIDs) > 0 {
		recipients := workoutRecipientsForStudents(draft.ID, students, now)
		if err := uc.workouts.SaveRecipients(ctx, recipients); err != nil {
			return nil, err
		}
	}
	log.DraftID = draft.ID
	log.Provider = output.Provider
	log.Model = output.Model
	log.Status = "success"
	_ = uc.workouts.SaveGenerationLog(ctx, &log)
	return &draft, nil
}

func (uc GenerateWorkoutDraftUseCase) resolveRecipients(ctx context.Context, boxID domain.ID, audience domain.MessageAudience, campaignID domain.ID, studentIDs []domain.ID) ([]domain.Student, error) {
	if len(studentIDs) > 0 {
		students := make([]domain.Student, 0, len(studentIDs))
		for _, studentID := range studentIDs {
			student, err := uc.students.FindByID(ctx, boxID, studentID)
			if err != nil {
				return nil, err
			}
			students = append(students, *student)
		}
		return students, nil
	}
	sender := audienceResolver{boxes: uc.boxes, students: uc.students, checkins: uc.checkins, campaigns: uc.campaigns}
	return sender.resolve(ctx, boxID, audience, campaignID)
}

type SendWorkoutDraftOutput struct {
	Total  int
	Sent   int
	Failed int
}

type SendWorkoutDraftUseCase struct {
	workouts  repositories.WorkoutRepository
	boxes     repositories.BoxRepository
	students  repositories.StudentRepository
	checkins  repositories.CheckinRepository
	campaigns repositories.CampaignRepository
	settings  repositories.WhatsappSettingsRepository
	gateway   services.WhatsappGateway
}

func NewSendWorkoutDraftUseCase(workouts repositories.WorkoutRepository, boxes repositories.BoxRepository, students repositories.StudentRepository, checkins repositories.CheckinRepository, campaigns repositories.CampaignRepository, settings repositories.WhatsappSettingsRepository, gateway services.WhatsappGateway) SendWorkoutDraftUseCase {
	return SendWorkoutDraftUseCase{workouts: workouts, boxes: boxes, students: students, checkins: checkins, campaigns: campaigns, settings: settings, gateway: gateway}
}

func (uc SendWorkoutDraftUseCase) Execute(ctx context.Context, boxID, draftID domain.ID) (*SendWorkoutDraftOutput, error) {
	draft, err := uc.workouts.FindDraftByID(ctx, boxID, draftID)
	if err != nil {
		return nil, err
	}
	if draft.Status != domain.WorkoutMessageDraftStatusApproved {
		return nil, errors.New("draft must be approved before sending")
	}
	body := strings.TrimSpace(draft.ApprovedBody)
	if body == "" {
		return nil, errors.New("approved body is required")
	}
	whatsappSettings, err := uc.settings.FindByBoxID(ctx, boxID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	recipients, err := uc.workouts.ListRecipients(ctx, draft.ID)
	if err != nil {
		return nil, err
	}
	if len(recipients) == 0 {
		return nil, errors.New("no workout recipients selected")
	}
	output := &SendWorkoutDraftOutput{Total: len(recipients)}
	for _, recipient := range recipients {
		student, err := uc.students.FindByID(ctx, boxID, recipient.StudentID)
		if err != nil {
			return nil, err
		}
		messageBody := personalizeWorkoutMessage(body, *student)
		sentAt := time.Now()
		recipient.SentAt = &sentAt
		if err := uc.gateway.Send(ctx, *whatsappSettings, services.WhatsappMessage{Phone: recipient.Phone, Body: messageBody}); err != nil {
			recipient.Status = domain.MessageRecipientFailed
			recipient.ErrorMessage = err.Error()
			output.Failed++
		} else {
			recipient.Status = domain.MessageRecipientSent
			output.Sent++
		}
		if err := uc.workouts.UpdateRecipient(ctx, recipient); err != nil {
			return nil, err
		}
	}
	draft.Status = domain.WorkoutMessageDraftStatusSent
	draft.SentAt = &now
	draft.TotalRecipients = output.Total
	draft.SentRecipients = output.Sent
	draft.FailedRecipients = output.Failed
	if err := uc.workouts.UpdateDraft(ctx, *draft); err != nil {
		return nil, err
	}
	return output, nil
}

type ListWorkoutRecipientsUseCase struct {
	workouts repositories.WorkoutRepository
}

func NewListWorkoutRecipientsUseCase(workouts repositories.WorkoutRepository) ListWorkoutRecipientsUseCase {
	return ListWorkoutRecipientsUseCase{workouts: workouts}
}
func (uc ListWorkoutRecipientsUseCase) Execute(ctx context.Context, draftID domain.ID) ([]domain.WorkoutMessageRecipient, error) {
	return uc.workouts.ListRecipients(ctx, draftID)
}

type audienceResolver struct {
	boxes     repositories.BoxRepository
	students  repositories.StudentRepository
	checkins  repositories.CheckinRepository
	campaigns repositories.CampaignRepository
}

func (r audienceResolver) resolve(ctx context.Context, boxID domain.ID, audience domain.MessageAudience, campaignID domain.ID) ([]domain.Student, error) {
	switch audience {
	case domain.MessageAudienceAll:
		return r.students.List(ctx, boxID, repositories.StudentFilters{})
	case domain.MessageAudienceInactive:
		return r.inactiveStudents(ctx, boxID)
	case domain.MessageAudienceAlmostThere:
		return r.almostThereStudents(ctx, boxID, campaignID)
	case domain.MessageAudienceNearGoal, domain.MessageAudienceAchieved:
		return r.progressAudience(ctx, boxID, campaignID, audience)
	default:
		return nil, fmt.Errorf("unsupported audience: %s", audience)
	}
}

func (r audienceResolver) inactiveStudents(ctx context.Context, boxID domain.ID) ([]domain.Student, error) {
	students, err := r.students.List(ctx, boxID, repositories.StudentFilters{})
	if err != nil {
		return nil, err
	}
	box, err := r.boxes.FindByID(ctx, boxID)
	if err != nil {
		return nil, err
	}
	inactiveDays := box.RiskInactiveDays
	if inactiveDays <= 0 {
		inactiveDays = 7
	}
	threshold := time.Now().AddDate(0, 0, -inactiveDays)
	now := time.Now()
	result := []domain.Student{}
	for _, student := range students {
		if !canReceiveRiskMessage(student, now, box.RiskMessageCooldownDays) {
			continue
		}
		lastCheckin, err := r.checkins.LastCheckinDate(ctx, boxID, student.ID)
		if err != nil || lastCheckin == nil || lastCheckin.Before(threshold) || lastCheckin.Equal(threshold) {
			result = append(result, student)
		}
	}
	return result, nil
}

func (r audienceResolver) progressAudience(ctx context.Context, boxID, campaignID domain.ID, audience domain.MessageAudience) ([]domain.Student, error) {
	campaign, err := r.campaigns.FindByID(ctx, boxID, campaignID)
	if err != nil {
		return nil, err
	}
	filters := repositories.StudentFilters{CampaignID: &campaign.ID}
	if audience == domain.MessageAudienceNearGoal {
		value := true
		filters.NearGoal = &value
	}
	if audience == domain.MessageAudienceAchieved {
		value := true
		filters.Achieved = &value
	}
	return r.students.List(ctx, boxID, filters)
}

func (r audienceResolver) almostThereStudents(ctx context.Context, boxID, campaignID domain.ID) ([]domain.Student, error) {
	campaign, err := r.campaigns.FindByID(ctx, boxID, campaignID)
	if err != nil {
		return nil, err
	}
	daysLeft := campaignDaysLeft(*campaign, time.Now())
	if daysLeft <= 0 {
		return []domain.Student{}, nil
	}
	progressList, err := r.campaigns.ListProgress(ctx, campaign.ID)
	if err != nil {
		return nil, err
	}
	result := []domain.Student{}
	for _, progress := range progressList {
		if !isAlmostThere(progress, daysLeft) {
			continue
		}
		student, err := r.students.FindByID(ctx, boxID, progress.StudentID)
		if err == nil {
			result = append(result, *student)
		}
	}
	return result, nil
}

func requiresCampaign(audience domain.MessageAudience) bool {
	return audience == domain.MessageAudienceAlmostThere || audience == domain.MessageAudienceNearGoal || audience == domain.MessageAudienceAchieved
}

func personalizeWorkoutMessage(body string, student domain.Student) string {
	firstName := strings.TrimSpace(student.Name)
	if parts := strings.Fields(firstName); len(parts) > 0 {
		firstName = parts[0]
	}
	message := body
	replaced := false
	for _, token := range []string{"{{first_name}}", "{{name}}"} {
		if strings.Contains(message, token) {
			message = strings.ReplaceAll(message, token, firstName)
			replaced = true
		}
	}
	if replaced || firstName == "" {
		return message
	}
	replacers := []struct {
		old string
		new string
	}{
		{"Olá, atleta!", "Olá, " + firstName + "!"},
		{"Ola, atleta!", "Ola, " + firstName + "!"},
		{"Olá, aluno!", "Olá, " + firstName + "!"},
		{"Ola, aluno!", "Ola, " + firstName + "!"},
	}
	for _, replacer := range replacers {
		if strings.HasPrefix(message, replacer.old) {
			return strings.Replace(message, replacer.old, replacer.new, 1)
		}
	}
	return firstName + ", " + message
}

func workoutRecipientsForStudents(draftID domain.ID, students []domain.Student, createdAt time.Time) []domain.WorkoutMessageRecipient {
	recipients := make([]domain.WorkoutMessageRecipient, 0, len(students))
	for _, student := range students {
		if strings.TrimSpace(student.Phone) == "" {
			continue
		}
		recipients = append(recipients, domain.WorkoutMessageRecipient{WorkoutMessageDraftID: draftID, StudentID: student.ID, Phone: student.Phone, Status: domain.MessageRecipientPending, CreatedAt: createdAt})
	}
	return recipients
}

func countStudentsWithPhone(students []domain.Student) int {
	count := 0
	for _, student := range students {
		if strings.TrimSpace(student.Phone) != "" {
			count++
		}
	}
	return count
}

func canReceiveRiskMessage(student domain.Student, now time.Time, cooldownDays int) bool {
	switch student.RiskStatus {
	case domain.StudentRiskStatusPaused, domain.StudentRiskStatusNotInterested:
		return false
	}
	if student.RiskLastMessageAt == nil {
		return true
	}
	if cooldownDays <= 0 {
		cooldownDays = 14
	}
	return now.Sub(*student.RiskLastMessageAt) >= time.Duration(cooldownDays)*24*time.Hour
}

func isAlmostThere(progress domain.CampaignProgress, daysLeft int) bool {
	if progress.Achieved || progress.CurrentCheckins >= progress.TargetCheckins {
		return false
	}
	remaining := progress.TargetCheckins - progress.CurrentCheckins
	return remaining >= 1 && remaining <= 3 && remaining <= daysLeft && progress.ProgressPercentage >= 80
}

func campaignDaysLeft(campaign domain.Campaign, now time.Time) int {
	today := dateOnly(now)
	endDate := dateOnly(campaign.EndDate)
	if endDate.Before(today) {
		return 0
	}
	return int(endDate.Sub(today).Hours()/24) + 1
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
