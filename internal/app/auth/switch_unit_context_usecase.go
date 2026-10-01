package auth

import (
	"context"
	"errors"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

var ErrUnitAccessDenied = errors.New("unit access denied")

type SwitchUnitContextUseCase struct {
	users  repositories.UserRepository
	access repositories.UnitAccessAuthorizer
	tokens services.TokenService
}

func NewSwitchUnitContextUseCase(users repositories.UserRepository, access repositories.UnitAccessAuthorizer, tokens services.TokenService) SwitchUnitContextUseCase {
	return SwitchUnitContextUseCase{users: users, access: access, tokens: tokens}
}

func (uc SwitchUnitContextUseCase) Execute(ctx context.Context, userID, unitID domain.ID) (string, error) {
	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return "", err
	}
	if user.Role != domain.UserRoleOwner && user.Role != domain.UserRoleCoach {
		return "", ErrUnitAccessDenied
	}
	access, err := uc.access.ResolveUserUnitAccess(ctx, userID, unitID)
	if err != nil || access.Role != user.Role || !access.Organization.IsActive() || !access.Unit.IsOperational() {
		return "", ErrUnitAccessDenied
	}
	return uc.tokens.Generate(ctx, services.AuthClaims{UserID: user.ID, BoxID: unitID, Role: user.Role, AuthVersion: normalizedAuthVersion(user.AuthVersion)})
}
