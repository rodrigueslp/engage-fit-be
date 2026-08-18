package people

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
	"github.com/google/uuid"
)

var (
	ErrInvalidInput          = errors.New("invalid people input")
	ErrCPFInvalid            = errors.New("person cpf invalid")
	ErrUnitNotFound          = errors.New("unit not found")
	ErrCPFConflict           = errors.New("person cpf conflict")
	ErrPossibleDuplicate     = errors.New("person possible duplicate")
	ErrIdempotencyKeyReused  = errors.New("idempotency key reused")
	ErrInvalidIdempotencyKey = errors.New("invalid idempotency key")
)

var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)

type AddressInput struct {
	PostalCode string `json:"postal_code"`
	Street     string `json:"street"`
	Number     string `json:"number"`
	Complement string `json:"complement"`
	District   string `json:"district"`
	City       string `json:"city"`
	State      string `json:"state"`
}

type PhoneDuplicateReviewInput struct {
	Decision        string `json:"decision"`
	MatchedPersonID string `json:"matched_person_id"`
	Reason          string `json:"reason"`
}

type CreateInput struct {
	Name                 string                     `json:"name"`
	Phone                string                     `json:"phone"`
	CPF                  string                     `json:"cpf"`
	Email                string                     `json:"email"`
	Address              *AddressInput              `json:"address"`
	PhoneDuplicateReview *PhoneDuplicateReviewInput `json:"phone_duplicate_review"`
}

type DuplicateCheckInput struct {
	CPF   string `json:"cpf"`
	Phone string `json:"phone"`
}

type Service struct {
	people  repositories.PeopleRepository
	boxes   repositories.BoxRepository
	cipher  services.SecretCipher
	hmacKey []byte
	now     func() time.Time
}

func NewService(peopleRepository repositories.PeopleRepository, boxRepository repositories.BoxRepository, cipher services.SecretCipher, encodedHMACKey string) (*Service, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encodedHMACKey))
	if err != nil || len(key) != 32 {
		return nil, errors.New("PERSON_LOOKUP_HMAC_KEY must be exactly 32 bytes encoded as base64")
	}
	if cipher == nil {
		return nil, errors.New("authenticated person CPF cipher is required")
	}
	return &Service{people: peopleRepository, boxes: boxRepository, cipher: cipher, hmacKey: key, now: func() time.Time { return time.Now().UTC() }}, nil
}

func (s *Service) DuplicateCheck(ctx context.Context, authenticatedUnitID, requestedUnitID domain.ID, input DuplicateCheckInput) (*domain.PersonDuplicateCheck, error) {
	organizationID, err := s.organizationForUnit(ctx, authenticatedUnitID, requestedUnitID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.CPF) == "" && strings.TrimSpace(input.Phone) == "" {
		return nil, ErrInvalidInput
	}
	cpfHash := ""
	if strings.TrimSpace(input.CPF) != "" {
		cpf, ok := NormalizeCPF(input.CPF)
		if !ok {
			return nil, ErrCPFInvalid
		}
		cpfHash = s.lookup("cpf", cpf)
	}
	phone := ""
	if strings.TrimSpace(input.Phone) != "" {
		var ok bool
		phone, ok = NormalizeBrazilianPhone(input.Phone)
		if !ok {
			return nil, ErrInvalidInput
		}
	}
	result, err := s.people.DuplicateCheck(ctx, organizationID, requestedUnitID, cpfHash, phone)
	if err != nil {
		return nil, err
	}
	maskDuplicateCheck(result)
	return result, nil
}

