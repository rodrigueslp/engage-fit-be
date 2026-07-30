package middleware

import (
	"net/http"

	"boxengage/backend/internal/adapters/http/apiresponse"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

func Owner() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, err := Role(c)
		if err != nil || role != domain.UserRoleOwner {
			apiresponse.AbortError(c, http.StatusForbidden, "owner_required", "acesso restrito ao proprietário da academia")
			return
		}
		c.Next()
	}
}
