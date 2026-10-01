package middleware

import (
	"net/http"
	"strings"

	"boxengage/backend/internal/adapters/http/apiresponse"
	"github.com/gin-gonic/gin"
)

type Capabilities struct {
	Whatsapp                    bool `json:"whatsapp"`
	Email                       bool `json:"email"`
	Automation                  bool `json:"automation"`
	Workouts                    bool `json:"workouts"`
	LLM                         bool `json:"llm"`
	Billing                     bool `json:"billing"`
	MemberBilling               bool `json:"member_billing"`
	Catalog                     bool `json:"catalog"`
	ServiceCatalog              bool `json:"service_catalog"`
	People                      bool `json:"people"`
	Memberships                 bool `json:"memberships"`
	Enrollments                 bool `json:"enrollments"`
	Contracts                   bool `json:"contracts"`
	ContractAcceptanceSimulator bool `json:"contract_acceptance_simulator"`
	ContractLifecycle           bool `json:"contract_lifecycle"`
	HealthForms                 bool `json:"health_forms"`
	Scheduling                  bool `json:"scheduling"`
	SchedulingGrid              bool `json:"scheduling_grid"`
	Reservations                bool `json:"reservations"`
	Attendance                  bool `json:"attendance"`
	Announcements               bool `json:"announcements"`
	Commerce                    bool `json:"commerce"`
}

func FeatureGates(capabilities Capabilities) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		disabled := false
		switch {
		case isCommercePath(path):
			disabled = !capabilities.Commerce
		case isAnnouncementPath(path):
			disabled = !capabilities.Announcements
		case strings.HasPrefix(path, "/api/v1/email"):
			disabled = !capabilities.Email
		case strings.HasPrefix(path, "/api/v1/automation"):
			disabled = !capabilities.Automation
		case strings.HasPrefix(path, "/api/v1/workouts") || strings.HasPrefix(path, "/api/v1/workout-message"):
			disabled = !capabilities.Workouts
			if !disabled && c.Request.Method == http.MethodPost && strings.HasSuffix(path, "/message-drafts") {
				disabled = !capabilities.LLM
			}
		case strings.HasPrefix(path, "/api/v1/whatsapp") || strings.HasPrefix(path, "/api/v1/message-") || strings.Contains(path, "/whatsapp-templates") || strings.Contains(path, "/whatsapp-settings") || strings.Contains(path, "/contact-activation") || path == "/api/v1/webhooks/twilio/whatsapp" || path == "/api/v1/messaging/usage":
			disabled = !capabilities.Whatsapp
		case strings.HasPrefix(path, "/api/v1/billing") || strings.HasPrefix(path, "/api/v1/admin/billing") || strings.HasPrefix(path, "/api/v1/webhooks/asaas"):
			disabled = !capabilities.Billing
		case isMemberBillingPath(path):
			disabled = !capabilities.MemberBilling || !capabilities.Enrollments
		case isServiceCatalogPath(path):
			disabled = !capabilities.Catalog || !capabilities.ServiceCatalog
		case isCatalogPath(path):
			disabled = !capabilities.Catalog
		case strings.HasPrefix(path, "/api/v1/public/contract-acceptance"):
			disabled = !capabilities.Contracts || !capabilities.ContractAcceptanceSimulator
		case isHealthFormPath(path):
			disabled = !capabilities.HealthForms || !capabilities.Enrollments || !capabilities.People
		case isContractLifecyclePath(path):
			disabled = !capabilities.Contracts || !capabilities.ContractLifecycle
		case isContractPath(path):
			disabled = !capabilities.Contracts
		case isReservationPath(path) || isAthleteBookingPath(path):
			disabled = !capabilities.Reservations || !capabilities.Scheduling || !capabilities.SchedulingGrid || !capabilities.Catalog || !capabilities.Memberships || !capabilities.Enrollments || !capabilities.People
		case isAttendancePath(path):
			disabled = !capabilities.Attendance || !capabilities.Reservations || !capabilities.Scheduling || !capabilities.SchedulingGrid || !capabilities.Catalog || !capabilities.Memberships || !capabilities.Enrollments || !capabilities.People
		case isEnrollmentPath(path):
			disabled = !capabilities.Enrollments
		case isPeoplePath(path):
			disabled = !capabilities.People
		case strings.HasPrefix(path, "/api/v1/membership-plans"):
			disabled = !capabilities.Memberships
		case isSchedulingGridPath(path):
			disabled = !capabilities.Scheduling || !capabilities.SchedulingGrid || !capabilities.Catalog
		case isSchedulingPath(path):
			disabled = !capabilities.Scheduling
		}
		if disabled {
			apiresponse.AbortError(c, http.StatusNotFound, "capability_disabled", "capability disabled")
			return
		}
		c.Next()
	}
}

