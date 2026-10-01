package handlers

import (
	"errors"
	"net/http"

	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/announcements"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type AnnouncementHandler struct{ service *announcements.Service }

func NewAnnouncementHandler(service *announcements.Service) AnnouncementHandler {
	return AnnouncementHandler{service: service}
}

func (h AnnouncementHandler) List(c *gin.Context) {
	unitID, _, ok := enrollmentContext(c)
	if !ok {
		return
	}
	items, err := h.service.List(c.Request.Context(), unitID)
	if err != nil {
		respondAnnouncementError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h AnnouncementHandler) Create(c *gin.Context) {
	unitID, actorID, ok := enrollmentContext(c)
	if !ok {
		return
	}
	var input announcements.SaveInput
	if c.ShouldBindJSON(&input) != nil {
		respondAnnouncementError(c, announcements.ErrInvalidInput)
		return
	}
	item, err := h.service.Create(c.Request.Context(), unitID, actorID, input)
	if err != nil {
		respondAnnouncementError(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h AnnouncementHandler) Update(c *gin.Context) {
	unitID, _, ok := enrollmentContext(c)
	if !ok {
		return
	}
	var input announcements.SaveInput
	if c.ShouldBindJSON(&input) != nil {
		respondAnnouncementError(c, announcements.ErrInvalidInput)
		return
	}
	item, err := h.service.Update(c.Request.Context(), unitID, domain.ID(c.Param("id")), input)
	if err != nil {
		respondAnnouncementError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h AnnouncementHandler) Publish(c *gin.Context) {
	unitID, _, ok := enrollmentContext(c)
	if !ok {
		return
	}
	var input announcements.PublishInput
	if c.ShouldBindJSON(&input) != nil {
		respondAnnouncementError(c, announcements.ErrInvalidInput)
		return
	}
	item, err := h.service.Publish(c.Request.Context(), unitID, domain.ID(c.Param("id")), input)
	if err != nil {
		respondAnnouncementError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h AnnouncementHandler) Archive(c *gin.Context) {
	unitID, _, ok := enrollmentContext(c)
	if !ok {
		return
	}
	if err := h.service.Archive(c.Request.Context(), unitID, domain.ID(c.Param("id"))); err != nil {
		respondAnnouncementError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h AnnouncementHandler) AthleteList(c *gin.Context) {
	athleteID, err := middleware.AthleteID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	items, err := h.service.AthleteList(c.Request.Context(), athleteID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h AnnouncementHandler) MarkViewed(c *gin.Context) {
	athleteID, err := middleware.AthleteID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	if err := h.service.MarkViewed(c.Request.Context(), athleteID, domain.ID(c.Param("id"))); err != nil {
		respondAnnouncementError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func respondAnnouncementError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, announcements.ErrInvalidInput):
		respondPublicError(c, http.StatusUnprocessableEntity, "announcement_invalid", "revise o conteúdo, prioridade, público ou agendamento")
	case errors.Is(err, announcements.ErrNotFound):
		respondPublicError(c, http.StatusNotFound, "announcement_not_found", "comunicado não encontrado")
	case errors.Is(err, announcements.ErrConflict):
		respondPublicError(c, http.StatusConflict, "announcement_conflict", "o comunicado não permite esta operação no estado atual")
	default:
		respondError(c, err)
	}
}
