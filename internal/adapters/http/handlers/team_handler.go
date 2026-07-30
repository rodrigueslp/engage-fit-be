package handlers

import (
	"errors"
	"net/http"
	"time"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/team"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type TeamHandler struct {
	service *team.Service
}

func NewTeamHandler(service *team.Service) TeamHandler {
	return TeamHandler{service: service}
}

func (h TeamHandler) List(c *gin.Context) {
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
	response := make([]dto.TeamMemberResponse, 0, len(items))
	for _, item := range items {
		response = append(response, teamMemberResponse(item))
	}
	c.JSON(http.StatusOK, response)
}

func (h TeamHandler) CreateCoach(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.CreateCoachRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	item, err := h.service.CreateCoach(c.Request.Context(), boxID, request.Name, request.Email, request.Password)
	if err != nil {
		switch {
		case errors.Is(err, team.ErrInvalidMember):
			respondBadRequest(c)
		case errors.Is(err, team.ErrEmailInUse):
			respondPublicError(c, http.StatusConflict, "email_in_use", "e-mail já cadastrado")
		default:
			respondError(c, err)
		}
		return
	}
	c.JSON(http.StatusCreated, teamMemberResponse(*item))
}

func (h TeamHandler) UpdateCoach(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.UpdateCoachRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	item, err := h.service.UpdateCoach(c.Request.Context(), boxID, domain.ID(c.Param("id")), request.Name, request.Active)
	if err != nil {
		if errors.Is(err, team.ErrInvalidMember) {
			respondBadRequest(c)
			return
		}
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, teamMemberResponse(*item))
}

func (h TeamHandler) ResetPassword(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.ResetCoachPasswordRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	if err := h.service.ResetCoachPassword(c.Request.Context(), boxID, domain.ID(c.Param("id")), request.Password); err != nil {
		if errors.Is(err, team.ErrInvalidMember) {
			respondBadRequest(c)
			return
		}
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func teamMemberResponse(item domain.User) dto.TeamMemberResponse {
	return dto.TeamMemberResponse{
		ID: string(item.ID), Name: item.Name, Email: item.Email, Role: string(item.Role),
		Active: item.Role == domain.UserRoleOwner || item.Active, CreatedAt: item.CreatedAt.Format(time.RFC3339),
	}
}
