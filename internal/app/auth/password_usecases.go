package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

const MinimumPasswordLength = 12

var (
	ErrInvalidCurrentPassword = errors.New("senha atual invalida")
	ErrInvalidNewPassword     = errors.New("nova senha invalida")
)

type ChangePasswordInput struct {
	UserID          domain.ID
	CurrentPassword string
	NewPassword     string
}

type ChangePasswordUseCase struct {
	users     repositories.UserRepository
	passwords services.PasswordService
}

type LogoutUseCase struct{ users repositories.UserRepository }

func NewLogoutUseCase(users repositories.UserRepository) LogoutUseCase {
	return LogoutUseCase{users: users}
}

func (uc LogoutUseCase) Execute(ctx context.Context, userID domain.ID) error {
	return uc.users.BumpAuthVersion(ctx, userID)
}

func NewChangePasswordUseCase(users repositories.UserRepository, passwords services.PasswordService) ChangePasswordUseCase {
	return ChangePasswordUseCase{users: users, passwords: passwords}
}

func (uc ChangePasswordUseCase) Execute(ctx context.Context, input ChangePasswordInput) error {
	if err := ValidateNewPassword(input.NewPassword); err != nil {
		return err
	}
	user, err := uc.users.FindByID(ctx, input.UserID)
	if err != nil {
		return err
	}
	if err := uc.passwords.Compare(ctx, user.PasswordHash, input.CurrentPassword); err != nil {
		return ErrInvalidCurrentPassword
	}
	if err := uc.passwords.Compare(ctx, user.PasswordHash, input.NewPassword); err == nil {
		return fmt.Errorf("%w: deve ser diferente da senha atual", ErrInvalidNewPassword)
	}
	hash, err := uc.passwords.Hash(ctx, input.NewPassword)
	if err != nil {
		return err
	}
	return uc.users.UpdatePassword(ctx, user.ID, hash)
}

func ValidateNewPassword(password string) error {
	if len(password) < MinimumPasswordLength {
		return fmt.Errorf("%w: deve ter ao menos 12 caracteres", ErrInvalidNewPassword)
	}
	if strings.TrimSpace(password) != password {
		return fmt.Errorf("%w: nao pode comecar ou terminar com espacos", ErrInvalidNewPassword)
	}
	return nil
}
