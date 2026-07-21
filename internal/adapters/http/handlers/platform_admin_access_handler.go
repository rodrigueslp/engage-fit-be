package handlers

import (
	"errors"
	"net/http"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/auth"
	"boxengage/backend/internal/app/platformadmin"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type PlatformAdminAccessHandler struct {
	resetOwnerPassword platformadmin.ResetOwnerPasswordUseCase
}

func NewPlatformAdminAccessHandler(resetOwnerPassword platformadmin.ResetOwnerPasswordUseCase) PlatformAdminAccessHandler {
	return PlatformAdminAccessHandler{resetOwnerPassword: resetOwnerPassword}
}

func (h PlatformAdminAccessHandler) ResetOwnerPassword(c *gin.Context) {
	adminID, err := middleware.UserID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.ResetOwnerPasswordRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	err = h.resetOwnerPassword.Execute(c.Request.Context(), platformadmin.ResetOwnerPasswordInput{
		BoxID:       domain.ID(c.Param("id")),
		AdminUserID: adminID,
		NewPassword: request.NewPassword,
		Reason:      request.Reason,
		IPAddress:   c.ClientIP(),
	})
	if errors.Is(err, auth.ErrInvalidNewPassword) || errors.Is(err, platformadmin.ErrResetReasonRequired) {
		respondPublicError(c, http.StatusBadRequest, "owner_password_reset_invalid", err.Error())
		return
	}
	if err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
