package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/auth"
)

type AuthHandler struct {
	login       auth.LoginUseCase
	currentUser auth.GetCurrentUserUseCase
	password    auth.ChangePasswordUseCase
	logout      auth.LogoutUseCase
	session     middleware.SessionConfig
}

func NewAuthHandler(login auth.LoginUseCase, currentUser auth.GetCurrentUserUseCase, password auth.ChangePasswordUseCase, logout auth.LogoutUseCase, session ...middleware.SessionConfig) AuthHandler {
	config := middleware.SessionConfig{}
	if len(session) > 0 {
		config = session[0]
	}
	return AuthHandler{login: login, currentUser: currentUser, password: password, logout: logout, session: config}
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

	c.JSON(http.StatusOK, dto.CurrentUserResponse{
		ID:    string(user.ID),
		BoxID: string(user.BoxID),
		Name:  user.Name,
		Email: user.Email,
		Role:  string(user.Role),
	})
}
