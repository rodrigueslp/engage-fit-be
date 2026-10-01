package healthforms

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"boxengage/backend/internal/domain"
)

const (
	testOrg        = domain.ID("10000000-0000-4000-8000-000000000001")
	testUnit       = domain.ID("10000000-0000-4000-8000-000000000001")
	testEnrollment = domain.ID("20000000-0000-4000-8000-000000000001")
	testPerson     = domain.ID("30000000-0000-4000-8000-000000000001")
	testActor      = domain.ID("40000000-0000-4000-8000-000000000001")
)

type healthRepoStub struct {
	created *domain.HealthFormSubmission
	answers domain.PARQAnswers
	forms   []domain.HealthFormSubmission
	session *domain.HealthFormSession
}

func (r *healthRepoStub) Create(_ context.Context, form *domain.HealthFormSubmission, answers domain.PARQAnswers) error {
	r.created = form
	r.answers = answers
	return nil
}
func (*healthRepoStub) ListForPerson(context.Context, domain.ID, domain.ID, domain.ID) ([]domain.HealthFormSubmission, error) {
	return nil, nil
}
func (*healthRepoStub) Find(context.Context, domain.ID, domain.ID, domain.ID, bool, domain.ID) (*domain.HealthFormSubmission, error) {
	return nil, nil
}
func (*healthRepoStub) RevokeConsent(context.Context, domain.ID, domain.ID, domain.ID, domain.ID, time.Time) (*domain.HealthFormSubmission, error) {
	return nil, nil
}
func (*healthRepoStub) ListPrivacy(context.Context, domain.ID, domain.ID) ([]domain.HealthFormPrivacyExport, error) {
	return nil, nil
}
func (*healthRepoStub) PurgeExpired(context.Context, time.Time, bool) (int64, error) { return 0, nil }
func (r *healthRepoStub) CreateSession(_ context.Context, session *domain.HealthFormSession) error {
	r.session = session
	return nil
}
func (r *healthRepoStub) FindSessionByTokenHash(_ context.Context, tokenHash string) (*domain.HealthFormSession, error) {
	if r.session == nil || r.session.TokenHash != tokenHash {
		return nil, ErrHealthFormSessionInvalid
	}
	return r.session, nil
}
func (r *healthRepoStub) CreateFromSession(_ context.Context, _ domain.ID, form *domain.HealthFormSubmission, answers domain.PARQAnswers, at time.Time) error {
	r.created, r.answers = form, answers
	r.session.Status = "submitted"
	r.session.SubmittedAt = &at
	return nil
}

type enrollmentResolverStub struct {
	status domain.EnrollmentEffectiveStatus
}

func (r enrollmentResolverStub) ResolveContractEnrollmentSnapshot(context.Context, domain.ID, domain.ID, domain.ID, time.Time) (*domain.ContractEnrollmentSnapshot, error) {
	return &domain.ContractEnrollmentSnapshot{PersonID: testPerson, EffectiveStatus: r.status}, nil
}

func TestSubmitPersistsConsentAndReviewWithoutReturningAnswers(t *testing.T) {
	repo := &healthRepoStub{}
	service := NewService(repo, enrollmentResolverStub{status: domain.EnrollmentEffectiveActive})
	fixed := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return fixed }
	result, err := service.Submit(context.Background(), testOrg, testUnit, testEnrollment, testActor, SubmitInput{ConsentAccepted: true, Answers: domain.PARQAnswers{ChestPain: true, Details: "Revisar com profissional"}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.RequiresReview || result.Answers != nil || repo.answers.Details != "Revisar com profissional" {
		t.Fatalf("unexpected submission: %+v answers=%+v", result, repo.answers)
	}
	if result.ConsentVersion != PARQConsentVersion || !result.RetentionDueAt.Equal(fixed.AddDate(0, 0, DefaultRetentionDays)) {
		t.Fatalf("unexpected consent/retention: %+v", result)
	}
}

func TestSubmitRejectsMissingConsentAndEndedEnrollment(t *testing.T) {
	repo := &healthRepoStub{}
	active := NewService(repo, enrollmentResolverStub{status: domain.EnrollmentEffectiveActive})
	if _, err := active.Submit(context.Background(), testOrg, testUnit, testEnrollment, testActor, SubmitInput{}); err != ErrInvalidHealthFormInput {
		t.Fatalf("expected invalid input, got %v", err)
	}
	ended := NewService(repo, enrollmentResolverStub{status: domain.EnrollmentEffectiveEnded})
	if _, err := ended.Submit(context.Background(), testOrg, testUnit, testEnrollment, testActor, SubmitInput{ConsentAccepted: true}); err != ErrHealthFormUnavailable {
		t.Fatalf("expected unavailable, got %v", err)
	}
}

func TestPublicSessionUsesHashedOneTimeTokenAndSubmitsConsent(t *testing.T) {
	repo := &healthRepoStub{}
	service := NewService(repo, enrollmentResolverStub{status: domain.EnrollmentEffectiveActive})
	fixed := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return fixed }
	launch, err := service.CreatePublicSession(context.Background(), testOrg, testUnit, testEnrollment, testActor)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(launch.LaunchToken))
	if repo.session.TokenHash != hex.EncodeToString(digest[:]) || repo.session.TokenHash == launch.LaunchToken {
		t.Fatalf("token was not stored as hash: %+v", repo.session)
	}
	public, err := service.GetPublicSession(context.Background(), launch.LaunchToken)
	if err != nil || public.ConsentVersion != PARQConsentVersion {
		t.Fatalf("unexpected public session: %+v err=%v", public, err)
	}
	result, err := service.SubmitPublic(context.Background(), launch.LaunchToken, SubmitInput{ConsentAccepted: true, Answers: domain.PARQAnswers{Dizziness: true}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.RequiresReview || repo.session.Status != "submitted" || result.Answers != nil {
		t.Fatalf("unexpected public submission: %+v session=%+v", result, repo.session)
	}
}
