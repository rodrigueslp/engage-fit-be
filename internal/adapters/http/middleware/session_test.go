package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSessionCookiesAndCSRF(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := SessionConfig{CookieName: "session", CSRFCookieName: "csrf", SameSite: http.SameSiteLaxMode, MaxAgeSeconds: 60}
	loginRouter := gin.New()
	loginRouter.GET("/login", func(c *gin.Context) {
		if err := SetSession(c, config, "signed-token"); err != nil {
			t.Fatal(err)
		}
		c.Status(http.StatusNoContent)
	})
	loginResponse := httptest.NewRecorder()
	loginRouter.ServeHTTP(loginResponse, httptest.NewRequest(http.MethodGet, "/login", nil))
	cookies := loginResponse.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("expected two cookies, got %d", len(cookies))
	}
	var sessionCookie, csrfCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "session" {
			sessionCookie = cookie
		}
		if cookie.Name == "csrf" {
			csrfCookie = cookie
		}
	}
	if sessionCookie == nil || !sessionCookie.HttpOnly {
		t.Fatal("session cookie must be HttpOnly")
	}
	if csrfCookie == nil || csrfCookie.HttpOnly || csrfCookie.Value == "" {
		t.Fatal("csrf cookie must be readable and non-empty")
	}

	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set(authTransportKey, "cookie"); c.Next() }, CSRF(config))
	router.POST("/change", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodPost, "/change", strings.NewReader("{}"))
	request.AddCookie(csrfCookie)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without header, got %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/change", strings.NewReader("{}"))
	request.AddCookie(csrfCookie)
	request.Header.Set("X-CSRF-Token", csrfCookie.Value)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected 204 with csrf, got %d", response.Code)
	}
}

func TestCORSAllowlist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS([]string{"https://app.example.com"}))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Origin", "https://app.example.com")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Fatal("allowed origin missing")
	}
	if response.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("credentials must be allowed")
	}
}

func TestCORSRejectsUnknownPreflightOrigin(t *testing.T) {
	router := gin.New()
	router.Use(CORS([]string{"https://app.example.com"}))
	request := httptest.NewRequest(http.MethodOptions, "/", nil)
	request.Header.Set("Origin", "https://evil.example")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", response.Code)
	}
}
