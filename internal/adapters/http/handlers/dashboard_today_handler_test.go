package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/dashboard"
	"boxengage/backend/internal/app/rewards"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type handlerTodayBoxes struct{}

func (handlerTodayBoxes) FindByID(context.Context, domain.ID) (*domain.Box, error) {
	return &domain.Box{ID: "unit", OrganizationID: "org", Timezone: "America/Sao_Paulo"}, nil
}
func (handlerTodayBoxes) ListAll(context.Context) ([]domain.Box, error)                  { return nil, nil }
func (handlerTodayBoxes) Save(context.Context, *domain.Box) error                        { return nil }
func (handlerTodayBoxes) SaveWithOwner(context.Context, *domain.Box, *domain.User) error { return nil }
func (handlerTodayBoxes) Update(context.Context, domain.Box) error                       { return nil }

type handlerTodayReader struct{}

func (handlerTodayReader) ReadToday(context.Context, domain.ID, domain.ID, string, time.Time, time.Time) (*domain.TodayOperations, error) {
	return &domain.TodayOperations{Charges: []domain.TodayCharge{{ChargeID: "charge"}}, PeopleActions: []domain.TodayPeopleAction{{Kind: "health_review"}}}, nil
}

func TestDashboardTodayRemovesSensitiveQueuesForCoach(t *testing.T) {
	gin.SetMode(gin.TestMode)
	today := dashboard.NewGetTodayOperationsUseCase(handlerTodayBoxes{}, handlerTodayReader{})
	handler := NewDashboardHandler(dashboard.GetDashboardSummaryUseCase{}, dashboard.ListActiveCampaignsUseCase{}, dashboard.ListNearGoalStudentsUseCase{}, dashboard.ListAtRiskStudentsUseCase{}, rewards.ListPendingRewardDeliveriesUseCase{}, today)
	router := gin.New()
	router.Use(func(c *gin.Context) { middleware.SetAuthContext(c, "coach", "unit", domain.UserRoleCoach); c.Next() })
	router.GET("/api/v1/dashboard/today", handler.Today)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/today", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body domain.TodayOperations
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Charges) != 0 || len(body.PeopleActions) != 0 {
		t.Fatalf("coach received sensitive queues: %+v", body)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"charges", "people_actions"} {
		if got := bytes.TrimSpace(payload[field]); !bytes.Equal(got, []byte("[]")) {
			t.Fatalf("coach %s must be an empty array, got %s", field, got)
		}
	}
}
