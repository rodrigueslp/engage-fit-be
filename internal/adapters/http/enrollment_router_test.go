package http

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/memberships"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

type enrollmentRepositoryStub struct {
	item     *domain.Enrollment
	commands map[string]domain.EnrollmentCommandRequest
	version  domain.MembershipPlanVersion
	plan     domain.MembershipPlan
}

func (*enrollmentRepositoryStub) LockCommand(context.Context, domain.ID, string, string) error {
	return nil
}
func (stub *enrollmentRepositoryStub) FindCommand(_ context.Context, organizationID domain.ID, command, key string) (*domain.EnrollmentCommandRequest, error) {
	item, ok := stub.commands[string(organizationID)+":"+command+":"+key]
	if !ok {
		return nil, nil
	}
	copy := item
	return &copy, nil
}
func (stub *enrollmentRepositoryStub) CreateCommand(_ context.Context, request *domain.EnrollmentCommandRequest) error {
	stub.commands[string(request.OrganizationID)+":"+request.Command+":"+request.IdempotencyKey] = *request
	return nil
}
func (*enrollmentRepositoryStub) LockSubject(context.Context, domain.ID, domain.ID, domain.ID) error {
	return nil
}
func (stub *enrollmentRepositoryStub) ListForPerson(context.Context, domain.ID, domain.ID, domain.ID) (*domain.EnrollmentHistory, error) {
	return &domain.EnrollmentHistory{Current: stub.item, History: []domain.Enrollment{}}, nil
}
func (stub *enrollmentRepositoryStub) GetByID(_ context.Context, organizationID, unitID, enrollmentID domain.ID) (*domain.Enrollment, error) {
	if stub.item == nil || stub.item.ID != enrollmentID || stub.item.OrganizationID != organizationID || stub.item.HomeUnitID != unitID {
		return nil, portrepo.ErrEnrollmentNotFound
	}
	copy := *stub.item
	return &copy, nil
}
func (stub *enrollmentRepositoryStub) GetOpenForUpdate(context.Context, domain.ID, domain.ID, domain.ID) (*domain.Enrollment, error) {
	return stub.item, nil
}
func (stub *enrollmentRepositoryStub) GetPlanVersionForEnrollment(_ context.Context, organizationID, unitID, planID, versionID domain.ID) (*domain.MembershipPlan, *domain.MembershipPlanVersion, error) {
	if stub.plan.ID != planID || stub.plan.OrganizationID != organizationID || stub.plan.HomeUnitID != unitID || stub.version.ID != versionID {
		return nil, nil, portrepo.ErrMembershipPlanNotFound
	}
	plan, version := stub.plan, stub.version
	return &plan, &version, nil
}
func (stub *enrollmentRepositoryStub) CreateEnrollment(_ context.Context, enrollment *domain.Enrollment) error {
	enrollment.ID = "66666666-6666-4666-8666-666666666666"
	copy := *enrollment
	stub.item = &copy
	return nil
}
func (stub *enrollmentRepositoryStub) EndEnrollment(_ context.Context, enrollmentID, organizationID, unitID domain.ID, expectedRevision int, endedOn, endedAt time.Time, reason string, actorID domain.ID) (*domain.Enrollment, error) {
	if stub.item == nil || stub.item.ID != enrollmentID || stub.item.Revision != expectedRevision {
		return nil, portrepo.ErrEnrollmentRevisionConflict
	}
	stub.item.LifecycleStatus, stub.item.EffectiveStatus = domain.EnrollmentLifecycleEnded, domain.EnrollmentEffectiveEnded
	stub.item.EndedOn, stub.item.EndedAt, stub.item.EndedReason, stub.item.EndedByUserID, stub.item.Revision = &endedOn, &endedAt, reason, actorID, expectedRevision+1
	copy := *stub.item
	return &copy, nil
}
func (*enrollmentRepositoryStub) CreateEvent(context.Context, *domain.EnrollmentEvent) error {
	return nil
}
func (*enrollmentRepositoryStub) ListPrivacy(context.Context, domain.ID, domain.ID) ([]domain.EnrollmentPrivacyExport, error) {
	return nil, nil
}

type enrollmentSubjectStub struct{ personID domain.ID }

func (stub enrollmentSubjectStub) ResolveEnrollmentSubject(context.Context, domain.ID, domain.ID, domain.ID) (*domain.EnrollmentSubject, error) {
	return &domain.EnrollmentSubject{PersonID: stub.personID, DisplayName: "Ana", Eligible: true}, nil
}

