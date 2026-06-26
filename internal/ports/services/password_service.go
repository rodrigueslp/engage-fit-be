package services

import "context"

type PasswordService interface {
	Hash(ctx context.Context, password string) (string, error)
	Compare(ctx context.Context, hash, password string) error
}
