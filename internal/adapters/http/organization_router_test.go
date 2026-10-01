package http

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/auth"
	"boxengage/backend/internal/app/organizations"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

type organizationRepositoryStub struct {
	organization domain.Organization
	units        []domain.Box
	access       map[domain.ID][]domain.ID
	members      map[domain.ID]domain.User
}

func (s *organizationRepositoryStub) ResolveUserUnitAccess(_ context.Context, userID, unitID domain.ID) (*domain.UnitAccessContext, error) {
	member, ok := s.members[userID]
	if !ok {
		return nil, portrepo.ErrOrganizationAccessNotFound
	}
	allowed := s.access[userID]
	for _, id := range allowed {
		if id == unitID {
			for _, unit := range s.units {
				if unit.ID == unitID {
					return &domain.UnitAccessContext{Organization: s.organization, Unit: unit, Role: member.Role}, nil
				}
			}
		}
	}
	return nil, portrepo.ErrOrganizationAccessNotFound
}

func (s *organizationRepositoryStub) ListAccessibleUnits(_ context.Context, userID, _ domain.ID) ([]domain.Box, error) {
	result := []domain.Box{}
	for _, unit := range s.units {
		for _, id := range s.access[userID] {
			if id == unit.ID {
				result = append(result, unit)
			}
		}
	}
	return result, nil
}

func (s *organizationRepositoryStub) FindOrganization(_ context.Context, organizationID domain.ID) (*domain.Organization, error) {
	if organizationID != s.organization.ID {
		return nil, portrepo.ErrOrganizationAccessNotFound
	}
	item := s.organization
	return &item, nil
}

func (s *organizationRepositoryStub) FindOrganizationOwner(_ context.Context, organizationID domain.ID) (*domain.User, error) {
	for _, member := range s.members {
		if organizationID == s.organization.ID && member.Role == domain.UserRoleOwner {
			item := member
			return &item, nil
		}
	}
	return nil, portrepo.ErrOrganizationMemberNotFound
}

func (s *organizationRepositoryStub) CreateUnit(_ context.Context, unit *domain.Box, grantUserID, _ domain.ID, _ domain.OrganizationAuditEvent) error {
	unit.ID = "unit-created"
	s.units = append(s.units, *unit)
	s.access[grantUserID] = append(s.access[grantUserID], unit.ID)
	return nil
}

func (s *organizationRepositoryStub) CountNonArchivedUnits(context.Context, domain.ID) (int64, error) {
	var count int64
	for _, unit := range s.units {
		if unit.EffectiveStatus() != domain.BoxStatusArchived {
			count++
		}
	}
	return count, nil
}

func (s *organizationRepositoryStub) UpdateUnit(_ context.Context, unit domain.Box, _ domain.OrganizationAuditEvent) error {
	for index := range s.units {
		if s.units[index].ID == unit.ID {
			s.units[index] = unit
			return nil
		}
	}
	return portrepo.ErrOrganizationUnitNotFound
}

func (s *organizationRepositoryStub) FindOrganizationMember(_ context.Context, _ domain.ID, userID domain.ID) (*domain.User, error) {
	member, ok := s.members[userID]
	if !ok {
		return nil, portrepo.ErrOrganizationMemberNotFound
	}
	return &member, nil
}

func (s *organizationRepositoryStub) ListMemberUnitIDs(_ context.Context, _ domain.ID, userID domain.ID) ([]domain.ID, error) {
	return append([]domain.ID(nil), s.access[userID]...), nil
}

func (s *organizationRepositoryStub) ReplaceMemberUnitAccess(_ context.Context, _ domain.ID, userID domain.ID, unitIDs []domain.ID, _ domain.ID, _ domain.OrganizationAuditEvent) error {
	s.access[userID] = append([]domain.ID(nil), unitIDs...)
	return nil
}

