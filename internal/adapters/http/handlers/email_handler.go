package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/email"
	"boxengage/backend/internal/domain"
)

type EmailHandler struct {
	getSettings    email.GetSettingsUseCase
	updateSettings email.UpdateSettingsUseCase
	testSettings   email.TestSettingsUseCase
	listTemplates  email.ListEmailTemplatesUseCase
	createTemplate email.CreateEmailTemplateUseCase
	getTemplate    email.GetEmailTemplateUseCase
	updateTemplate email.UpdateEmailTemplateUseCase
	deleteTemplate email.DeleteEmailTemplateUseCase
	listCampaigns  email.ListEmailCampaignsUseCase
	createCampaign email.CreateEmailCampaignUseCase
	getCampaign    email.GetEmailCampaignUseCase
	sendCampaign   email.SendEmailCampaignUseCase
	listRecipients email.ListEmailRecipientsUseCase
}

func NewEmailHandler(getSettings email.GetSettingsUseCase, updateSettings email.UpdateSettingsUseCase, testSettings email.TestSettingsUseCase, listTemplates email.ListEmailTemplatesUseCase, createTemplate email.CreateEmailTemplateUseCase, getTemplate email.GetEmailTemplateUseCase, updateTemplate email.UpdateEmailTemplateUseCase, deleteTemplate email.DeleteEmailTemplateUseCase, listCampaigns email.ListEmailCampaignsUseCase, createCampaign email.CreateEmailCampaignUseCase, getCampaign email.GetEmailCampaignUseCase, sendCampaign email.SendEmailCampaignUseCase, listRecipients email.ListEmailRecipientsUseCase) EmailHandler {
	return EmailHandler{getSettings: getSettings, updateSettings: updateSettings, testSettings: testSettings, listTemplates: listTemplates, createTemplate: createTemplate, getTemplate: getTemplate, updateTemplate: updateTemplate, deleteTemplate: deleteTemplate, listCampaigns: listCampaigns, createCampaign: createCampaign, getCampaign: getCampaign, sendCampaign: sendCampaign, listRecipients: listRecipients}
}

func (h EmailHandler) GetSettings(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	settings, err := h.getSettings.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, emailSettingsResponse(*settings))
}

func (h EmailHandler) UpdateSettings(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.EmailSettingsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	password := strings.TrimSpace(request.Password)
	if password == "" {
		if current, err := h.getSettings.Execute(c.Request.Context(), boxID); err == nil {
			password = current.PasswordEncrypted
		}
	}
	now := time.Now()
	settings := domain.EmailSettings{BoxID: boxID, Provider: normalizeEmailProvider(request.Provider), SMTPHost: request.SMTPHost, SMTPPort: request.SMTPPort, Username: request.Username, PasswordEncrypted: password, FromEmail: request.FromEmail, FromName: request.FromName, Enabled: request.Enabled, CreatedAt: now, UpdatedAt: now}
	if settings.SMTPPort == 0 {
		settings.SMTPPort = 587
	}
	if err := h.updateSettings.Execute(c.Request.Context(), &settings); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, emailSettingsResponse(settings))
}

func (h EmailHandler) TestSettings(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.EmailSettingsRequest
	hasDraft := c.ShouldBindJSON(&request) == nil && strings.TrimSpace(request.Provider) != ""
	var settings domain.EmailSettings
	current, err := h.getSettings.Execute(c.Request.Context(), boxID)
	if err == nil {
		settings = *current
	} else if !hasDraft || !errors.Is(err, gorm.ErrRecordNotFound) {
		respondError(c, err)
		return
	} else {
		settings = domain.EmailSettings{BoxID: boxID}
	}
	if hasDraft {
		settings.Provider = normalizeEmailProvider(request.Provider)
		settings.SMTPHost = request.SMTPHost
		settings.SMTPPort = request.SMTPPort
		settings.Username = request.Username
		settings.FromEmail = request.FromEmail
		settings.FromName = request.FromName
		settings.Enabled = request.Enabled
		if strings.TrimSpace(request.Password) != "" {
			settings.PasswordEncrypted = request.Password
		}
		if settings.SMTPPort == 0 {
			settings.SMTPPort = 587
		}
	}
	if err := h.testSettings.Execute(c.Request.Context(), settings); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h EmailHandler) ListTemplates(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	result, err := h.listTemplates.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}
	response := make([]dto.EmailTemplateResponse, 0, len(result))
	for _, template := range result {
		response = append(response, emailTemplateResponse(template))
	}
	c.JSON(http.StatusOK, response)
}

