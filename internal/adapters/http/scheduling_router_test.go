package http

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/http/middleware"
	schedulingapp "boxengage/backend/internal/app/scheduling"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

type schedulingRepoStub struct {
	professionals map[domain.ID]*domain.Professional
	commands      map[string]*domain.SchedulingCommandRequest
	unitActive    map[string]bool
	auditEvents   []domain.SchedulingAuditEvent
}

func newSchedulingRepoStub() *schedulingRepoStub {
	return &schedulingRepoStub{
		professionals: make(map[domain.ID]*domain.Professional),
		commands:      make(map[string]*domain.SchedulingCommandRequest),
		unitActive:    make(map[string]bool),
	}
}

func (*schedulingRepoStub) LockCommand(context.Context, domain.ID, string, string) error {
	return nil
}

func (s *schedulingRepoStub) FindCommand(_ context.Context, organizationID domain.ID, command, key string) (*domain.SchedulingCommandRequest, error) {
	return s.commands[string(organizationID)+":"+command+":"+key], nil
}

func (s *schedulingRepoStub) CreateCommand(_ context.Context, req *domain.SchedulingCommandRequest) error {
	s.commands[string(req.OrganizationID)+":"+req.Command+":"+req.IdempotencyKey] = req
	return nil
}

func (*schedulingRepoStub) LockProfessional(context.Context, domain.ID, domain.ID) error {
	return nil
}

func (*schedulingRepoStub) LockAccount(context.Context, domain.ID, domain.ID) error {
	return nil
}

func (s *schedulingRepoStub) ListProfessionals(_ context.Context, organizationID, unitID domain.ID, includeArchived bool, query string) ([]domain.Professional, error) {
	var list []domain.Professional
	for _, p := range s.professionals {
		if p.OrganizationID != organizationID {
			continue
		}
		if !includeArchived && p.Status != domain.ProfessionalStatusActive {
			continue
		}
		if query != "" && !strings.Contains(p.DisplayNameKey, strings.ToLower(query)) {
			continue
		}
		copy := *p
		copy.ActiveInUnit = s.unitActive[string(unitID)+":"+string(p.ID)]
		list = append(list, copy)
	}
	return list, nil
}

func (s *schedulingRepoStub) GetByID(_ context.Context, organizationID, unitID, professionalID domain.ID) (*domain.Professional, error) {
	p, ok := s.professionals[professionalID]
	if !ok || p.OrganizationID != organizationID {
		return nil, portrepo.ErrSchedulingProfessionalNotFound
	}
	copy := *p
	copy.ActiveInUnit = s.unitActive[string(unitID)+":"+string(p.ID)]
	return &copy, nil
}

func (s *schedulingRepoStub) GetByIDForUpdate(_ context.Context, organizationID, professionalID domain.ID) (*domain.Professional, error) {
	p, ok := s.professionals[professionalID]
	if !ok || p.OrganizationID != organizationID {
		return nil, portrepo.ErrSchedulingProfessionalNotFound
	}
	copy := *p
	return &copy, nil
}

func (s *schedulingRepoStub) FindByLinkedUser(_ context.Context, organizationID, userID domain.ID) (*domain.Professional, error) {
	for _, p := range s.professionals {
		if p.OrganizationID == organizationID && p.LinkedUserID != nil && *p.LinkedUserID == userID {
			copy := *p
			return &copy, nil
		}
	}
	return nil, nil
}

func (s *schedulingRepoStub) CreateProfessional(_ context.Context, prof *domain.Professional, unitID domain.ID) error {
	copy := *prof
	s.professionals[prof.ID] = &copy
	s.unitActive[string(unitID)+":"+string(prof.ID)] = true
	return nil
}

func (s *schedulingRepoStub) UpdateProfessional(_ context.Context, prof *domain.Professional) error {
	copy := *prof
	s.professionals[prof.ID] = &copy
	return nil
}

func (s *schedulingRepoStub) SetUnitActive(_ context.Context, _, unitID, professionalID domain.ID, active bool) error {
	s.unitActive[string(unitID)+":"+string(professionalID)] = active
	return nil
}

