package people

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

type peopleRepositoryStub struct {
	duplicate *domain.PersonDuplicateCheck
	command   domain.CreatePersonCommand
	result    *domain.PersonCreationResult
	replayed  bool
	err       error
}

func (s *peopleRepositoryStub) DuplicateCheck(context.Context, domain.ID, domain.ID, string, string) (*domain.PersonDuplicateCheck, error) {
	if s.duplicate == nil {
		return &domain.PersonDuplicateCheck{PhoneMatches: []domain.PersonDuplicateMatch{}}, s.err
	}
	return s.duplicate, s.err
}
func (s *peopleRepositoryStub) Create(_ context.Context, command domain.CreatePersonCommand) (*domain.PersonCreationResult, bool, error) {
	s.command = command
	if s.result == nil {
		s.result = &domain.PersonCreationResult{PersonID: command.Person.ID, StudentID: command.Student.ID, Name: command.Person.Name, CPFLast2: command.Person.CPFLast2}
	}
	return s.result, s.replayed, s.err
}
func (*peopleRepositoryStub) CountStudentsWithoutPerson(context.Context) (int64, error) {
	return 0, nil
}

type boxRepositoryStub struct{ box domain.Box }

func (s boxRepositoryStub) FindByID(_ context.Context, id domain.ID) (*domain.Box, error) {
	if id != s.box.ID {
		return nil, errors.New("not found")
	}
	copy := s.box
	return &copy, nil
}
func (boxRepositoryStub) ListAll(context.Context) ([]domain.Box, error)                  { return nil, nil }
func (boxRepositoryStub) Save(context.Context, *domain.Box) error                        { return nil }
func (boxRepositoryStub) SaveWithOwner(context.Context, *domain.Box, *domain.User) error { return nil }
func (boxRepositoryStub) Update(context.Context, domain.Box) error                       { return nil }

type authenticatedCipherStub struct {
	associatedData string
	plaintext      string
}

func (c *authenticatedCipherStub) Encrypt(plaintext, associatedData string) (string, error) {
	c.plaintext, c.associatedData = plaintext, associatedData
	return "enc:v1:test:ciphertext", nil
}
func (*authenticatedCipherStub) Decrypt(string, string) (string, error) { return "", nil }
func (*authenticatedCipherStub) NeedsRotation(string) bool              { return false }

func newPeopleServiceForTest(t *testing.T, repository *peopleRepositoryStub, cipher *authenticatedCipherStub) *Service {
	t.Helper()
	key := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))
	service, err := NewService(repository, boxRepositoryStub{box: domain.Box{ID: "unit-1", OrganizationID: "organization-1"}}, cipher, key)
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return time.Date(2026, 8, 18, 20, 0, 0, 0, time.UTC) }
	return service
}

func TestNormalizePeopleFields(t *testing.T) {
	if value, ok := NormalizeCPF("529.982.247-25"); !ok || value != "52998224725" {
		t.Fatalf("valid CPF = %q, %v", value, ok)
	}
	for _, value := range []string{"111.111.111-11", "529.982.247-24", "123"} {
		if _, ok := NormalizeCPF(value); ok {
			t.Fatalf("invalid CPF %q was accepted", value)
		}
	}
	if value, ok := NormalizeName("  Maria\t da\nSilva  "); !ok || value != "Maria da Silva" {
		t.Fatalf("name = %q, %v", value, ok)
	}
	if value, ok := NormalizeBrazilianPhone("(11) 99999-9999"); !ok || value != "+5511999999999" {
		t.Fatalf("phone = %q, %v", value, ok)
	}
	if value, ok := NormalizeEmail(" MARIA@Example.COM "); !ok || value != "maria@example.com" {
		t.Fatalf("email = %q, %v", value, ok)
	}
}

