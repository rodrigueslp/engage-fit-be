package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	whatsappadapter "boxengage/backend/internal/adapters/whatsapp"
	"boxengage/backend/internal/app/whatsapp"
	"boxengage/backend/internal/domain"
)

type WhatsappHandler struct {
	getSettings    whatsapp.GetSettingsUseCase
	updateSettings whatsapp.UpdateSettingsUseCase
	testSettings   whatsapp.TestSettingsUseCase
}

func NewWhatsappHandler(getSettings whatsapp.GetSettingsUseCase, updateSettings whatsapp.UpdateSettingsUseCase, testSettings whatsapp.TestSettingsUseCase) WhatsappHandler {
	return WhatsappHandler{getSettings: getSettings, updateSettings: updateSettings, testSettings: testSettings}
}

func (h WhatsappHandler) GetSettings(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	settings, err := h.getSettings.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, whatsappSettingsResponse(*settings))
}

func (h WhatsappHandler) UpdateSettings(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	var request dto.WhatsappSettingsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}

	apiKey := strings.TrimSpace(request.APIKey)
	if apiKey == "" {
		current, err := h.getSettings.Execute(c.Request.Context(), boxID)
		if err == nil {
			apiKey = current.APIKeyEncrypted
		}
	}

	now := time.Now()
	settings := domain.WhatsappSettings{
		BoxID:           boxID,
		Provider:        normalizeWhatsappProvider(request.Provider),
		BaseURL:         request.BaseURL,
		InstanceName:    request.InstanceName,
		APIKeyEncrypted: apiKey,
		Enabled:         request.Enabled,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := h.updateSettings.Execute(c.Request.Context(), &settings); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, whatsappSettingsResponse(settings))
}

func (h WhatsappHandler) TestSettings(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	var request dto.WhatsappSettingsRequest
	hasDraft := c.ShouldBindJSON(&request) == nil && strings.TrimSpace(request.Provider) != ""

	var settings domain.WhatsappSettings
	current, err := h.getSettings.Execute(c.Request.Context(), boxID)
	if err == nil {
		settings = *current
	} else if !hasDraft || !errors.Is(err, gorm.ErrRecordNotFound) {
		respondError(c, err)
		return
	} else {
		settings = domain.WhatsappSettings{BoxID: boxID}
	}

	if hasDraft {
		settings.Provider = normalizeWhatsappProvider(request.Provider)
		settings.BaseURL = request.BaseURL
		settings.InstanceName = request.InstanceName
		settings.Enabled = request.Enabled
		if strings.TrimSpace(request.APIKey) != "" {
			settings.APIKeyEncrypted = request.APIKey
		}
	}

	if err := h.testSettings.Execute(c.Request.Context(), settings); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func whatsappSettingsResponse(settings domain.WhatsappSettings) dto.WhatsappSettingsResponse {
	updatedAt := ""
	if !settings.UpdatedAt.IsZero() {
		updatedAt = settings.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return dto.WhatsappSettingsResponse{
		ID:           string(settings.ID),
		BoxID:        string(settings.BoxID),
		Provider:     normalizeWhatsappProvider(settings.Provider),
		BaseURL:      settings.BaseURL,
		InstanceName: settings.InstanceName,
		HasAPIKey:    strings.TrimSpace(settings.APIKeyEncrypted) != "",
		UpdatedAt:    updatedAt,
		Enabled:      settings.Enabled,
	}
}

func normalizeWhatsappProvider(provider string) string {
	provider = strings.TrimSpace(strings.ToLower(provider))
	if provider == "" {
		return whatsappadapter.ProviderEvolution
	}
	return provider
}