func (s *Service) Create(ctx context.Context, authenticatedUnitID, requestedUnitID, actorUserID domain.ID, idempotencyKey string, input CreateInput) (*domain.PersonCreationResult, bool, error) {
	if !idempotencyKeyPattern.MatchString(idempotencyKey) {
		return nil, false, ErrInvalidIdempotencyKey
	}
	organizationID, err := s.organizationForUnit(ctx, authenticatedUnitID, requestedUnitID)
	if err != nil {
		return nil, false, err
	}
	name, ok := NormalizeName(input.Name)
	if !ok {
		return nil, false, ErrInvalidInput
	}
	phone, ok := NormalizeBrazilianPhone(input.Phone)
	if !ok {
		return nil, false, ErrInvalidInput
	}
	cpf, ok := NormalizeCPF(input.CPF)
	if !ok {
		return nil, false, ErrCPFInvalid
	}
	email, ok := NormalizeEmail(input.Email)
	if !ok {
		return nil, false, ErrInvalidInput
	}
	address, ok := normalizeAddress(input.Address)
	if !ok {
		return nil, false, ErrInvalidInput
	}
	review, ok := normalizeReview(input.PhoneDuplicateReview)
	if !ok {
		return nil, false, ErrInvalidInput
	}

	personID := domain.ID(uuid.NewString())
	studentID := domain.ID(uuid.NewString())
	createdAt := s.now()
	cpfEncrypted, err := s.cipher.Encrypt(cpf, domain.PersonCPFAssociatedData(organizationID, personID))
	if err != nil || cpfEncrypted == "" || cpfEncrypted == cpf {
		return nil, false, errors.New("person CPF encryption failed")
	}

	canonical := struct {
		Name    string                             `json:"name"`
		Phone   string                             `json:"phone"`
		CPF     string                             `json:"cpf"`
		Email   string                             `json:"email"`
		Address *domain.PersonAddress              `json:"address,omitempty"`
		Review  *domain.PersonPhoneDuplicateReview `json:"phone_duplicate_review,omitempty"`
	}{Name: name, Phone: phone, CPF: cpf, Email: email, Address: address, Review: review}
	canonicalPayload, err := json.Marshal(canonical)
	if err != nil {
		return nil, false, err
	}
	if address != nil {
		address.ID = domain.ID(uuid.NewString())
		address.PersonID = personID
		address.OrganizationID = organizationID
		address.CreatedByUserID = actorUserID
		address.UpdatedByUserID = actorUserID
		address.CreatedAt = createdAt
		address.UpdatedAt = createdAt
	}
	command := domain.CreatePersonCommand{
		OrganizationID: organizationID,
		UnitID:         requestedUnitID,
		ActorUserID:    actorUserID,
		IdempotencyKey: idempotencyKey,
		Fingerprint:    s.lookup("request", string(canonicalPayload)),
		Person:         domain.Person{ID: personID, OrganizationID: organizationID, Name: name, PhoneE164: phone, Email: email, CPFEncrypted: cpfEncrypted, CPFLookupHash: s.lookup("cpf", cpf), CPFLast2: cpf[9:], ProfileStatus: domain.PersonProfileActive, CreatedAt: createdAt, UpdatedAt: createdAt},
		Address:        address,
		Review:         review,
		Student:        domain.Student{ID: studentID, PersonID: personID, BoxID: requestedUnitID, Name: name, Phone: strings.TrimPrefix(phone, "+"), Email: email, Source: domain.SourceBoxMember, ExternalID: "person:" + string(personID), RiskStatus: domain.StudentRiskStatusActive, ContactStatus: domain.ContactStatusUnknown, MembershipStartedAt: &createdAt, MembershipStartedSource: "manual", RetentionMonitoringStatus: domain.RetentionMonitoringMonitored, CreatedAt: createdAt, UpdatedAt: createdAt},
		CreatedAt:      createdAt,
	}
	return s.people.Create(ctx, command)
}

func (s *Service) organizationForUnit(ctx context.Context, authenticatedUnitID, requestedUnitID domain.ID) (domain.ID, error) {
	if authenticatedUnitID == "" || requestedUnitID == "" || authenticatedUnitID != requestedUnitID {
		return "", ErrUnitNotFound
	}
	box, err := s.boxes.FindByID(ctx, authenticatedUnitID)
	if err != nil || box.OrganizationID == "" {
		return "", ErrUnitNotFound
	}
	return box.OrganizationID, nil
}

func (s *Service) lookup(purpose, value string) string {
	mac := hmac.New(sha256.New, s.hmacKey)
	_, _ = mac.Write([]byte("people:" + purpose + ":" + value))
	return hex.EncodeToString(mac.Sum(nil))
}

func NormalizeName(value string) (string, bool) {
	value = strings.Join(strings.Fields(value), " ")
	return value, len([]rune(value)) >= 2 && len([]rune(value)) <= 160
}

func NormalizeCPF(value string) (string, bool) {
	digits := digitsOnly(value)
	if len(digits) != 11 || strings.Count(digits, digits[:1]) == 11 {
		return "", false
	}
	for position := 9; position <= 10; position++ {
		sum := 0
		for index := 0; index < position; index++ {
			sum += int(digits[index]-'0') * (position + 1 - index)
		}
		check := (sum * 10) % 11
		if check == 10 {
			check = 0
		}
		if check != int(digits[position]-'0') {
			return "", false
		}
	}
	return digits, true
}

