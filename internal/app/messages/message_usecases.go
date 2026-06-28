package messages

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

type ListMessageTemplatesUseCase struct {
	messages repositories.MessageRepository
}

func NewListMessageTemplatesUseCase(messages repositories.MessageRepository) ListMessageTemplatesUseCase {
	return ListMessageTemplatesUseCase{messages: messages}
}

func (uc ListMessageTemplatesUseCase) Execute(ctx context.Context, boxID domain.ID) ([]domain.MessageTemplate, error) {
	return uc.messages.ListTemplates(ctx, boxID)
}

type CreateMessageTemplateUseCase struct {
	messages repositories.MessageRepository
}

func NewCreateMessageTemplateUseCase(messages repositories.MessageRepository) CreateMessageTemplateUseCase {
	return CreateMessageTemplateUseCase{messages: messages}
}

func (uc CreateMessageTemplateUseCase) Execute(ctx context.Context, template *domain.MessageTemplate) error {
	return uc.messages.SaveTemplate(ctx, template)
}

type GetMessageTemplateUseCase struct {
	messages repositories.MessageRepository
}

func NewGetMessageTemplateUseCase(messages repositories.MessageRepository) GetMessageTemplateUseCase {
	return GetMessageTemplateUseCase{messages: messages}
}

func (uc GetMessageTemplateUseCase) Execute(ctx context.Context, boxID, templateID domain.ID) (*domain.MessageTemplate, error) {
	return uc.messages.FindTemplateByID(ctx, boxID, templateID)
}

type UpdateMessageTemplateUseCase struct {
	messages repositories.MessageRepository
}

func NewUpdateMessageTemplateUseCase(messages repositories.MessageRepository) UpdateMessageTemplateUseCase {
	return UpdateMessageTemplateUseCase{messages: messages}
}

func (uc UpdateMessageTemplateUseCase) Execute(ctx context.Context, template domain.MessageTemplate) error {
	return uc.messages.UpdateTemplate(ctx, template)
}

type DeleteMessageTemplateUseCase struct {
	messages repositories.MessageRepository
}

func NewDeleteMessageTemplateUseCase(messages repositories.MessageRepository) DeleteMessageTemplateUseCase {
	return DeleteMessageTemplateUseCase{messages: messages}
}

func (uc DeleteMessageTemplateUseCase) Execute(ctx context.Context, boxID, templateID domain.ID) error {
	return uc.messages.DeleteTemplate(ctx, boxID, templateID)
}

type ListMessageCampaignsUseCase struct {
	messages repositories.MessageRepository
}

func NewListMessageCampaignsUseCase(messages repositories.MessageRepository) ListMessageCampaignsUseCase {
	return ListMessageCampaignsUseCase{messages: messages}
}

func (uc ListMessageCampaignsUseCase) Execute(ctx context.Context, boxID domain.ID) ([]domain.MessageCampaign, error) {
	return uc.messages.ListCampaigns(ctx, boxID)
}

type CreateMessageCampaignUseCase struct {
	messages repositories.MessageRepository
}

func NewCreateMessageCampaignUseCase(messages repositories.MessageRepository) CreateMessageCampaignUseCase {
	return CreateMessageCampaignUseCase{messages: messages}
}

func (uc CreateMessageCampaignUseCase) Execute(ctx context.Context, campaign *domain.MessageCampaign) error {
	return uc.messages.SaveCampaign(ctx, campaign)
}

type GetMessageCampaignUseCase struct {
	messages repositories.MessageRepository
}

func NewGetMessageCampaignUseCase(messages repositories.MessageRepository) GetMessageCampaignUseCase {
	return GetMessageCampaignUseCase{messages: messages}
}

func (uc GetMessageCampaignUseCase) Execute(ctx context.Context, boxID, campaignID domain.ID) (*domain.MessageCampaign, error) {
	return uc.messages.FindCampaignByID(ctx, boxID, campaignID)
}

type ListMessageRecipientsUseCase struct {
	messages repositories.MessageRepository
}

func NewListMessageRecipientsUseCase(messages repositories.MessageRepository) ListMessageRecipientsUseCase {
	return ListMessageRecipientsUseCase{messages: messages}
}

func (uc ListMessageRecipientsUseCase) Execute(ctx context.Context, messageCampaignID domain.ID) ([]domain.MessageRecipient, error) {
	return uc.messages.ListRecipients(ctx, messageCampaignID)
}

