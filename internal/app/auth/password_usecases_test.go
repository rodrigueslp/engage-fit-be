package auth

import (
	"context"
	"testing"

	"boxengage/backend/internal/adapters/security"
	"boxengage/backend/internal/domain"
)

type passwordUserRepository struct {
	user domain.User
}

func (r *passwordUserRepository) FindByID(context.Context, domain.ID) (*domain.User, error) {
	user := r.user
	return &user, nil
}
func (r *passwordUserRepository) FindByEmail(context.Context, string) (*domain.User, error) {
	user := r.user
	return &user, nil
}
func (r *passwordUserRepository) FindOwnerByBoxID(context.Context, domain.ID) (*domain.User, error) {
	user := r.user
	return &user, nil
}
func (r *passwordUserRepository) Save(context.Context, *domain.User) error { return nil }
func (r *passwordUserRepository) UpdatePassword(_ context.Context, _ domain.ID, hash string) error {
	r.user.PasswordHash = hash
	r.user.AuthVersion++
	return nil
}
func (r *passwordUserRepository) BumpAuthVersion(context.Context, domain.ID) error {
	r.user.AuthVersion++
	return nil
}
func (r *passwordUserRepository) UpdatePlatformAdminCredentials(context.Context, domain.ID, string, string) error {
	return nil
}

func TestChangePassword(t *testing.T) {
	passwords := security.NewPasswordService()
	currentHash, err := passwords.Hash(context.Background(), "current-password")
	if err != nil {
		t.Fatal(err)
	}
	repository := &passwordUserRepository{user: domain.User{ID: "user-1", PasswordHash: currentHash, AuthVersion: 1}}
	useCase := NewChangePasswordUseCase(repository, passwords)

	if err := useCase.Execute(context.Background(), ChangePasswordInput{UserID: "user-1", CurrentPassword: "current-password", NewPassword: "new-password-123"}); err != nil {
		t.Fatalf("expected password change to succeed: %v", err)
	}
	if err := passwords.Compare(context.Background(), repository.user.PasswordHash, "new-password-123"); err != nil {
		t.Fatal("expected the new password hash to be persisted")
	}
	if repository.user.AuthVersion != 2 {
		t.Fatalf("expected auth version 2, got %d", repository.user.AuthVersion)
	}
}

func TestChangePasswordRejectsInvalidCredentials(t *testing.T) {
	passwords := security.NewPasswordService()
	currentHash, _ := passwords.Hash(context.Background(), "current-password")
	repository := &passwordUserRepository{user: domain.User{ID: "user-1", PasswordHash: currentHash, AuthVersion: 1}}
	useCase := NewChangePasswordUseCase(repository, passwords)

	if err := useCase.Execute(context.Background(), ChangePasswordInput{UserID: "user-1", CurrentPassword: "wrong", NewPassword: "new-password-123"}); err != ErrInvalidCurrentPassword {
		t.Fatalf("expected invalid current password, got %v", err)
	}
	if err := useCase.Execute(context.Background(), ChangePasswordInput{UserID: "user-1", CurrentPassword: "current-password", NewPassword: "short"}); err == nil {
		t.Fatal("expected a short password to be rejected")
	}
}