func (h EmailHandler) CreateTemplate(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.EmailTemplateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	now := time.Now()
	template := domain.EmailTemplate{BoxID: boxID, Name: request.Name, Subject: request.Subject, Content: request.Content, CreatedAt: now, UpdatedAt: now}
	if err := h.createTemplate.Execute(c.Request.Context(), &template); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, emailTemplateResponse(template))
}

func (h EmailHandler) GetTemplate(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	template, err := h.getTemplate.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, emailTemplateResponse(*template))
}

func (h EmailHandler) UpdateTemplate(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	template, err := h.getTemplate.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}
	var request dto.EmailTemplateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	template.Name = request.Name
	template.Subject = request.Subject
	template.Content = request.Content
	template.UpdatedAt = time.Now()
	if err := h.updateTemplate.Execute(c.Request.Context(), *template); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, emailTemplateResponse(*template))
}

func (h EmailHandler) DeleteTemplate(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	if err := h.deleteTemplate.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id"))); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h EmailHandler) ListCampaigns(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	result, err := h.listCampaigns.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}
	response := make([]dto.EmailCampaignResponse, 0, len(result))
	for _, campaign := range result {
		response = append(response, emailCampaignResponse(campaign))
	}
	c.JSON(http.StatusOK, response)
}

func (h EmailHandler) CreateCampaign(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.EmailCampaignRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	campaign := domain.EmailCampaign{BoxID: boxID, CampaignID: domain.ID(request.CampaignID), Name: request.Name, Audience: domain.MessageAudience(request.Audience), TemplateID: domain.ID(request.TemplateID), CreatedAt: time.Now()}
	if err := h.createCampaign.Execute(c.Request.Context(), &campaign); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, emailCampaignResponse(campaign))
}

func (h EmailHandler) GetCampaign(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	campaign, err := h.getCampaign.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, emailCampaignResponse(*campaign))
}

func (h EmailHandler) SendCampaign(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	output, err := h.sendCampaign.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SendEmailCampaignResponse{Total: output.Total, Sent: output.Sent, Failed: output.Failed})
}

func (h EmailHandler) PreviewCampaign(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	output, err := h.sendCampaign.Preview(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.EmailCampaignPreviewResponse{Total: output.Total, Subject: output.Subject, Body: output.Body, StudentID: string(output.StudentID), StudentName: output.StudentName, Email: output.Email})
}

func (h EmailHandler) ListRecipients(c *gin.Context) {
	result, err := h.listRecipients.Execute(c.Request.Context(), domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}
	response := make([]dto.EmailRecipientResponse, 0, len(result))
	for _, recipient := range result {
		item := dto.EmailRecipientResponse{ID: string(recipient.ID), EmailCampaignID: string(recipient.EmailCampaignID), StudentID: string(recipient.StudentID), Email: recipient.Email, Status: string(recipient.Status), ErrorMessage: recipient.ErrorMessage, CreatedAt: recipient.CreatedAt.Format("2006-01-02T15:04:05Z07:00")}
		if recipient.SentAt != nil {
			item.SentAt = recipient.SentAt.Format("2006-01-02T15:04:05Z07:00")
		}
		response = append(response, item)
	}
	c.JSON(http.StatusOK, response)
}

func emailSettingsResponse(settings domain.EmailSettings) dto.EmailSettingsResponse {
	updatedAt := ""
	if !settings.UpdatedAt.IsZero() {
		updatedAt = settings.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return dto.EmailSettingsResponse{ID: string(settings.ID), BoxID: string(settings.BoxID), Provider: normalizeEmailProvider(settings.Provider), SMTPHost: settings.SMTPHost, SMTPPort: settings.SMTPPort, Username: settings.Username, FromEmail: settings.FromEmail, FromName: settings.FromName, HasPassword: strings.TrimSpace(settings.PasswordEncrypted) != "", UpdatedAt: updatedAt, Enabled: settings.Enabled}
}

func emailTemplateResponse(template domain.EmailTemplate) dto.EmailTemplateResponse {
	return dto.EmailTemplateResponse{ID: string(template.ID), Name: template.Name, Subject: template.Subject, Content: template.Content}
}

func emailCampaignResponse(campaign domain.EmailCampaign) dto.EmailCampaignResponse {
	response := dto.EmailCampaignResponse{ID: string(campaign.ID), Name: campaign.Name, CampaignID: string(campaign.CampaignID), Audience: string(campaign.Audience), TemplateID: string(campaign.TemplateID)}
	if campaign.SentAt != nil {
		response.SentAt = campaign.SentAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return response
}

func normalizeEmailProvider(provider string) string {
	provider = strings.TrimSpace(strings.ToLower(provider))
	if provider == "" {
		return "smtp"
	}
	return provider
}
