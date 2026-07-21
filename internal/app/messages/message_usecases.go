package messages

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

const defaultWhatsappTemplateProvider = "twilio"

type ListMessageTemplatesUseCase struct {
	messages repositories.MessageRepository
}

func NewListMessageTemplatesUseCase(messages repositories.MessageRepository) ListMessageTemplatesUseCase {
	return ListMessageTemplatesUseCase{messages: messages}
}

func (uc ListMessageTemplatesUseCase) Execute(ctx context.Context, boxID domain.ID) ([]domain.MessageTemplate, error) {
	stored, err := uc.messages.ListTemplates(ctx, boxID)
	if err != nil {
		return nil, err
	}

	byType := map[domain.MessageTemplateType]domain.MessageTemplate{}
	for _, template := range stored {
		if template.TemplateType != "" {
			byType[template.TemplateType] = template
		}
	}

	result := make([]domain.MessageTemplate, 0, len(domain.OfficialWhatsappTemplates()))
	for _, official := range domain.OfficialWhatsappTemplates() {
		template := byType[official.Type]
		if template.ID == "" {
			template = defaultOfficialTemplate(boxID, official)
		}
		template.Name = official.Label
		template.Content = official.Body
		template.TemplateType = official.Type
		if template.ApprovalStatus == "" {
			template.ApprovalStatus = domain.MessageTemplateNotConfigured
		}
		if template.Provider == "" {
			template.Provider = defaultWhatsappTemplateProvider
		}
		if template.Language == "" {
			template.Language = "pt_BR"
		}
		result = append(result, template)
	}
	return result, nil
}

type CreateMessageTemplateUseCase struct {
	messages repositories.MessageRepository
}

func NewCreateMessageTemplateUseCase(messages repositories.MessageRepository) CreateMessageTemplateUseCase {
	return CreateMessageTemplateUseCase{messages: messages}
}

func (uc CreateMessageTemplateUseCase) Execute(ctx context.Context, template *domain.MessageTemplate) error {
	return errors.New("custom WhatsApp templates are no longer supported; use the official EngageFit template catalog")
}

type GetMessageTemplateUseCase struct {
	messages repositories.MessageRepository
}

func NewGetMessageTemplateUseCase(messages repositories.MessageRepository) GetMessageTemplateUseCase {
	return GetMessageTemplateUseCase{messages: messages}
}

func (uc GetMessageTemplateUseCase) Execute(ctx context.Context, boxID, templateID domain.ID) (*domain.MessageTemplate, error) {
	if official, ok := domain.OfficialWhatsappTemplateByType(domain.MessageTemplateType(templateID)); ok {
		return uc.officialByType(ctx, boxID, official.Type)
	}

	template, err := uc.messages.FindTemplateByID(ctx, boxID, templateID)
	if err != nil {
		return nil, err
	}
	if template.TemplateType == "" {
		return nil, errors.New("legacy custom WhatsApp template content is deprecated and not editable")
	}
	return uc.mergeOfficial(ctx, boxID, *template)
}

func (uc GetMessageTemplateUseCase) officialByType(ctx context.Context, boxID domain.ID, templateType domain.MessageTemplateType) (*domain.MessageTemplate, error) {
	template, err := uc.messages.FindTemplateByType(ctx, boxID, templateType)
	if err != nil {
		official, _ := domain.OfficialWhatsappTemplateByType(templateType)
		defaultTemplate := defaultOfficialTemplate(boxID, official)
		return &defaultTemplate, nil
	}
	return uc.mergeOfficial(ctx, boxID, *template)
}

func (uc GetMessageTemplateUseCase) mergeOfficial(ctx context.Context, boxID domain.ID, template domain.MessageTemplate) (*domain.MessageTemplate, error) {
	official, ok := domain.OfficialWhatsappTemplateByType(template.TemplateType)
	if !ok {
		return nil, fmt.Errorf("unsupported WhatsApp template type %q", template.TemplateType)
	}
	template.BoxID = boxID
	template.Name = official.Label
	template.Content = official.Body
	if template.Provider == "" {
		template.Provider = defaultWhatsappTemplateProvider
	}
	if template.ApprovalStatus == "" {
		template.ApprovalStatus = domain.MessageTemplateNotConfigured
	}
	if template.Language == "" {
		template.Language = "pt_BR"
	}
	return &template, nil
}

