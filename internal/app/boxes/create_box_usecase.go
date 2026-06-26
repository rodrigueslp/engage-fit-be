package boxes

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

type CreateBoxInput struct {
	Name       string
	OwnerName  string
	OwnerEmail string
	Password   string
}

type CreateBoxOutput struct {
	Box  domain.Box
	User domain.User
}

type CreateBoxUseCase struct {
	boxes     repositories.BoxRepository
	users     repositories.UserRepository
	passwords services.PasswordService
}

func NewCreateBoxUseCase(boxes repositories.BoxRepository, users repositories.UserRepository, passwords services.PasswordService) CreateBoxUseCase {
	return CreateBoxUseCase{boxes: boxes, users: users, passwords: passwords}
}

func (uc CreateBoxUseCase) Execute(ctx context.Context, input CreateBoxInput) (*CreateBoxOutput, error) {
	passwordHash, err := uc.passwords.Hash(ctx, input.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	box := domain.Box{
		Name:      input.Name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := uc.boxes.Save(ctx, &box); err != nil {
		return nil, err
	}

	user := domain.User{
		BoxID:        box.ID,
		Name:         input.OwnerName,
		Email:        input.OwnerEmail,
		PasswordHash: passwordHash,
		Role:         domain.UserRoleOwner,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := uc.users.Save(ctx, &user); err != nil {
		return nil, err
	}

	return &CreateBoxOutput{Box: box, User: user}, nil
}
