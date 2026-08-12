package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/catalog"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

type catalogTokenStub struct{ claims services.AuthClaims }

func (s catalogTokenStub) Generate(context.Context, services.AuthClaims) (string, error) {
	return "token", nil
}
func (s catalogTokenStub) Validate(context.Context, string) (*services.AuthClaims, error) {
	return &s.claims, nil
}

type catalogUserStub struct{ user domain.User }

func (s catalogUserStub) FindByID(context.Context, domain.ID) (*domain.User, error) {
	item := s.user
	return &item, nil
}
func (catalogUserStub) FindByEmail(context.Context, string) (*domain.User, error) {
	return nil, errors.New("not implemented")
}
func (catalogUserStub) FindOwnerByBoxID(context.Context, domain.ID) (*domain.User, error) {
	return nil, errors.New("not implemented")
}
func (catalogUserStub) Save(context.Context, *domain.User) error                { return nil }
func (catalogUserStub) UpdatePassword(context.Context, domain.ID, string) error { return nil }
func (catalogUserStub) BumpAuthVersion(context.Context, domain.ID) error        { return nil }
func (catalogUserStub) UpdatePlatformAdminCredentials(context.Context, domain.ID, string, string) error {
	return nil
}

type catalogBoxStub struct{ box domain.Box }

func (s catalogBoxStub) FindByID(context.Context, domain.ID) (*domain.Box, error) {
	item := s.box
	return &item, nil
}
func (catalogBoxStub) ListAll(context.Context) ([]domain.Box, error)                  { return nil, nil }
func (catalogBoxStub) Save(context.Context, *domain.Box) error                        { return nil }
func (catalogBoxStub) SaveWithOwner(context.Context, *domain.Box, *domain.User) error { return nil }
func (catalogBoxStub) Update(context.Context, domain.Box) error                       { return nil }

type catalogTransactionStub struct{}

func (catalogTransactionStub) WithinTransaction(ctx context.Context, operation func(context.Context) error) error {
	return operation(ctx)
}

type catalogRepositoryStub struct {
	current    domain.CatalogContext
	modalities []domain.Modality
}

func (s *catalogRepositoryStub) GetContext(_ context.Context, unitID domain.ID) (*domain.CatalogContext, error) {
	if unitID != s.current.Unit.ID {
		return nil, portrepo.ErrCatalogUnitNotFound
	}
	item := s.current
	return &item, nil
}
func (s *catalogRepositoryStub) ListModalities(context.Context, domain.ID, domain.ID, bool) ([]domain.Modality, error) {
	return append([]domain.Modality(nil), s.modalities...), nil
}
func (s *catalogRepositoryStub) FindModality(_ context.Context, organizationID, unitID, modalityID domain.ID) (*domain.Modality, error) {
	for _, item := range s.modalities {
		if item.ID == modalityID && item.OrganizationID == organizationID {
			copy := item
			return &copy, nil
		}
	}
	return nil, portrepo.ErrCatalogModalityNotFound
}
func (s *catalogRepositoryStub) CreateModality(_ context.Context, item *domain.Modality) error {
	for _, existing := range s.modalities {
		if existing.NameKey == item.NameKey {
			return portrepo.ErrCatalogModalityNameConflict
		}
	}
	item.ID = domain.ID("modality-created")
	s.modalities = append(s.modalities, *item)
	return nil
}
func (s *catalogRepositoryStub) UpdateModality(context.Context, domain.Modality) error { return nil }
func (s *catalogRepositoryStub) CreateUnitModality(context.Context, *domain.UnitModality) error {
	return nil
}
func (s *catalogRepositoryStub) SetUnitModalityActive(context.Context, domain.ID, domain.ID, domain.ID, bool) error {
	return nil
}
func (s *catalogRepositoryStub) ListSpaces(context.Context, domain.ID, bool) ([]domain.Space, error) {
	return []domain.Space{}, nil
}
func (s *catalogRepositoryStub) FindSpace(context.Context, domain.ID, domain.ID) (*domain.Space, error) {
	return nil, portrepo.ErrCatalogSpaceNotFound
}
func (s *catalogRepositoryStub) CreateSpace(context.Context, *domain.Space) error { return nil }
func (s *catalogRepositoryStub) UpdateSpace(context.Context, domain.Space) error  { return nil }
func (s *catalogRepositoryStub) CreateAuditEvent(context.Context, *domain.CatalogAuditEvent) error {
	return nil
}

