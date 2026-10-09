package handlers

import (
	"errors"
	"net/http"

	"boxengage/backend/internal/app/featureflags"
	"github.com/gin-gonic/gin"
)

type FeatureFlagsHandler struct{ service *featureflags.Service }

func NewFeatureFlagsHandler(service *featureflags.Service) FeatureFlagsHandler {
	return FeatureFlagsHandler{service: service}
}

func (h FeatureFlagsHandler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h FeatureFlagsHandler) Set(c *gin.Context) {
	adminID, ok := adminUserID(c)
	if !ok {
		return
	}
	var request struct {
		Enabled *bool  `json:"enabled" binding:"required"`
		Reason  string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || request.Enabled == nil {
		respondBadRequest(c)
		return
	}
	err := h.service.Set(c.Request.Context(), c.Param("key"), *request.Enabled, request.Reason, string(adminID), c.ClientIP())
	if errors.Is(err, featureflags.ErrNotEditable) || errors.Is(err, featureflags.ErrReasonRequired) || errors.Is(err, featureflags.ErrDependencyDisabled) {
		respondPublicError(c, http.StatusBadRequest, "feature_flag_invalid", err.Error())
		return
	}
	if err != nil {
		respondError(c, err)
		return
	}
	h.List(c)
}
