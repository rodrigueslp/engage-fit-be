package middleware

import (
	"crypto/subtle"
	"net/http"

	"boxengage/backend/internal/adapters/http/apiresponse"
	"github.com/gin-gonic/gin"
)

func OwnerSetupAccess(enabled bool, expectedToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !enabled {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		if expectedToken != "" {
			provided := c.GetHeader("X-Setup-Token")
			if subtle.ConstantTimeCompare([]byte(provided), []byte(expectedToken)) != 1 {
				apiresponse.AbortError(c, http.StatusUnauthorized, "setup_token_invalid", "invalid setup token")
				return
			}
		}
		c.Next()
	}
}
