package auth

import (
	"context"
	"errors"
	"strings"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

var ErrBoxAccessInactive = errors.New("academia inativa")
var ErrBillingAccessBlocked = errors.New("acesso bloqueado por pendência financeira")

type LoginInput struct {
	Email    string
	Password string
}

type LoginOutput struct {
	AccessToken string
}

type LoginUseCase struct {
	users     repositories.UserRepository
	boxes     repositories.BoxRepository
	passwords services.PasswordService
	tokens    services.TokenService
}

func NewLoginUseCase(users repositories.UserRepository, boxes repositories.BoxRepository, passwords services.PasswordService, tokens services.TokenService) LoginUseCase {
	return LoginUseCase{users: users, boxes: boxes, passwords: passwords, tokens: tokens}
}

func (uc LoginUseCase) Execute(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	user, err := uc.users.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(input.Email)))
	if err != nil {
		return nil, err
	}
	if err := uc.passwords.Compare(ctx, user.PasswordHash, input.Password); err != nil {
		return nil, err
	}
	if user.Role == domain.UserRoleOwner {
		box, err := uc.boxes.FindByID(ctx, user.BoxID)
		if err != nil {
			return nil, err
		}
		if !box.IsActive() {
			return nil, ErrBoxAccessInactive
		}
		if box.BillingAccessBlocked {
			return nil, ErrBillingAccessBlocked
		}
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
