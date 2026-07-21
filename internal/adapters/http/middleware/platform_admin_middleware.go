package middleware

import (
	"net/http"

	"boxengage/backend/internal/adapters/http/apiresponse"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

func PlatformAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, err := Role(c)
		if err != nil || role != domain.UserRolePlatformAdmin {
			apiresponse.AbortError(c, http.StatusForbidden, "platform_admin_required", "acesso restrito ao administrador da plataforma")
			return
		}
		c.Next()
	}
}