func TestEnrollmentHTTPContractsCapabilityRolesCSRFAndTenant(t *testing.T) {
	unitID := domain.ID("11111111-1111-4111-8111-111111111111")
	organizationID := domain.ID("22222222-2222-4222-8222-222222222222")
	userID := domain.ID("33333333-3333-4333-8333-333333333333")
	studentID := domain.ID("44444444-4444-4444-8444-444444444444")
	personID := domain.ID("55555555-5555-4555-8555-555555555555")
	planID := domain.ID("77777777-7777-4777-8777-777777777777")
	versionID := domain.ID("88888888-8888-4888-8888-888888888888")
	modalityID := domain.ID("99999999-9999-4999-8999-999999999999")
	createBody := `{"plan_id":"` + string(planID) + `","plan_version_id":"` + string(versionID) + `","starts_on":"2026-08-19"}`
	newRouter := func(role domain.UserRole, enabled bool) http.Handler {
		version := domain.MembershipPlanVersion{ID: versionID, OrganizationID: organizationID, PlanID: planID, VersionNumber: 1, Status: domain.MembershipVersionPublished, Name: "CrossFit Mensal", PriceCents: 25000, Currency: "BRL", BillingCycle: domain.MembershipBillingMonthly, Entitlements: []domain.MembershipPlanEntitlement{{ModalityID: modalityID, AccessType: domain.MembershipAccessOpen}}, Summary: "CrossFit · acesso livre · R$ 250/mês"}
		repository := &enrollmentRepositoryStub{commands: map[string]domain.EnrollmentCommandRequest{}, version: version, plan: domain.MembershipPlan{ID: planID, OrganizationID: organizationID, HomeUnitID: unitID, Status: domain.MembershipPlanActive}}
		box := domain.Box{ID: unitID, OrganizationID: organizationID, Name: "Unit", Status: domain.BoxStatusActive}
		service := memberships.NewEnrollmentService(repository, enrollmentSubjectStub{personID: personID}, membershipCatalogResolverStub{}, catalogBoxStub{box: box}, catalogTransactionStub{}, memberships.WithEnrollmentClock(func() time.Time { return time.Date(2026, 8, 19, 15, 0, 0, 0, time.UTC) }))
		user := domain.User{ID: userID, BoxID: unitID, Role: role, AuthVersion: 1, Active: true}
		return NewRouter(RouterDependencies{TokenService: catalogTokenStub{claims: services.AuthClaims{UserID: userID, BoxID: unitID, Role: role, AuthVersion: 1}}, UserRepository: catalogUserStub{user: user}, BoxRepository: catalogBoxStub{box: box}, EnrollmentService: service, Capabilities: middleware.Capabilities{Enrollments: enabled, Memberships: true, People: true}, SessionConfig: middleware.SessionConfig{CookieName: "session", CSRFCookieName: "csrf"}})
	}
	path := "/api/v1/units/" + string(unitID) + "/students/" + string(studentID) + "/enrollments"

	disabled := catalogRequest(newRouter(domain.UserRoleOwner, false), http.MethodGet, path, "", true, false)
	if disabled.Code != http.StatusNotFound || !strings.Contains(disabled.Body.String(), "capability_disabled") {
		t.Fatalf("disabled capability: %d %s", disabled.Code, disabled.Body.String())
	}
	for _, role := range []domain.UserRole{domain.UserRoleCoach, domain.UserRolePlatformAdmin} {
		response := catalogRequest(newRouter(role, true), http.MethodGet, path, "", true, false)
		if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "owner_required") {
			t.Fatalf("role %s access: %d %s", role, response.Code, response.Body.String())
		}
	}
	withoutCSRF := catalogRequest(newRouter(domain.UserRoleOwner, true), http.MethodPost, path, createBody, false, false)
	if withoutCSRF.Code != http.StatusForbidden || !strings.Contains(withoutCSRF.Body.String(), "csrf_invalid") {
		t.Fatalf("csrf: %d %s", withoutCSRF.Code, withoutCSRF.Body.String())
	}
	router := newRouter(domain.UserRoleOwner, true)
	created := membershipBearerRequest(router, http.MethodPost, path, createBody, "enrollment.aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	if created.Code != http.StatusCreated || !strings.Contains(created.Body.String(), `"effective_status":"active"`) || !strings.Contains(created.Body.String(), `"valid_until":null`) {
		t.Fatalf("create contract: %d %s", created.Code, created.Body.String())
	}
	replay := membershipBearerRequest(router, http.MethodPost, path, createBody, "enrollment.aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	if replay.Code != http.StatusOK || replay.Header().Get("Idempotency-Replayed") != "true" {
		t.Fatalf("replay contract: %d headers=%v body=%s", replay.Code, replay.Header(), replay.Body.String())
	}
	list := catalogRequest(router, http.MethodGet, path, "", true, false)
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"current"`) || !strings.Contains(list.Body.String(), `"history":[]`) {
		t.Fatalf("list contract: %d %s", list.Code, list.Body.String())
	}
	wrongUnitPath := "/api/v1/units/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa/students/" + string(studentID) + "/enrollments"
	wrongUnit := catalogRequest(router, http.MethodGet, wrongUnitPath, "", true, false)
	if wrongUnit.Code != http.StatusNotFound || !strings.Contains(wrongUnit.Body.String(), "enrollment_not_found") {
		t.Fatalf("tenant mismatch: %d %s", wrongUnit.Code, wrongUnit.Body.String())
	}
	malformed := membershipBearerRequest(router, http.MethodPost, path, `{"plan_id":"bad","plan_version_id":"bad","starts_on":"2026-08-19"}`, "enrollment.bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb")
	if malformed.Code != http.StatusBadRequest || !strings.Contains(malformed.Body.String(), "invalid_request") {
		t.Fatalf("malformed UUID: %d %s", malformed.Code, malformed.Body.String())
	}
}
