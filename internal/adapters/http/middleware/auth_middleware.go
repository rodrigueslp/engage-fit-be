package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"boxengage/backend/internal/ports/services"
)

func Auth(tokens services.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "missing authorization header"})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "invalid authorization header"})
			return
		}

		claims, err := tokens.Validate(c.Request.Context(), parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
			return
		}

		SetAuthContext(c, claims.UserID, claims.BoxID, claims.Role)
		c.Next()
	}
}
