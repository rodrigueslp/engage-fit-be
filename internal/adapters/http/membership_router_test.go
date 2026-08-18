package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/memberships"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

type membershipRepositoryStub struct {
	plans    map[domain.ID]domain.MembershipPlan
	commands map[string]domain.MembershipCommandRequest
}

func (s *membershipRepositoryStub) ListPlans(_ context.Context, organizationID, unitID domain.ID, includeArchived bool) ([]domain.MembershipPlan, error) {
	result := []domain.MembershipPlan{}
	for _, plan := range s.plans {
		if plan.OrganizationID == organizationID && plan.HomeUnitID == unitID && (includeArchived || plan.Status == domain.MembershipPlanActive) {
			result = append(result, plan)
		}
	}
	return result, nil
}
func (s *membershipRepositoryStub) GetPlan(_ context.Context, organizationID, unitID, planID domain.ID) (*domain.MembershipPlan, error) {
	plan, ok := s.plans[planID]
	if !ok || plan.OrganizationID != organizationID || plan.HomeUnitID != unitID {
		return nil, portrepo.ErrMembershipPlanNotFound
	}
	copy := plan
	return &copy, nil
}
func (*membershipRepositoryStub) LockCommand(context.Context, domain.ID, string, string) error {
	return nil
}
func (s *membershipRepositoryStub) FindCommand(_ context.Context, organizationID domain.ID, command, key string) (*domain.MembershipCommandRequest, error) {
	item, ok := s.commands[string(organizationID)+":"+command+":"+key]
	if !ok {
		return nil, nil
	}
	copy := item
	return &copy, nil
}
func (s *membershipRepositoryStub) CreateCommand(_ context.Context, request *domain.MembershipCommandRequest) error {
	s.commands[string(request.OrganizationID)+":"+request.Command+":"+request.IdempotencyKey] = *request
	return nil
}
func (s *membershipRepositoryStub) CreatePlan(_ context.Context, plan *domain.MembershipPlan, version *domain.MembershipPlanVersion, entitlements []domain.MembershipPlanEntitlement) error {
	plan.ID = "plan-created"
	version.ID = "version-created"
	version.PlanID = plan.ID
	version.OrganizationID = plan.OrganizationID
	version.Entitlements = entitlements
	version.Summary = domain.MembershipPlanSummary(*version)
	copyVersion := *version
	plan.Draft = &copyVersion
	plan.Versions = []domain.MembershipPlanVersion{copyVersion}
	s.plans[plan.ID] = *plan
	return nil
}
func (*membershipRepositoryStub) UpdateDraft(context.Context, domain.ID, domain.ID, domain.ID, int, domain.MembershipPlanVersion, []domain.MembershipPlanEntitlement) (*domain.MembershipPlan, error) {
	return nil, portrepo.ErrMembershipDraftNotFound
}
func (*membershipRepositoryStub) PublishDraft(context.Context, domain.ID, domain.ID, domain.ID, int, time.Time) (*domain.MembershipPlan, error) {
	return nil, portrepo.ErrMembershipDraftNotFound
}
func (*membershipRepositoryStub) CreateNewVersion(context.Context, domain.ID, domain.ID, domain.ID, time.Time) (*domain.MembershipPlan, error) {
	return nil, portrepo.ErrMembershipDraftExists
}
func (*membershipRepositoryStub) SetPlanStatus(context.Context, domain.ID, domain.ID, domain.ID, domain.MembershipPlanStatus, time.Time) (*domain.MembershipPlan, error) {
	return nil, portrepo.ErrMembershipPlanNotFound
}
func (*membershipRepositoryStub) CreateAuditEvent(context.Context, *domain.MembershipPlanAuditEvent) error {
	return nil
}

type membershipCatalogResolverStub struct{}

func (membershipCatalogResolverStub) ResolvePlanReferences(_ context.Context, _ domain.ID, _ domain.ID, modalityIDs, serviceIDs []domain.ID) ([]domain.CatalogPlanReference, error) {
	result := []domain.CatalogPlanReference{}
	for _, id := range modalityIDs {
		result = append(result, domain.CatalogPlanReference{ID: id, Type: "modality", Name: "Pilates", Status: domain.CatalogStatusActive, ActiveInUnit: true})
	}
	for _, id := range serviceIDs {
		result = append(result, domain.CatalogPlanReference{ID: id, Type: "service_offering", Name: "Recovery", Status: domain.CatalogStatusActive, ActiveInUnit: true})
	}
	return result, nil
}

