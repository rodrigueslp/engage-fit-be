package auth

import (
	"context"
	"strings"

	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

type LoginInput struct {
	Email    string
	Password string
}

type LoginOutput struct {
	AccessToken string
}

type LoginUseCase struct {
	users     repositories.UserRepository
	passwords services.PasswordService
	tokens    services.TokenService
}

func NewLoginUseCase(users repositories.UserRepository, passwords services.PasswordService, tokens services.TokenService) LoginUseCase {
	return LoginUseCase{users: users, passwords: passwords, tokens: tokens}
}

func (uc LoginUseCase) Execute(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	user, err := uc.users.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(input.Email)))
	if err != nil {
		return nil, err
	}
	if err := uc.passwords.Compare(ctx, user.PasswordHash, input.Password); err != nil {
		return nil, err
	}

	token, err := uc.tokens.Generate(ctx, services.AuthClaims{
		UserID:      user.ID,
		BoxID:       user.BoxID,
		Role:        user.Role,
		AuthVersion: normalizedAuthVersion(user.AuthVersion),
	})
	if err != nil {
		return nil, err
	}

	return &LoginOutput{AccessToken: token}, nil
}

func normalizedAuthVersion(version int) int {
	if version < 1 {
		return 1
	}
	return version
}
