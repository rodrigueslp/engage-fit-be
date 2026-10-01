package healthforms

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
	"github.com/google/uuid"
)

var (
	ErrInvalidHealthFormInput   = errors.New("invalid health form input")
	ErrHealthFormUnavailable    = errors.New("health form enrollment unavailable")
	ErrHealthFormSessionInvalid = repositories.ErrHealthFormSessionInvalid
)

const (
	PARQConsentVersion   = "parq-v1"
	PARQConsentText      = "Confirmo que respondi com veracidade e autorizo o tratamento destas informações de saúde para segurança da prática física."
	DefaultRetentionDays = 730
	PublicSessionTTL     = 30 * time.Minute
)

type Service struct {
	repository  repositories.HealthFormRepository
	enrollments repositories.ContractEnrollmentSnapshotResolver
	now         func() time.Time
}

func NewService(repository repositories.HealthFormRepository, enrollments repositories.ContractEnrollmentSnapshotResolver) *Service {
	return &Service{repository: repository, enrollments: enrollments, now: func() time.Time { return time.Now().UTC() }}
}

type SubmitInput struct {
	Answers         domain.PARQAnswers `json:"answers"`
	ConsentAccepted bool               `json:"consent_accepted"`
}

func (s *Service) Submit(ctx context.Context, organizationID, unitID, enrollmentID, actorUserID domain.ID, input SubmitInput) (*domain.HealthFormSubmission, error) {
	if !validID(organizationID) || !validID(unitID) || !validID(enrollmentID) || !validID(actorUserID) || !input.ConsentAccepted || len(strings.TrimSpace(input.Answers.Details)) > 2000 {
		return nil, ErrInvalidHealthFormInput
	}
	snapshot, err := s.enrollments.ResolveContractEnrollmentSnapshot(ctx, organizationID, unitID, enrollmentID, s.now())
	if err != nil {
		return nil, err
	}
	if snapshot.EffectiveStatus != domain.EnrollmentEffectiveScheduled && snapshot.EffectiveStatus != domain.EnrollmentEffectiveActive {
		return nil, ErrHealthFormUnavailable
	}
	now := s.now()
	form := &domain.HealthFormSubmission{ID: domain.ID(uuid.NewString()), OrganizationID: organizationID, UnitID: unitID, PersonID: snapshot.PersonID, EnrollmentID: enrollmentID, SchemaVersion: domain.HealthFormSchemaVersion, Status: domain.HealthFormStatusSubmitted, ConsentVersion: PARQConsentVersion, ConsentText: PARQConsentText, ConsentAcceptedAt: now, RequiresReview: input.Answers.RequiresReview(), RetentionDueAt: now.AddDate(0, 0, DefaultRetentionDays), CreatedByUserID: actorUserID, CreatedAt: now, UpdatedAt: now}
	if err := s.repository.Create(ctx, form, input.Answers); err != nil {
		return nil, err
	}
	return form, nil
}

func (s *Service) ListForEnrollment(ctx context.Context, organizationID, unitID, enrollmentID domain.ID) ([]domain.HealthFormSubmission, error) {
	if !validID(organizationID) || !validID(unitID) || !validID(enrollmentID) {
		return nil, ErrInvalidHealthFormInput
	}
	snapshot, err := s.enrollments.ResolveContractEnrollmentSnapshot(ctx, organizationID, unitID, enrollmentID, s.now())
	if err != nil {
		return nil, err
	}
	return s.repository.ListForPerson(ctx, organizationID, unitID, snapshot.PersonID)
}

func (s *Service) Get(ctx context.Context, organizationID, unitID, formID, actorUserID domain.ID) (*domain.HealthFormSubmission, error) {
	if !validID(organizationID) || !validID(unitID) || !validID(formID) || !validID(actorUserID) {
		return nil, ErrInvalidHealthFormInput
	}
	return s.repository.Find(ctx, organizationID, unitID, formID, true, actorUserID)
}

func (s *Service) Revoke(ctx context.Context, organizationID, unitID, formID, actorUserID domain.ID) (*domain.HealthFormSubmission, error) {
	if !validID(organizationID) || !validID(unitID) || !validID(formID) || !validID(actorUserID) {
		return nil, ErrInvalidHealthFormInput
	}
	return s.repository.RevokeConsent(ctx, organizationID, unitID, formID, actorUserID, s.now())
}

