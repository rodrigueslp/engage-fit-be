package http

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/adapters/security"
	"boxengage/backend/internal/app/people"
	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/services"
)

type peopleRepositoryHTTPStub struct {
	command domain.CreatePersonCommand
}

func (s *peopleRepositoryHTTPStub) DuplicateCheck(context.Context, domain.ID, domain.ID, string, string) (*domain.PersonDuplicateCheck, error) {
	return &domain.PersonDuplicateCheck{PhoneMatches: []domain.PersonDuplicateMatch{}}, nil
}
func (s *peopleRepositoryHTTPStub) Create(_ context.Context, command domain.CreatePersonCommand) (*domain.PersonCreationResult, bool, error) {
	s.command = command
	return &domain.PersonCreationResult{PersonID: command.Person.ID, StudentID: command.Student.ID, Name: command.Person.Name, PhoneE164: command.Person.PhoneE164, Email: command.Person.Email, CPFLast2: command.Person.CPFLast2, RelationshipStatus: domain.PersonRelationshipActive, ContactStatus: domain.ContactStatusUnknown, CreatedAt: command.CreatedAt}, false, nil
}
func (*peopleRepositoryHTTPStub) CountStudentsWithoutPerson(context.Context) (int64, error) {
	return 0, nil
}
func (*peopleRepositoryHTTPStub) ResolveContractParty(context.Context, domain.ID, domain.ID) (*domain.ContractPartyResolution, error) {
	return nil, nil
}

func TestPeopleHTTPContractAuthorizationTenancyAndSafeResponse(t *testing.T) {
	unitID := domain.ID("11111111-1111-4111-8111-111111111111")
	userID := domain.ID("22222222-2222-4222-8222-222222222222")
	now := time.Now().UTC()
	for _, role := range []domain.UserRole{domain.UserRoleOwner, domain.UserRoleCoach, domain.UserRolePlatformAdmin} {
		t.Run(string(role), func(t *testing.T) {
			repository := &peopleRepositoryHTTPStub{}
			key := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))
			cipher, err := security.NewSecretCipher("test", "test:"+key)
			if err != nil {
				t.Fatal(err)
			}
			service, err := people.NewService(repository, catalogBoxStub{box: domain.Box{ID: unitID, OrganizationID: "33333333-3333-4333-8333-333333333333", Status: domain.BoxStatusActive}}, cipher, key)
			if err != nil {
				t.Fatal(err)
			}
			boxID := unitID
			if role == domain.UserRolePlatformAdmin {
				boxID = ""
			}
			user := domain.User{ID: userID, BoxID: boxID, Role: role, AuthVersion: 1, Active: true, CreatedAt: now, UpdatedAt: now}
			router := NewRouter(RouterDependencies{TokenService: catalogTokenStub{claims: servicesAuthClaims(userID, boxID, role)}, UserRepository: catalogUserStub{user: user}, BoxRepository: catalogBoxStub{box: domain.Box{ID: unitID, OrganizationID: "33333333-3333-4333-8333-333333333333", Status: domain.BoxStatusActive}}, PeopleService: service, Capabilities: middleware.Capabilities{People: true}, SessionConfig: middleware.SessionConfig{CookieName: "session", CSRFCookieName: "csrf"}})
			body := `{"name":"Maria da Silva","phone":"(11) 99999-9999","cpf":"529.982.247-25"}`
			response := catalogRequest(router, http.MethodPost, "/api/v1/units/"+string(unitID)+"/people", body, true, false)
			if role != domain.UserRoleOwner {
				if response.Code != http.StatusForbidden && response.Code != http.StatusUnauthorized {
					t.Fatalf("%s create = %d %s", role, response.Code, response.Body.String())
				}
				return
			}
			if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "invalid_idempotency_key") {
				t.Fatalf("missing idempotency key = %d %s", response.Code, response.Body.String())
			}
			requestWithKey := func(path, payload string) *httptest.ResponseRecorder {
				request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(payload))
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Authorization", "Bearer token")
				request.Header.Set("Idempotency-Key", "people-http-001")
				result := httptest.NewRecorder()
				router.ServeHTTP(result, request)
				return result
			}
			created := requestWithKey("/api/v1/units/"+string(unitID)+"/people", body)
			if created.Code != http.StatusCreated || !strings.Contains(created.Body.String(), `"cpf_masked":"***.***.***-25"`) || strings.Contains(created.Body.String(), "52998224725") || strings.Contains(created.Body.String(), "529.982.247-25") {
				t.Fatalf("unsafe create response = %d %s", created.Code, created.Body.String())
			}
			crossUnit := requestWithKey("/api/v1/units/44444444-4444-4444-8444-444444444444/people", body)
			if crossUnit.Code != http.StatusNotFound || !strings.Contains(crossUnit.Body.String(), "unit_not_found") {
				t.Fatalf("cross-unit response = %d %s", crossUnit.Code, crossUnit.Body.String())
			}
			invalidCPF := requestWithKey("/api/v1/units/"+string(unitID)+"/people", `{"name":"Maria","phone":"11999999999","cpf":"111.111.111-11"}`)
			if invalidCPF.Code != http.StatusBadRequest || !strings.Contains(invalidCPF.Body.String(), "person_cpf_invalid") || strings.Contains(invalidCPF.Body.String(), "111.111.111-11") {
				t.Fatalf("CPF error leaked input = %d %s", invalidCPF.Code, invalidCPF.Body.String())
			}
		})
	}
}

func TestPeopleCapabilityDisabledBeforeOperationalAccess(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))
	cipher, err := security.NewSecretCipher("test", "test:"+key)
	if err != nil {
		t.Fatal(err)
	}
	service, err := people.NewService(&peopleRepositoryHTTPStub{}, catalogBoxStub{box: domain.Box{ID: "11111111-1111-4111-8111-111111111111", OrganizationID: "33333333-3333-4333-8333-333333333333"}}, cipher, key)
	if err != nil {
		t.Fatal(err)
	}
	router := NewRouter(RouterDependencies{PeopleService: service, Capabilities: middleware.Capabilities{People: false}})
	response := catalogRequest(router, http.MethodPost, "/api/v1/units/11111111-1111-4111-8111-111111111111/people", `{}`, true, false)
	if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "capability_disabled") {
		t.Fatalf("disabled people route = %d %s", response.Code, response.Body.String())
	}
}

func servicesAuthClaims(userID, boxID domain.ID, role domain.UserRole) services.AuthClaims {
	return services.AuthClaims{UserID: userID, BoxID: boxID, Role: role, AuthVersion: 1}
}
