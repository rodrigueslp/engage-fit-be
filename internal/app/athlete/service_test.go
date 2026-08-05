package athlete

import (
	"context"
	"errors"
	"testing"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

type athleteRepositoryStub struct {
	invitation     domain.AthleteInvitation
	account        *domain.AthleteAccount
	accountErr     error
	claimedAccount *domain.AthleteAccount
	claimedID      domain.ID
	context        domain.AthleteContext
	session        domain.AthleteSession
}

func (s *athleteRepositoryStub) SaveInvitation(context.Context, *domain.AthleteInvitation) error {
	return nil
}
func (s *athleteRepositoryStub) FindInvitationByTokenHash(context.Context, string) (*domain.AthleteInvitation, error) {
	copy := s.invitation
	return &copy, nil
}
func (s *athleteRepositoryStub) FindAccountByEmail(context.Context, string) (*domain.AthleteAccount, error) {
	return s.account, s.accountErr
}
func (s *athleteRepositoryStub) ClaimInvitation(_ context.Context, _ domain.ID, account *domain.AthleteAccount, athleteID domain.ID, _ time.Time) (*domain.AthleteMembership, error) {
	s.claimedAccount, s.claimedID = account, athleteID
	if account != nil {
		account.ID = "athlete-new"
		s.context.Account = *account
	}
	return &domain.AthleteMembership{ID: "membership-1"}, nil
}
func (s *athleteRepositoryStub) SaveSession(_ context.Context, session *domain.AthleteSession) error {
	session.ID = "session-1"
	s.session = *session
	return nil
}
func (s *athleteRepositoryStub) FindContextBySessionHash(context.Context, string, time.Time) (*domain.AthleteContext, error) {
	copy := s.context
	return &copy, nil
}
func (s *athleteRepositoryStub) RevokeSession(context.Context, string, time.Time) error { return nil }
func (s *athleteRepositoryStub) ListPublishedWorkouts(context.Context, domain.ID) ([]domain.AthleteWorkout, error) {
	return nil, nil
}

type studentRepositoryStub struct{ student domain.Student }

func (s studentRepositoryStub) FindByID(context.Context, domain.ID, domain.ID) (*domain.Student, error) {
	copy := s.student
	return &copy, nil
}
func (studentRepositoryStub) FindByExternalID(context.Context, domain.ID, domain.Source, string) (*domain.Student, error) {
	return nil, errors.New("not implemented")
}
func (studentRepositoryStub) List(context.Context, domain.ID, repositories.StudentFilters) ([]domain.Student, error) {
	return nil, nil
}
func (studentRepositoryStub) Save(context.Context, *domain.Student) error { return nil }
func (studentRepositoryStub) UpdateRiskStatus(context.Context, domain.ID, domain.ID, domain.StudentRiskStatus) error {
	return nil
}
func (studentRepositoryStub) MarkRiskMessageSent(context.Context, domain.ID, domain.ID, time.Time) error {
	return nil
}
func (studentRepositoryStub) UpdateContactPreference(context.Context, domain.ID, domain.ID, domain.ContactStatus, string, time.Time) error {
	return nil
}

type passwordServiceStub struct{ compareErr error }

func (passwordServiceStub) Hash(context.Context, string) (string, error)    { return "password-hash", nil }
func (s passwordServiceStub) Compare(context.Context, string, string) error { return s.compareErr }

func TestClaimInvitationCreatesUniqueAccountAndSession(t *testing.T) {
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	repository := &athleteRepositoryStub{
		invitation: domain.AthleteInvitation{ID: "invite-1", ExpiresAt: now.Add(time.Hour)},
		accountErr: repositories.ErrAthleteAccountNotFound,
		context:    domain.AthleteContext{Memberships: []domain.AthleteMembership{{ID: "membership-1", BoxID: "box-1", BoxName: "CrossFit Aurora"}}},
	}
	service := NewService(repository, studentRepositoryStub{}, passwordServiceStub{})
	service.now = func() time.Time { return now }

	result, err := service.ClaimInvitation(context.Background(), ClaimInput{Token: "valid-token", Name: "  Maria Silva  ", Email: " MARIA@EXAMPLE.COM ", Password: "uma-senha-forte"})
	if err != nil {
		t.Fatalf("claim invitation: %v", err)
	}
	if repository.claimedAccount == nil || repository.claimedAccount.Email != "maria@example.com" || repository.claimedAccount.Name != "Maria Silva" {
		t.Fatalf("unexpected account: %+v", repository.claimedAccount)
	}
	if result.Token == "" || repository.session.TokenHash == "" || repository.session.ExpiresAt != now.Add(sessionLifetime) {
		t.Fatalf("session was not created correctly: %+v", repository.session)
	}
}

func TestClaimInvitationRequiresExistingAccountPassword(t *testing.T) {
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	repository := &athleteRepositoryStub{
		invitation: domain.AthleteInvitation{ID: "invite-1", ExpiresAt: now.Add(time.Hour)},
		account:    &domain.AthleteAccount{ID: "athlete-1", Email: "maria@example.com", PasswordHash: "hash", Status: "active"},
	}
	service := NewService(repository, studentRepositoryStub{}, passwordServiceStub{compareErr: errors.New("wrong password")})
	service.now = func() time.Time { return now }

	_, err := service.ClaimInvitation(context.Background(), ClaimInput{Token: "valid-token", Name: "Maria", Email: "maria@example.com", Password: "senha-incorreta"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
	if repository.claimedAccount != nil || repository.claimedID != "" {
		t.Fatal("invitation must not be claimed with a wrong password")
	}
}

func TestClaimInvitationDoesNotTreatDatabaseFailureAsMissingAccount(t *testing.T) {
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	databaseErr := errors.New("database unavailable")
	repository := &athleteRepositoryStub{invitation: domain.AthleteInvitation{ID: "invite-1", ExpiresAt: now.Add(time.Hour)}, accountErr: databaseErr}
	service := NewService(repository, studentRepositoryStub{}, passwordServiceStub{})
	service.now = func() time.Time { return now }

	_, err := service.ClaimInvitation(context.Background(), ClaimInput{Token: "valid-token", Name: "Maria", Email: "maria@example.com", Password: "uma-senha-forte"})
	if !errors.Is(err, databaseErr) {
		t.Fatalf("expected database error, got %v", err)
	}
	if repository.claimedAccount != nil {
		t.Fatal("must not attempt to create a duplicate identity after a lookup failure")
	}
}
