package email

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

type GetSettingsUseCase struct {
	settings repositories.EmailSettingsRepository
}

func NewGetSettingsUseCase(settings repositories.EmailSettingsRepository) GetSettingsUseCase {
	return GetSettingsUseCase{settings: settings}
}
func (uc GetSettingsUseCase) Execute(ctx context.Context, boxID domain.ID) (*domain.EmailSettings, error) {
	return uc.settings.FindByBoxID(ctx, boxID)
}

type UpdateSettingsUseCase struct {
	settings repositories.EmailSettingsRepository
}

func NewUpdateSettingsUseCase(settings repositories.EmailSettingsRepository) UpdateSettingsUseCase {
	return UpdateSettingsUseCase{settings: settings}
}
func (uc UpdateSettingsUseCase) Execute(ctx context.Context, settings *domain.EmailSettings) error {
	return uc.settings.Upsert(ctx, settings)
}

type TestSettingsUseCase struct{ gateway services.EmailGateway }

func NewTestSettingsUseCase(gateway services.EmailGateway) TestSettingsUseCase {
	return TestSettingsUseCase{gateway: gateway}
}
func (uc TestSettingsUseCase) Execute(ctx context.Context, settings domain.EmailSettings) error {
	return uc.gateway.Test(ctx, settings)
}

type ListEmailTemplatesUseCase struct{ email repositories.EmailRepository }

func NewListEmailTemplatesUseCase(email repositories.EmailRepository) ListEmailTemplatesUseCase {
	return ListEmailTemplatesUseCase{email: email}
}
func (uc ListEmailTemplatesUseCase) Execute(ctx context.Context, boxID domain.ID) ([]domain.EmailTemplate, error) {
	return uc.email.ListTemplates(ctx, boxID)
}

type CreateEmailTemplateUseCase struct{ email repositories.EmailRepository }

func NewCreateEmailTemplateUseCase(email repositories.EmailRepository) CreateEmailTemplateUseCase {
	return CreateEmailTemplateUseCase{email: email}
}
func (uc CreateEmailTemplateUseCase) Execute(ctx context.Context, template *domain.EmailTemplate) error {
	return uc.email.SaveTemplate(ctx, template)
}

type GetEmailTemplateUseCase struct{ email repositories.EmailRepository }

func NewGetEmailTemplateUseCase(email repositories.EmailRepository) GetEmailTemplateUseCase {
	return GetEmailTemplateUseCase{email: email}
}
func (uc GetEmailTemplateUseCase) Execute(ctx context.Context, boxID, templateID domain.ID) (*domain.EmailTemplate, error) {
	return uc.email.FindTemplateByID(ctx, boxID, templateID)
}

type UpdateEmailTemplateUseCase struct{ email repositories.EmailRepository }

func NewUpdateEmailTemplateUseCase(email repositories.EmailRepository) UpdateEmailTemplateUseCase {
	return UpdateEmailTemplateUseCase{email: email}
}
func (uc UpdateEmailTemplateUseCase) Execute(ctx context.Context, template domain.EmailTemplate) error {
	return uc.email.UpdateTemplate(ctx, template)
}

type DeleteEmailTemplateUseCase struct{ email repositories.EmailRepository }

func NewDeleteEmailTemplateUseCase(email repositories.EmailRepository) DeleteEmailTemplateUseCase {
	return DeleteEmailTemplateUseCase{email: email}
}
func (uc DeleteEmailTemplateUseCase) Execute(ctx context.Context, boxID, templateID domain.ID) error {
	return uc.email.DeleteTemplate(ctx, boxID, templateID)
}

type ListEmailCampaignsUseCase struct{ email repositories.EmailRepository }

func NewListEmailCampaignsUseCase(email repositories.EmailRepository) ListEmailCampaignsUseCase {
	return ListEmailCampaignsUseCase{email: email}
}
func (uc ListEmailCampaignsUseCase) Execute(ctx context.Context, boxID domain.ID) ([]domain.EmailCampaign, error) {
	return uc.email.ListCampaigns(ctx, boxID)
}

type CreateEmailCampaignUseCase struct{ email repositories.EmailRepository }

func NewCreateEmailCampaignUseCase(email repositories.EmailRepository) CreateEmailCampaignUseCase {
	return CreateEmailCampaignUseCase{email: email}
}
func (uc CreateEmailCampaignUseCase) Execute(ctx context.Context, campaign *domain.EmailCampaign) error {
	return uc.email.SaveCampaign(ctx, campaign)
}

type GetEmailCampaignUseCase struct{ email repositories.EmailRepository }

