package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/boxes"
	"boxengage/backend/internal/domain"
)

type BoxesHandler struct {
	getBox    boxes.GetBoxUseCase
	updateBox boxes.UpdateBoxUseCase
}

func NewBoxesHandler(getBox boxes.GetBoxUseCase, updateBox boxes.UpdateBoxUseCase) BoxesHandler {
	return BoxesHandler{getBox: getBox, updateBox: updateBox}
}

func (h BoxesHandler) Get(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	box, err := h.getBox.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, boxResponse(*box))
}

func (h BoxesHandler) Update(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	var request dto.UpdateBoxRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}

	box, err := h.getBox.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}
	box.Name = request.Name
	box.RiskInactiveDays = request.RiskInactiveDays
	box.RiskMessageCooldownDays = request.RiskMessageCooldownDays

	if err := h.updateBox.Execute(c.Request.Context(), *box); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, boxResponse(*box))
}

func boxResponse(box domain.Box) dto.BoxResponse {
	return dto.BoxResponse{
		ID:                      string(box.ID),
		Name:                    box.Name,
		RiskInactiveDays:        box.RiskInactiveDays,
		RiskMessageCooldownDays: box.RiskMessageCooldownDays,
	}
}
