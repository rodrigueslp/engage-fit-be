package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"boxengage/backend/internal/adapters/http/apiresponse"
	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

func Auth(tokens services.TokenService, users repositories.UserRepository, boxes repositories.BoxRepository, sessionConfig ...SessionConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		config := SessionConfig{}
		if len(sessionConfig) > 0 {
			config = sessionConfig[0]
		}
		token, transport := tokenFromRequest(c, config)
		if token == "" {
			apiresponse.AbortError(c, http.StatusUnauthorized, "session_missing", "missing session")
			return
		}
		claims, err := tokens.Validate(c.Request.Context(), token)
		if err != nil {
			apiresponse.AbortError(c, http.StatusUnauthorized, "session_invalid", "invalid session")
			return
		}
		user, err := users.FindByID(c.Request.Context(), claims.UserID)
		if err != nil || claims.AuthVersion < 1 || user.AuthVersion != claims.AuthVersion || user.Role != claims.Role || user.BoxID != claims.BoxID {
			apiresponse.AbortError(c, http.StatusUnauthorized, "session_invalid", "invalid session")
			return
		}
		if user.Role == domain.UserRoleOwner {
			box, err := boxes.FindByID(c.Request.Context(), user.BoxID)
			if err != nil {
				apiresponse.AbortError(c, http.StatusUnauthorized, "session_invalid", "invalid session")
				return
			}
			if !box.IsActive() {
				apiresponse.AbortError(c, http.StatusForbidden, "box_inactive", "academia suspensa ou arquivada; procure o suporte")
				return
			}
		}

		SetAuthContext(c, user.ID, user.BoxID, user.Role)
		c.Set(authTransportKey, transport)
		c.Next()
	}
}
