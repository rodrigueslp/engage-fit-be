package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestWindowRateLimiterChecksAllKeys(t *testing.T) {
	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	limiter := NewWindowRateLimiter(2, time.Minute)
	limiter.now = func() time.Time { return now }

	if allowed, _ := limiter.Allow("ip:one", "identity:shared"); !allowed {
		t.Fatal("expected first request to be allowed")
	}
	if allowed, _ := limiter.Allow("ip:two", "identity:shared"); !allowed {
		t.Fatal("expected second request to be allowed")
	}
	if allowed, retry := limiter.Allow("ip:three", "identity:shared"); allowed || retry <= 0 {
		t.Fatal("expected shared identity to be rate limited")
	}

	now = now.Add(time.Minute)
	if allowed, _ := limiter.Allow("ip:three", "identity:shared"); !allowed {
		t.Fatal("expected request after window expiration to be allowed")
	}
}

func TestBodySizeLimitRejectsLargeLoginBeforeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(BodySizeLimit(16, 32))
	limiter := NewWindowRateLimiter(10, time.Minute)
	router.POST("/api/v1/auth/login", JSONRateLimit(limiter, "email"), func(c *gin.Context) { c.Status(http.StatusNoContent) })

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"long@example.com"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, response.Code)
	}
}
