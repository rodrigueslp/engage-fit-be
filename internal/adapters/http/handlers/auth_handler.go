package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/auth"
)

type AuthHandler struct {
	login       auth.LoginUseCase
	currentUser auth.GetCurrentUserUseCase
}

func NewAuthHandler(login auth.LoginUseCase, currentUser auth.GetCurrentUserUseCase) AuthHandler {
	return AuthHandler{login: login, currentUser: currentUser}
}

func (h AuthHandler) Login(c *gin.Context) {
	var request dto.LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}

	output, err := h.login.Execute(c.Request.Context(), auth.LoginInput{
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, dto.LoginResponse{AccessToken: output.AccessToken})
}

func (h AuthHandler) Logout(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func (h AuthHandler) Me(c *gin.Context) {
	userID, err := middleware.UserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "missing user context"})
		return
	}

	user, err := h.currentUser.Execute(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "user not found"})
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
