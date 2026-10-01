package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/auth"
	"boxengage/backend/internal/domain"
)

type AuthHandler struct {
	login       auth.LoginUseCase
	currentUser auth.GetCurrentUserUseCase
	password    auth.ChangePasswordUseCase
	logout      auth.LogoutUseCase
	switchUnit  auth.SwitchUnitContextUseCase
	session     middleware.SessionConfig
}

func NewAuthHandler(login auth.LoginUseCase, currentUser auth.GetCurrentUserUseCase, password auth.ChangePasswordUseCase, logout auth.LogoutUseCase, session ...middleware.SessionConfig) AuthHandler {
	config := middleware.SessionConfig{}
	if len(session) > 0 {
		config = session[0]
	}
	return AuthHandler{login: login, currentUser: currentUser, password: password, logout: logout, session: config}
}

func (h AuthHandler) WithUnitContext(useCase auth.SwitchUnitContextUseCase) AuthHandler {
	h.switchUnit = useCase
	return h
}

func (h AuthHandler) ChangePassword(c *gin.Context) {
	userID, err := middleware.UserID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	err = h.password.Execute(c.Request.Context(), auth.ChangePasswordInput{UserID: userID, CurrentPassword: request.CurrentPassword, NewPassword: request.NewPassword})
	if errors.Is(err, auth.ErrInvalidCurrentPassword) || errors.Is(err, auth.ErrInvalidNewPassword) {
		respondPublicError(c, http.StatusBadRequest, "password_invalid", err.Error())
		return
	}
	if err != nil {
		respondError(c, err)
		return
	}
	middleware.ClearSession(c, h.session)
	c.Status(http.StatusNoContent)
}

func (h AuthHandler) Login(c *gin.Context) {
	var request dto.LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondPublicError(c, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}

	output, err := h.login.Execute(c.Request.Context(), auth.LoginInput{
		Email:    request.Email,
		Password: request.Password,
	})
	if errors.Is(err, auth.ErrBoxAccessInactive) {
		respondPublicError(c, http.StatusForbidden, "box_inactive", "academia suspensa ou arquivada; procure o suporte")
		return
	}
	if errors.Is(err, auth.ErrUserAccessInactive) {
		respondPublicError(c, http.StatusForbidden, "user_inactive", "acesso desativado pelo proprietário da academia")
		return
	}
	if errors.Is(err, auth.ErrBillingAccessBlocked) {
		respondPublicError(c, http.StatusPaymentRequired, "billing_access_blocked", "assinatura pendente ou vencida; procure o financeiro")
		return
	}
	if err != nil {
		respondPublicError(c, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
		return
	}
	if err := middleware.SetSession(c, h.session, output.AccessToken); err != nil {
		respondPublicError(c, http.StatusInternalServerError, "session_creation_failed", "could not create session")
		return
	}

	c.JSON(http.StatusOK, dto.LoginResponse{AccessToken: output.AccessToken})
}

func (h AuthHandler) Logout(c *gin.Context) {
	userID, err := middleware.UserID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	if err := h.logout.Execute(c.Request.Context(), userID); err != nil {
		respondError(c, err)
		return
	}
	middleware.ClearSession(c, h.session)
	c.Status(http.StatusNoContent)
}

func (h AuthHandler) Me(c *gin.Context) {
	userID, err := middleware.UserID(c)
	if err != nil {
		respondPublicError(c, http.StatusUnauthorized, "user_context_missing", "missing user context")
		return
	}

	user, err := h.currentUser.Execute(c.Request.Context(), userID)
	if err != nil {
		respondPublicError(c, http.StatusNotFound, "user_not_found", "user not found")
		return
	}

	boxID, boxErr := middleware.BoxID(c)
	if boxErr != nil && user.Role != "PLATFORM_ADMIN" {
		respondUnauthorized(c)
		return
	}
	c.JSON(http.StatusOK, dto.CurrentUserResponse{
		ID:    string(user.ID),
		BoxID: string(boxID),
		Name:  user.Name,
		Email: user.Email,
		Role:  string(user.Role),
	})
}

func (h AuthHandler) SwitchUnitContext(c *gin.Context) {
	userID, err := middleware.UserID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.SwitchUnitContextRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondPublicError(c, http.StatusBadRequest, "invalid_unit_context", "unidade inválida")
		return
	}
	token, err := h.switchUnit.Execute(c.Request.Context(), userID, domain.ID(request.UnitID))
	if errors.Is(err, auth.ErrUnitAccessDenied) {
		respondPublicError(c, http.StatusNotFound, "unit_not_found", "unidade não encontrada")
		return
	}
	if err != nil {
		respondError(c, err)
		return
	}
	if err := middleware.SetSession(c, h.session, token); err != nil {
		respondPublicError(c, http.StatusInternalServerError, "session_creation_failed", "could not create session")
		return
	}
	c.JSON(http.StatusOK, dto.LoginResponse{AccessToken: token})
}
