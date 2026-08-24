package repositories

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/app/people"
	"boxengage/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type duplicateMatchRow struct {
	PersonID           string
	Name               string
	PhoneE164          string
	RelationshipStatus string
}

func (r PeopleGormRepository) ResolveEnrollmentSubject(ctx context.Context, organizationID, unitID, studentID domain.ID) (*domain.EnrollmentSubject, error) {
	var row struct {
		PersonID           *string
		DisplayName        string
		AnonymizedAt       *string
		ProfileStatus      *string
		RelationshipStatus *string
	}
	err := databaseForContext(r.db, ctx).Table("students s").
		Select("s.person_id, s.name AS display_name, s.anonymized_at::text AS anonymized_at, p.profile_status, r.relationship_status").
		Joins("JOIN boxes b ON b.id = s.box_id AND b.organization_id = ?", stringID(organizationID)).
		Joins("LEFT JOIN persons p ON p.id = s.person_id AND p.organization_id = b.organization_id").
		Joins("LEFT JOIN person_unit_relationships r ON r.person_id = p.id AND r.organization_id = p.organization_id AND r.unit_id = s.box_id").
		Where("s.id = ? AND s.box_id = ?", stringID(studentID), stringID(unitID)).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if row.PersonID == nil || *row.PersonID == "" {
		return nil, nil
	}
	subject := &domain.EnrollmentSubject{PersonID: domainID(*row.PersonID), DisplayName: row.DisplayName, Eligible: true}
	switch {
	case row.AnonymizedAt != nil || row.ProfileStatus == nil || *row.ProfileStatus == string(domain.PersonProfileAnonymized):
		subject.Eligible, subject.IneligibleReason = false, "person_anonymized"
	case *row.ProfileStatus != string(domain.PersonProfileActive):
		subject.Eligible, subject.IneligibleReason = false, "person_inactive"
	case row.RelationshipStatus == nil || *row.RelationshipStatus != string(domain.PersonRelationshipActive):
		subject.Eligible, subject.IneligibleReason = false, "relationship_inactive"
	}
	return subject, nil
}

func (r PeopleGormRepository) ResolveContractParty(ctx context.Context, organizationID, personID domain.ID) (*domain.ContractPartyResolution, error) {
	var row struct {
		ID            string
		Name          string
		ProfileStatus string
	}
	err := databaseForContext(r.db, ctx).Table("persons").
		Select("id, name, profile_status").
		Where("id = ? AND organization_id = ?", stringID(personID), stringID(organizationID)).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	eligible := row.ProfileStatus == string(domain.PersonProfileActive)
	return &domain.ContractPartyResolution{
		PersonID:      domainID(row.ID),
		DisplayName:   row.Name,
		ProfileStatus: domain.PersonProfileStatus(row.ProfileStatus),
		Eligible:      eligible,
	}, nil
}

func (r PeopleGormRepository) DuplicateCheck(ctx context.Context, organizationID, unitID domain.ID, cpfLookupHash, phoneE164 string) (*domain.PersonDuplicateCheck, error) {
	db := databaseForContext(r.db, ctx)
	result := &domain.PersonDuplicateCheck{PhoneMatches: []domain.PersonDuplicateMatch{}}
	if cpfLookupHash != "" {
		var row duplicateMatchRow
		err := db.Table("persons p").
			Select("p.id AS person_id, p.name, COALESCE(p.phone_e164, '') AS phone_e164, r.relationship_status").
			Joins("JOIN person_unit_relationships r ON r.person_id = p.id AND r.organization_id = p.organization_id").
			Where("p.organization_id = ? AND p.cpf_lookup_hash = ? AND p.profile_status <> 'anonymized' AND r.unit_id = ?", stringID(organizationID), cpfLookupHash, stringID(unitID)).
			Take(&row).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if err == nil {
			match := duplicateMatchToDomain(row)
			result.CPFMatch = &match
		}
	}
	if phoneE164 != "" {
		var rows []duplicateMatchRow
		if err := db.Table("persons p").
			Select("p.id AS person_id, p.name, COALESCE(p.phone_e164, '') AS phone_e164, r.relationship_status").
			Joins("JOIN person_unit_relationships r ON r.person_id = p.id AND r.organization_id = p.organization_id").
			Where("p.organization_id = ? AND p.phone_e164 = ? AND p.profile_status <> 'anonymized' AND r.unit_id = ?", stringID(organizationID), phoneE164, stringID(unitID)).
			Order("p.created_at ASC").Limit(5).Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			result.PhoneMatches = append(result.PhoneMatches, duplicateMatchToDomain(row))
		}
	}
	return result, nil
}