type SendMessageCampaignOutput struct {
	Total  int
	Sent   int
	Failed int
}

type MessageCampaignPreviewOutput struct {
	Total       int
	Body        string
	StudentID   domain.ID
	StudentName string
	Phone       string
}

type SendMessageCampaignUseCase struct {
	messages  repositories.MessageRepository
	boxes     repositories.BoxRepository
	students  repositories.StudentRepository
	checkins  repositories.CheckinRepository
	campaigns repositories.CampaignRepository
	rewards   repositories.RewardRepository
	settings  repositories.WhatsappSettingsRepository
	gateway   services.WhatsappGateway
}

func NewSendMessageCampaignUseCase(messages repositories.MessageRepository, boxes repositories.BoxRepository, students repositories.StudentRepository, checkins repositories.CheckinRepository, campaigns repositories.CampaignRepository, rewards repositories.RewardRepository, settings repositories.WhatsappSettingsRepository, gateway services.WhatsappGateway) SendMessageCampaignUseCase {
	return SendMessageCampaignUseCase{messages: messages, boxes: boxes, students: students, checkins: checkins, campaigns: campaigns, rewards: rewards, settings: settings, gateway: gateway}
}

func (uc SendMessageCampaignUseCase) Execute(ctx context.Context, boxID, campaignID domain.ID) (*SendMessageCampaignOutput, error) {
	messageCampaign, err := uc.messages.FindCampaignByID(ctx, boxID, campaignID)
	if err != nil {
		return nil, err
	}
	template, err := uc.messages.FindTemplateByID(ctx, boxID, messageCampaign.TemplateID)
	if err != nil {
		return nil, err
	}
	whatsappSettings, err := uc.settings.FindByBoxID(ctx, boxID)
	if err != nil {
		return nil, err
	}
	if requiresTwilioContentTemplate(*whatsappSettings) && strings.TrimSpace(template.ContentSID) == "" {
		return nil, fmt.Errorf("twilio whatsapp requires an approved Content SID (HX...) on the template; freeform messages only work within 24h after the recipient replies (Twilio error 63016)")
	}

	audience, err := uc.resolveAudience(ctx, boxID, *messageCampaign)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	recipients := make([]domain.MessageRecipient, 0, len(audience))
	for _, student := range audience {
		if strings.TrimSpace(student.Phone) == "" {
			continue
		}
		recipients = append(recipients, domain.MessageRecipient{
			MessageCampaignID: campaignID,
			StudentID:         student.ID,
			Phone:             student.Phone,
			Status:            domain.MessageRecipientPending,
			CreatedAt:         now,
		})
	}
	if err := uc.messages.SaveRecipients(ctx, recipients); err != nil {
		return nil, err
	}

	output := &SendMessageCampaignOutput{Total: len(recipients)}
	for _, recipient := range recipients {
		student := audienceByID(audience, recipient.StudentID)
		templateValues := uc.templateValues(ctx, boxID, student, messageCampaign.CampaignID)
		body := renderTemplate(template.Content, templateValues)
		err := uc.gateway.Send(ctx, *whatsappSettings, services.WhatsappMessage{
			Phone:            recipient.Phone,
			Body:             body,
			ContentSID:       template.ContentSID,
			ContentVariables: twilioContentVariables(templateValues),
		})
		sentAt := time.Now()
		recipient.SentAt = &sentAt
		if err != nil {
			recipient.Status = domain.MessageRecipientFailed
			recipient.ErrorMessage = err.Error()
			output.Failed++
		} else {
			recipient.Status = domain.MessageRecipientSent
			output.Sent++
			if messageCampaign.Audience == domain.MessageAudienceInactive {
				_ = uc.students.MarkRiskMessageSent(ctx, boxID, student.ID, sentAt)
			}
		}
		if err := uc.messages.UpdateRecipient(ctx, recipient); err != nil {
			return nil, err
		}
	}

	messageCampaign.SentAt = &now
	if err := uc.messages.UpdateCampaign(ctx, *messageCampaign); err != nil {
		return nil, err
	}

	return output, nil
}