func NewGetEmailCampaignUseCase(email repositories.EmailRepository) GetEmailCampaignUseCase {
	return GetEmailCampaignUseCase{email: email}
}
func (uc GetEmailCampaignUseCase) Execute(ctx context.Context, boxID, campaignID domain.ID) (*domain.EmailCampaign, error) {
	return uc.email.FindCampaignByID(ctx, boxID, campaignID)
}

type ListEmailRecipientsUseCase struct{ email repositories.EmailRepository }

func NewListEmailRecipientsUseCase(email repositories.EmailRepository) ListEmailRecipientsUseCase {
	return ListEmailRecipientsUseCase{email: email}
}
func (uc ListEmailRecipientsUseCase) Execute(ctx context.Context, emailCampaignID domain.ID) ([]domain.EmailRecipient, error) {
	return uc.email.ListRecipients(ctx, emailCampaignID)
}

type SendEmailCampaignOutput struct {
	Total  int
	Sent   int
	Failed int
}
type EmailCampaignPreviewOutput struct {
	Total       int
	Subject     string
	Body        string
	StudentID   domain.ID
	StudentName string
	Email       string
}

type SendEmailCampaignUseCase struct {
	email     repositories.EmailRepository
	boxes     repositories.BoxRepository
	students  repositories.StudentRepository
	checkins  repositories.CheckinRepository
	campaigns repositories.CampaignRepository
	rewards   repositories.RewardRepository
	settings  repositories.EmailSettingsRepository
	gateway   services.EmailGateway
}

func NewSendEmailCampaignUseCase(email repositories.EmailRepository, boxes repositories.BoxRepository, students repositories.StudentRepository, checkins repositories.CheckinRepository, campaigns repositories.CampaignRepository, rewards repositories.RewardRepository, settings repositories.EmailSettingsRepository, gateway services.EmailGateway) SendEmailCampaignUseCase {
	return SendEmailCampaignUseCase{email: email, boxes: boxes, students: students, checkins: checkins, campaigns: campaigns, rewards: rewards, settings: settings, gateway: gateway}
}

func (uc SendEmailCampaignUseCase) Execute(ctx context.Context, boxID, campaignID domain.ID) (*SendEmailCampaignOutput, error) {
	emailCampaign, err := uc.email.FindCampaignByID(ctx, boxID, campaignID)
	if err != nil {
		return nil, err
	}
	template, err := uc.email.FindTemplateByID(ctx, boxID, emailCampaign.TemplateID)
	if err != nil {
		return nil, err
	}
	settings, err := uc.settings.FindByBoxID(ctx, boxID)
	if err != nil {
		return nil, err
	}
	if !settings.Enabled {
		return nil, fmt.Errorf("email provider is disabled")
	}
	audience, err := uc.resolveAudience(ctx, boxID, *emailCampaign)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	recipients := make([]domain.EmailRecipient, 0, len(audience))
	for _, student := range audience {
		if strings.TrimSpace(student.Email) == "" {
			continue
		}
		recipients = append(recipients, domain.EmailRecipient{EmailCampaignID: campaignID, StudentID: student.ID, Email: student.Email, Status: domain.MessageRecipientPending, CreatedAt: now})
	}
	if err := uc.email.SaveRecipients(ctx, recipients); err != nil {
		return nil, err
	}
	output := &SendEmailCampaignOutput{Total: len(recipients)}
	for _, recipient := range recipients {
		student := audienceByID(audience, recipient.StudentID)
		values := uc.templateValues(ctx, boxID, student, emailCampaign.CampaignID)
		message := services.EmailMessage{ToEmail: recipient.Email, Subject: renderTemplate(template.Subject, values), Body: renderTemplate(template.Content, values)}
		err := uc.gateway.Send(ctx, *settings, message)
		sentAt := time.Now()
		recipient.SentAt = &sentAt
		if err != nil {
			recipient.Status = domain.MessageRecipientFailed
			recipient.ErrorMessage = err.Error()
			output.Failed++
			slog.WarnContext(ctx, "email_recipient_send_failed", "box_id", string(boxID), "email_campaign_id", string(campaignID), "student_id", string(recipient.StudentID), "to_email", recipient.Email, "provider", settings.Provider, "smtp_host", settings.SMTPHost, "error", err.Error())
		} else {
			recipient.Status = domain.MessageRecipientSent
			output.Sent++
		}
		if err := uc.email.UpdateRecipient(ctx, recipient); err != nil {
			return nil, err
		}
	}
	emailCampaign.SentAt = &now
	if err := uc.email.UpdateCampaign(ctx, *emailCampaign); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "email_campaign_send_finished", "box_id", string(boxID), "email_campaign_id", string(campaignID), "total", output.Total, "sent", output.Sent, "failed", output.Failed)
	return output, nil
}

