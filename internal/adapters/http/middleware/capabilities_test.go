package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestFeatureGates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	capabilities := Capabilities{Whatsapp: false, Email: false, Automation: true, Workouts: true, LLM: false}
	router := gin.New()
	router.Use(FeatureGates(capabilities))
	for _, path := range []string{"/api/v1/email/settings", "/api/v1/message-campaigns"} {
		router.GET(path, func(c *gin.Context) { c.Status(http.StatusNoContent) })
	}
	router.GET("/api/v1/automation/runs", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.POST("/api/v1/workouts/123/message-drafts", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	assert := func(method, path string, expected int) {
		t.Helper()
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(method, path, nil))
		if response.Code != expected {
			t.Fatalf("%s %s: expected %d, got %d", method, path, expected, response.Code)
		}
	}
	assert(http.MethodGet, "/api/v1/email/settings", http.StatusNotFound)
	assert(http.MethodGet, "/api/v1/message-campaigns", http.StatusNotFound)
	assert(http.MethodGet, "/api/v1/automation/runs", http.StatusNoContent)
	assert(http.MethodPost, "/api/v1/workouts/123/message-drafts", http.StatusNotFound)
}

func TestAthleteBookingRequiresReservationsCapability(t *testing.T) {
	gin.SetMode(gin.TestMode)
	paths := []string{"/api/v1/athlete/memberships/membership/sessions/session/book", "/api/v1/athlete/memberships/membership/sessions/session/reservations/reservation/cancel"}
	for _, enabled := range []bool{false, true} {
		for _, path := range paths {
			capabilities := Capabilities{Reservations: enabled, Scheduling: true, SchedulingGrid: true, Catalog: true, Memberships: true, Enrollments: true, People: true}
			router := gin.New()
			router.Use(FeatureGates(capabilities))
			router.POST(path, func(c *gin.Context) { c.Status(http.StatusNoContent) })
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))
			want := http.StatusNotFound
			if enabled {
				want = http.StatusNoContent
			}
			if response.Code != want {
				t.Fatalf("path=%s enabled=%v: expected %d, got %d", path, enabled, want, response.Code)
			}
		}
	}
}

func TestAttendanceCapabilityRequiresFullFoundation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	path := "/api/v1/units/unit-id/attendance/today"
	for _, capabilities := range []Capabilities{
		{Attendance: false, Reservations: true, Scheduling: true, SchedulingGrid: true, Catalog: true, Memberships: true, Enrollments: true, People: true},
		{Attendance: true, Reservations: false, Scheduling: true, SchedulingGrid: true, Catalog: true, Memberships: true, Enrollments: true, People: true},
	} {
		router := gin.New()
		router.Use(FeatureGates(capabilities))
		router.GET(path, func(c *gin.Context) { c.Status(http.StatusNoContent) })
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("attendance should be blocked: %d", response.Code)
		}
	}
	router := gin.New()
	router.Use(FeatureGates(Capabilities{Attendance: true, Reservations: true, Scheduling: true, SchedulingGrid: true, Catalog: true, Memberships: true, Enrollments: true, People: true}))
	router.GET(path, func(c *gin.Context) { c.Status(http.StatusNoContent) })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("attendance should be enabled: %d", response.Code)
	}
}

func TestCatalogCapabilityBlocksAllCatalogRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(FeatureGates(Capabilities{Catalog: false, People: true}))
	router.Any("/*path", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	for _, path := range []string{"/api/v1/organization", "/api/v1/modalities", "/api/v1/modalities/id", "/api/v1/units/unit-id/spaces", "/api/v1/units/unit-id/modalities/id"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "capability_disabled") {
			t.Fatalf("%s should be blocked by catalog capability: %d %s", path, response.Code, response.Body.String())
		}
	}

	for _, path := range []string{"/api/v1/units/unit-id/people", "/api/v1/units/unit-id", "/api/v1/units"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNoContent {
			t.Fatalf("%s is not a catalog route and should not be blocked: %d %s", path, response.Code, response.Body.String())
		}
	}
}

func TestServiceCatalogRequiresBothCapabilities(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, capabilities := range []Capabilities{
		{Catalog: false, ServiceCatalog: true},
		{Catalog: true, ServiceCatalog: false},
	} {
		router := gin.New()
		router.Use(FeatureGates(capabilities))
		router.Any("/*path", func(c *gin.Context) { c.Status(http.StatusNoContent) })
		for _, path := range []string{"/api/v1/service-offerings", "/api/v1/service-offerings/id", "/api/v1/units/unit-id/service-offerings/id"} {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "capability_disabled") {
				t.Fatalf("%s should require both service catalog capabilities: %d %s", path, response.Code, response.Body.String())
			}
		}
	}

	router := gin.New()
	router.Use(FeatureGates(Capabilities{Catalog: true, ServiceCatalog: true}))
	router.Any("/*path", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/service-offerings", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("enabled service catalog should pass: %d %s", response.Code, response.Body.String())
	}
}

