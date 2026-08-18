package dto

import "boxengage/backend/internal/app/people"

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