func (r PeopleGormRepository) Create(ctx context.Context, command domain.CreatePersonCommand) (*domain.PersonCreationResult, bool, error) {
	var result *domain.PersonCreationResult
	var replayed bool
	err := databaseForContext(r.db, ctx).Transaction(func(tx *gorm.DB) error {
		lockKey := fmt.Sprintf("%s:%s:%s", command.OrganizationID, command.UnitID, command.IdempotencyKey)
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", lockKey).Error; err != nil {
			return err
		}
		var previous models.PersonCreationRequestModel
		err := tx.Where("organization_id = ? AND unit_id = ? AND idempotency_key = ?", stringID(command.OrganizationID), stringID(command.UnitID), command.IdempotencyKey).Take(&previous).Error
		if err == nil {
			if !hmacEqual(previous.RequestFingerprint, command.Fingerprint) {
				return people.ErrIdempotencyKeyReused
			}
			loaded, loadErr := loadCreationResult(tx, previous.PersonID, previous.StudentID)
			if loadErr != nil {
				return loadErr
			}
			result, replayed = loaded, true
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var cpfCount int64
		if err := tx.Model(&models.PersonModel{}).
			Where("organization_id = ? AND cpf_lookup_hash = ? AND profile_status <> 'anonymized'", stringID(command.OrganizationID), command.Person.CPFLookupHash).
			Count(&cpfCount).Error; err != nil {
			return err
		}
		if cpfCount > 0 {
			return people.ErrCPFConflict
		}

		duplicates, err := (PeopleGormRepository{db: tx}).DuplicateCheck(ctx, command.OrganizationID, command.UnitID, "", command.Person.PhoneE164)
		if err != nil {
			return err
		}
		if command.Review == nil {
			if len(duplicates.PhoneMatches) > 0 {
				return people.ErrPossibleDuplicate
			}
		} else {
			validReview := false
			for _, match := range duplicates.PhoneMatches {
				if command.Review.Decision == "create_separate" && match.PersonID == command.Review.MatchedPersonID {
					validReview = true
					break
				}
			}
			if !validReview {
				return people.ErrPossibleDuplicate
			}
		}

		if err := tx.Create(personToModel(command.Person)).Error; err != nil {
			return mapPeopleWriteError(err)
		}
		if command.Address != nil {
			if err := tx.Create(addressToModel(*command.Address)).Error; err != nil {
				return err
			}
		}
		relationship := models.PersonUnitRelationshipModel{ID: stringID(command.Person.ID), OrganizationID: stringID(command.OrganizationID), PersonID: stringID(command.Person.ID), UnitID: stringID(command.UnitID), RelationshipStatus: string(domain.PersonRelationshipActive), StartedAt: command.CreatedAt, CreatedAt: command.CreatedAt, UpdatedAt: command.CreatedAt}
		if err := tx.Create(&relationship).Error; err != nil {
			return err
		}
		studentModel := studentToModel(command.Student)
		if err := tx.Create(&studentModel).Error; err != nil {
			return err
		}
		if err := createPeopleAuditEvents(tx, command); err != nil {
			return err
		}
		requestID, err := newID()
		if err != nil {
			return err
		}
		request := models.PersonCreationRequestModel{ID: stringID(requestID), OrganizationID: stringID(command.OrganizationID), UnitID: stringID(command.UnitID), ActorUserID: stringID(command.ActorUserID), IdempotencyKey: command.IdempotencyKey, RequestFingerprint: command.Fingerprint, PersonID: stringID(command.Person.ID), StudentID: stringID(command.Student.ID), CreatedAt: command.CreatedAt}
		if err := tx.Create(&request).Error; err != nil {
			return mapPeopleWriteError(err)
		}
		result = creationResult(command.Person, command.Student)
		return nil
	})
	return result, replayed, err
}

func (r PeopleGormRepository) CountStudentsWithoutPerson(ctx context.Context) (int64, error) {
	var count int64
	err := databaseForContext(r.db, ctx).Model(&models.StudentModel{}).Where("person_id IS NULL").Count(&count).Error
	return count, err
}

func loadCreationResult(db *gorm.DB, personID, studentID string) (*domain.PersonCreationResult, error) {
	var person models.PersonModel
	if err := db.Where("id = ?", personID).Take(&person).Error; err != nil {
		return nil, err
	}
	var student models.StudentModel
	if err := db.Where("id = ?", studentID).Take(&student).Error; err != nil {
		return nil, err
	}
	return creationResult(personToDomain(person), studentToDomain(student)), nil
}

func creationResult(person domain.Person, student domain.Student) *domain.PersonCreationResult {
	return &domain.PersonCreationResult{PersonID: person.ID, StudentID: student.ID, Name: person.Name, PhoneE164: person.PhoneE164, Email: person.Email, CPFLast2: person.CPFLast2, RelationshipStatus: domain.PersonRelationshipActive, ContactStatus: student.ContactStatus, CreatedAt: person.CreatedAt}
}