func TestNewServiceRejectsInvalidDedicatedHMACKey(t *testing.T) {
	if _, err := NewService(&peopleRepositoryStub{}, boxRepositoryStub{}, &authenticatedCipherStub{}, "not-base64"); err == nil {
		t.Fatal("invalid HMAC key should fail startup construction")
	}
	short := base64.StdEncoding.EncodeToString([]byte("short"))
	if _, err := NewService(&peopleRepositoryStub{}, boxRepositoryStub{}, &authenticatedCipherStub{}, short); err == nil {
		t.Fatal("short HMAC key should fail startup construction")
	}
}

func TestCreateBuildsEncryptedCompatiblePersonAndStudent(t *testing.T) {
	repository := &peopleRepositoryStub{}
	cipher := &authenticatedCipherStub{}
	service := newPeopleServiceForTest(t, repository, cipher)
	result, replayed, err := service.Create(context.Background(), "unit-1", "unit-1", "actor-1", "intent-123", CreateInput{Name: " Maria  da Silva ", Phone: "(11) 99999-9999", CPF: "529.982.247-25", Email: " MARIA@example.com ", Address: &AddressInput{PostalCode: "01234-567", State: "sp", Street: "Rua Teste"}})
	if err != nil || replayed || result == nil {
		t.Fatalf("create = %+v, %v, %v", result, replayed, err)
	}
	command := repository.command
	if command.Person.CPFEncrypted == "52998224725" || command.Person.CPFEncrypted == "" {
		t.Fatal("CPF was not encrypted")
	}
	plainSHA := sha256.Sum256([]byte("52998224725"))
	if command.Person.CPFLookupHash == hex.EncodeToString(plainSHA[:]) || len(command.Person.CPFLookupHash) != 64 {
		t.Fatal("CPF lookup must be a dedicated keyed HMAC")
	}
	if cipher.associatedData != domain.PersonCPFAssociatedData("organization-1", command.Person.ID) {
		t.Fatalf("associated data = %q", cipher.associatedData)
	}
	if command.Student.PersonID != command.Person.ID || command.Student.Source != domain.SourceBoxMember || command.Student.ContactStatus != domain.ContactStatusUnknown || command.Student.Phone != "5511999999999" || command.Student.MembershipStartedSource != "manual" {
		t.Fatalf("incompatible student projection: %+v", command.Student)
	}
	if command.Address == nil || command.Address.PostalCode != "01234567" || command.Address.State != "SP" {
		t.Fatalf("address = %+v", command.Address)
	}
}

func TestCreateRejectsTenantMismatchAndInvalidIdempotency(t *testing.T) {
	service := newPeopleServiceForTest(t, &peopleRepositoryStub{}, &authenticatedCipherStub{})
	valid := CreateInput{Name: "Maria", Phone: "11999999999", CPF: "52998224725"}
	if _, _, err := service.Create(context.Background(), "unit-1", "unit-2", "actor", "intent-123", valid); !errors.Is(err, ErrUnitNotFound) {
		t.Fatalf("cross-unit create = %v", err)
	}
	if _, _, err := service.Create(context.Background(), "unit-1", "unit-1", "actor", "short", valid); !errors.Is(err, ErrInvalidIdempotencyKey) {
		t.Fatalf("invalid key = %v", err)
	}
}

func TestDuplicateCheckMasksMinimalCandidateData(t *testing.T) {
	repository := &peopleRepositoryStub{duplicate: &domain.PersonDuplicateCheck{PhoneMatches: []domain.PersonDuplicateMatch{{PersonID: "person-1", Name: "Maria da Silva", PhoneE164: "+5511999999999", RelationshipStatus: domain.PersonRelationshipActive}}}}
	service := newPeopleServiceForTest(t, repository, &authenticatedCipherStub{})
	result, err := service.DuplicateCheck(context.Background(), "unit-1", "unit-1", DuplicateCheckInput{Phone: "11999999999"})
	if err != nil || len(result.PhoneMatches) != 1 {
		t.Fatal(err)
	}
	match := result.PhoneMatches[0]
	if match.Name != "Maria S." || match.PhoneE164 != "+55 11 *****-9999" {
		t.Fatalf("candidate was not minimized: %+v", match)
	}
}

var _ repositories.PeopleRepository = (*peopleRepositoryStub)(nil)
