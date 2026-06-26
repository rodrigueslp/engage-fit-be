package auth

import (
	"context"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

type GetCurrentUserUseCase struct {
	users repositories.UserRepository
}

func NewGetCurrentUserUseCase(users repositories.UserRepository) GetCurrentUserUseCase {
	return GetCurrentUserUseCase{users: users}
}

func (uc GetCurrentUserUseCase) Execute(ctx context.Context, userID domain.ID) (*domain.User, error) {
	return uc.users.FindByID(ctx, userID)
}
