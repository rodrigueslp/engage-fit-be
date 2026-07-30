package repositories

import (
	"context"

	"boxengage/backend/internal/domain"
)

type TeamRepository interface {
	ListMembers(ctx context.Context, boxID domain.ID) ([]domain.User, error)
	FindMember(ctx context.Context, boxID, userID domain.ID) (*domain.User, error)
	UpdateMember(ctx context.Context, boxID, userID domain.ID, name string, active bool) error
}