func TestMembershipHTTPContractsCapabilityRolesAndCSRF(t *testing.T) {
	unitID, userID := domain.ID("11111111-1111-4111-8111-111111111111"), domain.ID("22222222-2222-4222-8222-222222222222")
	planBody := `{"name":"Pilates 2x","description":"","price_cents":26000,"billing_cycle":"monthly","validity_count":null,"validity_unit":null,"entitlements":[{"modality_id":"33333333-3333-4333-8333-333333333333","service_offering_id":null,"access_type":"sessions_per_period","quantity":2,"period_unit":"week","daily_limit":1,"delivery_mode":"group","booking_required":true}]}`
	newRouter := func(role domain.UserRole, enabled bool) http.Handler {
		repository := &membershipRepositoryStub{plans: map[domain.ID]domain.MembershipPlan{}, commands: map[string]domain.MembershipCommandRequest{}}
		box := domain.Box{ID: unitID, OrganizationID: "organization-1", Name: "Unit", Status: domain.BoxStatusActive}
		service := memberships.NewService(repository, membershipCatalogResolverStub{}, catalogBoxStub{box: box}, catalogTransactionStub{})
		user := domain.User{ID: userID, BoxID: unitID, Role: role, AuthVersion: 1, Active: true}
		return NewRouter(RouterDependencies{TokenService: catalogTokenStub{claims: services.AuthClaims{UserID: userID, BoxID: unitID, Role: role, AuthVersion: 1}}, UserRepository: catalogUserStub{user: user}, BoxRepository: catalogBoxStub{box: box}, MembershipService: service, Capabilities: middleware.Capabilities{Memberships: enabled}, SessionConfig: middleware.SessionConfig{CookieName: "session", CSRFCookieName: "csrf"}})
	}

	disabled := catalogRequest(newRouter(domain.UserRoleOwner, false), http.MethodGet, "/api/v1/membership-plans", "", true, false)
	if disabled.Code != http.StatusNotFound || !strings.Contains(disabled.Body.String(), "capability_disabled") {
		t.Fatalf("disabled capability: %d %s", disabled.Code, disabled.Body.String())
	}
	coach := catalogRequest(newRouter(domain.UserRoleCoach, true), http.MethodGet, "/api/v1/membership-plans", "", true, false)
	if coach.Code != http.StatusForbidden || !strings.Contains(coach.Body.String(), "owner_required") {
		t.Fatalf("coach access: %d %s", coach.Code, coach.Body.String())
	}
	platformAdmin := catalogRequest(newRouter(domain.UserRolePlatformAdmin, true), http.MethodGet, "/api/v1/membership-plans", "", true, false)
	if platformAdmin.Code != http.StatusForbidden || !strings.Contains(platformAdmin.Body.String(), "owner_required") {
		t.Fatalf("platform admin access: %d %s", platformAdmin.Code, platformAdmin.Body.String())
	}
	withoutCSRF := catalogRequest(newRouter(domain.UserRoleOwner, true), http.MethodPost, "/api/v1/membership-plans", planBody, false, false)
	if withoutCSRF.Code != http.StatusForbidden || !strings.Contains(withoutCSRF.Body.String(), "csrf_invalid") {
		t.Fatalf("csrf: %d %s", withoutCSRF.Code, withoutCSRF.Body.String())
	}

	router := newRouter(domain.UserRoleOwner, true)
	request := catalogRequest(router, http.MethodPost, "/api/v1/membership-plans", planBody, true, false)
	if request.Code != http.StatusUnprocessableEntity || !strings.Contains(request.Body.String(), "membership_plan_invalid") {
		t.Fatalf("missing key: %d %s", request.Code, request.Body.String())
	}
	response := membershipBearerRequest(router, http.MethodPost, "/api/v1/membership-plans", planBody, "create-plan-http-1")
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), `"name":"Pilates 2x"`) || !strings.Contains(response.Body.String(), `"summary":"Pilates · 2 sessões por semana · R$ 260/mês"`) {
		t.Fatalf("create contract: %d %s", response.Code, response.Body.String())
	}
	list := catalogRequest(router, http.MethodGet, "/api/v1/membership-plans", "", true, false)
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), "plan-created") {
		t.Fatalf("list contract: %d %s", list.Code, list.Body.String())
	}
}

func membershipBearerRequest(handler http.Handler, method, path, body, idempotencyKey string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer token")
	request.Header.Set("Idempotency-Key", idempotencyKey)
	handler.ServeHTTP(response, request)
	return response
}