type UpdateMessageTemplateUseCase struct {
	messages repositories.MessageRepository
}

func NewUpdateMessageTemplateUseCase(messages repositories.MessageRepository) UpdateMessageTemplateUseCase {
	return UpdateMessageTemplateUseCase{messages: messages}
}

func (uc UpdateMessageTemplateUseCase) Execute(ctx context.Context, template domain.MessageTemplate) error {
	return errors.New("WhatsApp template body editing is disabled; only official template metadata can be configured")
}

func (uc UpdateMessageTemplateUseCase) ConfigureOfficial(ctx context.Context, boxID domain.ID, templateID domain.ID, contentSID string, approvalStatus domain.MessageTemplateApprovalStatus) (*domain.MessageTemplate, error) {
	templateType := domain.MessageTemplateType(templateID)
	if _, ok := domain.OfficialWhatsappTemplateByType(templateType); !ok {
		stored, err := uc.messages.FindTemplateByID(ctx, boxID, templateID)
		if err != nil {
			return nil, err
		}
		templateType = stored.TemplateType
	}
	if err := domain.ValidateOfficialWhatsappTemplateType(templateType); err != nil {
		return nil, err
	}
	if !validApprovalStatus(approvalStatus) {
		return nil, fmt.Errorf("invalid WhatsApp template approval status %q", approvalStatus)
	}

	official, _ := domain.OfficialWhatsappTemplateByType(templateType)
	template, err := uc.messages.FindTemplateByType(ctx, boxID, templateType)
	if err != nil {
		now := time.Now()
		template = &domain.MessageTemplate{
			BoxID:          boxID,
			TemplateType:   templateType,
			CreatedAt:      now,
			ApprovalStatus: domain.MessageTemplateNotConfigured,
		}
	}

	template.Name = official.Label
	template.Content = official.Body
	template.ContentSID = strings.TrimSpace(contentSID)
	template.Provider = defaultWhatsappTemplateProvider
	template.ApprovalStatus = approvalStatus
	template.Language = "pt_BR"
	template.UpdatedAt = time.Now()
	if template.ID == "" {
		if err := uc.messages.SaveTemplate(ctx, template); err != nil {
			return nil, err
		}
	} else if err := uc.messages.UpdateTemplate(ctx, *template); err != nil {
		return nil, err
	}
	return template, nil
}

type DeleteMessageTemplateUseCase struct {
	messages repositories.MessageRepository
}

func NewDeleteMessageTemplateUseCase(messages repositories.MessageRepository) DeleteMessageTemplateUseCase {
	return DeleteMessageTemplateUseCase{messages: messages}
}

func (uc DeleteMessageTemplateUseCase) Execute(ctx context.Context, boxID, templateID domain.ID) error {
	return errors.New("official WhatsApp templates cannot be deleted")
}

type ListMessageCampaignsUseCase struct {
	messages repositories.MessageRepository
}

func NewListMessageCampaignsUseCase(messages repositories.MessageRepository) ListMessageCampaignsUseCase {
	return ListMessageCampaignsUseCase{messages: messages}
}

func (uc ListMessageCampaignsUseCase) Execute(ctx context.Context, boxID domain.ID) ([]domain.MessageCampaign, error) {
	campaigns, err := uc.messages.ListCampaigns(ctx, boxID)
	if err != nil {
		return nil, err
	}
	for i := range campaigns {
		if campaigns[i].TemplateType == "" {
			campaigns[i].TemplateType = templateTypeForAudience(campaigns[i].Audience)
		}
	}
	return campaigns, nil
}

type CreateMessageCampaignUseCase struct {
	messages repositories.MessageRepository
}

func NewCreateMessageCampaignUseCase(messages repositories.MessageRepository) CreateMessageCampaignUseCase {
	return CreateMessageCampaignUseCase{messages: messages}
}