func createPeopleAuditEvents(tx *gorm.DB, command domain.CreatePersonCommand) error {
	createdMetadata := map[string]any{"profile_status": "active", "has_email": command.Person.Email != "", "has_phone": true, "has_cpf": true, "phone_duplicate_reviewed": command.Review != nil}
	if command.Review != nil {
		createdMetadata["matched_person_id"] = command.Review.MatchedPersonID
	}
	actions := []struct {
		action   string
		metadata map[string]any
	}{
		{"person.created", createdMetadata},
		{"person.unit_linked", map[string]any{"relationship_status": "active", "unit_id": command.UnitID}},
	}
	if command.Address != nil {
		actions = append(actions, struct {
			action   string
			metadata map[string]any
		}{"person.address_added", map[string]any{"address_type": "home", "fields_present": addressFieldsPresent(*command.Address)}})
	}
	studentID := stringID(command.Student.ID)
	for _, item := range actions {
		id, err := newID()
		if err != nil {
			return err
		}
		metadata, err := json.Marshal(item.metadata)
		if err != nil {
			return err
		}
		event := models.PeopleAuditEventModel{ID: stringID(id), OrganizationID: stringID(command.OrganizationID), UnitID: stringID(command.UnitID), ActorUserID: stringID(command.ActorUserID), Action: item.action, PersonID: stringID(command.Person.ID), StudentID: &studentID, Metadata: metadata, CreatedAt: command.CreatedAt}
		if err := tx.Create(&event).Error; err != nil {
			return err
		}
	}
	return nil
}

func addressFieldsPresent(address domain.PersonAddress) []string {
	fields := make([]string, 0, 7)
	values := []struct{ name, value string }{{"postal_code", address.PostalCode}, {"street", address.Street}, {"number", address.Number}, {"complement", address.Complement}, {"district", address.District}, {"city", address.City}, {"state", address.State}}
	for _, item := range values {
		if item.value != "" {
			fields = append(fields, item.name)
		}
	}
	return fields
}

func personToModel(person domain.Person) *models.PersonModel {
	return &models.PersonModel{ID: stringID(person.ID), OrganizationID: stringID(person.OrganizationID), Name: person.Name, PhoneE164: peopleNullableString(person.PhoneE164), Email: peopleNullableString(person.Email), CPFEncrypted: peopleNullableString(person.CPFEncrypted), CPFLookupHash: peopleNullableString(person.CPFLookupHash), CPFLast2: peopleNullableString(person.CPFLast2), ProfileStatus: string(person.ProfileStatus), CreatedAt: person.CreatedAt, UpdatedAt: person.UpdatedAt}
}

func personToDomain(person models.PersonModel) domain.Person {
	return domain.Person{ID: domainID(person.ID), OrganizationID: domainID(person.OrganizationID), Name: person.Name, PhoneE164: optionalString(person.PhoneE164), Email: optionalString(person.Email), CPFEncrypted: optionalString(person.CPFEncrypted), CPFLookupHash: optionalString(person.CPFLookupHash), CPFLast2: optionalString(person.CPFLast2), ProfileStatus: domain.PersonProfileStatus(person.ProfileStatus), CreatedAt: person.CreatedAt, UpdatedAt: person.UpdatedAt}
}

func addressToModel(address domain.PersonAddress) *models.PersonAddressModel {
	return &models.PersonAddressModel{ID: stringID(address.ID), PersonID: stringID(address.PersonID), OrganizationID: stringID(address.OrganizationID), AddressType: address.AddressType, PostalCode: peopleNullableString(address.PostalCode), Street: peopleNullableString(address.Street), Number: peopleNullableString(address.Number), Complement: peopleNullableString(address.Complement), District: peopleNullableString(address.District), City: peopleNullableString(address.City), State: peopleNullableString(address.State), Country: address.Country, CreatedByUserID: stringID(address.CreatedByUserID), UpdatedByUserID: stringID(address.UpdatedByUserID), CreatedAt: address.CreatedAt, UpdatedAt: address.UpdatedAt}
}

func duplicateMatchToDomain(row duplicateMatchRow) domain.PersonDuplicateMatch {
	return domain.PersonDuplicateMatch{PersonID: domainID(row.PersonID), Name: row.Name, PhoneE164: row.PhoneE164, RelationshipStatus: domain.PersonRelationshipStatus(row.RelationshipStatus)}
}

func peopleNullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func hmacEqual(left, right string) bool {
	return len(left) == len(right) && subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func mapPeopleWriteError(err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23505" {
		switch postgresError.ConstraintName {
		case "idx_persons_organization_cpf_lookup":
			return people.ErrCPFConflict
		case "person_creation_requests_scope_key":
			return people.ErrIdempotencyKeyReused
		}
	}
	return err
}
