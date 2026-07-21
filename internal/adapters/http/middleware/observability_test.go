package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMetricsAccessAllowsUnprotectedDevelopmentEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/metrics", MetricsAccess(""), func(c *gin.Context) { c.Status(http.StatusNoContent) })

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", response.Code)
	}
}

func TestMetricsAccessRequiresExactBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/metrics", MetricsAccess("a-secure-observability-token"), func(c *gin.Context) { c.Status(http.StatusNoContent) })

	unauthorized := httptest.NewRecorder()
	router.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", unauthorized.Code)
	}

	authorizedRequest := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	authorizedRequest.Header.Set("Authorization", "Bearer a-secure-observability-token")
	authorized := httptest.NewRecorder()
	router.ServeHTTP(authorized, authorizedRequest)
	if authorized.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", authorized.Code)
	}
}

func TestRequestIDRejectsUnsafeOrOversizedValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Request-ID", "unsafe\nlog-entry")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	requestID := response.Header().Get("X-Request-ID")
	if requestID == "" || requestID == "unsafe\nlog-entry" || !validRequestID.MatchString(requestID) {
		t.Fatalf("expected a generated safe request id, got %q", requestID)
	}
}
