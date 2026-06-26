package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/app/boxes"
)

type SetupHandler struct {
	createBox boxes.CreateBoxUseCase
}

func NewSetupHandler(createBox boxes.CreateBoxUseCase) SetupHandler {
	return SetupHandler{createBox: createBox}
}

func (h SetupHandler) CreateOwner(c *gin.Context) {
	var request dto.CreateOwnerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}

	output, err := h.createBox.Execute(c.Request.Context(), boxes.CreateBoxInput{
		Name:       request.BoxName,
		OwnerName:  request.OwnerName,
		OwnerEmail: request.OwnerEmail,
		Password:   request.Password,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "could not create owner"})
		return
	}

	c.JSON(http.StatusCreated, dto.CreateOwnerResponse{
		BoxID:  string(output.Box.ID),
		UserID: string(output.User.ID),
	})
}
