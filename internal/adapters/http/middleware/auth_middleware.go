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
	return auth(tokens, users, boxes, nil, sessionConfig...)
}

func AuthWithUnitAccess(tokens services.TokenService, users repositories.UserRepository, boxes repositories.BoxRepository, access repositories.UnitAccessAuthorizer, sessionConfig ...SessionConfig) gin.HandlerFunc {
	return auth(tokens, users, boxes, access, sessionConfig...)
}

func auth(tokens services.TokenService, users repositories.UserRepository, boxes repositories.BoxRepository, access repositories.UnitAccessAuthorizer, sessionConfig ...SessionConfig) gin.HandlerFunc {
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
		if err != nil || claims.AuthVersion < 1 || user.AuthVersion != claims.AuthVersion || user.Role != claims.Role {
			apiresponse.AbortError(c, http.StatusUnauthorized, "session_invalid", "invalid session")
			return
		}
		if access == nil && user.BoxID != claims.BoxID {
			apiresponse.AbortError(c, http.StatusUnauthorized, "session_invalid", "invalid session")
			return
		}
		if user.Role == domain.UserRoleCoach && !user.Active {
			apiresponse.AbortError(c, http.StatusForbidden, "user_inactive", "acesso desativado pelo proprietário da academia")
			return
		}
		if user.Role == domain.UserRoleOwner || user.Role == domain.UserRoleCoach {
			if access != nil {
				context, accessErr := access.ResolveUserUnitAccess(c.Request.Context(), user.ID, claims.BoxID)
				if accessErr != nil || context.Role != user.Role || !context.Organization.IsActive() {
					apiresponse.AbortError(c, http.StatusForbidden, "unit_access_denied", "acesso à unidade não autorizado")
					return
				}
			}
			box, err := boxes.FindByID(c.Request.Context(), claims.BoxID)
			if err != nil {
				apiresponse.AbortError(c, http.StatusUnauthorized, "session_invalid", "invalid session")
				return
			}
			if !box.IsActive() {
				apiresponse.AbortError(c, http.StatusForbidden, "box_inactive", "academia suspensa ou arquivada; procure o suporte")
				return
			}
			if box.BillingAccessBlocked {
				apiresponse.AbortError(c, http.StatusPaymentRequired, "billing_access_blocked", "assinatura pendente ou vencida; procure o financeiro")
				return
			}
		}

		SetAuthContext(c, user.ID, claims.BoxID, user.Role)
		c.Set(authTransportKey, transport)
		c.Next()
	}
}
