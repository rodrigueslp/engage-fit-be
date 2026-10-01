package people

import (
	"context"
	"errors"
	"strings"

	"boxengage/backend/internal/domain"
	"github.com/google/uuid"
)

var (
	ErrPersonNotFound              = errors.New("person not found")
	ErrPersonRevisionConflict      = errors.New("person revision conflict")
	ErrPersonAlreadyLinked         = errors.New("person unit already linked")
	ErrExternalIdentityConflict    = errors.New("person external identity conflict")
	ErrIdentityReviewNotFound      = errors.New("person identity review not found")
	ErrMergeNotAllowed             = errors.New("person merge not allowed")
	ErrOpenEnrollmentPhoneRequired = errors.New("open enrollment phone required")
)

type SearchInput struct {
	Query string `json:"query"`
}

type UpdateInput struct {
	Revision      int                        `json:"revision"`
	Name          *string                    `json:"name"`
	PreferredName *string                    `json:"preferred_name"`
	Phone         *string                    `json:"phone"`
	Email         *string                    `json:"email"`
	CPF           *string                    `json:"cpf"`
	ProfileStatus *string                    `json:"profile_status"`
	Address       *AddressInput              `json:"address"`
	ClearAddress  bool                       `json:"clear_address"`
	PhoneReview   *PhoneDuplicateReviewInput `json:"phone_duplicate_review"`
}

type LinkUnitInput struct {
	Reason string `json:"reason"`
}

type ExternalIdentityInput struct {
	Provider   string `json:"provider"`
	ExternalID string `json:"external_id"`
	StudentID  string `json:"student_id"`
	Verified   bool   `json:"verified"`
}

type MergeInput struct {
	SourcePersonID string `json:"source_person_id"`
	Reason         string `json:"reason"`
}

type ResolveReviewInput struct {
	Resolution string `json:"resolution"`
	Reason     string `json:"reason"`
}

func (s *Service) List(ctx context.Context, authenticatedUnitID, requestedUnitID domain.ID, input SearchInput) ([]domain.PersonSummary, error) {
	organizationID, err := s.organizationForUnit(ctx, authenticatedUnitID, requestedUnitID)
	if err != nil {
		return nil, err
	}
	criteria, err := s.searchCriteria(input.Query)
	if err != nil {
		return nil, err
	}
	return s.people.List(ctx, organizationID, requestedUnitID, criteria)
}

func (s *Service) Get(ctx context.Context, authenticatedUnitID, requestedUnitID, personID domain.ID, includePrivate bool) (*domain.PersonProfile, error) {
	organizationID, err := s.organizationForUnit(ctx, authenticatedUnitID, requestedUnitID)
	if err != nil {
		return nil, err
	}
	if !validID(personID) {
		return nil, ErrPersonNotFound
	}
	profile, err := s.people.Get(ctx, organizationID, requestedUnitID, personID)
	if err != nil || profile == nil {
		return profile, err
	}
	if !includePrivate {
		profile.Person.PhoneE164 = MaskPhone(profile.Person.PhoneE164)
		profile.Person.Email = MaskEmail(profile.Person.Email)
		profile.Address = nil
		profile.CPFMasked = ""
		profile.Identities = nil
		profile.AthleteAccount = nil
	}
	return profile, nil
}

