package security

import (
	"context"
	"strings"
	"testing"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/services"
)

func TestJWTServiceGenerateAndValidate(t *testing.T) {
	service := NewJWTService("test-secret")

	token, err := service.Generate(context.Background(), services.AuthClaims{
		UserID:      domain.ID("user-id"),
		BoxID:       domain.ID("box-id"),
		Role:        domain.UserRoleOwner,
		AuthVersion: 3,
	})
	if err != nil {
		t.Fatalf("expected token generation to succeed: %v", err)
	}

	claims, err := service.Validate(context.Background(), token)
	if err != nil {
		t.Fatalf("expected token validation to succeed: %v", err)
	}

	if claims.UserID != "user-id" {
		t.Fatalf("expected user-id, got %q", claims.UserID)
	}
	if claims.BoxID != "box-id" {
		t.Fatalf("expected box-id, got %q", claims.BoxID)
	}
	if claims.Role != domain.UserRoleOwner {
		t.Fatalf("expected OWNER, got %q", claims.Role)
	}
	if claims.AuthVersion != 3 {
		t.Fatalf("expected auth version 3, got %d", claims.AuthVersion)
	}
}

func TestJWTServiceRejectsTamperedToken(t *testing.T) {
	service := NewJWTService("test-secret")

	token, err := service.Generate(context.Background(), services.AuthClaims{
		UserID:      domain.ID("user-id"),
		BoxID:       domain.ID("box-id"),
		Role:        domain.UserRoleOwner,
		AuthVersion: 1,
	})
	if err != nil {
		t.Fatalf("expected token generation to succeed: %v", err)
	}

	tampered := token[:len(token)-1] + "x"
	if strings.EqualFold(tampered, token) {
		t.Fatal("test setup failed to tamper token")
	}

	if _, err := service.Validate(context.Background(), tampered); err == nil {
		t.Fatal("expected tampered token to fail validation")
	}
}
