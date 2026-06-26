package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Tenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := BoxID(c); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "missing tenant context"})
			return
		}
		c.Next()
	}
}
