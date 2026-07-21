package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"

	"boxengage/backend/internal/adapters/http/apiresponse"
	"github.com/gin-gonic/gin"
)

const authTransportKey = "auth_transport"

type SessionConfig struct {
	CookieName     string
	CSRFCookieName string
	Secure         bool
	SameSite       http.SameSite
	MaxAgeSeconds  int
}

func (s SessionConfig) normalized() SessionConfig {
	if s.CookieName == "" {
		s.CookieName = "engagefit_session"
	}
	if s.CSRFCookieName == "" {
		s.CSRFCookieName = "engagefit_csrf"
	}
	if s.SameSite == 0 {
		s.SameSite = http.SameSiteLaxMode
	}
	if s.MaxAgeSeconds <= 0 {
		s.MaxAgeSeconds = 86400
	}
	return s
}

func ParseSameSite(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func SetSession(c *gin.Context, config SessionConfig, token string) error {
	config = config.normalized()
	csrfBytes := make([]byte, 32)
	if _, err := rand.Read(csrfBytes); err != nil {
		return err
	}
	csrf := base64.RawURLEncoding.EncodeToString(csrfBytes)
	c.SetSameSite(config.SameSite)
	c.SetCookie(config.CookieName, token, config.MaxAgeSeconds, "/", "", config.Secure, true)
	c.SetCookie(config.CSRFCookieName, csrf, config.MaxAgeSeconds, "/", "", config.Secure, false)
	return nil
}

func ClearSession(c *gin.Context, config SessionConfig) {
	config = config.normalized()
	c.SetSameSite(config.SameSite)
	c.SetCookie(config.CookieName, "", -1, "/", "", config.Secure, true)
	c.SetCookie(config.CSRFCookieName, "", -1, "/", "", config.Secure, false)
}

func tokenFromRequest(c *gin.Context, config SessionConfig) (string, string) {
	header := c.GetHeader("Authorization")
	if header != "" {
		parts := strings.SplitN(header, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1], "bearer"
		}
		return "", ""
	}
	config = config.normalized()
	token, err := c.Cookie(config.CookieName)
	if err != nil {
		return "", ""
	}
	return token, "cookie"
}

func CSRF(config SessionConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		transport, _ := c.Get(authTransportKey)
		if transport != "cookie" || c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		config = config.normalized()
		cookieToken, err := c.Cookie(config.CSRFCookieName)
		headerToken := c.GetHeader("X-CSRF-Token")
		if err != nil || cookieToken == "" || headerToken == "" || !hmac.Equal([]byte(cookieToken), []byte(headerToken)) {
			apiresponse.AbortError(c, http.StatusForbidden, "csrf_invalid", "invalid csrf token")
			return
		}
		c.Next()
	}
}
