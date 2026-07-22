package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/services"
	"github.com/gin-gonic/gin"
)

type authTokenStub struct {
	claims *services.AuthClaims
	err    error
}

func (s authTokenStub) Generate(context.Context, services.AuthClaims) (string, error) {
	return "token", nil
}
func (s authTokenStub) Validate(context.Context, string) (*services.AuthClaims, error) {
	return s.claims, s.err
}

type authUserRepositoryStub struct {
	user *domain.User
	err  error
}

type authBoxRepositoryStub struct {
	box *domain.Box
	err error
}

func (s authBoxRepositoryStub) FindByID(context.Context, domain.ID) (*domain.Box, error) {
	if s.box == nil && s.err == nil {
		return &domain.Box{Status: domain.BoxStatusActive}, nil
	}
	return s.box, s.err
}
func (authBoxRepositoryStub) ListAll(context.Context) ([]domain.Box, error) { return nil, nil }
func (authBoxRepositoryStub) Save(context.Context, *domain.Box) error       { return nil }
func (authBoxRepositoryStub) SaveWithOwner(context.Context, *domain.Box, *domain.User) error {
	return nil
}
func (authBoxRepositoryStub) Update(context.Context, domain.Box) error { return nil }

func (s authUserRepositoryStub) FindByID(context.Context, domain.ID) (*domain.User, error) {
	return s.user, s.err
}
func (authUserRepositoryStub) FindByEmail(context.Context, string) (*domain.User, error) {
	return nil, errors.New("not implemented")
}
func (authUserRepositoryStub) FindOwnerByBoxID(context.Context, domain.ID) (*domain.User, error) {
	return nil, errors.New("not implemented")
}
func (authUserRepositoryStub) Save(context.Context, *domain.User) error { return nil }
func (authUserRepositoryStub) UpdatePassword(context.Context, domain.ID, string) error {
	return nil
}
func (authUserRepositoryStub) BumpAuthVersion(context.Context, domain.ID) error { return nil }
func (authUserRepositoryStub) UpdatePlatformAdminCredentials(context.Context, domain.ID, string, string) error {
	return nil
}

func TestAuthAndTenantAcceptMatchingOwnerSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	claims := &services.AuthClaims{UserID: "user-1", BoxID: "box-1", Role: domain.UserRoleOwner, AuthVersion: 2}
	user := &domain.User{ID: "user-1", BoxID: "box-1", Role: domain.UserRoleOwner, AuthVersion: 2}
	router := gin.New()
	router.Use(Auth(authTokenStub{claims: claims}, authUserRepositoryStub{user: user}, authBoxRepositoryStub{}), Tenant())
	router.GET("/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer valid")
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", response.Code)
	}
}

func TestAuthRejectsTenantOrVersionMismatch(t *testing.T) {
	tests := []struct {
		name   string
		claims services.AuthClaims
		user   domain.User
	}{
		{name: "tenant", claims: services.AuthClaims{UserID: "user-1", BoxID: "box-2", Role: domain.UserRoleOwner, AuthVersion: 2}, user: domain.User{ID: "user-1", BoxID: "box-1", Role: domain.UserRoleOwner, AuthVersion: 2}},
		{name: "version", claims: services.AuthClaims{UserID: "user-1", BoxID: "box-1", Role: domain.UserRoleOwner, AuthVersion: 1}, user: domain.User{ID: "user-1", BoxID: "box-1", Role: domain.UserRoleOwner, AuthVersion: 2}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(Auth(authTokenStub{claims: &test.claims}, authUserRepositoryStub{user: &test.user}, authBoxRepositoryStub{}))
			router.GET("/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("Authorization", "Bearer stale")
			router.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", response.Code)
			}
		})
	}
}

func TestAuthRejectsOwnerWhenBoxIsSuspended(t *testing.T) {
	gin.SetMode(gin.TestMode)
	claims := &services.AuthClaims{UserID: "user-1", BoxID: "box-1", Role: domain.UserRoleOwner, AuthVersion: 2}
	user := &domain.User{ID: "user-1", BoxID: "box-1", Role: domain.UserRoleOwner, AuthVersion: 2}
	router := gin.New()
	router.Use(Auth(authTokenStub{claims: claims}, authUserRepositoryStub{user: user}, authBoxRepositoryStub{box: &domain.Box{ID: "box-1", Status: domain.BoxStatusSuspended}}))
	router.GET("/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer valid")
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", response.Code)
	}
}

func TestPlatformAdminAndTenantBoundaries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ownerRouter := gin.New()
	ownerRouter.Use(func(c *gin.Context) {
		SetAuthContext(c, "owner", "box-1", domain.UserRoleOwner)
		c.Next()
	}, PlatformAdmin())
	ownerRouter.GET("/admin", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	ownerResponse := httptest.NewRecorder()
	ownerRouter.ServeHTTP(ownerResponse, httptest.NewRequest(http.MethodGet, "/admin", nil))
	if ownerResponse.Code != http.StatusForbidden {
		t.Fatalf("owner must not access platform admin route: %d", ownerResponse.Code)
	}

	adminRouter := gin.New()
	adminRouter.Use(func(c *gin.Context) {
		SetAuthContext(c, "admin", "", domain.UserRolePlatformAdmin)
		c.Next()
	}, Tenant())
	adminRouter.GET("/tenant", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	adminResponse := httptest.NewRecorder()
	adminRouter.ServeHTTP(adminResponse, httptest.NewRequest(http.MethodGet, "/tenant", nil))
	if adminResponse.Code != http.StatusUnauthorized {
		t.Fatalf("platform admin must not inherit tenant access: %d", adminResponse.Code)
	}
}
