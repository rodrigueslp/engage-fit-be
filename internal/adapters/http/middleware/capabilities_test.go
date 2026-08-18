package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestCatalogCapabilityBlocksAllCatalogRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(FeatureGates(Capabilities{Catalog: false, People: true}))
	router.Any("/*path", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	for _, path := range []string{"/api/v1/organization", "/api/v1/modalities", "/api/v1/modalities/id", "/api/v1/units/unit-id/spaces", "/api/v1/units/unit-id/modalities/id"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "capability_disabled") {
			t.Fatalf("%s should be blocked by catalog capability: %d %s", path, response.Code, response.Body.String())
		}
	}

	for _, path := range []string{"/api/v1/units/unit-id/people", "/api/v1/units/unit-id", "/api/v1/units"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNoContent {
			t.Fatalf("%s is not a catalog route and should not be blocked: %d %s", path, response.Code, response.Body.String())
		}
	}
}

func TestPeopleCapabilityBlocksOnlyPeopleUnitRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(FeatureGates(Capabilities{People: false, Catalog: true}))
	router.POST("/api/v1/units/:unit_id/people", func(c *gin.Context) { c.Status(http.StatusCreated) })
	router.POST("/api/v1/units/:unit_id/people/duplicate-check", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/api/v1/units/:unit_id/spaces", func(c *gin.Context) { c.Status(http.StatusOK) })
	for _, path := range []string{"/api/v1/units/unit-1/people", "/api/v1/units/unit-1/people/duplicate-check"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))
		if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "capability_disabled") {
			t.Fatalf("%s should be blocked by people capability: %d %s", path, response.Code, response.Body.String())
		}
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/units/unit-1/spaces", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("spaces route should not be blocked by people capability: %d", response.Code)
	}
}
