package middleware

import (
	"net/http"

	"boxengage/backend/internal/adapters/http/apiresponse"
	"github.com/gin-gonic/gin"
)

func Tenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		boxID, err := BoxID(c)
		if err != nil || boxID == "" {
			apiresponse.AbortError(c, http.StatusUnauthorized, "tenant_missing", "missing tenant context")
			return
		}
		c.Next()
	}
}