func (uc SendMessageCampaignUseCase) Preview(ctx context.Context, boxID, campaignID domain.ID) (*MessageCampaignPreviewOutput, error) {
	messageCampaign, err := uc.messages.FindCampaignByID(ctx, boxID, campaignID)
	if err != nil {
		return nil, err
	}
	template, err := uc.messages.FindTemplateByID(ctx, boxID, messageCampaign.TemplateID)
	if err != nil {
		return nil, err
	}
	audience, err := uc.resolveAudience(ctx, boxID, *messageCampaign)
	if err != nil {
		return nil, err
	}
	output := &MessageCampaignPreviewOutput{Total: len(audience)}
	if len(audience) == 0 {
		return output, nil
	}

	student := firstPreviewStudent(audience)
	templateValues := uc.templateValues(ctx, boxID, student, messageCampaign.CampaignID)
	output.Body = renderTemplate(template.Content, templateValues)
	output.StudentID = student.ID
	output.StudentName = student.Name
	output.Phone = student.Phone
	return output, nil
}

func firstPreviewStudent(students []domain.Student) domain.Student {
	for _, student := range students {
		if strings.TrimSpace(student.Phone) != "" {
			return student
		}
	}
	return students[0]
}

func (uc SendMessageCampaignUseCase) resolveAudience(ctx context.Context, boxID domain.ID, messageCampaign domain.MessageCampaign) ([]domain.Student, error) {
	switch messageCampaign.Audience {
	case domain.MessageAudienceAll:
		return uc.students.List(ctx, boxID, repositories.StudentFilters{})
	case domain.MessageAudienceInactive:
		return uc.inactiveStudents(ctx, boxID)
	case domain.MessageAudienceAlmostThere:
		return uc.almostThereStudents(ctx, boxID, messageCampaign.CampaignID)
	case domain.MessageAudienceNearGoal, domain.MessageAudienceAchieved:
		return uc.progressAudience(ctx, boxID, messageCampaign.CampaignID, messageCampaign.Audience)
	default:
		return []domain.Student{}, nil
	}
}

func (uc SendMessageCampaignUseCase) inactiveStudents(ctx context.Context, boxID domain.ID) ([]domain.Student, error) {
	students, err := uc.students.List(ctx, boxID, repositories.StudentFilters{})
	if err != nil {
		return nil, err
	}
	box, err := uc.boxes.FindByID(ctx, boxID)
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
		lastCheckin, err := uc.checkins.LastCheckinDate(ctx, boxID, student.ID)
		if err != nil || lastCheckin.Before(threshold) || lastCheckin.Equal(threshold) {
			result = append(result, student)
		}
	}
	return result, nil
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

