package security

import (
	"context"

	"golang.org/x/crypto/bcrypt"
)

type PasswordService struct{}

func NewPasswordService() PasswordService {
	return PasswordService{}
}

func (s PasswordService) Hash(ctx context.Context, password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (s PasswordService) Compare(ctx context.Context, hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
