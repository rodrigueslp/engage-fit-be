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
var ErrUserAccessInactive = errors.New("acesso do usuário desativado")

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
	access    repositories.OrganizationRepository
}

func NewLoginUseCase(users repositories.UserRepository, boxes repositories.BoxRepository, passwords services.PasswordService, tokens services.TokenService, access ...repositories.OrganizationRepository) LoginUseCase {
	uc := LoginUseCase{users: users, boxes: boxes, passwords: passwords, tokens: tokens}
	if len(access) > 0 {
		uc.access = access[0]
	}
	return uc
}

func (uc LoginUseCase) Execute(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	user, err := uc.users.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(input.Email)))
	if err != nil {
		return nil, err
	}
	if err := uc.passwords.Compare(ctx, user.PasswordHash, input.Password); err != nil {
		return nil, err
	}
	if user.Role == domain.UserRoleCoach && !user.Active {
		return nil, ErrUserAccessInactive
	}
	selectedUnitID := user.BoxID
	if user.Role == domain.UserRoleOwner || user.Role == domain.UserRoleCoach {
		box, err := uc.boxes.FindByID(ctx, user.BoxID)
		if err != nil {
			return nil, err
		}
		if uc.access != nil {
			accessContext, accessErr := uc.access.ResolveUserUnitAccess(ctx, user.ID, selectedUnitID)
			if accessErr != nil || !accessContext.Organization.IsActive() || !accessContext.Unit.IsOperational() {
				units, listErr := uc.access.ListAccessibleUnits(ctx, user.ID, box.OrganizationID)
				if listErr != nil {
					return nil, listErr
				}
				selectedUnitID = ""
				hasAuthorizedUnit := false
				hasBillingBlockedUnit := false
				for _, unit := range units {
					candidate, resolveErr := uc.access.ResolveUserUnitAccess(ctx, user.ID, unit.ID)
					if resolveErr != nil {
						continue
					}
					hasAuthorizedUnit = true
					if candidate.Organization.IsActive() && candidate.Unit.IsOperational() {
						selectedUnitID = unit.ID
						break
					}
					if candidate.Organization.IsActive() && candidate.Unit.IsActive() && candidate.Unit.BillingAccessBlocked {
						hasBillingBlockedUnit = true
					}
				}
				if selectedUnitID == "" {
					if hasBillingBlockedUnit {
						return nil, ErrBillingAccessBlocked
					}
					if hasAuthorizedUnit {
						return nil, ErrBoxAccessInactive
					}
					return nil, ErrUserAccessInactive
				}
			}
		} else {
			if !box.IsActive() {
				return nil, ErrBoxAccessInactive
			}
			if box.BillingAccessBlocked {
				return nil, ErrBillingAccessBlocked
			}
		}
	}

	token, err := uc.tokens.Generate(ctx, services.AuthClaims{
		UserID:      user.ID,
		BoxID:       selectedUnitID,
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
