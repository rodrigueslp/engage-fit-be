package imports

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
	"gorm.io/gorm"
)

func TestImportErrorDetailsClassifiesParameterLimitWithoutExposingMessage(t *testing.T) {
	kind, sqlState := importErrorDetails("checkin_persistence", errors.New("extended protocol limited to 65535 parameters: sensitive detail"))
	if kind != "database_parameter_limit" || sqlState != "" {
		t.Fatalf("unexpected classification: %q %q", kind, sqlState)
	}
}

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

func TestImportFailureMarksHistoryFailedWithoutReturningOutput(t *testing.T) {
	expectedError := errors.New("student persistence failed")
	studentRepository := &studentRepositoryStub{findErr: gorm.ErrRecordNotFound, saveErr: expectedError}
	importRepository := &importHistoryRepositoryStub{}
	useCase := ImportCheckinsUseCase{
		parser:       checkinParserStub{result: []services.ParsedCheckin{{StudentName: "Student", CheckinDate: time.Now().UTC()}}},
		imports:      importRepository,
		students:     studentRepository,
		privacy:      privacyRepositoryStub{},
		transactions: transactionManagerStub{},
	}

	output, err := useCase.Execute(context.Background(), ImportCheckinsInput{
		BoxID: "box-1", Source: domain.SourceTotalPass, Filename: "failed.xlsx", File: strings.NewReader("fixture"),
	})
	if !errors.Is(err, expectedError) || output != nil {
		t.Fatalf("expected failed import, output=%+v err=%v", output, err)
	}
	if importRepository.saved == nil || importRepository.saved.Status != domain.ImportStatusProcessing {
		t.Fatalf("expected processing history, got %+v", importRepository.saved)
	}
	if importRepository.failedCode != "student_processing_failed" {
		t.Fatalf("expected normalized failure code, got %q", importRepository.failedCode)
	}
}

type studentRepositoryStub struct {
	students []domain.Student
	findErr  error
	saveErr  error
}

func (s *studentRepositoryStub) FindByID(context.Context, domain.ID, domain.ID) (*domain.Student, error) {
	return nil, nil
}
func (s *studentRepositoryStub) FindByExternalID(context.Context, domain.ID, domain.Source, string) (*domain.Student, error) {
	return nil, s.findErr
}
func (s *studentRepositoryStub) List(context.Context, domain.ID, repositories.StudentFilters) ([]domain.Student, error) {
	return s.students, nil
}
func (s *studentRepositoryStub) Save(context.Context, *domain.Student) error { return s.saveErr }
func (s *studentRepositoryStub) UpdateRiskStatus(context.Context, domain.ID, domain.ID, domain.StudentRiskStatus) error {
	return nil
}
func (s *studentRepositoryStub) MarkRiskMessageSent(context.Context, domain.ID, domain.ID, time.Time) error {
	return nil
}
func (s *studentRepositoryStub) UpdateContactPreference(context.Context, domain.ID, domain.ID, domain.ContactStatus, string, time.Time) error {
	return nil
}

type checkinParserStub struct {
	result []services.ParsedCheckin
	err    error
}

func (s checkinParserStub) Parse(context.Context, io.Reader, domain.Source, string) ([]services.ParsedCheckin, error) {
	return s.result, s.err
}

type transactionManagerStub struct{}

func (transactionManagerStub) WithinTransaction(ctx context.Context, operation func(context.Context) error) error {
	return operation(ctx)
}

type privacyRepositoryStub struct{}

func (privacyRepositoryStub) ExportStudent(context.Context, domain.ID, domain.ID) (*domain.StudentPrivacyExport, error) {
	return nil, nil
}
func (privacyRepositoryStub) AnonymizeStudent(context.Context, domain.ID, domain.ID, domain.ID, string) error {
	return nil
}
func (privacyRepositoryStub) IsIdentitySuppressed(context.Context, domain.ID, domain.Source, string) (bool, error) {
	return false, nil
}
func (privacyRepositoryStub) RecordAudit(context.Context, domain.ID, domain.ID, domain.ID, string, string) error {
	return nil
}

type importHistoryRepositoryStub struct {
	saved      *domain.ImportHistory
	failedCode string
}

func (s *importHistoryRepositoryStub) FindByID(context.Context, domain.ID, domain.ID) (*domain.ImportHistory, error) {
	return nil, nil
}
func (s *importHistoryRepositoryStub) List(context.Context, domain.ID) ([]domain.ImportHistory, error) {
	return nil, nil
}
func (s *importHistoryRepositoryStub) Save(_ context.Context, history *domain.ImportHistory) error {
	history.ID = "import-1"
	copy := *history
	s.saved = &copy
	return nil
}
func (s *importHistoryRepositoryStub) MarkCompleted(context.Context, domain.ID, domain.ID, int, int, time.Time) error {
	return nil
}
func (s *importHistoryRepositoryStub) MarkFailed(_ context.Context, _, _ domain.ID, errorCode string, _ time.Time) error {
	s.failedCode = errorCode
	return nil
}
func (s *importHistoryRepositoryStub) SetRetentionBaselineIfEmpty(context.Context, domain.ID, time.Time) error {
	return nil
}
