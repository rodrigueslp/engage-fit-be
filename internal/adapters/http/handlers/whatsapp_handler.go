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

	h.getForBox(c, boxID)
}

func (h WhatsappHandler) AdminGetSettings(c *gin.Context) {
	h.getForBox(c, domain.ID(c.Param("id")))
}

func (h WhatsappHandler) getForBox(c *gin.Context, boxID domain.ID) {
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
	current, err := h.getSettings.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}
	current.ConnectionMode = normalizeWhatsappConnectionMode(request.ConnectionMode)
	current.UpdatedAt = time.Now()
	if err := h.updateSettings.Execute(c.Request.Context(), current); err != nil {
		respondError(c, err)
		return
	}
	h.getForBox(c, boxID)
}

func (h WhatsappHandler) AdminUpdateSettings(c *gin.Context) {
	h.updateForBox(c, domain.ID(c.Param("id")))
}

func (h WhatsappHandler) updateForBox(c *gin.Context, boxID domain.ID) {
	var request dto.WhatsappSettingsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}

	apiKey := strings.TrimSpace(request.APIKey)
	current, currentErr := h.getSettings.Execute(c.Request.Context(), boxID)
	if apiKey == "" && currentErr == nil {
		apiKey = current.APIKeyEncrypted
	}

	connectionMode := normalizeWhatsappConnectionMode(request.ConnectionMode)
	provider := normalizeWhatsappProvider(request.Provider)
	baseURL := request.BaseURL
	instanceName := request.InstanceName
	enabled := request.Enabled
	if connectionMode == domain.WhatsappConnectionPlatform && currentErr == nil {
		provider = current.Provider
		baseURL = current.BaseURL
		instanceName = current.InstanceName
	}

	now := time.Now()
	settings := domain.WhatsappSettings{
		BoxID:           boxID,
		ConnectionMode:  connectionMode,
		Provider:        provider,
		BaseURL:         baseURL,
		InstanceName:    instanceName,
		APIKeyEncrypted: apiKey,
		Enabled:         enabled,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := h.updateSettings.Execute(c.Request.Context(), &settings); err != nil {
		respondError(c, err)
		return
	}

	saved, err := h.getSettings.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, whatsappSettingsResponse(*saved))
}

func (h WhatsappHandler) TestSettings(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	current, err := h.getSettings.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}
	if err := h.testSettings.Execute(c.Request.Context(), *current); err != nil {
		respondPublicError(c, http.StatusBadGateway, "whatsapp_provider_failed", "WhatsApp provider request failed")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h WhatsappHandler) AdminTestSettings(c *gin.Context) {
	h.testForBox(c, domain.ID(c.Param("id")))
}

func (h WhatsappHandler) testForBox(c *gin.Context, boxID domain.ID) {
	var request dto.WhatsappSettingsRequest
	hasDraft := c.ShouldBindJSON(&request) == nil

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
		settings.ConnectionMode = normalizeWhatsappConnectionMode(request.ConnectionMode)
		settings.Provider = normalizeWhatsappProvider(request.Provider)
		settings.BaseURL = request.BaseURL
		settings.InstanceName = request.InstanceName
		settings.Enabled = request.Enabled
		if strings.TrimSpace(request.APIKey) != "" {
			settings.APIKeyEncrypted = request.APIKey
		}
	}

	if err := h.testSettings.Execute(c.Request.Context(), settings); err != nil {
		respondPublicError(c, http.StatusBadGateway, "whatsapp_provider_failed", "WhatsApp provider request failed")
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
		ID:                string(settings.ID),
		BoxID:             string(settings.BoxID),
		ConnectionMode:    string(normalizeWhatsappConnectionMode(string(settings.ConnectionMode))),
		Provider:          normalizeWhatsappProvider(settings.Provider),
		BaseURL:           settings.BaseURL,
		InstanceName:      settings.InstanceName,
		HasAPIKey:         strings.TrimSpace(settings.APIKeyEncrypted) != "",
		UpdatedAt:         updatedAt,
		Enabled:           settings.Enabled,
		PlatformAvailable: settings.PlatformAvailable,
		PlatformSender:    settings.PlatformSender,
	}
}

func normalizeWhatsappConnectionMode(mode string) domain.WhatsappConnectionMode {
	if strings.TrimSpace(strings.ToLower(mode)) == string(domain.WhatsappConnectionDedicated) {
		return domain.WhatsappConnectionDedicated
	}
	return domain.WhatsappConnectionPlatform
}

func normalizeWhatsappProvider(provider string) string {
	provider = strings.TrimSpace(strings.ToLower(provider))
	if provider == "" || provider == "evolution" {
		return "twilio"
	}
	return provider
}
