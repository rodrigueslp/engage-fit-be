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
	PreferredName  string
	PhoneE164      string
	Email          string
	CPFEncrypted   string
	CPFLookupHash  string
	CPFLast2       string
	ProfileStatus  PersonProfileStatus
	MergedIntoID   ID
	Revision       int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type PersonUnitRelationship struct {
	ID                 ID                       `json:"id"`
	UnitID             ID                       `json:"unit_id"`
	UnitName           string                   `json:"unit_name"`
	RelationshipStatus PersonRelationshipStatus `json:"relationship_status"`
	StartedAt          time.Time                `json:"started_at"`
	EndedAt            *time.Time               `json:"ended_at,omitempty"`
	StatusReason       string                   `json:"status_reason,omitempty"`
}

type PersonSourceProfile struct {
	StudentID     ID            `json:"student_id"`
	UnitID        ID            `json:"unit_id"`
	Source        Source        `json:"source"`
	ContactStatus ContactStatus `json:"contact_status"`
	AnonymizedAt  *time.Time    `json:"anonymized_at,omitempty"`
}

type PersonExternalIdentity struct {
	ID             ID         `json:"id"`
	OrganizationID ID         `json:"-"`
	PersonID       ID         `json:"person_id"`
	UnitID         ID         `json:"unit_id,omitempty"`
	Provider       string     `json:"provider"`
	ExternalID     string     `json:"external_id"`
	VerifiedAt     *time.Time `json:"verified_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type PersonAthleteAccount struct {
	AthleteAccountID ID        `json:"athlete_account_id"`
	LinkedAt         time.Time `json:"linked_at"`
	LinkMethod       string    `json:"link_method"`
}

type PeopleAuditEvent struct {
	ID        ID             `json:"id"`
	UnitID    ID             `json:"unit_id"`
	Action    string         `json:"action"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
}

type PersonSummary struct {
	ID                 ID                       `json:"id"`
	PrimaryStudentID   ID                       `json:"primary_student_id,omitempty"`
	Name               string                   `json:"name"`
	PreferredName      string                   `json:"preferred_name,omitempty"`
	PhoneMasked        string                   `json:"phone_masked,omitempty"`
	EmailMasked        string                   `json:"email_masked,omitempty"`
	CPFMasked          string                   `json:"cpf_masked,omitempty"`
	ProfileStatus      PersonProfileStatus      `json:"profile_status"`
	RelationshipStatus PersonRelationshipStatus `json:"relationship_status"`
	Sources            []Source                 `json:"sources"`
	Revision           int                      `json:"revision"`
}

type PersonProfile struct {
	Person         Person                   `json:"-"`
	CPFMasked      string                   `json:"cpf_masked,omitempty"`
	Address        *PersonAddress           `json:"address,omitempty"`
	Relationships  []PersonUnitRelationship `json:"relationships"`
	Profiles       []PersonSourceProfile    `json:"source_profiles"`
	Identities     []PersonExternalIdentity `json:"external_identities"`
	AthleteAccount *PersonAthleteAccount    `json:"athlete_account,omitempty"`
	History        []PeopleAuditEvent       `json:"history"`
}

type PersonSearchCriteria struct {
	NameQuery     string
	PhoneE164     string
	CPFLookupHash string
}

type UpdatePersonCommand struct {
	OrganizationID   ID
	UnitID           ID
	PersonID         ID
	ActorUserID      ID
	ExpectedRevision int
	Name             *string
	PreferredName    *string
	PhoneE164        *string
	Email            *string
	CPFEncrypted     *string
	CPFLookupHash    *string
	CPFLast2         *string
	ProfileStatus    *PersonProfileStatus
	PhoneReview      *PersonPhoneDuplicateReview
	Address          *PersonAddress
	ClearAddress     bool
	UpdatedAt        time.Time
}

type LinkPersonUnitCommand struct {
	OrganizationID ID
	UnitID         ID
	PersonID       ID
	ActorUserID    ID
	Student        Student
	Reason         string
	CreatedAt      time.Time
}

type LinkExternalIdentityCommand struct {
	OrganizationID ID
	UnitID         ID
	PersonID       ID
	ActorUserID    ID
	Provider       string
	ExternalID     string
	StudentID      ID
	Verified       bool
	CreatedAt      time.Time
}

type PersonIdentityReview struct {
	ID             ID         `json:"id"`
	OrganizationID ID         `json:"-"`
	UnitID         ID         `json:"unit_id"`
	TargetPersonID ID         `json:"target_person_id"`
	SourcePersonID ID         `json:"source_person_id"`
	Reason         string     `json:"reason"`
	Provider       string     `json:"provider,omitempty"`
	Status         string     `json:"status"`
	Resolution     string     `json:"resolution,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
}

type MergePeopleCommand struct {
	OrganizationID ID
	UnitID         ID
	TargetPersonID ID
	SourcePersonID ID
	ActorUserID    ID
	ReviewID       ID
	Reason         string
	CreatedAt      time.Time
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
	ID             ID
	Name           string
	PhoneE164      string
	Email          string
	CPF            string
	ProfileStatus  PersonProfileStatus
	Address        *PersonAddress
	Relationships  []PersonUnitRelationship
	Identities     []PersonExternalIdentity
	AthleteAccount *PersonAthleteAccount
}

func PersonCPFAssociatedData(organizationID, personID ID) string {
	return "person-cpf:" + string(organizationID) + ":" + string(personID)
}
