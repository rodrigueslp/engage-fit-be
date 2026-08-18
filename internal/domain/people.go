package domain

import "time"

type PersonProfileStatus string
type PersonRelationshipStatus string

const (
	PersonProfileActive     PersonProfileStatus = "active"
	PersonProfileArchived   PersonProfileStatus = "archived"
	PersonProfileAnonymized PersonProfileStatus = "anonymized"

	PersonRelationshipActive   PersonRelationshipStatus = "active"
	PersonRelationshipInactive PersonRelationshipStatus = "inactive"
	PersonRelationshipBlocked  PersonRelationshipStatus = "blocked"
)

type Person struct {
	ID             ID
	OrganizationID ID
	Name           string
	PhoneE164      string
	Email          string
	CPFEncrypted   string
	CPFLookupHash  string
	CPFLast2       string
	ProfileStatus  PersonProfileStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type PersonAddress struct {
	ID              ID
	PersonID        ID
	OrganizationID  ID
	AddressType     string
	PostalCode      string
	Street          string
	Number          string
	Complement      string
	District        string
	City            string
	State           string
	Country         string
	CreatedByUserID ID
	UpdatedByUserID ID
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type PersonDuplicateMatch struct {
	PersonID           ID
	Name               string
	PhoneE164          string
	RelationshipStatus PersonRelationshipStatus
}

type PersonDuplicateCheck struct {
	CPFMatch     *PersonDuplicateMatch
	PhoneMatches []PersonDuplicateMatch
}

type PersonPhoneDuplicateReview struct {
	Decision        string
	MatchedPersonID ID
	Reason          string
}

type CreatePersonCommand struct {
	OrganizationID ID
	UnitID         ID
	ActorUserID    ID
	IdempotencyKey string
	Fingerprint    string
	Person         Person
	Address        *PersonAddress
	Review         *PersonPhoneDuplicateReview
	Student        Student
	CreatedAt      time.Time
}

type PersonCreationResult struct {
	PersonID           ID
	StudentID          ID
	Name               string
	PhoneE164          string
	Email              string
	CPFLast2           string
	RelationshipStatus PersonRelationshipStatus
	ContactStatus      ContactStatus
	CreatedAt          time.Time
}

type PersonPrivacyExport struct {
	ID            ID
	Name          string
	PhoneE164     string
	Email         string
	CPF           string
	ProfileStatus PersonProfileStatus
	Address       *PersonAddress
}

func PersonCPFAssociatedData(organizationID, personID ID) string {
	return "person-cpf:" + string(organizationID) + ":" + string(personID)
}