func (s *Service) Update(ctx context.Context, authenticatedUnitID, requestedUnitID, personID, actorUserID domain.ID, input UpdateInput) (*domain.PersonProfile, error) {
	organizationID, err := s.organizationForUnit(ctx, authenticatedUnitID, requestedUnitID)
	if err != nil {
		return nil, err
	}
	if !validID(personID) || input.Revision < 1 || (input.Address != nil && input.ClearAddress) {
		return nil, ErrInvalidInput
	}
	command := domain.UpdatePersonCommand{OrganizationID: organizationID, UnitID: requestedUnitID, PersonID: personID, ActorUserID: actorUserID, ExpectedRevision: input.Revision, ClearAddress: input.ClearAddress, UpdatedAt: s.now()}
	if input.Name != nil {
		value, ok := NormalizeName(*input.Name)
		if !ok {
			return nil, ErrInvalidInput
		}
		command.Name = &value
	}
	if input.PreferredName != nil {
		value := strings.Join(strings.Fields(*input.PreferredName), " ")
		if value != "" {
			if normalized, ok := NormalizeName(value); !ok {
				return nil, ErrInvalidInput
			} else {
				value = normalized
			}
		}
		command.PreferredName = &value
	}
	if input.Phone != nil {
		value := ""
		if strings.TrimSpace(*input.Phone) != "" {
			var ok bool
			value, ok = NormalizeBrazilianPhone(*input.Phone)
			if !ok {
				return nil, ErrInvalidInput
			}
			duplicates, checkErr := s.people.DuplicateCheck(ctx, organizationID, requestedUnitID, "", value)
			if checkErr != nil {
				return nil, checkErr
			}
			var other *domain.PersonDuplicateMatch
			for index := range duplicates.PhoneMatches {
				if duplicates.PhoneMatches[index].PersonID != personID {
					other = &duplicates.PhoneMatches[index]
					break
				}
			}
			if other != nil {
				review, valid := normalizeReview(input.PhoneReview)
				if !valid || review == nil || review.MatchedPersonID != other.PersonID {
					return nil, ErrPossibleDuplicate
				}
				command.PhoneReview = review
			}
		}
		command.PhoneE164 = &value
	}
	if input.Email != nil {
		value, ok := NormalizeEmail(*input.Email)
		if !ok {
			return nil, ErrInvalidInput
		}
		command.Email = &value
	}
	if input.CPF != nil {
		cpf, ok := NormalizeCPF(*input.CPF)
		if !ok {
			return nil, ErrCPFInvalid
		}
		ciphertext, cipherErr := s.cipher.Encrypt(cpf, domain.PersonCPFAssociatedData(organizationID, personID))
		if cipherErr != nil || ciphertext == "" || ciphertext == cpf {
			return nil, errors.New("person CPF encryption failed")
		}
		hash, last2 := s.lookup("cpf", cpf), cpf[9:]
		command.CPFEncrypted, command.CPFLookupHash, command.CPFLast2 = &ciphertext, &hash, &last2
	}
	if input.ProfileStatus != nil {
		status := domain.PersonProfileStatus(strings.TrimSpace(*input.ProfileStatus))
		if status != domain.PersonProfileActive && status != domain.PersonProfileArchived {
			return nil, ErrInvalidInput
		}
		command.ProfileStatus = &status
	}
	if input.Address != nil {
		address, ok := normalizeAddress(input.Address)
		if !ok {
			return nil, ErrInvalidInput
		}
		if address == nil {
			command.ClearAddress = true
		} else {
			address.ID = domain.ID(uuid.NewString())
			address.PersonID, address.OrganizationID = personID, organizationID
			address.CreatedByUserID, address.UpdatedByUserID = actorUserID, actorUserID
			address.CreatedAt, address.UpdatedAt = command.UpdatedAt, command.UpdatedAt
			command.Address = address
		}
	}
	if command.Name == nil && command.PreferredName == nil && command.PhoneE164 == nil && command.Email == nil && command.CPFEncrypted == nil && command.ProfileStatus == nil && command.Address == nil && !command.ClearAddress {
		return nil, ErrInvalidInput
	}
	return s.people.Update(ctx, command)
}

func (s *Service) LinkUnit(ctx context.Context, authenticatedUnitID, requestedUnitID, personID, actorUserID domain.ID, input LinkUnitInput) (*domain.PersonProfile, error) {
	organizationID, err := s.organizationForUnit(ctx, authenticatedUnitID, requestedUnitID)
	if err != nil {
		return nil, err
	}
	if !validID(personID) {
		return nil, ErrPersonNotFound
	}
	reason := strings.Join(strings.Fields(input.Reason), " ")
	if reason != "" && (len([]rune(reason)) < 5 || len([]rune(reason)) > 280) {
		return nil, ErrInvalidInput
	}
	now := s.now()
	studentID := domain.ID(uuid.NewString())
	command := domain.LinkPersonUnitCommand{OrganizationID: organizationID, UnitID: requestedUnitID, PersonID: personID, ActorUserID: actorUserID, Reason: reason, CreatedAt: now,
		Student: domain.Student{ID: studentID, PersonID: personID, BoxID: requestedUnitID, Source: domain.SourceBoxMember, ExternalID: "person:" + string(personID), RiskStatus: domain.StudentRiskStatusActive, ContactStatus: domain.ContactStatusUnknown, MembershipStartedAt: &now, MembershipStartedSource: "manual", RetentionMonitoringStatus: domain.RetentionMonitoringMonitored, CreatedAt: now, UpdatedAt: now},
	}
	return s.people.LinkUnit(ctx, command)
}