func (s *schedulingRepoStub) CreateAuditEvent(_ context.Context, ev *domain.SchedulingAuditEvent) error {
	s.auditEvents = append(s.auditEvents, *ev)
	return nil
}

func (s *schedulingRepoStub) ListAuditEventsForProfessional(_ context.Context, organizationID, _, professionalID domain.ID) ([]domain.SchedulingAuditEvent, error) {
	var list []domain.SchedulingAuditEvent
	for _, ev := range s.auditEvents {
		if ev.OrganizationID == organizationID && ev.EntityID == professionalID {
			list = append(list, ev)
		}
	}
	return list, nil
}

func (s *schedulingRepoStub) CountActiveInUnit(_ context.Context, organizationID, unitID domain.ID) (int64, error) {
	var count int64
	for _, p := range s.professionals {
		if p.OrganizationID == organizationID && p.Status == domain.ProfessionalStatusActive && s.unitActive[string(unitID)+":"+string(p.ID)] {
			count++
		}
	}
	return count, nil
}

type schedulingContextStub struct {
	orgID  domain.ID
	unitID domain.ID
	role   string
}

func (stub schedulingContextStub) ResolveSchedulingUnitContext(_ context.Context, _, unitID domain.ID) (*domain.SchedulingUnitContext, error) {
	if unitID != stub.unitID {
		return &domain.SchedulingUnitContext{Authorized: false}, nil
	}
	return &domain.SchedulingUnitContext{
		OrganizationID: stub.orgID,
		UnitID:         stub.unitID,
		UnitName:       "CrossFit Test",
		Role:           stub.role,
		Authorized:     true,
	}, nil
}

type schedulingAccountStub struct {
	candidates []domain.ProfessionalAccountCandidate
}

func (stub schedulingAccountStub) ListProfessionalAccountCandidates(_ context.Context, _, _ domain.ID) ([]domain.ProfessionalAccountCandidate, error) {
	return stub.candidates, nil
}

func (stub schedulingAccountStub) ResolveProfessionalAccount(_ context.Context, _, _, userID domain.ID) (*domain.ProfessionalAccountCandidate, error) {
	for _, c := range stub.candidates {
		if c.UserID == userID {
			return &c, nil
		}
	}
	return nil, nil
}