func TestPeopleCapabilityBlocksOnlyPeopleUnitRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(FeatureGates(Capabilities{People: false, Catalog: true}))
	router.POST("/api/v1/units/:unit_id/people", func(c *gin.Context) { c.Status(http.StatusCreated) })
	router.POST("/api/v1/units/:unit_id/people/duplicate-check", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/api/v1/units/:unit_id/spaces", func(c *gin.Context) { c.Status(http.StatusOK) })
	for _, path := range []string{"/api/v1/units/unit-1/people", "/api/v1/units/unit-1/people/duplicate-check"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))
		if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "capability_disabled") {
			t.Fatalf("%s should be blocked by people capability: %d %s", path, response.Code, response.Body.String())
		}
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/units/unit-1/spaces", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("spaces route should not be blocked by people capability: %d", response.Code)
	}
}

func TestEnrollmentCapabilityBlocksOnlyEnrollmentRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(FeatureGates(Capabilities{Enrollments: false, Memberships: true, People: true}))
	router.Any("/*path", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	for _, path := range []string{"/api/v1/units/unit-1/students/student-1/enrollments", "/api/v1/units/unit-1/enrollments/enrollment-1", "/api/v1/units/unit-1/enrollments/enrollment-1/end"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "capability_disabled") {
			t.Fatalf("%s should be blocked by enrollment capability: %d %s", path, response.Code, response.Body.String())
		}
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/units/unit-1/students/student-1", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("student route should not be blocked by enrollment capability: %d", response.Code)
	}
}

func TestMemberBillingRequiresItsOwnCapabilityAndEnrollments(t *testing.T) {
	paths := []string{"/api/v1/units/unit-1/enrollments/enrollment-1/member-billing", "/api/v1/units/unit-1/member-billing/charges/charge-1/manual-payments"}
	for _, caps := range []Capabilities{{MemberBilling: false, Enrollments: true}, {MemberBilling: true, Enrollments: false}} {
		router := gin.New()
		router.Use(FeatureGates(caps))
		router.Any("/*path", func(c *gin.Context) { c.Status(http.StatusNoContent) })
		for _, path := range paths {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusNotFound {
				t.Fatalf("%s should be blocked: %d", path, response.Code)
			}
		}
	}
	router := gin.New()
	router.Use(FeatureGates(Capabilities{MemberBilling: true, Enrollments: true}))
	router.Any("/*path", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	for _, path := range paths {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNoContent {
			t.Fatalf("%s should pass: %d", path, response.Code)
		}
	}
}

func TestCommerceCapabilityOwnsKioskRoutes(t *testing.T) {
	path := "/api/v1/units/unit-1/commerce/sales"
	for _, enabled := range []bool{false, true} {
		router := gin.New()
		router.Use(FeatureGates(Capabilities{Commerce: enabled}))
		router.Any("/*path", func(c *gin.Context) { c.Status(http.StatusNoContent) })
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		expected := http.StatusNotFound
		if enabled {
			expected = http.StatusNoContent
		}
		if response.Code != expected {
			t.Fatalf("enabled=%v: got %d, want %d", enabled, response.Code, expected)
		}
	}
}

func TestReservationCapabilityOwnsAllReservationRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	paths := []string{"/api/v1/units/unit-1/reservations", "/api/v1/units/unit-1/reservations/id/cancel", "/api/v1/units/unit-1/scheduling/sessions/session-1/reservations", "/api/v1/units/unit-1/people/person-1/reservations"}
	for _, capabilities := range []Capabilities{{Reservations: false, Scheduling: true, SchedulingGrid: true, Catalog: true, Memberships: true, Enrollments: true, People: true}, {Reservations: true, Scheduling: true, SchedulingGrid: false, Catalog: true, Memberships: true, Enrollments: true, People: true}} {
		router := gin.New()
		router.Use(FeatureGates(capabilities))
		router.Any("/*path", func(c *gin.Context) { c.Status(http.StatusNoContent) })
		for _, path := range paths {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "capability_disabled") {
				t.Fatalf("%s should be blocked: %d %s", path, response.Code, response.Body.String())
			}
		}
	}
	router := gin.New()
	router.Use(FeatureGates(Capabilities{Reservations: true, Scheduling: true, SchedulingGrid: true, Catalog: true, Memberships: true, Enrollments: true, People: true}))
	router.Any("/*path", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	for _, path := range paths {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNoContent {
			t.Fatalf("%s should pass: %d", path, response.Code)
		}
	}
}
