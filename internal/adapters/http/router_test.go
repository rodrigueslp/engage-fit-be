package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthEndpoints(t *testing.T) {
	ready := true
	router := NewRouter(RouterDependencies{ReadinessCheck: func(ctx context.Context) error {
		if ready {
			return nil
		}
		return errors.New("database unavailable")
	}})

	assertStatus := func(path string, expected int) {
		t.Helper()
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		router.ServeHTTP(response, request)
		if response.Code != expected {
			t.Fatalf("%s: expected %d, got %d", path, expected, response.Code)
		}
	}
	assertStatus("/health/live", http.StatusOK)
	assertStatus("/health/ready", http.StatusOK)
	ready = false
	assertStatus("/health/live", http.StatusOK)
	assertStatus("/health/ready", http.StatusServiceUnavailable)
}

func TestSecurityHeaders(t *testing.T) {
	router := NewRouter(RouterDependencies{})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	router.ServeHTTP(response, request)

	expected := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"Permissions-Policy":     "camera=(), microphone=(), geolocation=()",
		"X-Frame-Options":        "DENY",
	}
	for name, value := range expected {
		if actual := response.Header().Get(name); actual != value {
			t.Errorf("%s: expected %q, got %q", name, value, actual)
		}
	}
	if response.Header().Get("Content-Security-Policy") == "" {
		t.Error("Content-Security-Policy must be present")
	}
}

func TestBuildInformationEndpoint(t *testing.T) {
	router := NewRouter(RouterDependencies{BuildVersion: "1.2.3", BuildCommit: "abc123", BuildTime: "2026-07-21T16:00:00Z"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/build", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	for _, expected := range []string{"1.2.3", "abc123", "2026-07-21T16:00:00Z"} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("build response missing %q: %s", expected, response.Body.String())
		}
	}
}

func TestErrorEnvelopeIncludesStableCodeAndRequestID(t *testing.T) {
	router := NewRouter(RouterDependencies{})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/email/settings", nil)
	request.Header.Set("X-Request-ID", "support-test-123")
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", response.Code)
	}
	for _, expected := range []string{"capability_disabled", "support-test-123", "message"} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("error response missing %q: %s", expected, response.Body.String())
		}
	}
}
