package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

func TestCoachAccessIsLimitedToOperationalRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		SetAuthContext(c, "coach-1", "box-1", domain.UserRoleCoach)
		c.Next()
	}, TenantRoleAccess())
	router.GET("/api/v1/retention/radar", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.GET("/api/v1/imports", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	allowed := httptest.NewRecorder()
	router.ServeHTTP(allowed, httptest.NewRequest(http.MethodGet, "/api/v1/retention/radar", nil))
	if allowed.Code != http.StatusNoContent {
		t.Fatalf("coach should access retention radar, got %d", allowed.Code)
	}
	forbidden := httptest.NewRecorder()
	router.ServeHTTP(forbidden, httptest.NewRequest(http.MethodGet, "/api/v1/imports", nil))
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("coach must not access imports, got %d", forbidden.Code)
	}
}
