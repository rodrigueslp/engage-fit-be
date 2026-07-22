package boxes

import (
	"context"
	"errors"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
	"gorm.io/gorm"
)

var (
	ErrInvalidBoxOnboarding        = errors.New("dados de onboarding invalidos")
	ErrOwnerEmailAlreadyRegistered = errors.New("email de owner ja cadastrado")
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
	input.Name = strings.TrimSpace(input.Name)
	input.OwnerName = strings.TrimSpace(input.OwnerName)
	input.OwnerEmail = strings.ToLower(strings.TrimSpace(input.OwnerEmail))
	if input.Name == "" || input.OwnerName == "" || input.OwnerEmail == "" || len(input.Password) < 8 {
		return nil, ErrInvalidBoxOnboarding
	}
	if _, err := uc.users.FindByEmail(ctx, input.OwnerEmail); err == nil {
		return nil, ErrOwnerEmailAlreadyRegistered
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	passwordHash, err := uc.passwords.Hash(ctx, input.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	box := domain.Box{
		Name:      input.Name,
		Status:    domain.BoxStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	user := domain.User{
		Name:         input.OwnerName,
		Email:        input.OwnerEmail,
		PasswordHash: passwordHash,
		AuthVersion:  1,
		Role:         domain.UserRoleOwner,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := uc.boxes.SaveWithOwner(ctx, &box, &user); err != nil {
		return nil, err
	}

	return &CreateBoxOutput{Box: box, User: user}, nil
}