func TestMultiUnitHTTPContextAndAuthorization(t *testing.T) {
	ownerID, coachID := domain.ID("owner-1"), domain.ID("coach-1")
	unitOne := domain.Box{ID: "unit-1", OrganizationID: "organization-1", Name: "Centro", NameKey: "centro", Timezone: domain.DefaultUnitTimezone, Status: domain.BoxStatusActive}
	unitTwo := domain.Box{ID: "unit-2", OrganizationID: "organization-1", Name: "Norte", NameKey: "norte", Timezone: "America/Manaus", Status: domain.BoxStatusActive}
	hiddenUnit := domain.Box{ID: "unit-hidden", OrganizationID: "organization-1", Name: "Restrita", NameKey: "restrita", Timezone: domain.DefaultUnitTimezone, Status: domain.BoxStatusActive}
	owner := domain.User{ID: ownerID, BoxID: unitOne.ID, Role: domain.UserRoleOwner, AuthVersion: 1, Active: true}
	repository := &organizationRepositoryStub{
		organization: domain.Organization{ID: "organization-1", Name: "Academia", Status: domain.OrganizationStatusActive},
		units:        []domain.Box{unitOne, unitTwo, hiddenUnit},
		access:       map[domain.ID][]domain.ID{ownerID: {unitOne.ID, unitTwo.ID}, coachID: {unitOne.ID}},
		members:      map[domain.ID]domain.User{ownerID: owner, coachID: {ID: coachID, BoxID: unitOne.ID, Role: domain.UserRoleCoach, AuthVersion: 1, Active: true}},
	}
	organizationService := organizations.NewService(repository)
	tokenService := catalogTokenStub{claims: services.AuthClaims{UserID: ownerID, BoxID: unitOne.ID, Role: domain.UserRoleOwner, AuthVersion: 1}}
	router := NewRouter(RouterDependencies{
		TokenService: tokenService, UserRepository: catalogUserStub{user: owner}, BoxRepository: catalogBoxStub{box: unitOne},
		OrganizationService: organizationService, UnitAccessAuthorizer: repository,
		SwitchUnitContextUseCase: auth.NewSwitchUnitContextUseCase(catalogUserStub{user: owner}, repository, tokenService),
		SessionConfig:            middleware.SessionConfig{CookieName: "session", CSRFCookieName: "csrf"},
	})

	list := catalogRequest(router, http.MethodGet, "/api/v1/units", "", true, false)
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"name":"Centro"`) || !strings.Contains(list.Body.String(), `"name":"Norte"`) || strings.Contains(list.Body.String(), `"name":"Restrita"`) {
		t.Fatalf("list units: %d %s", list.Code, list.Body.String())
	}
	hiddenUpdate := catalogRequest(router, http.MethodPatch, "/api/v1/units/unit-hidden", `{"name":"Não autorizado"}`, true, false)
	if hiddenUpdate.Code != http.StatusNotFound {
		t.Fatalf("update unit outside allowlist: %d %s", hiddenUpdate.Code, hiddenUpdate.Body.String())
	}
	create := catalogRequest(router, http.MethodPost, "/api/v1/units", `{"name":"  Unidade   Sul ","timezone":"America/Sao_Paulo"}`, true, false)
	if create.Code != http.StatusCreated || !strings.Contains(create.Body.String(), `"name":"Unidade Sul"`) {
		t.Fatalf("create unit: %d %s", create.Code, create.Body.String())
	}
	switchResponse := catalogRequest(router, http.MethodPost, "/api/v1/auth/unit-context", `{"unit_id":"unit-2"}`, true, false)
	if switchResponse.Code != http.StatusOK || !strings.Contains(switchResponse.Body.String(), "access_token") {
		t.Fatalf("switch unit: %d %s", switchResponse.Code, switchResponse.Body.String())
	}
	switchedToken := catalogTokenStub{claims: services.AuthClaims{UserID: ownerID, BoxID: unitTwo.ID, Role: domain.UserRoleOwner, AuthVersion: 1}}
	switchedRouter := NewRouter(RouterDependencies{
		TokenService: switchedToken, UserRepository: catalogUserStub{user: owner}, BoxRepository: catalogBoxStub{box: unitTwo},
		OrganizationService: organizationService, UnitAccessAuthorizer: repository,
		SwitchUnitContextUseCase: auth.NewSwitchUnitContextUseCase(catalogUserStub{user: owner}, repository, switchedToken),
		CurrentUserUseCase:       auth.NewGetCurrentUserUseCase(catalogUserStub{user: owner}),
		SessionConfig:            middleware.SessionConfig{CookieName: "session", CSRFCookieName: "csrf"},
	})
	switchedMe := catalogRequest(switchedRouter, http.MethodGet, "/api/v1/auth/me", "", true, false)
	if switchedMe.Code != http.StatusOK || !strings.Contains(switchedMe.Body.String(), `"box_id":"unit-2"`) {
		t.Fatalf("selected unit context: %d %s", switchedMe.Code, switchedMe.Body.String())
	}
	unknown := catalogRequest(router, http.MethodPost, "/api/v1/auth/unit-context", `{"unit_id":"other-organization-unit"}`, true, false)
	if unknown.Code != http.StatusNotFound || !strings.Contains(unknown.Body.String(), "unit_not_found") {
		t.Fatalf("unknown unit switch: %d %s", unknown.Code, unknown.Body.String())
	}
	coachAccess := catalogRequest(router, http.MethodPut, "/api/v1/team/coaches/coach-1/units", `{"unit_ids":["unit-1","unit-2"]}`, true, false)
	if coachAccess.Code != http.StatusOK || !strings.Contains(coachAccess.Body.String(), "unit-2") {
		t.Fatalf("replace coach access: %d %s", coachAccess.Code, coachAccess.Body.String())
	}
	hiddenCoachAccess := catalogRequest(router, http.MethodPut, "/api/v1/team/coaches/coach-1/units", `{"unit_ids":["unit-hidden"]}`, true, false)
	if hiddenCoachAccess.Code != http.StatusBadRequest {
		t.Fatalf("grant unit outside owner allowlist: %d %s", hiddenCoachAccess.Code, hiddenCoachAccess.Body.String())
	}
	coach := repository.members[coachID]
	coachToken := catalogTokenStub{claims: services.AuthClaims{UserID: coachID, BoxID: unitOne.ID, Role: domain.UserRoleCoach, AuthVersion: 1}}
	coachRouter := NewRouter(RouterDependencies{TokenService: coachToken, UserRepository: catalogUserStub{user: coach}, BoxRepository: catalogBoxStub{box: unitOne}, OrganizationService: organizationService, UnitAccessAuthorizer: repository, SwitchUnitContextUseCase: auth.NewSwitchUnitContextUseCase(catalogUserStub{user: coach}, repository, coachToken)})
	coachList := catalogRequest(coachRouter, http.MethodGet, "/api/v1/units", "", true, false)
	if coachList.Code != http.StatusOK {
		t.Fatalf("coach unit list: %d %s", coachList.Code, coachList.Body.String())
	}
	coachCreate := catalogRequest(coachRouter, http.MethodPost, "/api/v1/units", `{"name":"Forbidden"}`, true, false)
	if coachCreate.Code != http.StatusForbidden || !strings.Contains(coachCreate.Body.String(), "owner_required") {
		t.Fatalf("coach create unit: %d %s", coachCreate.Code, coachCreate.Body.String())
	}
	archiveCurrent := catalogRequest(router, http.MethodPatch, "/api/v1/units/unit-1", `{"status":"archived","reason":"reorganização"}`, true, false)
	if archiveCurrent.Code != http.StatusConflict || !strings.Contains(archiveCurrent.Body.String(), "current_unit_archive") {
		t.Fatalf("archive current unit: %d %s", archiveCurrent.Code, archiveCurrent.Body.String())
	}
	archiveOther := catalogRequest(router, http.MethodPatch, "/api/v1/units/unit-2", `{"status":"archived","reason":"reorganização"}`, true, false)
	if archiveOther.Code != http.StatusOK || !strings.Contains(archiveOther.Body.String(), `"status":"archived"`) {
		t.Fatalf("archive other unit: %d %s", archiveOther.Code, archiveOther.Body.String())
	}
}
