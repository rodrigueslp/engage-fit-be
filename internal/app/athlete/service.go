package athlete

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

var (
	ErrInvalidInput       = errors.New("invalid athlete input")
	ErrInvalidCredentials = errors.New("invalid athlete credentials")
	ErrInvitationExpired  = errors.New("athlete invitation expired")
)

const (
	invitationLifetime = 7 * 24 * time.Hour
	sessionLifetime    = 30 * 24 * time.Hour
)

type Service struct {
	athletes  repositories.AthleteRepository
	students  repositories.StudentRepository
	passwords services.PasswordService
	now       func() time.Time
}

type InvitationOutput struct {
	Token     string
	ExpiresAt time.Time
}

type ClaimInput struct {
	Token    string
	Name     string
	Email    string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

type SessionOutput struct {
	Token     string
	ExpiresAt time.Time
	Context   domain.AthleteContext
}

func NewService(athletes repositories.AthleteRepository, students repositories.StudentRepository, passwords services.PasswordService) *Service {
	return &Service{athletes: athletes, students: students, passwords: passwords, now: time.Now}
}

func (s *Service) CreateInvitation(ctx context.Context, boxID, studentID, actorUserID domain.ID) (*InvitationOutput, error) {
	student, err := s.students.FindByID(ctx, boxID, studentID)
	if err != nil {
		return nil, err
	}
	if student.AnonymizedAt != nil {
		return nil, ErrInvalidInput
	}
	token, tokenHash, err := randomToken()
	if err != nil {
		return nil, err
	}
	now := s.now()
	invitation := domain.AthleteInvitation{BoxID: boxID, StudentID: studentID, TokenHash: tokenHash, CreatedByUserID: actorUserID, ExpiresAt: now.Add(invitationLifetime), CreatedAt: now}
	if err := s.athletes.SaveInvitation(ctx, &invitation); err != nil {
		return nil, err
	}
	return &InvitationOutput{Token: token, ExpiresAt: invitation.ExpiresAt}, nil
}

func (s *Service) PreviewInvitation(ctx context.Context, token string) (*domain.AthleteInvitation, error) {
	invitation, err := s.athletes.FindInvitationByTokenHash(ctx, hashToken(token))
	if err != nil {
		return nil, ErrInvitationExpired
	}
	if invitation.ClaimedAt != nil || !invitation.ExpiresAt.After(s.now()) {
		return nil, ErrInvitationExpired
	}
	invitation.TokenHash = ""
	return invitation, nil
}

func (s *Service) ClaimInvitation(ctx context.Context, input ClaimInput) (*SessionOutput, error) {
	name := strings.TrimSpace(input.Name)
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if name == "" || len([]rune(name)) > 160 || !validEmail(email) || len(input.Password) < 12 || len(input.Password) > 128 {
		return nil, ErrInvalidInput
	}
	invitation, err := s.PreviewInvitation(ctx, input.Token)
	if err != nil {
		return nil, err
	}

	existing, findErr := s.athletes.FindAccountByEmail(ctx, email)
	var account *domain.AthleteAccount
	var athleteID domain.ID
	if findErr == nil {
		if err := s.passwords.Compare(ctx, existing.PasswordHash, input.Password); err != nil {
			return nil, ErrInvalidCredentials
		}
		athleteID = existing.ID
	} else if errors.Is(findErr, repositories.ErrAthleteAccountNotFound) {
		passwordHash, err := s.passwords.Hash(ctx, input.Password)
		if err != nil {
			return nil, err
		}
		now := s.now()
		account = &domain.AthleteAccount{Name: name, Email: email, PasswordHash: passwordHash, Status: "active", CreatedAt: now, UpdatedAt: now}
	} else {
		return nil, findErr
	}

	if _, err := s.athletes.ClaimInvitation(ctx, invitation.ID, account, athleteID, s.now()); err != nil {
		if errors.Is(err, repositories.ErrAthleteInvitationUnavailable) {
			return nil, ErrInvitationExpired
		}
		return nil, err
	}
	if account != nil {
		athleteID = account.ID
	}
	return s.newSession(ctx, athleteID)
}

func (s *Service) Login(ctx context.Context, input LoginInput) (*SessionOutput, error) {
	account, err := s.athletes.FindAccountByEmail(ctx, strings.ToLower(strings.TrimSpace(input.Email)))
	if err != nil || account.Status != "active" {
		return nil, ErrInvalidCredentials
	}
	if err := s.passwords.Compare(ctx, account.PasswordHash, input.Password); err != nil {
		return nil, ErrInvalidCredentials
	}
	return s.newSession(ctx, account.ID)
}

func (s *Service) Authenticate(ctx context.Context, token string) (*domain.AthleteContext, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrInvalidCredentials
	}
	return s.athletes.FindContextBySessionHash(ctx, hashToken(token), s.now())
}

func (s *Service) Logout(ctx context.Context, token string) error {
	return s.athletes.RevokeSession(ctx, hashToken(token), s.now())
}

func (s *Service) Workouts(ctx context.Context, athleteID domain.ID) ([]domain.AthleteWorkout, error) {
	return s.athletes.ListPublishedWorkouts(ctx, athleteID)
}

func (s *Service) newSession(ctx context.Context, athleteID domain.ID) (*SessionOutput, error) {
	token, tokenHash, err := randomToken()
	if err != nil {
		return nil, err
	}
	now := s.now()
	session := domain.AthleteSession{AthleteID: athleteID, TokenHash: tokenHash, ExpiresAt: now.Add(sessionLifetime), CreatedAt: now}
	if err := s.athletes.SaveSession(ctx, &session); err != nil {
		return nil, err
	}
	context, err := s.athletes.FindContextBySessionHash(ctx, tokenHash, now)
	if err != nil {
		return nil, err
	}
	return &SessionOutput{Token: token, ExpiresAt: session.ExpiresAt, Context: *context}, nil
}

func randomToken() (string, string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(value)
	return token, hashToken(token), nil
}

func hashToken(value string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(digest[:])
}

func validEmail(value string) bool {
	parts := strings.Split(value, "@")
	return len(parts) == 2 && parts[0] != "" && strings.Contains(parts[1], ".")
}
