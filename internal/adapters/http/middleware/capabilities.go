package middleware

import (
	"net/http"
	"strings"

	"boxengage/backend/internal/adapters/http/apiresponse"
	"github.com/gin-gonic/gin"
)

type Capabilities struct {
	Whatsapp   bool `json:"whatsapp"`
	Email      bool `json:"email"`
	Automation bool `json:"automation"`
	Workouts   bool `json:"workouts"`
	LLM        bool `json:"llm"`
}

func FeatureGates(capabilities Capabilities) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		disabled := false
		switch {
		case strings.HasPrefix(path, "/api/v1/email"):
			disabled = !capabilities.Email
		case strings.HasPrefix(path, "/api/v1/automation"):
			disabled = !capabilities.Automation
		case strings.HasPrefix(path, "/api/v1/workouts") || strings.HasPrefix(path, "/api/v1/workout-message"):
			disabled = !capabilities.Workouts
			if !disabled && c.Request.Method == http.MethodPost && strings.HasSuffix(path, "/message-drafts") {
				disabled = !capabilities.LLM
			}
		case strings.HasPrefix(path, "/api/v1/whatsapp") || strings.HasPrefix(path, "/api/v1/message-") || strings.Contains(path, "/whatsapp-templates") || strings.Contains(path, "/whatsapp-settings") || path == "/api/v1/messaging/usage":
			disabled = !capabilities.Whatsapp
		}
		if disabled {
			apiresponse.AbortError(c, http.StatusNotFound, "capability_disabled", "capability disabled")
			return
		}
		c.Next()
	}
}
