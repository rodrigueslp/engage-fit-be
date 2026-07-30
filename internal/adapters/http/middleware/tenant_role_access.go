package middleware

import (
	"net/http"

	"boxengage/backend/internal/adapters/http/apiresponse"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

var coachRoutes = map[string]map[string]bool{
	http.MethodGet: {
		"/api/v1/box":                                  true,
		"/api/v1/dashboard/summary":                    true,
		"/api/v1/dashboard/active-campaigns":           true,
		"/api/v1/dashboard/near-goal-students":         true,
		"/api/v1/dashboard/at-risk-students":           true,
		"/api/v1/dashboard/pending-rewards":            true,
		"/api/v1/retention/radar":                      true,
		"/api/v1/retention/summary":                    true,
		"/api/v1/retention/onboarding":                 true,
		"/api/v1/students":                             true,
		"/api/v1/students/:id":                         true,
		"/api/v1/students/:id/checkins":                true,
		"/api/v1/students/:id/retention-interventions": true,
		"/api/v1/checkins/summary":                     true,
		"/api/v1/team/members":                         true,
	},
	http.MethodPost: {
		"/api/v1/students/:id/retention-interventions": true,
	},
	http.MethodPatch: {
		"/api/v1/retention-interventions/:id":   true,
		"/api/v1/students/:id/membership-start": true,
		"/api/v1/students/:id/risk-status":      true,
	},
}

func TenantRoleAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, err := Role(c)
		if err != nil {
			apiresponse.AbortError(c, http.StatusForbidden, "tenant_role_required", "perfil operacional inválido")
			return
		}
		if role == domain.UserRoleOwner {
			c.Next()
			return
		}
		if role == domain.UserRoleCoach && coachRoutes[c.Request.Method][c.FullPath()] {
			c.Next()
			return
		}
		apiresponse.AbortError(c, http.StatusForbidden, "owner_required", "ação restrita ao proprietário da academia")
	}
}