func (uc SendEmailCampaignUseCase) Preview(ctx context.Context, boxID, campaignID domain.ID) (*EmailCampaignPreviewOutput, error) {
	emailCampaign, err := uc.email.FindCampaignByID(ctx, boxID, campaignID)
	if err != nil {
		return nil, err
	}
	template, err := uc.email.FindTemplateByID(ctx, boxID, emailCampaign.TemplateID)
	if err != nil {
		return nil, err
	}
	audience, err := uc.resolveAudience(ctx, boxID, *emailCampaign)
	if err != nil {
		return nil, err
	}
	output := &EmailCampaignPreviewOutput{Total: len(audience)}
	if len(audience) == 0 {
		return output, nil
	}
	student := firstPreviewStudent(audience)
	values := uc.templateValues(ctx, boxID, student, emailCampaign.CampaignID)
	output.Subject = renderTemplate(template.Subject, values)
	output.Body = renderTemplate(template.Content, values)
	output.StudentID = student.ID
	output.StudentName = student.Name
	output.Email = student.Email
	return output, nil
}

func firstPreviewStudent(students []domain.Student) domain.Student {
	for _, student := range students {
		if strings.TrimSpace(student.Email) != "" {
			return student
		}
	}
	return students[0]
}

func (uc SendEmailCampaignUseCase) resolveAudience(ctx context.Context, boxID domain.ID, emailCampaign domain.EmailCampaign) ([]domain.Student, error) {
	switch emailCampaign.Audience {
	case domain.MessageAudienceAll:
		return uc.students.List(ctx, boxID, repositories.StudentFilters{})
	case domain.MessageAudienceInactive:
		return uc.inactiveStudents(ctx, boxID)
	case domain.MessageAudienceAlmostThere:
		return uc.almostThereStudents(ctx, boxID, emailCampaign.CampaignID)
	case domain.MessageAudienceNearGoal, domain.MessageAudienceAchieved:
		return uc.progressAudience(ctx, boxID, emailCampaign.CampaignID, emailCampaign.Audience)
	default:
		return []domain.Student{}, nil
	}
}

func (uc SendEmailCampaignUseCase) inactiveStudents(ctx context.Context, boxID domain.ID) ([]domain.Student, error) {
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
	result := []domain.Student{}
	for _, student := range students {
		if student.RiskStatus == domain.StudentRiskStatusPaused || student.RiskStatus == domain.StudentRiskStatusNotInterested {
			continue
		}
		lastCheckin, err := uc.checkins.LastCheckinDate(ctx, boxID, student.ID)
		if err != nil || lastCheckin.Before(threshold) || lastCheckin.Equal(threshold) {
			result = append(result, student)
		}
	}
	return result, nil
}

func (uc SendEmailCampaignUseCase) progressAudience(ctx context.Context, boxID, campaignID domain.ID, audience domain.MessageAudience) ([]domain.Student, error) {
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

func (uc SendEmailCampaignUseCase) almostThereStudents(ctx context.Context, boxID, campaignID domain.ID) ([]domain.Student, error) {
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
			if err == nil {
				unique[student.ID] = *student
			}
		}
	}
	result := make([]domain.Student, 0, len(unique))
	for _, student := range unique {
		result = append(result, student)
	}
	return result, nil
}

func (uc SendEmailCampaignUseCase) campaignScope(ctx context.Context, boxID, campaignID domain.ID) ([]domain.Campaign, error) {
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

func (uc SendEmailCampaignUseCase) templateValues(ctx context.Context, boxID domain.ID, student domain.Student, campaignID domain.ID) map[string]string {
	context := uc.templateContext(ctx, boxID, student, campaignID)
	return map[string]string{
		"name": student.Name, "nome": student.Name, "email": student.Email, "phone": student.Phone, "telefone": student.Phone,
		"source": string(student.Source), "platform": string(student.Source), "plataforma": string(student.Source), "box_name": context.boxName,
		"campaign_name": context.campaignName, "reward_name": context.rewardName, "current_checkins": strconv.Itoa(context.currentCheckins),
		"checkins": strconv.Itoa(context.currentCheckins), "target_checkins": strconv.Itoa(context.targetCheckins), "goal_checkins": strconv.Itoa(context.targetCheckins),
		"remaining_checkins": strconv.Itoa(context.remainingCheckins), "faltam_checkins": strconv.Itoa(context.remainingCheckins),
	}
}

func renderTemplate(content string, values map[string]string) string {
	for key, value := range values {
		content = strings.ReplaceAll(content, "{{"+key+"}}", value)
	}
	return content
}

func (uc SendEmailCampaignUseCase) templateContext(ctx context.Context, boxID domain.ID, student domain.Student, campaignID domain.ID) templateContext {
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