func isCommercePath(path string) bool {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/units/"), "/")
	return len(parts) >= 2 && parts[0] != "" && parts[1] == "commerce"
}

func isAnnouncementPath(path string) bool {
	return strings.Contains(path, "/announcements")
}

func isHealthFormPath(path string) bool {
	return strings.Contains(path, "/health-forms") || strings.Contains(path, "/health-form-sessions")
}
func isContractLifecyclePath(path string) bool {
	return strings.Contains(path, "/signature-envelopes") || strings.Contains(path, "/legal-actions") || strings.HasPrefix(path, "/api/v1/webhooks/contracts/")
}

func isMemberBillingPath(path string) bool {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/units/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		return false
	}
	if parts[1] == "member-billing" {
		return true
	}
	return len(parts) >= 4 && parts[1] == "enrollments" && parts[3] == "member-billing"
}

func isAttendancePath(path string) bool {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/units/"), "/")
	return len(parts) >= 2 && parts[0] != "" && parts[1] == "attendance"
}

func isReservationPath(path string) bool {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/units/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		return false
	}
	if parts[1] == "reservations" {
		return true
	}
	if parts[1] == "scheduling" && len(parts) >= 5 && parts[2] == "sessions" && parts[4] == "reservations" {
		return true
	}
	return parts[1] == "people" && len(parts) >= 4 && parts[3] == "reservations"
}

func isAthleteBookingPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/athlete/memberships/") && (strings.HasSuffix(path, "/book") || strings.HasSuffix(path, "/cancel"))
}

func isSchedulingGridPath(path string) bool {
	if !isSchedulingPath(path) {
		return false
	}
	return !strings.Contains(path, "/professionals") && !strings.Contains(path, "/professional-account-candidates")
}

func isEnrollmentPath(path string) bool {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/units/"), "/")
	if len(parts) < 3 || parts[0] == "" {
		return false
	}
	return parts[1] == "enrollments" || (parts[1] == "students" && len(parts) >= 4 && parts[3] == "enrollments")
}

func isServiceCatalogPath(path string) bool {
	if path == "/api/v1/service-offerings" || strings.HasPrefix(path, "/api/v1/service-offerings/") {
		return true
	}
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/units/"), "/")
	return len(parts) == 3 && parts[0] != "" && parts[1] == "service-offerings" && parts[2] != ""
}

func isPeoplePath(path string) bool {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/units/"), "/")
	return len(parts) >= 2 && parts[0] != "" && parts[1] == "people"
}

func isCatalogPath(path string) bool {
	if path == "/api/v1/organization" || path == "/api/v1/modalities" || strings.HasPrefix(path, "/api/v1/modalities/") {
		return true
	}
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/units/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		return false
	}
	if parts[1] == "spaces" {
		return len(parts) == 2 || (len(parts) == 3 && parts[2] != "")
	}
	return parts[1] == "modalities" && len(parts) == 3 && parts[2] != ""
}

func isContractPath(path string) bool {
	if path == "/api/v1/contract-templates" || strings.HasPrefix(path, "/api/v1/contract-templates/") {
		return true
	}
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/units/"), "/")
	if len(parts) >= 2 && parts[0] != "" {
		if parts[1] == "contracts" {
			return true
		}
		if len(parts) >= 4 && parts[1] == "enrollments" && parts[3] == "contracts" {
			return true
		}
	}
	return false
}

func isSchedulingPath(path string) bool {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/units/"), "/")
	return len(parts) >= 2 && parts[0] != "" && parts[1] == "scheduling"
}
