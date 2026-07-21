package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestFeatureGates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	capabilities := Capabilities{Whatsapp: false, Email: false, Automation: true, Workouts: true, LLM: false}
	router := gin.New()
	router.Use(FeatureGates(capabilities))
	for _, path := range []string{"/api/v1/email/settings", "/api/v1/message-campaigns"} {
		router.GET(path, func(c *gin.Context) { c.Status(http.StatusNoContent) })
	}
	router.GET("/api/v1/automation/runs", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.POST("/api/v1/workouts/123/message-drafts", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	assert := func(method, path string, expected int) {
		t.Helper()
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(method, path, nil))
		if response.Code != expected {
			t.Fatalf("%s %s: expected %d, got %d", method, path, expected, response.Code)
		}
	}
	assert(http.MethodGet, "/api/v1/email/settings", http.StatusNotFound)
	assert(http.MethodGet, "/api/v1/message-campaigns", http.StatusNotFound)
	assert(http.MethodGet, "/api/v1/automation/runs", http.StatusNoContent)
	assert(http.MethodPost, "/api/v1/workouts/123/message-drafts", http.StatusNotFound)
}