func (s *Service) LinkExternalIdentity(ctx context.Context, authenticatedUnitID, requestedUnitID, personID, actorUserID domain.ID, input ExternalIdentityInput) (*domain.PersonProfile, error) {
	organizationID, err := s.organizationForUnit(ctx, authenticatedUnitID, requestedUnitID)
	if err != nil {
		return nil, err
	}
	provider := strings.ToLower(strings.TrimSpace(input.Provider))
	externalID := strings.TrimSpace(input.ExternalID)
	if !validID(personID) || (provider != "wellhub" && provider != "totalpass") || len([]rune(externalID)) < 1 || len([]rune(externalID)) > 255 {
		return nil, ErrInvalidInput
	}
	studentID := domain.ID("")
	if input.StudentID != "" {
		parsed, parseErr := uuid.Parse(input.StudentID)
		if parseErr != nil {
			return nil, ErrInvalidInput
		}
		studentID = domain.ID(parsed.String())
	}
	return s.people.LinkExternalIdentity(ctx, domain.LinkExternalIdentityCommand{OrganizationID: organizationID, UnitID: requestedUnitID, PersonID: personID, ActorUserID: actorUserID, Provider: provider, ExternalID: externalID, StudentID: studentID, Verified: input.Verified, CreatedAt: s.now()})
}

func (s *Service) ListIdentityReviews(ctx context.Context, authenticatedUnitID, requestedUnitID domain.ID) ([]domain.PersonIdentityReview, error) {
	organizationID, err := s.organizationForUnit(ctx, authenticatedUnitID, requestedUnitID)
	if err != nil {
		return nil, err
	}
	return s.people.ListIdentityReviews(ctx, organizationID, requestedUnitID)
}

func (s *Service) Merge(ctx context.Context, authenticatedUnitID, requestedUnitID, targetPersonID, actorUserID domain.ID, input MergeInput) (*domain.PersonProfile, error) {
	organizationID, err := s.organizationForUnit(ctx, authenticatedUnitID, requestedUnitID)
	if err != nil {
		return nil, err
	}
	sourceID, parseErr := uuid.Parse(strings.TrimSpace(input.SourcePersonID))
	reason := strings.Join(strings.Fields(input.Reason), " ")
	if !validID(targetPersonID) || parseErr != nil || domain.ID(sourceID.String()) == targetPersonID || len([]rune(reason)) < 5 || len([]rune(reason)) > 280 {
		return nil, ErrInvalidInput
	}
	return s.people.Merge(ctx, domain.MergePeopleCommand{OrganizationID: organizationID, UnitID: requestedUnitID, TargetPersonID: targetPersonID, SourcePersonID: domain.ID(sourceID.String()), ActorUserID: actorUserID, Reason: reason, CreatedAt: s.now()})
}

func (s *Service) ResolveIdentityReview(ctx context.Context, authenticatedUnitID, requestedUnitID, reviewID, actorUserID domain.ID, input ResolveReviewInput) (*domain.PersonIdentityReview, *domain.PersonProfile, error) {
	organizationID, err := s.organizationForUnit(ctx, authenticatedUnitID, requestedUnitID)
	if err != nil {
		return nil, nil, err
	}
	if !validID(reviewID) || (input.Resolution != "merge" && input.Resolution != "dismiss") {
		return nil, nil, ErrInvalidInput
	}
	reason := strings.Join(strings.Fields(input.Reason), " ")
	if len([]rune(reason)) < 5 || len([]rune(reason)) > 280 {
		return nil, nil, ErrInvalidInput
	}
	return s.people.ResolveIdentityReview(ctx, domain.MergePeopleCommand{OrganizationID: organizationID, UnitID: requestedUnitID, ActorUserID: actorUserID, ReviewID: reviewID, Reason: reason, CreatedAt: s.now()}, input.Resolution)
}

func (s *Service) searchCriteria(query string) (domain.PersonSearchCriteria, error) {
	query = strings.TrimSpace(query)
	if len([]rune(query)) > 160 {
		return domain.PersonSearchCriteria{}, ErrInvalidInput
	}
	if query == "" {
		return domain.PersonSearchCriteria{}, nil
	}
	digits := digitsOnly(query)
	if len(digits) == 11 {
		criteria := domain.PersonSearchCriteria{}
		if cpf, ok := NormalizeCPF(query); ok {
			criteria.CPFLookupHash = s.lookup("cpf", cpf)
		}
		if phone, ok := NormalizeBrazilianPhone(query); ok {
			criteria.PhoneE164 = phone
		}
		if criteria.CPFLookupHash != "" || criteria.PhoneE164 != "" {
			return criteria, nil
		}
	}
	if len(digits) == 10 || len(digits) == 12 || len(digits) == 13 {
		if phone, ok := NormalizeBrazilianPhone(query); ok {
			return domain.PersonSearchCriteria{PhoneE164: phone}, nil
		}
	}
	return domain.PersonSearchCriteria{NameQuery: query}, nil
}

func MaskEmail(value string) string {
	parts := strings.Split(value, "@")
	if len(parts) != 2 || parts[0] == "" {
		return ""
	}
	return string([]rune(parts[0])[0]) + "***@" + parts[1]
}

func validID(value domain.ID) bool {
	_, err := uuid.Parse(string(value))
	return err == nil
}