func TestSchedulingHTTPContractsCapabilityRolesCSRFAndTenant(t *testing.T) {
	unitID := domain.ID("11111111-1111-4111-8111-111111111111")
	orgID := domain.ID("22222222-2222-4222-8222-222222222222")
	userID := domain.ID("33333333-3333-4333-8333-333333333333")
	coachUserID := domain.ID("44444444-4444-4444-8444-444444444444")

	repo := newSchedulingRepoStub()
	candidates := []domain.ProfessionalAccountCandidate{
		{UserID: coachUserID, DisplayName: "Coach Pedro", Role: "COACH", Active: true, AuthorizedInUnit: true},
	}

	newRouter := func(role domain.UserRole, enabled bool) http.Handler {
		ctxAdapter := schedulingContextStub{orgID: orgID, unitID: unitID, role: string(role)}
		accAdapter := schedulingAccountStub{candidates: candidates}
		service := schedulingapp.NewService(repo, ctxAdapter, accAdapter, catalogTransactionStub{}, schedulingapp.WithClock(func() time.Time { return time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC) }))
		box := domain.Box{ID: unitID, OrganizationID: orgID, Name: "CrossFit Test", Status: domain.BoxStatusActive}
		user := domain.User{ID: userID, BoxID: unitID, Role: role, AuthVersion: 1, Active: true}

		return NewRouter(RouterDependencies{
			TokenService:      catalogTokenStub{claims: services.AuthClaims{UserID: userID, BoxID: unitID, Role: role, AuthVersion: 1}},
			UserRepository:    catalogUserStub{user: user},
			BoxRepository:     catalogBoxStub{box: box},
			SchedulingService: service,
			Capabilities:      middleware.Capabilities{Scheduling: enabled},
			SessionConfig:     middleware.SessionConfig{CookieName: "session", CSRFCookieName: "csrf"},
		})
	}

	path := "/api/v1/units/" + string(unitID) + "/scheduling/professionals"

	// 1. Feature disabled returns 404 capability_disabled
	disabled := catalogRequest(newRouter(domain.UserRoleOwner, false), http.MethodGet, path, "", true, false)
	if disabled.Code != http.StatusNotFound || !strings.Contains(disabled.Body.String(), "capability_disabled") {
		t.Fatalf("disabled capability: %d %s", disabled.Code, disabled.Body.String())
	}

	// 2. CSRF enforcement on POST
	withoutCSRF := catalogRequest(newRouter(domain.UserRoleOwner, true), http.MethodPost, path, `{"display_name":"Marina"}`, false, false)
	if withoutCSRF.Code != http.StatusForbidden || !strings.Contains(withoutCSRF.Body.String(), "csrf_invalid") {
		t.Fatalf("csrf check failed: %d %s", withoutCSRF.Code, withoutCSRF.Body.String())
	}

	// 3. Owner creates professional
	router := newRouter(domain.UserRoleOwner, true)
	createBody := `{"display_name":"  Marina Souza  ","linked_user_id":"` + string(coachUserID) + `"}`
	created := membershipBearerRequest(router, http.MethodPost, path, createBody, "prof.create.11111111-1111-4111-8111-111111111111")
	if created.Code != http.StatusCreated || !strings.Contains(created.Body.String(), `"display_name":"Marina Souza"`) || !strings.Contains(created.Body.String(), `"revision":1`) {
		t.Fatalf("create failed: %d %s", created.Code, created.Body.String())
	}

	// 4. Idempotency replay returns 200 with Idempotency-Replayed: true
	replay := membershipBearerRequest(router, http.MethodPost, path, createBody, "prof.create.11111111-1111-4111-8111-111111111111")
	if replay.Code != http.StatusOK || replay.Header().Get("Idempotency-Replayed") != "true" {
		t.Fatalf("replay failed: %d header=%v body=%s", replay.Code, replay.Header(), replay.Body.String())
	}

	// 5. Coach lists active professionals (read-only, linked_user_id stripped)
	coachRouter := newRouter(domain.UserRoleCoach, true)
	coachList := catalogRequest(coachRouter, http.MethodGet, path, "", true, false)
	if coachList.Code != http.StatusOK || !strings.Contains(coachList.Body.String(), `"display_name":"Marina Souza"`) {
		t.Fatalf("coach list failed: %d %s", coachList.Code, coachList.Body.String())
	}
	if strings.Contains(coachList.Body.String(), string(coachUserID)) {
		t.Fatalf("coach list must NOT expose linked user ID: %s", coachList.Body.String())
	}

	// 6. Coach cannot mutate (POST/candidates is Owner only)
	coachCreate := membershipBearerRequest(coachRouter, http.MethodPost, path, `{"display_name":"Test"}`, "prof.create.22222222-2222-4222-8222-222222222222")
	if coachCreate.Code != http.StatusForbidden || !strings.Contains(coachCreate.Body.String(), "owner_required") {
		t.Fatalf("coach create must be forbidden: %d %s", coachCreate.Code, coachCreate.Body.String())
	}

	candPath := "/api/v1/units/" + string(unitID) + "/scheduling/professional-account-candidates"
	candResp := catalogRequest(router, http.MethodGet, candPath, "", true, false)
	if candResp.Code != http.StatusOK || !strings.Contains(candResp.Body.String(), "Coach Pedro") {
		t.Fatalf("candidates list failed: %d %s", candResp.Code, candResp.Body.String())
	}

	// 7. Wrong unit returns 404 unit_not_found
	wrongUnitPath := "/api/v1/units/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa/scheduling/professionals"
	wrongUnit := catalogRequest(router, http.MethodGet, wrongUnitPath, "", true, false)
	if wrongUnit.Code != http.StatusNotFound || !strings.Contains(wrongUnit.Body.String(), "unit_not_found") {
		t.Fatalf("wrong unit must return unit_not_found: %d %s", wrongUnit.Code, wrongUnit.Body.String())
	}
}
