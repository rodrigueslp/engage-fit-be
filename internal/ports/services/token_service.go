package services

import (
	"context"

	"boxengage/backend/internal/domain"
)

type AuthClaims struct {
	UserID domain.ID
	BoxID  domain.ID
	Role   domain.UserRole
}

type TokenService interface {
	Generate(ctx context.Context, claims AuthClaims) (string, error)
	Validate(ctx context.Context, token string) (*AuthClaims, error)
}
