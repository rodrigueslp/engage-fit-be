package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestOwnerSetupAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name    string
		enabled bool
		token   string
		given   string
		status  int
	}{
		{name: "disabled is hidden", status: http.StatusNotFound},
		{name: "enabled without token", enabled: true, status: http.StatusNoContent},
		{name: "rejects invalid token", enabled: true, token: "expected", given: "wrong", status: http.StatusUnauthorized},
		{name: "accepts valid token", enabled: true, token: "expected", given: "expected", status: http.StatusNoContent},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			router.POST("/setup", OwnerSetupAccess(test.enabled, test.token), func(c *gin.Context) { c.Status(http.StatusNoContent) })
			request := httptest.NewRequest(http.MethodPost, "/setup", nil)
			if test.given != "" {
				request.Header.Set("X-Setup-Token", test.given)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("expected status %d, got %d", test.status, response.Code)
			}
		})
	}
}