func (uc SendMessageCampaignUseCase) progressAudience(ctx context.Context, boxID, campaignID domain.ID, audience domain.MessageAudience) ([]domain.Student, error) {
	campaigns, err := uc.campaignScope(ctx, boxID, campaignID)
	if err != nil {
		return nil, err
	}
	unique := map[domain.ID]domain.Student{}
	for _, campaign := range campaigns {
		filters := repositories.StudentFilters{CampaignID: &campaign.ID}
		if audience == domain.MessageAudienceNearGoal {
			value := true
			filters.NearGoal = &value
		}
		if audience == domain.MessageAudienceAchieved {
			value := true
			filters.Achieved = &value
		}
		students, err := uc.students.List(ctx, boxID, filters)
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

func (uc SendMessageCampaignUseCase) almostThereStudents(ctx context.Context, boxID, campaignID domain.ID) ([]domain.Student, error) {
	campaigns, err := uc.campaignScope(ctx, boxID, campaignID)
	if err != nil {
		return nil, err
	}

	unique := map[domain.ID]domain.Student{}
	for _, campaign := range campaigns {
		daysLeft := campaignDaysLeft(campaign, time.Now())
		if daysLeft <= 0 {
			continue
		}

		progressList, err := uc.campaigns.ListProgress(ctx, campaign.ID)
		if err != nil {
			return nil, err
		}

		for _, progress := range progressList {
			if !isAlmostThere(progress, daysLeft) {
				continue
			}
			student, err := uc.students.FindByID(ctx, boxID, progress.StudentID)
			if err != nil {
				continue
			}
			unique[student.ID] = *student
		}
	}

	result := make([]domain.Student, 0, len(unique))
	for _, student := range unique {
		result = append(result, student)
	}
	return result, nil
}

func (uc SendMessageCampaignUseCase) campaignScope(ctx context.Context, boxID, campaignID domain.ID) ([]domain.Campaign, error) {
	if campaignID != "" {
		campaign, err := uc.campaigns.FindByID(ctx, boxID, campaignID)
		if err != nil {
			return nil, err
		}
		return []domain.Campaign{*campaign}, nil
	}
	return uc.campaigns.ListActive(ctx, boxID)
}

func isAlmostThere(progress domain.CampaignProgress, daysLeft int) bool {
	if progress.Achieved || progress.CurrentCheckins >= progress.TargetCheckins {
		return false
	}
	remaining := progress.TargetCheckins - progress.CurrentCheckins
	return remaining >= 1 &&
		remaining <= 3 &&
		remaining <= daysLeft &&
		progress.ProgressPercentage >= 80
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

func audienceByID(students []domain.Student, studentID domain.ID) domain.Student {
	for _, student := range students {
		if student.ID == studentID {
			return student
		}
	}
	return domain.Student{}
}

type templateContext struct {
	boxName           string
	campaignName      string
	rewardName        string
	currentCheckins   int
	targetCheckins    int
	remainingCheckins int
}

func (uc SendMessageCampaignUseCase) templateValues(ctx context.Context, boxID domain.ID, student domain.Student, campaignID domain.ID) map[string]string {
	templateContext := uc.templateContext(ctx, boxID, student, campaignID)
	return map[string]string{
		"name":               student.Name,
		"nome":               student.Name,
		"email":              student.Email,
		"phone":              student.Phone,
		"telefone":           student.Phone,
		"source":             string(student.Source),
		"platform":           string(student.Source),
		"plataforma":         string(student.Source),
		"box_name":           templateContext.boxName,
		"campaign_name":      templateContext.campaignName,
		"reward_name":        templateContext.rewardName,
		"current_checkins":   strconv.Itoa(templateContext.currentCheckins),
		"checkins":           strconv.Itoa(templateContext.currentCheckins),
		"target_checkins":    strconv.Itoa(templateContext.targetCheckins),
		"goal_checkins":      strconv.Itoa(templateContext.targetCheckins),
		"remaining_checkins": strconv.Itoa(templateContext.remainingCheckins),
		"faltam_checkins":    strconv.Itoa(templateContext.remainingCheckins),
	}
}

func renderTemplate(content string, values map[string]string) string {
	for key, value := range values {
		content = strings.ReplaceAll(content, "{{"+key+"}}", value)
	}
	return content
}

func twilioContentVariables(values map[string]string) map[string]string {
	return map[string]string{
		"1": values["name"],
		"2": values["box_name"],
		"3": values["current_checkins"],
		"4": values["remaining_checkins"],
		"5": values["target_checkins"],
		"6": values["reward_name"],
		"7": values["platform"],
	}
}

func (uc SendMessageCampaignUseCase) templateContext(ctx context.Context, boxID domain.ID, student domain.Student, campaignID domain.ID) templateContext {
	context := templateContext{}
	if box, err := uc.boxes.FindByID(ctx, boxID); err == nil {
		context.boxName = box.Name
	}

	campaigns, err := uc.campaignScope(ctx, boxID, campaignID)
	if err != nil {
		return context
	}
	for _, campaign := range campaigns {
		progressList, err := uc.campaigns.ListProgress(ctx, campaign.ID)
		if err != nil {
			continue
		}
		for _, progress := range progressList {
			if progress.StudentID != student.ID {
				continue
			}
			context.campaignName = campaign.Name
			context.currentCheckins = progress.CurrentCheckins
			context.targetCheckins = progress.TargetCheckins
			context.remainingCheckins = progress.TargetCheckins - progress.CurrentCheckins
			if context.remainingCheckins < 0 {
				context.remainingCheckins = 0
			}
			if rewards, err := uc.rewards.ListByCampaign(ctx, campaign.ID); err == nil && len(rewards) > 0 {
				context.rewardName = rewards[0].Name
			}
			return context
		}
	}
	return context
}

func requiresTwilioContentTemplate(settings domain.WhatsappSettings) bool {
	if strings.HasPrefix(strings.ToLower(settings.BaseURL), "mock://") {
		return false
	}
	provider := strings.TrimSpace(strings.ToLower(settings.Provider))
	return provider == "" || provider == "twilio"
}