func (uc CreateMessageCampaignUseCase) Execute(ctx context.Context, campaign *domain.MessageCampaign) error {
	if campaign.TemplateType == "" {
		campaign.TemplateType = domain.MessageTemplateType(campaign.TemplateID)
	}
	if campaign.TemplateType == "" {
		campaign.TemplateType = templateTypeForAudience(campaign.Audience)
	}
	if err := domain.ValidateOfficialWhatsappTemplateType(campaign.TemplateType); err != nil {
		return err
	}
	official, _ := domain.OfficialWhatsappTemplateByType(campaign.TemplateType)
	campaign.Audience = official.Audience
	if strings.TrimSpace(campaign.Name) == "" {
		campaign.Name = official.Label
	}

	template, err := uc.ensureOfficialTemplate(ctx, campaign.BoxID, official)
	if err != nil {
		return err
	}
	campaign.TemplateID = template.ID
	return uc.messages.SaveCampaign(ctx, campaign)
}

func (uc CreateMessageCampaignUseCase) ensureOfficialTemplate(ctx context.Context, boxID domain.ID, official domain.OfficialWhatsappTemplate) (*domain.MessageTemplate, error) {
	template, err := uc.messages.FindTemplateByType(ctx, boxID, official.Type)
	if err == nil {
		return template, nil
	}
	now := time.Now()
	template = &domain.MessageTemplate{
		BoxID:          boxID,
		Name:           official.Label,
		Content:        official.Body,
		TemplateType:   official.Type,
		Provider:       defaultWhatsappTemplateProvider,
		ApprovalStatus: domain.MessageTemplateNotConfigured,
		Language:       "pt_BR",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := uc.messages.SaveTemplate(ctx, template); err != nil {
		return nil, err
	}
	return template, nil
}

type GetMessageCampaignUseCase struct {
	messages repositories.MessageRepository
}

func NewGetMessageCampaignUseCase(messages repositories.MessageRepository) GetMessageCampaignUseCase {
	return GetMessageCampaignUseCase{messages: messages}
}

func (uc GetMessageCampaignUseCase) Execute(ctx context.Context, boxID, campaignID domain.ID) (*domain.MessageCampaign, error) {
	campaign, err := uc.messages.FindCampaignByID(ctx, boxID, campaignID)
	if err != nil {
		return nil, err
	}
	if campaign.TemplateType == "" {
		campaign.TemplateType = templateTypeForAudience(campaign.Audience)
	}
	return campaign, nil
}

type ListMessageRecipientsUseCase struct {
	messages repositories.MessageRepository
}

func NewListMessageRecipientsUseCase(messages repositories.MessageRepository) ListMessageRecipientsUseCase {
	return ListMessageRecipientsUseCase{messages: messages}
}

func (uc ListMessageRecipientsUseCase) Execute(ctx context.Context, boxID, messageCampaignID domain.ID) ([]domain.MessageRecipient, error) {
	if _, err := uc.messages.FindCampaignByID(ctx, boxID, messageCampaignID); err != nil {
		return nil, err
	}
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

type OfficialWhatsappTemplatePreviewOutput struct {
	Type               domain.MessageTemplateType
	Label              string
	Description        string
	Editable           bool
	ApprovalStatus     domain.MessageTemplateApprovalStatus
	ProviderTemplateID string
	Preview            string
}

type SendMessageCampaignUseCase struct {
	messages   repositories.MessageRepository
	boxes      repositories.BoxRepository
	students   repositories.StudentRepository
	checkins   repositories.CheckinRepository
	campaigns  repositories.CampaignRepository
	rewards    repositories.RewardRepository
	settings   services.WhatsappSettingsResolver
	gateway    services.WhatsappGateway
	governance services.MessagingGovernance
}

func NewSendMessageCampaignUseCase(messages repositories.MessageRepository, boxes repositories.BoxRepository, students repositories.StudentRepository, checkins repositories.CheckinRepository, campaigns repositories.CampaignRepository, rewards repositories.RewardRepository, settings services.WhatsappSettingsResolver, gateway services.WhatsappGateway, governance ...services.MessagingGovernance) SendMessageCampaignUseCase {
	uc := SendMessageCampaignUseCase{messages: messages, boxes: boxes, students: students, checkins: checkins, campaigns: campaigns, rewards: rewards, settings: settings, gateway: gateway}
	if len(governance) > 0 {
		uc.governance = governance[0]
	}
	return uc
}

func (uc SendMessageCampaignUseCase) Execute(ctx context.Context, boxID, campaignID domain.ID) (*SendMessageCampaignOutput, error) {
	return uc.ExecuteAs(ctx, boxID, campaignID, "")
}

func (uc SendMessageCampaignUseCase) ExecuteAs(ctx context.Context, boxID, campaignID, requestedByUserID domain.ID) (*SendMessageCampaignOutput, error) {
	messageCampaign, err := uc.messages.FindCampaignByID(ctx, boxID, campaignID)
	if err != nil {
		return nil, err
	}
	template, _, err := uc.templateForCampaign(ctx, boxID, *messageCampaign)
	if err != nil {
		return nil, err
	}
	whatsappSettings, err := uc.settings.Resolve(ctx, boxID)
	if err != nil {
		return nil, err
	}
	applyEffectiveTemplateMetadata(template, *whatsappSettings)
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
	var dispatch *domain.MessageDispatch
	if uc.governance != nil && len(recipients) > 0 {
		dispatch, err = uc.governance.Reserve(ctx, services.MessagingReservationRequest{BoxID: boxID, RequestedByUserID: requestedByUserID, SourceType: "message_campaign", SourceID: campaignID, ConnectionMode: whatsappSettings.ConnectionMode, Recipients: len(recipients)})
		if err != nil {
			return nil, err
		}
		for index := range recipients {
			recipients[index].DispatchID = dispatch.ID
		}
	}
	if err := uc.messages.SaveRecipients(ctx, recipients); err != nil {
		if dispatch != nil {
			_ = uc.governance.Complete(ctx, dispatch.ID, 0, len(recipients))
		}
		return nil, err
	}

	output := &SendMessageCampaignOutput{Total: len(recipients)}
	for _, recipient := range recipients {
		student := audienceByID(audience, recipient.StudentID)
		templateValues := uc.templateValues(ctx, boxID, student, messageCampaign.CampaignID)
		body, err := domain.RenderOfficialWhatsappTemplate(template.TemplateType, templateValues)
		if err == nil {
			err = validateTemplateReady(*template)
		}
		if err == nil {
			err = validateTemplateProvider(*whatsappSettings)
		}
		var sendResult *services.WhatsappSendResult
		if err == nil {
			sendResult, err = uc.gateway.Send(ctx, *whatsappSettings, services.WhatsappMessage{
				Phone:            recipient.Phone,
				Body:             body,
				ContentSID:       template.ContentSID,
				ContentVariables: twilioContentVariables(templateValues),
			})
		}
		sentAt := time.Now()
		recipient.SentAt = &sentAt
		if err != nil {
			recipient.Status = domain.MessageRecipientFailed
			recipient.ErrorMessage = err.Error()
			output.Failed++
		} else {
			recipient.Status = domain.MessageRecipientSent
			if sendResult != nil {
				recipient.ProviderMessageSID = sendResult.ProviderMessageID
				recipient.ProviderStatus = sendResult.InitialStatus
			}
			output.Sent++
			if messageCampaign.Audience == domain.MessageAudienceInactive {
				_ = uc.students.MarkRiskMessageSent(ctx, boxID, student.ID, sentAt)
			}
		}
		if err := uc.messages.UpdateRecipient(ctx, recipient); err != nil {
			if dispatch != nil {
				_ = uc.governance.Complete(ctx, dispatch.ID, output.Sent, len(recipients)-output.Sent)
			}
			return nil, err
		}
	}

	messageCampaign.SentAt = &now
	if messageCampaign.TemplateType == "" {
		messageCampaign.TemplateType = template.TemplateType
	}
	if err := uc.messages.UpdateCampaign(ctx, *messageCampaign); err != nil {
		if dispatch != nil {
			_ = uc.governance.Complete(ctx, dispatch.ID, output.Sent, output.Failed)
		}
		return nil, err
	}
	if dispatch != nil {
		if err := uc.governance.Complete(ctx, dispatch.ID, output.Sent, output.Failed); err != nil {
			return nil, err
		}
	}

	return output, nil
}

func (uc SendMessageCampaignUseCase) Preview(ctx context.Context, boxID, campaignID domain.ID) (*MessageCampaignPreviewOutput, error) {
	messageCampaign, err := uc.messages.FindCampaignByID(ctx, boxID, campaignID)
	if err != nil {
		return nil, err
	}
	template, _, err := uc.templateForCampaign(ctx, boxID, *messageCampaign)
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
	body, err := domain.RenderOfficialWhatsappTemplate(template.TemplateType, templateValues)
	if err != nil {
		return nil, err
	}
	output.Body = body
	output.StudentID = student.ID
	output.StudentName = student.Name
	output.Phone = student.Phone
	return output, nil
}

func (uc SendMessageCampaignUseCase) OfficialPreviews(ctx context.Context, boxID, campaignID domain.ID) ([]OfficialWhatsappTemplatePreviewOutput, error) {
	values, err := uc.previewValues(ctx, boxID, campaignID)
	if err != nil {
		return nil, err
	}
	whatsappSettings, err := uc.settings.ResolveMetadata(ctx, boxID)
	if err != nil {
		return nil, err
	}

	result := make([]OfficialWhatsappTemplatePreviewOutput, 0, len(domain.OfficialWhatsappTemplates()))
	for _, official := range domain.OfficialWhatsappTemplates() {
		preview, err := domain.RenderOfficialWhatsappTemplate(official.Type, values)
		if err != nil {
			return nil, err
		}
		metadata := domain.MessageTemplate{ApprovalStatus: domain.MessageTemplateNotConfigured}
		if stored, err := uc.messages.FindTemplateByType(ctx, boxID, official.Type); err == nil {
			metadata = *stored
		}
		if metadata.ApprovalStatus == "" {
			metadata.ApprovalStatus = domain.MessageTemplateNotConfigured
		}
		applyEffectiveTemplateMetadata(&metadata, *whatsappSettings)
		result = append(result, OfficialWhatsappTemplatePreviewOutput{
			Type:               official.Type,
			Label:              official.Label,
			Description:        official.Description,
			Editable:           false,
			ApprovalStatus:     metadata.ApprovalStatus,
			ProviderTemplateID: metadata.ContentSID,
			Preview:            preview,
		})
	}
	return result, nil
}

func applyEffectiveTemplateMetadata(template *domain.MessageTemplate, settings domain.WhatsappSettings) {
	if settings.ConnectionMode != domain.WhatsappConnectionPlatform {
		return
	}
	contentSID := strings.TrimSpace(settings.ContentSIDs[template.TemplateType])
	if contentSID == "" {
		template.ContentSID = ""
		template.ApprovalStatus = domain.MessageTemplateNotConfigured
		return
	}
	template.ContentSID = contentSID
	template.Provider = defaultWhatsappTemplateProvider
	template.ApprovalStatus = domain.MessageTemplateApproved
}

func (uc SendMessageCampaignUseCase) templateForCampaign(ctx context.Context, boxID domain.ID, messageCampaign domain.MessageCampaign) (*domain.MessageTemplate, domain.OfficialWhatsappTemplate, error) {
	templateType := messageCampaign.TemplateType
	if templateType == "" && messageCampaign.TemplateID != "" {
		if template, err := uc.messages.FindTemplateByID(ctx, boxID, messageCampaign.TemplateID); err == nil {
			templateType = template.TemplateType
		}
	}
	if templateType == "" {
		templateType = templateTypeForAudience(messageCampaign.Audience)
	}
	official, ok := domain.OfficialWhatsappTemplateByType(templateType)
	if !ok {
		return nil, official, fmt.Errorf("message campaign must use one official WhatsApp template type")
	}
	template, err := uc.messages.FindTemplateByType(ctx, boxID, templateType)
	if err != nil {
		defaultTemplate := defaultOfficialTemplate(boxID, official)
		return &defaultTemplate, official, nil
	}
	template.Name = official.Label
	template.Content = official.Body
	template.TemplateType = official.Type
	if template.ApprovalStatus == "" {
		template.ApprovalStatus = domain.MessageTemplateNotConfigured
	}
	return template, official, nil
}

func defaultOfficialTemplate(boxID domain.ID, official domain.OfficialWhatsappTemplate) domain.MessageTemplate {
	return domain.MessageTemplate{
		ID:             domain.ID(official.Type),
		BoxID:          boxID,
		Name:           official.Label,
		Content:        official.Body,
		TemplateType:   official.Type,
		Provider:       defaultWhatsappTemplateProvider,
		ApprovalStatus: domain.MessageTemplateNotConfigured,
		Language:       "pt_BR",
	}
}

func templateTypeForAudience(audience domain.MessageAudience) domain.MessageTemplateType {
	switch audience {
	case domain.MessageAudienceAlmostThere, domain.MessageAudienceNearGoal:
		return domain.MessageTemplateAlmostThere
	case domain.MessageAudienceAchieved:
		return domain.MessageTemplateGoalReached
	case domain.MessageAudienceInactive:
		return domain.MessageTemplateWeMissYou
	default:
		return ""
	}
}

func validateTemplateReady(template domain.MessageTemplate) error {
	if template.ApprovalStatus != domain.MessageTemplateApproved {
		return fmt.Errorf("WhatsApp template %s is not approved/configured: %s", template.TemplateType, template.ApprovalStatus)
	}
	if strings.TrimSpace(template.ContentSID) == "" {
		return fmt.Errorf("WhatsApp template %s has no provider template id/content_sid configured", template.TemplateType)
	}
	return nil
}

func validateTemplateProvider(settings domain.WhatsappSettings) error {
	provider := strings.TrimSpace(strings.ToLower(settings.Provider))
	if provider == "" || provider == "twilio" {
		return nil
	}
	return fmt.Errorf("official proactive WhatsApp templates are currently implemented only for Twilio content templates; provider %s is not supported for this flow", settings.Provider)
}

func validApprovalStatus(status domain.MessageTemplateApprovalStatus) bool {
	switch status {
	case domain.MessageTemplateNotConfigured, domain.MessageTemplatePending, domain.MessageTemplateApproved, domain.MessageTemplateRejected:
		return true
	default:
		return false
	}
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
		return uc.students.List(ctx, boxID, repositories.StudentFilters{ContactableOnly: true})
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
	students, err := uc.students.List(ctx, boxID, repositories.StudentFilters{ContactableOnly: true})
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
		filters := repositories.StudentFilters{CampaignID: &campaign.ID, ContactableOnly: true}
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
			if err != nil || !student.CanContact() {
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
	campaignEndDate   string
	rewardName        string
	currentCheckins   int
	targetCheckins    int
	remainingCheckins int
}

func (uc SendMessageCampaignUseCase) templateValues(ctx context.Context, boxID domain.ID, student domain.Student, campaignID domain.ID) map[string]string {
	templateContext := uc.templateContext(ctx, boxID, student, campaignID)
	return map[string]string{
		"student_name":       student.Name,
		"name":               student.Name,
		"nome":               student.Name,
		"email":              student.Email,
		"phone":              student.Phone,
		"telefone":           student.Phone,
		"source":             string(student.Source),
		"platform":           string(student.Source),
		"platform_name":      string(student.Source),
		"plataforma":         string(student.Source),
		"box_name":           templateContext.boxName,
		"campaign_name":      templateContext.campaignName,
		"campaign_end_date":  templateContext.campaignEndDate,
		"reward_name":        templateContext.rewardName,
		"current_checkins":   strconv.Itoa(templateContext.currentCheckins),
		"checkins":           strconv.Itoa(templateContext.currentCheckins),
		"target_checkins":    strconv.Itoa(templateContext.targetCheckins),
		"goal_checkins":      strconv.Itoa(templateContext.targetCheckins),
		"remaining_checkins": strconv.Itoa(templateContext.remainingCheckins),
		"missing_checkins":   strconv.Itoa(templateContext.remainingCheckins),
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
		"1": values["student_name"],
		"2": values["box_name"],
		"3": values["current_checkins"],
		"4": values["missing_checkins"],
		"5": values["target_checkins"],
		"6": values["reward_name"],
		"7": values["platform_name"],
	}
}

func (uc SendMessageCampaignUseCase) templateContext(ctx context.Context, boxID domain.ID, student domain.Student, campaignID domain.ID) templateContext {
	context := templateContext{rewardName: "recompensa da campanha"}
	if box, err := uc.boxes.FindByID(ctx, boxID); err == nil {
		context.boxName = box.Name
	}

	campaigns, err := uc.campaignScope(ctx, boxID, campaignID)
	if err != nil {
		return context
	}
	for _, campaign := range campaigns {
		context.campaignName = campaign.Name
		context.campaignEndDate = campaign.EndDate.Format("02/01/2006")
		if rewards, err := uc.rewards.ListByCampaign(ctx, boxID, campaign.ID); err == nil && len(rewards) > 0 {
			context.rewardName = rewards[0].Name
		}
		if goals, err := uc.campaigns.ListGoals(ctx, campaign.ID); err == nil {
			for _, goal := range goals {
				if goal.Source == student.Source {
					context.targetCheckins = goal.TargetCheckins
					break
				}
			}
			if context.targetCheckins == 0 && len(goals) > 0 {
				context.targetCheckins = goals[0].TargetCheckins
			}
		}
		progressList, err := uc.campaigns.ListProgress(ctx, campaign.ID)
		if err != nil {
			continue
		}
		for _, progress := range progressList {
			if progress.StudentID != student.ID {
				continue
			}
			context.currentCheckins = progress.CurrentCheckins
			context.targetCheckins = progress.TargetCheckins
			context.remainingCheckins = progress.TargetCheckins - progress.CurrentCheckins
			if context.remainingCheckins < 0 {
				context.remainingCheckins = 0
			}
			return context
		}
	}
	if context.remainingCheckins == 0 && context.targetCheckins > context.currentCheckins {
		context.remainingCheckins = context.targetCheckins - context.currentCheckins
	}
	return context
}

func (uc SendMessageCampaignUseCase) previewValues(ctx context.Context, boxID, campaignID domain.ID) (map[string]string, error) {
	campaign, err := uc.campaigns.FindByID(ctx, boxID, campaignID)
	if err != nil {
		return nil, err
	}
	boxName := "seu box"
	if box, err := uc.boxes.FindByID(ctx, boxID); err == nil && strings.TrimSpace(box.Name) != "" {
		boxName = box.Name
	}
	rewardName := "recompensa da campanha"
	if rewards, err := uc.rewards.ListByCampaign(ctx, boxID, campaign.ID); err == nil && len(rewards) > 0 && strings.TrimSpace(rewards[0].Name) != "" {
		rewardName = rewards[0].Name
	}

	studentName := "Luiz"
	currentCheckins := 8
	targetCheckins := 12
	progressList, _ := uc.campaigns.ListProgress(ctx, campaign.ID)
	if len(progressList) > 0 {
		progress := progressList[0]
		currentCheckins = progress.CurrentCheckins
		targetCheckins = progress.TargetCheckins
		if student, err := uc.students.FindByID(ctx, boxID, progress.StudentID); err == nil && strings.TrimSpace(student.Name) != "" {
			studentName = student.Name
		}
	}
	if targetCheckins == 0 {
		if goals, err := uc.campaigns.ListGoals(ctx, campaign.ID); err == nil && len(goals) > 0 {
			targetCheckins = goals[0].TargetCheckins
		}
	}
	if targetCheckins == 0 {
		targetCheckins = 12
	}
	missingCheckins := targetCheckins - currentCheckins
	if missingCheckins < 0 {
		missingCheckins = 0
	}

	return map[string]string{
		"student_name":      studentName,
		"box_name":          boxName,
		"campaign_name":     campaign.Name,
		"current_checkins":  strconv.Itoa(currentCheckins),
		"target_checkins":   strconv.Itoa(targetCheckins),
		"campaign_end_date": campaign.EndDate.Format("02/01/2006"),
		"reward_name":       rewardName,
		"platform_name":     "Wellhub",
		"missing_checkins":  strconv.Itoa(missingCheckins),
	}, nil
}

func requiresTwilioContentTemplate(settings domain.WhatsappSettings) bool {
	if strings.HasPrefix(strings.ToLower(settings.BaseURL), "mock://") {
		return false
	}
	provider := strings.TrimSpace(strings.ToLower(settings.Provider))
	return provider == "" || provider == "twilio"
}