func TestCatalogHTTPContractsRolesAndCSRF(t *testing.T) {
	for _, role := range []domain.UserRole{domain.UserRoleOwner, domain.UserRoleCoach} {
		t.Run(string(role), func(t *testing.T) {
			unitID, userID := domain.ID("11111111-1111-4111-8111-111111111111"), domain.ID("22222222-2222-4222-8222-222222222222")
			now := time.Now().UTC()
			repository := &catalogRepositoryStub{current: domain.CatalogContext{Organization: domain.Organization{ID: "organization-1", Name: "Organization", Status: domain.OrganizationStatusActive}, Unit: domain.Box{ID: unitID, OrganizationID: "organization-1", Name: "Unit", Status: domain.BoxStatusActive}}}
			service := catalog.NewService(repository, catalogTransactionStub{})
			user := domain.User{ID: userID, BoxID: unitID, Role: role, AuthVersion: 1, Active: true, CreatedAt: now, UpdatedAt: now}
			router := NewRouter(RouterDependencies{TokenService: catalogTokenStub{claims: services.AuthClaims{UserID: userID, BoxID: unitID, Role: role, AuthVersion: 1}}, UserRepository: catalogUserStub{user: user}, BoxRepository: catalogBoxStub{box: repository.current.Unit}, CatalogService: service, Capabilities: middleware.Capabilities{Catalog: true}, SessionConfig: middleware.SessionConfig{CookieName: "session", CSRFCookieName: "csrf"}})

			organization := catalogRequest(router, http.MethodGet, "/api/v1/organization", "", true, false)
			if organization.Code != http.StatusOK || !strings.Contains(organization.Body.String(), "current_unit") {
				t.Fatalf("organization contract: %d %s", organization.Code, organization.Body.String())
			}
			list := catalogRequest(router, http.MethodGet, "/api/v1/modalities", "", true, false)
			if list.Code != http.StatusOK {
				t.Fatalf("catalog list for %s: %d %s", role, list.Code, list.Body.String())
			}

			create := catalogRequest(router, http.MethodPost, "/api/v1/modalities", `{"name":"  Cross   Training "}`, true, false)
			if role == domain.UserRoleCoach {
				if create.Code != http.StatusForbidden || !strings.Contains(create.Body.String(), "owner_required") {
					t.Fatalf("coach mutation: %d %s", create.Code, create.Body.String())
				}
				return
			}
			if create.Code != http.StatusCreated || !strings.Contains(create.Body.String(), `"name":"Cross Training"`) || !strings.Contains(create.Body.String(), `"color":"#0F766E"`) {
				t.Fatalf("owner create contract: %d %s", create.Code, create.Body.String())
			}
			duplicate := catalogRequest(router, http.MethodPost, "/api/v1/modalities", `{"name":"cross training"}`, true, false)
			if duplicate.Code != http.StatusConflict || !strings.Contains(duplicate.Body.String(), "modality_name_conflict") {
				t.Fatalf("duplicate contract: %d %s", duplicate.Code, duplicate.Body.String())
			}
			crossUnit := catalogRequest(router, http.MethodGet, "/api/v1/units/other-unit/spaces", "", true, false)
			if crossUnit.Code != http.StatusNotFound || !strings.Contains(crossUnit.Body.String(), "unit_not_found") {
				t.Fatalf("cross-unit contract: %d %s", crossUnit.Code, crossUnit.Body.String())
			}

			withoutCSRF := catalogRequest(router, http.MethodPost, "/api/v1/modalities", `{"name":"Pilates"}`, false, false)
			if withoutCSRF.Code != http.StatusForbidden || !strings.Contains(withoutCSRF.Body.String(), "csrf_invalid") {
				t.Fatalf("missing csrf: %d %s", withoutCSRF.Code, withoutCSRF.Body.String())
			}
			withCSRF := catalogRequest(router, http.MethodPost, "/api/v1/modalities", `{"name":"Pilates"}`, false, true)
			if withCSRF.Code != http.StatusCreated {
				t.Fatalf("valid csrf: %d %s", withCSRF.Code, withCSRF.Body.String())
			}
		})
	}
}

func catalogRequest(handler http.Handler, method, path, body string, bearer, csrf bool) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if bearer {
		request.Header.Set("Authorization", "Bearer token")
	} else {
		request.AddCookie(&http.Cookie{Name: "session", Value: "token"})
		request.AddCookie(&http.Cookie{Name: "csrf", Value: "csrf-token"})
		if csrf {
			request.Header.Set("X-CSRF-Token", "csrf-token")
		}
	}
	handler.ServeHTTP(response, request)
	return response
}
