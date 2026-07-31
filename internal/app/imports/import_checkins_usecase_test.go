package imports

import (
	"context"
	"testing"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

func TestStudentIdentityNormalizesNameWhenNoStableIdentifierExists(t *testing.T) {
	first := studentIdentity(services.ParsedCheckin{StudentName: "  Adriana   Segatelli "})
	second := studentIdentity(services.ParsedCheckin{StudentName: "ADRIANA SEGATELLI"})

	if first != "adriana segatelli" {
		t.Fatalf("expected normalized identity, got %q", first)
	}
	if first != second {
		t.Fatalf("expected name variants to share an identity: %q != %q", first, second)
	}
}

func TestStudentIdentityStillPrefersStableExternalID(t *testing.T) {
	identity := studentIdentity(services.ParsedCheckin{
		StudentName:       "Adriana Segatelli",
		StudentExternalID: " MEMBER-123 ",
	})

	if identity != "member-123" {
		t.Fatalf("expected stable external ID, got %q", identity)
	}
}

func TestFindSelfRegisteredStudentReconcilesNormalizedName(t *testing.T) {
	source := domain.SourceWellhub
	student := domain.Student{
		ID: domain.ID("student-1"), BoxID: domain.ID("box-1"), Name: "Pessoa Nova", Source: source,
		MembershipStartedSource: "self_registration",
	}
	uc := ImportCheckinsUseCase{students: &studentRepositoryStub{students: []domain.Student{student}}}

	matched, err := uc.findSelfRegisteredStudent(context.Background(), domain.ID("box-1"), source, "  PESSOA   NOVA ")
	if err != nil {
		t.Fatal(err)
	}
	if matched == nil || matched.ID != student.ID {
		t.Fatalf("expected self-registered student, got %+v", matched)
	}
}

type studentRepositoryStub struct {
	students []domain.Student
}

func (s *studentRepositoryStub) FindByID(context.Context, domain.ID, domain.ID) (*domain.Student, error) {
	return nil, nil
}
func (s *studentRepositoryStub) FindByExternalID(context.Context, domain.ID, domain.Source, string) (*domain.Student, error) {
	return nil, nil
}
func (s *studentRepositoryStub) List(context.Context, domain.ID, repositories.StudentFilters) ([]domain.Student, error) {
	return s.students, nil
}
func (s *studentRepositoryStub) Save(context.Context, *domain.Student) error { return nil }
func (s *studentRepositoryStub) UpdateRiskStatus(context.Context, domain.ID, domain.ID, domain.StudentRiskStatus) error {
	return nil
}
func (s *studentRepositoryStub) MarkRiskMessageSent(context.Context, domain.ID, domain.ID, time.Time) error {
	return nil
}
func (s *studentRepositoryStub) UpdateContactPreference(context.Context, domain.ID, domain.ID, domain.ContactStatus, string, time.Time) error {
	return nil
}
