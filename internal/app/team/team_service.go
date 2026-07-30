package team

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
	ErrInvalidMember = errors.New("invalid team member")
	ErrEmailInUse    = errors.New("team member email already in use")
)

type Service struct {
	users     repositories.UserRepository
	team      repositories.TeamRepository
	passwords services.PasswordService
	now       func() time.Time
}

func NewService(users repositories.UserRepository, team repositories.TeamRepository, passwords services.PasswordService) *Service {
	return &Service{users: users, team: team, passwords: passwords, now: time.Now}
}

func (s Service) List(ctx context.Context, boxID domain.ID) ([]domain.User, error) {
	return s.team.ListMembers(ctx, boxID)
}

func (s Service) CreateCoach(ctx context.Context, boxID domain.ID, name, email, password string) (*domain.User, error) {
	name, email = strings.TrimSpace(name), strings.ToLower(strings.TrimSpace(email))
	if name == "" || len(name) > 255 || email == "" || len(email) > 255 || len(password) < 12 {
		return nil, ErrInvalidMember
	}
	if _, err := s.users.FindByEmail(ctx, email); err == nil {
		return nil, ErrEmailInUse
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	hash, err := s.passwords.Hash(ctx, password)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	item := domain.User{
		BoxID: boxID, Name: name, Email: email, PasswordHash: hash,
		AuthVersion: 1, Role: domain.UserRoleCoach, Active: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.users.Save(ctx, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s Service) UpdateCoach(ctx context.Context, boxID, userID domain.ID, name string, active bool) (*domain.User, error) {
	item, err := s.team.FindMember(ctx, boxID, userID)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if item.Role != domain.UserRoleCoach || name == "" || len(name) > 255 {
		return nil, ErrInvalidMember
	}
	if err := s.team.UpdateMember(ctx, boxID, userID, name, active); err != nil {
		return nil, err
	}
	item.Name, item.Active = name, active
	return item, nil
}

func (s Service) ResetCoachPassword(ctx context.Context, boxID, userID domain.ID, password string) error {
	item, err := s.team.FindMember(ctx, boxID, userID)
	if err != nil {
		return err
	}
	if item.Role != domain.UserRoleCoach || len(password) < 12 {
		return ErrInvalidMember
	}
	hash, err := s.passwords.Hash(ctx, password)
	if err != nil {
		return err
	}
	return s.users.UpdatePassword(ctx, userID, hash)
}
