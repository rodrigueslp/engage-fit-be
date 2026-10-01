package dto

import (
	"time"

	"boxengage/backend/internal/app/people"
	"boxengage/backend/internal/domain"
)

type PersonDuplicateMatchResponse struct {
	PersonID           string `json:"person_id"`
	Name               string `json:"name"`
	PhoneMasked        string `json:"phone_masked"`
	RelationshipStatus string `json:"relationship_status"`
}

type PersonDuplicateCheckResponse struct {
	CPFMatch     *PersonDuplicateMatchResponse  `json:"cpf_match"`
	PhoneMatches []PersonDuplicateMatchResponse `json:"phone_matches"`
}

type CreatePersonRequest = people.CreateInput
type PersonDuplicateCheckRequest = people.DuplicateCheckInput

type CreatePersonResponse struct {
	PersonID           string `json:"person_id"`
	StudentID          string `json:"student_id"`
	Name               string `json:"name"`
	Phone              string `json:"phone"`
	Email              string `json:"email,omitempty"`
	CPFMasked          string `json:"cpf_masked"`
	RelationshipStatus string `json:"relationship_status"`
	ContactStatus      string `json:"contact_status"`
	CreatedAt          string `json:"created_at"`
}

type PersonProfileResponse struct {
	ID                 string                          `json:"id"`
	Name               string                          `json:"name"`
	PreferredName      string                          `json:"preferred_name,omitempty"`
	Phone              string                          `json:"phone,omitempty"`
	Email              string                          `json:"email,omitempty"`
	CPFMasked          string                          `json:"cpf_masked,omitempty"`
	ProfileStatus      string                          `json:"profile_status"`
	Revision           int                             `json:"revision"`
	Address            *PersonAddressResponse          `json:"address,omitempty"`
	Relationships      []domain.PersonUnitRelationship `json:"relationships"`
	SourceProfiles     []domain.PersonSourceProfile    `json:"source_profiles"`
	ExternalIdentities []domain.PersonExternalIdentity `json:"external_identities,omitempty"`
	AthleteAccount     *domain.PersonAthleteAccount    `json:"athlete_account,omitempty"`
	History            []domain.PeopleAuditEvent       `json:"history"`
	CreatedAt          time.Time                       `json:"created_at"`
	UpdatedAt          time.Time                       `json:"updated_at"`
}

type PersonSearchRequest = people.SearchInput
type UpdatePersonRequest = people.UpdateInput
type LinkPersonUnitRequest = people.LinkUnitInput
type LinkExternalIdentityRequest = people.ExternalIdentityInput
type MergePeopleRequest = people.MergeInput
type ResolveIdentityReviewRequest = people.ResolveReviewInput
