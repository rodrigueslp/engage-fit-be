package handlers

import (
	"errors"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/contactactivation"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ContactActivationHandler struct {
	service *contactactivation.Service
}

func NewContactActivationHandler(service *contactactivation.Service) ContactActivationHandler {
	return ContactActivationHandler{service: service}
}

func (h ContactActivationHandler) PublicConfig(c *gin.Context) {
	config, err := h.service.PublicConfig(c.Request.Context(), c.Param("code"))
	if err != nil {
		respondContactActivationError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ContactActivationConfigResponse{
		BoxName: config.BoxName, ActivationCode: config.ActivationCode, SenderPhone: config.SenderPhone,
		ConsentVersion: config.ConsentVersion, ConsentText: config.ConsentText,
	})
}

func (h ContactActivationHandler) Start(c *gin.Context) {
	var request dto.StartContactActivationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	var checkinDate *time.Time
	if !request.IsNewStudent {
		parsed, err := time.Parse("2006-01-02", request.RecentCheckinDate)
		if err != nil {
			respondBadRequest(c)
			return
		}
		checkinDate = &parsed
	}
	result, err := h.service.Start(c.Request.Context(), contactactivation.StartInput{
		ActivationCode: c.Param("code"), Name: request.Name, Source: domain.Source(request.Source),
		RecentCheckinDate: checkinDate, IsNewStudent: request.IsNewStudent, ConsentAccepted: request.ConsentAccepted,
	})
	if err != nil {
		respondContactActivationError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.StartContactActivationResponse{
		WhatsappURL: result.WhatsappURL, ExpiresAt: result.ExpiresAt,
	})
}

func (h ContactActivationHandler) Inbound(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		respondBadRequest(c)
		return
	}
	requestURL := inboundRequestURL(c)
	result, err := h.service.HandleInbound(c.Request.Context(), requestURL, c.GetHeader("X-Twilio-Signature"), url.Values(c.Request.PostForm))
	if err != nil {
		if errors.Is(err, contactactivation.ErrUnsupportedMessage) {
			c.Data(http.StatusOK, "application/xml; charset=utf-8", twiml("Para ativar seu acompanhamento, use o QR Code da academia. Para cancelar mensagens, envie SAIR."))
			return
		}
		if errors.Is(err, contactactivation.ErrActivationExpired) {
			c.Data(http.StatusOK, "application/xml; charset=utf-8", twiml("Esse código expirou. Abra novamente o link ou QR Code da academia para gerar uma nova ativação."))
			return
		}
		respondContactActivationError(c, err)
		return
	}
	c.Data(http.StatusOK, "application/xml; charset=utf-8", twiml(result.Message))
}

func (h ContactActivationHandler) Summary(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	summary, err := h.service.AdminSummary(c.Request.Context(), boxID)
	if err != nil {
		respondContactActivationError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ContactActivationSummaryResponse{
		TotalStudents: summary.TotalStudents, WithPhone: summary.WithPhone, OptedIn: summary.OptedIn,
		OptedOut: summary.OptedOut, PendingReview: summary.PendingReview, PendingSync: summary.PendingSync,
		AwaitingMessage: summary.AwaitingMessage, ActivationCode: summary.ActivationCode,
		SenderPhone: summary.SenderPhone, WhatsappReady: summary.WhatsappReady,
	})
}

func (h ContactActivationHandler) List(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	items, err := h.service.List(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}
	response := make([]dto.ContactActivationResponse, 0, len(items))
	for _, item := range items {
		response = append(response, contactActivationResponse(item))
	}
	c.JSON(http.StatusOK, response)
}

func (h ContactActivationHandler) StartForStudent(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	result, err := h.service.StartForStudent(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		respondContactActivationError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.StartContactActivationResponse{
		WhatsappURL: result.WhatsappURL, ExpiresAt: result.ExpiresAt,
	})
}

func (h ContactActivationHandler) Resolve(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.ResolveContactActivationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	item, err := h.service.Resolve(c.Request.Context(), boxID, domain.ID(c.Param("id")), domain.ID(request.StudentID))
	if err != nil {
		respondContactActivationError(c, err)
		return
	}
	c.JSON(http.StatusOK, contactActivationResponse(*item))
}

func contactActivationResponse(item domain.ContactActivationRequest) dto.ContactActivationResponse {
	return dto.ContactActivationResponse{
		ID: string(item.ID), StudentID: string(item.StudentID), StudentName: item.StudentName,
		ClaimedName: item.ClaimedName, Source: string(item.Source), RecentCheckinDate: item.RecentCheckinDate,
		IsNewStudent: item.IsNewStudent,
		Phone:        item.Phone, MatchStrategy: item.MatchStrategy, Status: string(item.Status), ConsentedAt: item.ConsentedAt,
		ExpiresAt: item.ExpiresAt, ResolvedAt: item.ResolvedAt, CreatedAt: item.CreatedAt,
	}
}

func respondContactActivationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, contactactivation.ErrInvalidInput):
		respondPublicError(c, http.StatusBadRequest, "invalid_activation", "Confira os dados informados e tente novamente.")
	case errors.Is(err, contactactivation.ErrWhatsappUnavailable):
		respondPublicError(c, http.StatusConflict, "whatsapp_unavailable", "A ativação pelo WhatsApp ainda não está disponível para esta academia.")
	case errors.Is(err, contactactivation.ErrInvalidSignature):
		respondPublicError(c, http.StatusForbidden, "invalid_signature", "invalid webhook signature")
	case errors.Is(err, gorm.ErrRecordNotFound):
		respondPublicError(c, http.StatusNotFound, "activation_not_found", "Ativação não encontrada.")
	default:
		respondError(c, err)
	}
}

func inboundRequestURL(c *gin.Context) string {
	scheme := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto"))
	if scheme == "" {
		scheme = "https"
		if c.Request.TLS == nil {
			scheme = "http"
		}
	}
	return scheme + "://" + c.Request.Host + c.Request.URL.RequestURI()
}

func twiml(message string) []byte {
	return []byte(`<?xml version="1.0" encoding="UTF-8"?><Response><Message>` + html.EscapeString(message) + `</Message></Response>`)
}
