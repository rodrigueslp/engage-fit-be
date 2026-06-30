package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/messages"
	"boxengage/backend/internal/domain"
)

type MessagesHandler struct {
	listTemplates  messages.ListMessageTemplatesUseCase
	createTemplate messages.CreateMessageTemplateUseCase
	getTemplate    messages.GetMessageTemplateUseCase
	updateTemplate messages.UpdateMessageTemplateUseCase
	deleteTemplate messages.DeleteMessageTemplateUseCase
	listCampaigns  messages.ListMessageCampaignsUseCase
	createCampaign messages.CreateMessageCampaignUseCase
	getCampaign    messages.GetMessageCampaignUseCase
	sendCampaign   messages.SendMessageCampaignUseCase
	listRecipients messages.ListMessageRecipientsUseCase
}

func NewMessagesHandler(listTemplates messages.ListMessageTemplatesUseCase, createTemplate messages.CreateMessageTemplateUseCase, getTemplate messages.GetMessageTemplateUseCase, updateTemplate messages.UpdateMessageTemplateUseCase, deleteTemplate messages.DeleteMessageTemplateUseCase, listCampaigns messages.ListMessageCampaignsUseCase, createCampaign messages.CreateMessageCampaignUseCase, getCampaign messages.GetMessageCampaignUseCase, sendCampaign messages.SendMessageCampaignUseCase, listRecipients messages.ListMessageRecipientsUseCase) MessagesHandler {
	return MessagesHandler{listTemplates: listTemplates, createTemplate: createTemplate, getTemplate: getTemplate, updateTemplate: updateTemplate, deleteTemplate: deleteTemplate, listCampaigns: listCampaigns, createCampaign: createCampaign, getCampaign: getCampaign, sendCampaign: sendCampaign, listRecipients: listRecipients}
}

func (h MessagesHandler) ListTemplates(c *gin.Context) {
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

	response := make([]dto.MessageTemplateResponse, 0, len(result))
	for _, template := range result {
		response = append(response, messageTemplateResponse(template))
	}
	c.JSON(http.StatusOK, response)
}

func (h MessagesHandler) CreateTemplate(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	var request dto.MessageTemplateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}

	now := time.Now()
	template := domain.MessageTemplate{
		BoxID:      boxID,
		Name:       request.Name,
		Content:    request.Content,
		ContentSID: request.ContentSID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := h.createTemplate.Execute(c.Request.Context(), &template); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, messageTemplateResponse(template))
}

func (h MessagesHandler) GetTemplate(c *gin.Context) {
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

	c.JSON(http.StatusOK, messageTemplateResponse(*template))
}

func (h MessagesHandler) UpdateTemplate(c *gin.Context) {
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

	var request dto.MessageTemplateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}

	template.Name = request.Name
	template.Content = request.Content
	template.ContentSID = request.ContentSID
	template.UpdatedAt = time.Now()
	if err := h.updateTemplate.Execute(c.Request.Context(), *template); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, messageTemplateResponse(*template))
}
func (h MessagesHandler) DeleteTemplate(c *gin.Context) {
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

func (h MessagesHandler) ListCampaigns(c *gin.Context) {
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

	response := make([]dto.MessageCampaignResponse, 0, len(result))
	for _, campaign := range result {
		item := dto.MessageCampaignResponse{
			ID:         string(campaign.ID),
			Name:       campaign.Name,
			CampaignID: string(campaign.CampaignID),
			Audience:   string(campaign.Audience),
			TemplateID: string(campaign.TemplateID),
		}
		if campaign.SentAt != nil {
			item.SentAt = campaign.SentAt.Format("2006-01-02T15:04:05Z07:00")
		}
		response = append(response, item)
	}
	c.JSON(http.StatusOK, response)
}

func (h MessagesHandler) CreateCampaign(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	var request dto.MessageCampaignRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}

	campaign := domain.MessageCampaign{
		BoxID:      boxID,
		CampaignID: domain.ID(request.CampaignID),
		Name:       request.Name,
		Audience:   domain.MessageAudience(request.Audience),
		TemplateID: domain.ID(request.TemplateID),
		CreatedAt:  time.Now(),
	}
	if err := h.createCampaign.Execute(c.Request.Context(), &campaign); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, messageCampaignResponse(campaign))
}
func (h MessagesHandler) GetCampaign(c *gin.Context) {
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

	c.JSON(http.StatusOK, messageCampaignResponse(*campaign))
}
func (h MessagesHandler) SendCampaign(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	output, err := h.sendCampaign.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		log.Printf("message campaign send failed: box_id=%s campaign_id=%s error=%v", boxID, c.Param("id"), err)
		c.JSON(http.StatusBadGateway, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.SendMessageCampaignResponse{
		Total:  output.Total,
		Sent:   output.Sent,
		Failed: output.Failed,
	})
}

func (h MessagesHandler) PreviewCampaign(c *gin.Context) {
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

	c.JSON(http.StatusOK, dto.MessageCampaignPreviewResponse{
		Total:       output.Total,
		Body:        output.Body,
		StudentID:   string(output.StudentID),
		StudentName: output.StudentName,
		Phone:       output.Phone,
	})
}

func (h MessagesHandler) ListRecipients(c *gin.Context) {
	result, err := h.listRecipients.Execute(c.Request.Context(), domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}

	response := make([]dto.MessageRecipientResponse, 0, len(result))
	for _, recipient := range result {
		item := dto.MessageRecipientResponse{
			ID:                string(recipient.ID),
			MessageCampaignID: string(recipient.MessageCampaignID),
			StudentID:         string(recipient.StudentID),
			Phone:             recipient.Phone,
			Status:            string(recipient.Status),
			ErrorMessage:      recipient.ErrorMessage,
			CreatedAt:         recipient.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		if recipient.SentAt != nil {
			item.SentAt = recipient.SentAt.Format("2006-01-02T15:04:05Z07:00")
		}
		response = append(response, item)
	}
	c.JSON(http.StatusOK, response)
}

func messageTemplateResponse(template domain.MessageTemplate) dto.MessageTemplateResponse {
	return dto.MessageTemplateResponse{
		ID:         string(template.ID),
		Name:       template.Name,
		Content:    template.Content,
		ContentSID: template.ContentSID,
	}
}

func messageCampaignResponse(campaign domain.MessageCampaign) dto.MessageCampaignResponse {
	response := dto.MessageCampaignResponse{
		ID:         string(campaign.ID),
		Name:       campaign.Name,
		CampaignID: string(campaign.CampaignID),
		Audience:   string(campaign.Audience),
		TemplateID: string(campaign.TemplateID),
	}
	if campaign.SentAt != nil {
		response.SentAt = campaign.SentAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return response
}