func (s *Service) CreatePublicSession(ctx context.Context, organizationID, unitID, enrollmentID, actorUserID domain.ID) (*domain.HealthFormSessionLaunch, error) {
	if !validID(organizationID) || !validID(unitID) || !validID(enrollmentID) || !validID(actorUserID) {
		return nil, ErrInvalidHealthFormInput
	}
	snapshot, err := s.enrollments.ResolveContractEnrollmentSnapshot(ctx, organizationID, unitID, enrollmentID, s.now())
	if err != nil {
		return nil, err
	}
	if snapshot.EffectiveStatus != domain.EnrollmentEffectiveScheduled && snapshot.EffectiveStatus != domain.EnrollmentEffectiveActive {
		return nil, ErrHealthFormUnavailable
	}
	forms, err := s.repository.ListForPerson(ctx, organizationID, unitID, snapshot.PersonID)
	if err != nil {
		return nil, err
	}
	for _, form := range forms {
		if form.EnrollmentID == enrollmentID && form.SchemaVersion == domain.HealthFormSchemaVersion && form.Status == domain.HealthFormStatusSubmitted {
			return nil, repositories.ErrHealthFormAlreadySubmitted
		}
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	digest := sha256.Sum256([]byte(token))
	now := s.now()
	session := &domain.HealthFormSession{ID: domain.ID(uuid.NewString()), OrganizationID: organizationID, UnitID: unitID, PersonID: snapshot.PersonID, EnrollmentID: enrollmentID, SchemaVersion: domain.HealthFormSchemaVersion, TokenHash: hex.EncodeToString(digest[:]), Status: "active", ExpiresAt: now.Add(PublicSessionTTL), CreatedByUserID: actorUserID, CreatedAt: now, UpdatedAt: now}
	if err := s.repository.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return &domain.HealthFormSessionLaunch{LaunchToken: token, LaunchPath: "/api/v1/public/health-forms/" + token, ExpiresAt: session.ExpiresAt}, nil
}

func (s *Service) GetPublicSession(ctx context.Context, token string) (*domain.PublicHealthFormSession, error) {
	session, err := s.findValidSession(ctx, token)
	if err != nil {
		return nil, err
	}
	return &domain.PublicHealthFormSession{SchemaVersion: session.SchemaVersion, ConsentVersion: PARQConsentVersion, ConsentText: PARQConsentText, ExpiresAt: session.ExpiresAt}, nil
}

func (s *Service) SubmitPublic(ctx context.Context, token string, input SubmitInput) (*domain.HealthFormSubmission, error) {
	if !input.ConsentAccepted || len(strings.TrimSpace(input.Answers.Details)) > 2000 {
		return nil, ErrInvalidHealthFormInput
	}
	session, err := s.findValidSession(ctx, token)
	if err != nil {
		return nil, err
	}
	now := s.now()
	form := &domain.HealthFormSubmission{ID: domain.ID(uuid.NewString()), OrganizationID: session.OrganizationID, UnitID: session.UnitID, PersonID: session.PersonID, EnrollmentID: session.EnrollmentID, SchemaVersion: domain.HealthFormSchemaVersion, Status: domain.HealthFormStatusSubmitted, ConsentVersion: PARQConsentVersion, ConsentText: PARQConsentText, ConsentAcceptedAt: now, RequiresReview: input.Answers.RequiresReview(), RetentionDueAt: now.AddDate(0, 0, DefaultRetentionDays), CreatedByUserID: session.CreatedByUserID, CreatedAt: now, UpdatedAt: now}
	if err := s.repository.CreateFromSession(ctx, session.ID, form, input.Answers, now); err != nil {
		return nil, err
	}
	return form, nil
}

func (s *Service) findValidSession(ctx context.Context, token string) (*domain.HealthFormSession, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(decoded) != 32 {
		return nil, ErrHealthFormSessionInvalid
	}
	digest := sha256.Sum256([]byte(token))
	session, err := s.repository.FindSessionByTokenHash(ctx, hex.EncodeToString(digest[:]))
	if err != nil {
		return nil, err
	}
	if session.Status != "active" || !s.now().Before(session.ExpiresAt) {
		return nil, ErrHealthFormSessionInvalid
	}
	return session, nil
}

func validID(value domain.ID) bool { _, err := uuid.Parse(string(value)); return err == nil }