func NormalizeBrazilianPhone(value string) (string, bool) {
	digits := digitsOnly(value)
	if len(digits) == 12 || len(digits) == 13 {
		if !strings.HasPrefix(digits, "55") {
			return "", false
		}
		digits = digits[2:]
	}
	if len(digits) != 10 && len(digits) != 11 {
		return "", false
	}
	if digits[0] == '0' || digits[1] == '0' || (len(digits) == 11 && digits[2] != '9') {
		return "", false
	}
	return "+55" + digits, true
}

func NormalizeEmail(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", true
	}
	if len(value) > 254 {
		return "", false
	}
	address, err := mail.ParseAddress(value)
	return value, err == nil && address.Address == value
}

func normalizeAddress(input *AddressInput) (*domain.PersonAddress, bool) {
	if input == nil {
		return nil, true
	}
	address := &domain.PersonAddress{AddressType: "home", PostalCode: digitsOnly(input.PostalCode), Street: strings.TrimSpace(input.Street), Number: strings.TrimSpace(input.Number), Complement: strings.TrimSpace(input.Complement), District: strings.TrimSpace(input.District), City: strings.TrimSpace(input.City), State: strings.ToUpper(strings.TrimSpace(input.State)), Country: "BR"}
	if input.PostalCode != "" && len(address.PostalCode) != 8 {
		return nil, false
	}
	if address.State != "" && (len(address.State) != 2 || !allASCIIUpper(address.State)) {
		return nil, false
	}
	if len([]rune(address.Street)) > 160 || len([]rune(address.Number)) > 30 || len([]rune(address.Complement)) > 100 || len([]rune(address.District)) > 100 || len([]rune(address.City)) > 100 {
		return nil, false
	}
	if address.PostalCode == "" && address.Street == "" && address.Number == "" && address.Complement == "" && address.District == "" && address.City == "" && address.State == "" {
		return nil, true
	}
	return address, true
}

func normalizeReview(input *PhoneDuplicateReviewInput) (*domain.PersonPhoneDuplicateReview, bool) {
	if input == nil {
		return nil, true
	}
	reason := strings.Join(strings.Fields(input.Reason), " ")
	matchedID, err := uuid.Parse(strings.TrimSpace(input.MatchedPersonID))
	if input.Decision != "create_separate" || err != nil || len([]rune(reason)) < 5 || len([]rune(reason)) > 280 {
		return nil, false
	}
	return &domain.PersonPhoneDuplicateReview{Decision: input.Decision, MatchedPersonID: domain.ID(matchedID.String()), Reason: reason}, true
}

func digitsOnly(value string) string {
	var builder strings.Builder
	for _, current := range value {
		if unicode.IsDigit(current) && current <= unicode.MaxASCII {
			builder.WriteRune(current)
		}
	}
	return builder.String()
}

func allASCIIUpper(value string) bool {
	for _, current := range value {
		if current < 'A' || current > 'Z' {
			return false
		}
	}
	return true
}

func maskDuplicateCheck(result *domain.PersonDuplicateCheck) {
	if result == nil {
		return
	}
	if result.CPFMatch != nil {
		result.CPFMatch.Name = MaskName(result.CPFMatch.Name)
		result.CPFMatch.PhoneE164 = MaskPhone(result.CPFMatch.PhoneE164)
	}
	for index := range result.PhoneMatches {
		result.PhoneMatches[index].Name = MaskName(result.PhoneMatches[index].Name)
		result.PhoneMatches[index].PhoneE164 = MaskPhone(result.PhoneMatches[index].PhoneE164)
	}
}

func MaskName(value string) string {
	parts := strings.Fields(value)
	if len(parts) <= 1 {
		return value
	}
	return parts[0] + " " + string([]rune(parts[len(parts)-1])[0]) + "."
}

func MaskPhone(value string) string {
	digits := strings.TrimPrefix(value, "+55")
	if len(digits) == 11 {
		return "+55 " + digits[:2] + " *****-" + digits[7:]
	}
	if len(digits) == 10 {
		return "+55 " + digits[:2] + " ****-" + digits[6:]
	}
	return ""
}
