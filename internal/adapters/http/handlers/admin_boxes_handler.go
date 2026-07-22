package handlers

import (
	"errors"
	"net/http"
	"time"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/auth"
	"boxengage/backend/internal/app/boxes"
	"boxengage/backend/internal/app/platformadmin"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type AdminBoxesHandler struct {
	useCases platformadmin.BoxAdminUseCases
}

func NewAdminBoxesHandler(useCases platformadmin.BoxAdminUseCases) AdminBoxesHandler {
	return AdminBoxesHandler{useCases: useCases}
}

func (h AdminBoxesHandler) List(c *gin.Context) {
	items, err := h.useCases.List(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	response := make([]dto.AdminBoxResponse, 0, len(items))
	for _, item := range items {
		response = append(response, adminBoxResponse(item))
	}
	c.JSON(http.StatusOK, response)
}

func (h AdminBoxesHandler) Create(c *gin.Context) {
	adminID, ok := adminUserID(c)
	if !ok {
		return
	}
	var request dto.CreateAdminBoxRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	item, err := h.useCases.Create(c.Request.Context(), platformadmin.CreateAdminBoxInput{
		BoxName: request.BoxName, OwnerName: request.OwnerName, OwnerEmail: request.OwnerEmail, Password: request.Password,
		Reason: request.Reason, AdminUserID: adminID, IPAddress: c.ClientIP(),
	})
	if err != nil {
		respondAdminBoxError(c, err)
		return
	}
	c.JSON(http.StatusCreated, adminBoxResponse(*item))
}

func (h AdminBoxesHandler) Update(c *gin.Context) {
	adminID, ok := adminUserID(c)
	if !ok {
		return
	}
	var request dto.UpdateAdminBoxRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	item, err := h.useCases.Update(c.Request.Context(), platformadmin.UpdateAdminBoxInput{
		BoxID: domain.ID(c.Param("id")), Name: request.Name, Reason: request.Reason, AdminUserID: adminID, IPAddress: c.ClientIP(),
	})
	if err != nil {
		respondAdminBoxError(c, err)
		return
	}
	c.JSON(http.StatusOK, adminBoxResponse(*item))
}

func (h AdminBoxesHandler) Suspend(c *gin.Context)    { h.changeStatus(c, domain.BoxStatusSuspended) }
func (h AdminBoxesHandler) Reactivate(c *gin.Context) { h.changeStatus(c, domain.BoxStatusActive) }
func (h AdminBoxesHandler) Archive(c *gin.Context)    { h.changeStatus(c, domain.BoxStatusArchived) }

func (h AdminBoxesHandler) changeStatus(c *gin.Context, status domain.BoxStatus) {
	adminID, ok := adminUserID(c)
	if !ok {
		return
	}
	var request dto.ChangeAdminBoxStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	item, err := h.useCases.ChangeStatus(c.Request.Context(), platformadmin.ChangeBoxStatusInput{
		BoxID: domain.ID(c.Param("id")), Status: status, Reason: request.Reason, AdminUserID: adminID, IPAddress: c.ClientIP(),
	})
	if err != nil {
		respondAdminBoxError(c, err)
		return
	}
	c.JSON(http.StatusOK, adminBoxResponse(*item))
}

func adminUserID(c *gin.Context) (domain.ID, bool) {
	id, err := middleware.UserID(c)
	if err != nil {
		respondUnauthorized(c)
		return "", false
	}
	return id, true
}

func respondAdminBoxError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, boxes.ErrOwnerEmailAlreadyRegistered):
		respondPublicError(c, http.StatusConflict, "owner_email_already_registered", err.Error())
	case errors.Is(err, boxes.ErrInvalidBoxOnboarding), errors.Is(err, auth.ErrInvalidNewPassword),
		errors.Is(err, platformadmin.ErrBoxNameRequired), errors.Is(err, platformadmin.ErrLifecycleReasonRequired), errors.Is(err, platformadmin.ErrInvalidBoxStatus):
		respondPublicError(c, http.StatusBadRequest, "box_lifecycle_invalid", err.Error())
	case errors.Is(err, platformadmin.ErrInvalidStatusTransition):
		respondPublicError(c, http.StatusConflict, "box_status_transition_invalid", err.Error())
	default:
		respondError(c, err)
	}
}

func adminBoxResponse(item platformadmin.AdminBoxOverview) dto.AdminBoxResponse {
	changedAt := ""
	if item.Box.StatusChangedAt != nil {
		changedAt = item.Box.StatusChangedAt.UTC().Format(time.RFC3339)
	}
	return dto.AdminBoxResponse{
		ID: string(item.Box.ID), Name: item.Box.Name, Status: string(item.Box.EffectiveStatus()), StatusReason: item.Box.StatusReason,
		StatusChangedAt: changedAt, OwnerID: string(item.Owner.ID), OwnerName: item.Owner.Name, OwnerEmail: item.Owner.Email,
		CreatedAt: item.Box.CreatedAt.UTC().Format(time.RFC3339),
	}
}
