package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/app/people"
	"boxengage/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type personListRow struct {
	PersonID           string
	PrimaryStudentID   *string
	Name               string
	PreferredName      *string
	PhoneE164          *string
	Email              *string
	CPFLast2           *string
	ProfileStatus      string
	RelationshipStatus string
	SourcesCSV         string
	Revision           int
}

func (r PeopleGormRepository) List(ctx context.Context, organizationID, unitID domain.ID, criteria domain.PersonSearchCriteria) ([]domain.PersonSummary, error) {
	query := databaseForContext(r.db, ctx).Table("persons p").
		Select(`p.id AS person_id,
            min(s.id::text) FILTER (WHERE s.box_id = r.unit_id) AS primary_student_id,
            p.name, p.preferred_name, p.phone_e164, p.email, p.cpf_last2,
            p.profile_status, r.relationship_status,
            COALESCE(string_agg(DISTINCT s.source, ',' ORDER BY s.source), '') AS sources_csv,
            p.revision`).
		Joins("JOIN person_unit_relationships r ON r.person_id = p.id AND r.organization_id = p.organization_id AND r.unit_id = ?", stringID(unitID)).
		Joins("LEFT JOIN students s ON s.person_id = p.id AND s.box_id = r.unit_id AND s.anonymized_at IS NULL").
		Where("p.organization_id = ? AND p.merged_into_person_id IS NULL", stringID(organizationID))
	if criteria.CPFLookupHash != "" && criteria.PhoneE164 != "" {
		query = query.Where("(p.cpf_lookup_hash = ? OR p.phone_e164 = ?)", criteria.CPFLookupHash, criteria.PhoneE164)
	} else if criteria.CPFLookupHash != "" {
		query = query.Where("p.cpf_lookup_hash = ?", criteria.CPFLookupHash)
	} else if criteria.PhoneE164 != "" {
		query = query.Where("p.phone_e164 = ?", criteria.PhoneE164)
	} else if criteria.NameQuery != "" {
		query = query.Where("p.name ILIKE ? OR COALESCE(p.preferred_name, '') ILIKE ?", "%"+criteria.NameQuery+"%", "%"+criteria.NameQuery+"%")
	}
	var rows []personListRow
	if err := query.Group("p.id, r.relationship_status").Order("lower(COALESCE(p.preferred_name, p.name)), p.id").Limit(200).Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.PersonSummary, 0, len(rows))
	for _, row := range rows {
		summary := domain.PersonSummary{ID: domainID(row.PersonID), Name: row.Name, PreferredName: optionalString(row.PreferredName), PhoneMasked: people.MaskPhone(optionalString(row.PhoneE164)), EmailMasked: people.MaskEmail(optionalString(row.Email)), ProfileStatus: domain.PersonProfileStatus(row.ProfileStatus), RelationshipStatus: domain.PersonRelationshipStatus(row.RelationshipStatus), Sources: []domain.Source{}, Revision: row.Revision}
		if row.PrimaryStudentID != nil {
			summary.PrimaryStudentID = domainID(*row.PrimaryStudentID)
		}
		if row.CPFLast2 != nil {
			summary.CPFMasked = "***.***.***-" + *row.CPFLast2
		}
		for _, source := range strings.Split(row.SourcesCSV, ",") {
			if source != "" {
				summary.Sources = append(summary.Sources, domain.Source(source))
			}
		}
		result = append(result, summary)
	}
	return result, nil
}

func (r PeopleGormRepository) Get(ctx context.Context, organizationID, unitID, personID domain.ID) (*domain.PersonProfile, error) {
	return r.loadProfile(databaseForContext(r.db, ctx), organizationID, unitID, personID)
}

func (r PeopleGormRepository) loadProfile(db *gorm.DB, organizationID, unitID, personID domain.ID) (*domain.PersonProfile, error) {
	var person models.PersonModel
	err := db.Table("persons p").Select("p.*").
		Joins("JOIN person_unit_relationships current_relationship ON current_relationship.person_id = p.id AND current_relationship.organization_id = p.organization_id AND current_relationship.unit_id = ?", stringID(unitID)).
		Where("p.id = ? AND p.organization_id = ? AND p.merged_into_person_id IS NULL", stringID(personID), stringID(organizationID)).Take(&person).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, people.ErrPersonNotFound
	}
	if err != nil {
		return nil, err
	}
	profile := &domain.PersonProfile{Person: personToDomain(person), Relationships: []domain.PersonUnitRelationship{}, Profiles: []domain.PersonSourceProfile{}, Identities: []domain.PersonExternalIdentity{}, History: []domain.PeopleAuditEvent{}}
	if person.CPFLast2 != nil {
		profile.CPFMasked = "***.***.***-" + *person.CPFLast2
	}
	var address models.PersonAddressModel
	if err := db.Where("person_id = ? AND organization_id = ? AND address_type = 'home'", person.ID, person.OrganizationID).Take(&address).Error; err == nil {
		value := profileAddressToDomain(address)
		profile.Address = &value
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	var relationshipRows []struct {
		ID, UnitID, UnitName, RelationshipStatus, StatusReason string
		StartedAt                                              time.Time
		EndedAt                                                *time.Time
	}
	if err := db.Table("person_unit_relationships r").Select("r.id, r.unit_id, b.name AS unit_name, r.relationship_status, r.status_reason, r.started_at, r.ended_at").Joins("JOIN boxes b ON b.id = r.unit_id AND b.organization_id = r.organization_id").Where("r.person_id = ? AND r.organization_id = ?", person.ID, person.OrganizationID).Order("r.started_at").Scan(&relationshipRows).Error; err != nil {
		return nil, err
	}
	for _, row := range relationshipRows {
		rel := domain.PersonUnitRelationship{ID: domainID(row.ID), UnitID: domainID(row.UnitID), UnitName: row.UnitName, RelationshipStatus: domain.PersonRelationshipStatus(row.RelationshipStatus), StartedAt: row.StartedAt, StatusReason: row.StatusReason}
		if row.EndedAt != nil {
			value := *row.EndedAt
			rel.EndedAt = &value
		}
		profile.Relationships = append(profile.Relationships, rel)
	}
	var students []models.StudentModel
	if err := db.Where("person_id = ?", person.ID).Order("created_at").Find(&students).Error; err != nil {
		return nil, err
	}
	for _, student := range students {
		profile.Profiles = append(profile.Profiles, domain.PersonSourceProfile{StudentID: domainID(student.ID), UnitID: domainID(student.BoxID), Source: domain.Source(student.Source), ContactStatus: domain.ContactStatus(student.ContactStatus), AnonymizedAt: student.AnonymizedAt})
	}
	var identities []models.PersonExternalIdentityModel
	if err := db.Where("person_id = ? AND organization_id = ?", person.ID, person.OrganizationID).Order("provider, created_at").Find(&identities).Error; err != nil {
		return nil, err
	}
	for _, identity := range identities {
		profile.Identities = append(profile.Identities, externalIdentityToDomain(identity))
	}
	var athlete models.PersonAthleteAccountModel
	if err := db.Where("person_id = ? AND organization_id = ?", person.ID, person.OrganizationID).Take(&athlete).Error; err == nil {
		profile.AthleteAccount = &domain.PersonAthleteAccount{AthleteAccountID: domainID(athlete.AthleteAccountID), LinkedAt: athlete.LinkedAt, LinkMethod: athlete.LinkMethod}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	var events []models.PeopleAuditEventModel
	if err := db.Where("person_id = ? AND organization_id = ?", person.ID, person.OrganizationID).Order("created_at DESC").Limit(50).Find(&events).Error; err != nil {
		return nil, err
	}
	for _, event := range events {
		metadata := map[string]any{}
		_ = json.Unmarshal(event.Metadata, &metadata)
		profile.History = append(profile.History, domain.PeopleAuditEvent{ID: domainID(event.ID), UnitID: domainID(event.UnitID), Action: event.Action, Metadata: metadata, CreatedAt: event.CreatedAt})
	}
	return profile, nil
}

func (r PeopleGormRepository) Update(ctx context.Context, command domain.UpdatePersonCommand) (*domain.PersonProfile, error) {
	err := databaseForContext(r.db, ctx).Transaction(func(tx *gorm.DB) error {
		var current models.PersonModel
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND organization_id = ? AND merged_into_person_id IS NULL", stringID(command.PersonID), stringID(command.OrganizationID)).Take(&current).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return people.ErrPersonNotFound
		}
		if err != nil {
			return err
		}
		var relationshipCount int64
		if err := tx.Model(&models.PersonUnitRelationshipModel{}).Where("person_id = ? AND organization_id = ? AND unit_id = ?", current.ID, current.OrganizationID, stringID(command.UnitID)).Count(&relationshipCount).Error; err != nil {
			return err
		}
		if relationshipCount == 0 {
			return people.ErrPersonNotFound
		}
		if current.ProfileStatus == string(domain.PersonProfileAnonymized) {
			return people.ErrPersonNotFound
		}
		if current.Revision != command.ExpectedRevision {
			return people.ErrPersonRevisionConflict
		}
		updates := map[string]any{"revision": gorm.Expr("revision + 1"), "updated_at": command.UpdatedAt}
		changed := []string{}
		if command.Name != nil {
			updates["name"], changed = *command.Name, append(changed, "name")
		}
		if command.PreferredName != nil {
			updates["preferred_name"], changed = nullableSQLString(*command.PreferredName), append(changed, "preferred_name")
		}
		if command.PhoneE164 != nil {
			updates["phone_e164"], changed = nullableSQLString(*command.PhoneE164), append(changed, "phone")
		}
		if command.Email != nil {
			updates["email"], changed = nullableSQLString(*command.Email), append(changed, "email")
		}
		if command.CPFEncrypted != nil {
			updates["cpf_encrypted"], updates["cpf_lookup_hash"], updates["cpf_last2"], changed = *command.CPFEncrypted, *command.CPFLookupHash, *command.CPFLast2, append(changed, "cpf")
		}
		if command.ProfileStatus != nil {
			updates["profile_status"], changed = string(*command.ProfileStatus), append(changed, "profile_status")
		}
		result := tx.Model(&models.PersonModel{}).Where("id = ? AND organization_id = ? AND revision = ?", current.ID, current.OrganizationID, command.ExpectedRevision).Updates(updates)
		if result.Error != nil {
			return mapPeopleWriteError(result.Error)
		}
		if result.RowsAffected != 1 {
			return people.ErrPersonRevisionConflict
		}
		studentUpdates := map[string]any{}
		if command.Name != nil {
			studentUpdates["name"] = *command.Name
		}
		if command.PhoneE164 != nil {
			studentUpdates["phone"] = strings.TrimPrefix(*command.PhoneE164, "+")
		}
		if command.Email != nil {
			studentUpdates["email"] = *command.Email
		}
		if len(studentUpdates) > 0 {
			studentUpdates["updated_at"] = command.UpdatedAt
			if err := tx.Model(&models.StudentModel{}).Where("person_id = ?", current.ID).Updates(studentUpdates).Error; err != nil {
				return err
			}
		}
		if command.ClearAddress {
			if err := tx.Where("person_id = ? AND organization_id = ? AND address_type = 'home'", current.ID, current.OrganizationID).Delete(&models.PersonAddressModel{}).Error; err != nil {
				return err
			}
			changed = append(changed, "address")
		} else if command.Address != nil {
			model := addressToModel(*command.Address)
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "person_id"}, {Name: "address_type"}}, DoUpdates: clause.Assignments(map[string]any{"postal_code": model.PostalCode, "street": model.Street, "number": model.Number, "complement": model.Complement, "district": model.District, "city": model.City, "state": model.State, "country": model.Country, "updated_by_user_id": model.UpdatedByUserID, "updated_at": model.UpdatedAt})}).Create(model).Error; err != nil {
				return err
			}
			changed = append(changed, "address")
		}
		sort.Strings(changed)
		metadata := map[string]any{"fields": changed, "revision": command.ExpectedRevision + 1, "phone_duplicate_reviewed": command.PhoneReview != nil}
		if command.PhoneReview != nil {
			metadata["matched_person_id"] = command.PhoneReview.MatchedPersonID
		}
		return appendPeopleAudit(tx, command.OrganizationID, command.UnitID, command.ActorUserID, "person.updated", command.PersonID, "", metadata, command.UpdatedAt)
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, command.OrganizationID, command.UnitID, command.PersonID)
}

func (r PeopleGormRepository) LinkUnit(ctx context.Context, command domain.LinkPersonUnitCommand) (*domain.PersonProfile, error) {
	err := databaseForContext(r.db, ctx).Transaction(func(tx *gorm.DB) error {
		var person models.PersonModel
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND organization_id = ? AND profile_status <> 'anonymized' AND merged_into_person_id IS NULL", stringID(command.PersonID), stringID(command.OrganizationID)).Take(&person).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return people.ErrPersonNotFound
		}
		if err != nil {
			return err
		}
		var existing int64
		if err := tx.Model(&models.PersonUnitRelationshipModel{}).Where("person_id = ? AND unit_id = ?", person.ID, stringID(command.UnitID)).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			return people.ErrPersonAlreadyLinked
		}
		command.Student.Name, command.Student.Email, command.Student.Phone = person.Name, optionalString(person.Email), strings.TrimPrefix(optionalString(person.PhoneE164), "+")
		relationship := models.PersonUnitRelationshipModel{ID: uuid.NewString(), OrganizationID: person.OrganizationID, PersonID: person.ID, UnitID: stringID(command.UnitID), RelationshipStatus: string(domain.PersonRelationshipActive), StartedAt: command.CreatedAt, StatusReason: command.Reason, CreatedAt: command.CreatedAt, UpdatedAt: command.CreatedAt}
		if err := tx.Create(&relationship).Error; err != nil {
			return err
		}
		student := studentToModel(command.Student)
		if err := tx.Create(&student).Error; err != nil {
			return err
		}
		return appendPeopleAudit(tx, command.OrganizationID, command.UnitID, command.ActorUserID, "person.unit_linked", command.PersonID, command.Student.ID, map[string]any{"relationship_status": "active", "unit_id": command.UnitID}, command.CreatedAt)
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, command.OrganizationID, command.UnitID, command.PersonID)
}

func (r PeopleGormRepository) LinkExternalIdentity(ctx context.Context, command domain.LinkExternalIdentityCommand) (*domain.PersonProfile, error) {
	db := databaseForContext(r.db, ctx)
	var current models.PersonExternalIdentityModel
	err := db.Where("organization_id = ? AND provider = ? AND external_id = ?", stringID(command.OrganizationID), command.Provider, command.ExternalID).Take(&current).Error
	if err == nil {
		if current.PersonID == stringID(command.PersonID) {
			return r.Get(ctx, command.OrganizationID, command.UnitID, command.PersonID)
		}
		review := models.PersonIdentityReviewModel{ID: uuid.NewString(), OrganizationID: stringID(command.OrganizationID), UnitID: stringID(command.UnitID), TargetPersonID: stringID(command.PersonID), SourcePersonID: current.PersonID, Reason: "external_identity_conflict", Provider: &command.Provider, Status: "pending", CreatedAt: command.CreatedAt}
		if createErr := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&review).Error; createErr != nil {
			return nil, createErr
		}
		return nil, people.ErrExternalIdentityConflict
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		var relationshipCount int64
		if err := tx.Model(&models.PersonUnitRelationshipModel{}).Where("person_id = ? AND organization_id = ? AND unit_id = ?", stringID(command.PersonID), stringID(command.OrganizationID), stringID(command.UnitID)).Count(&relationshipCount).Error; err != nil {
			return err
		}
		if relationshipCount == 0 {
			return people.ErrPersonNotFound
		}
		if command.StudentID != "" {
			var student models.StudentModel
			if err := tx.Where("id = ? AND box_id = ?", stringID(command.StudentID), stringID(command.UnitID)).Take(&student).Error; err != nil {
				return people.ErrPersonNotFound
			}
			if student.PersonID == nil || *student.PersonID != stringID(command.PersonID) {
				return people.ErrMergeNotAllowed
			}
		}
		actor := stringID(command.ActorUserID)
		model := models.PersonExternalIdentityModel{ID: uuid.NewString(), OrganizationID: stringID(command.OrganizationID), PersonID: stringID(command.PersonID), UnitID: peopleNullableID(command.UnitID), Provider: command.Provider, ExternalID: command.ExternalID, CreatedByUserID: &actor, CreatedAt: command.CreatedAt, UpdatedAt: command.CreatedAt}
		if command.Verified {
			value := command.CreatedAt
			model.VerifiedAt = &value
		}
		if err := tx.Create(&model).Error; err != nil {
			return mapIdentityWriteError(err)
		}
		return appendPeopleAudit(tx, command.OrganizationID, command.UnitID, command.ActorUserID, "person.external_identity_linked", command.PersonID, command.StudentID, map[string]any{"provider": command.Provider, "verified": command.Verified}, command.CreatedAt)
	})
	if err != nil {
		if errors.Is(err, people.ErrExternalIdentityConflict) {
			var existing models.PersonExternalIdentityModel
			if findErr := db.Where("organization_id = ? AND provider = ? AND external_id = ?", stringID(command.OrganizationID), command.Provider, command.ExternalID).Take(&existing).Error; findErr == nil {
				if existing.PersonID == stringID(command.PersonID) {
					return r.Get(ctx, command.OrganizationID, command.UnitID, command.PersonID)
				}
				review := models.PersonIdentityReviewModel{ID: uuid.NewString(), OrganizationID: stringID(command.OrganizationID), UnitID: stringID(command.UnitID), TargetPersonID: stringID(command.PersonID), SourcePersonID: existing.PersonID, Reason: "external_identity_conflict", Provider: &command.Provider, Status: "pending", CreatedAt: command.CreatedAt}
				if createErr := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&review).Error; createErr != nil {
					return nil, createErr
				}
			}
		}
		return nil, err
	}
	return r.Get(ctx, command.OrganizationID, command.UnitID, command.PersonID)
}

func (r PeopleGormRepository) ListIdentityReviews(ctx context.Context, organizationID, unitID domain.ID) ([]domain.PersonIdentityReview, error) {
	var rows []models.PersonIdentityReviewModel
	if err := databaseForContext(r.db, ctx).Where("organization_id = ? AND unit_id = ? AND status = 'pending'", stringID(organizationID), stringID(unitID)).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.PersonIdentityReview, 0, len(rows))
	for _, row := range rows {
		result = append(result, reviewToDomain(row))
	}
	return result, nil
}

func (r PeopleGormRepository) ReconcileAthleteAccount(ctx context.Context, unitID, studentID, athleteAccountID domain.ID, linkedAt time.Time) error {
	return databaseForContext(r.db, ctx).Transaction(func(tx *gorm.DB) error {
		var scope struct{ OrganizationID, PersonID string }
		err := tx.Table("students s").Select("b.organization_id, s.person_id").Joins("JOIN boxes b ON b.id = s.box_id").Where("s.id = ? AND s.box_id = ? AND s.person_id IS NOT NULL", stringID(studentID), stringID(unitID)).Take(&scope).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		var personIDs []string
		if err := tx.Table("athlete_student_links asl").Distinct("s.person_id").
			Joins("JOIN athlete_box_memberships abm ON abm.id = asl.membership_id").
			Joins("JOIN boxes b ON b.id = abm.box_id").
			Joins("JOIN students s ON s.id = asl.student_id AND s.person_id IS NOT NULL").
			Where("abm.athlete_account_id = ? AND b.organization_id = ?", stringID(athleteAccountID), scope.OrganizationID).
			Order("s.person_id").Pluck("s.person_id", &personIDs).Error; err != nil {
			return err
		}
		if len(personIDs) == 1 {
			model := models.PersonAthleteAccountModel{ID: uuid.NewString(), OrganizationID: scope.OrganizationID, PersonID: personIDs[0], AthleteAccountID: stringID(athleteAccountID), LinkMethod: "confirmed_student_links", LinkedAt: linkedAt, CreatedAt: linkedAt}
			return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model).Error
		}
		if len(personIDs) > 1 {
			for _, sourceID := range personIDs[1:] {
				review := models.PersonIdentityReviewModel{ID: uuid.NewString(), OrganizationID: scope.OrganizationID, UnitID: stringID(unitID), TargetPersonID: personIDs[0], SourcePersonID: sourceID, Reason: "athlete_account_conflict", Status: "pending", CreatedAt: linkedAt}
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&review).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (r PeopleGormRepository) ResolveIdentityReview(ctx context.Context, command domain.MergePeopleCommand, resolution string) (*domain.PersonIdentityReview, *domain.PersonProfile, error) {
	var review domain.PersonIdentityReview
	err := databaseForContext(r.db, ctx).Transaction(func(tx *gorm.DB) error {
		var model models.PersonIdentityReviewModel
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND organization_id = ? AND unit_id = ? AND status = 'pending'", stringID(command.ReviewID), stringID(command.OrganizationID), stringID(command.UnitID)).Take(&model).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return people.ErrIdentityReviewNotFound
		}
		if err != nil {
			return err
		}
		command.TargetPersonID, command.SourcePersonID = domainID(model.TargetPersonID), domainID(model.SourcePersonID)
		resolved := "dismissed"
		if resolution == "merge" {
			if err := mergePeopleTx(tx, command); err != nil {
				return err
			}
			resolved = "merged"
		} else if err := appendPeopleAudit(tx, command.OrganizationID, command.UnitID, command.ActorUserID, "person.identity_review_dismissed", command.TargetPersonID, "", map[string]any{"review_id": command.ReviewID, "reason": model.Reason}, command.CreatedAt); err != nil {
			return err
		}
		actor, value := stringID(command.ActorUserID), resolved
		if err := tx.Model(&model).Updates(map[string]any{"status": "resolved", "resolution": &value, "resolved_at": command.CreatedAt, "resolved_by_user_id": &actor}).Error; err != nil {
			return err
		}
		model.Status, model.Resolution, model.ResolvedAt, model.ResolvedByUserID = "resolved", &value, &command.CreatedAt, &actor
		review = reviewToDomain(model)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	if resolution != "merge" {
		return &review, nil, nil
	}
	profile, err := r.Get(ctx, command.OrganizationID, command.UnitID, review.TargetPersonID)
	return &review, profile, err
}

func (r PeopleGormRepository) Merge(ctx context.Context, command domain.MergePeopleCommand) (*domain.PersonProfile, error) {
	err := databaseForContext(r.db, ctx).Transaction(func(tx *gorm.DB) error { return mergePeopleTx(tx, command) })
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, command.OrganizationID, command.UnitID, command.TargetPersonID)
}

func mergePeopleTx(tx *gorm.DB, command domain.MergePeopleCommand) error {
	ids := []string{stringID(command.TargetPersonID), stringID(command.SourcePersonID)}
	var persons []models.PersonModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id = ? AND id IN ? AND merged_into_person_id IS NULL AND profile_status <> 'anonymized'", stringID(command.OrganizationID), ids).Order("id").Find(&persons).Error; err != nil {
		return err
	}
	if len(persons) != 2 {
		return people.ErrPersonNotFound
	}
	var targetRelationship int64
	if err := tx.Model(&models.PersonUnitRelationshipModel{}).Where("organization_id = ? AND unit_id = ? AND person_id = ?", stringID(command.OrganizationID), stringID(command.UnitID), stringID(command.TargetPersonID)).Count(&targetRelationship).Error; err != nil {
		return err
	}
	if targetRelationship == 0 {
		return people.ErrPersonNotFound
	}
	for _, table := range []string{"enrollments", "contract_instances"} {
		var count int64
		if err := tx.Table(table).Where("organization_id = ? AND person_id = ?", stringID(command.OrganizationID), stringID(command.SourcePersonID)).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return people.ErrMergeNotAllowed
		}
	}
	var sourceAddressCount, targetAddressCount int64
	if err := tx.Model(&models.PersonAddressModel{}).Where("person_id = ?", stringID(command.SourcePersonID)).Count(&sourceAddressCount).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.PersonAddressModel{}).Where("person_id = ?", stringID(command.TargetPersonID)).Count(&targetAddressCount).Error; err != nil {
		return err
	}
	if sourceAddressCount > 0 && targetAddressCount > 0 {
		return people.ErrMergeNotAllowed
	}
	var sourceAthlete, targetAthlete int64
	if err := tx.Model(&models.PersonAthleteAccountModel{}).Where("person_id = ? AND organization_id = ?", stringID(command.SourcePersonID), stringID(command.OrganizationID)).Count(&sourceAthlete).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.PersonAthleteAccountModel{}).Where("person_id = ? AND organization_id = ?", stringID(command.TargetPersonID), stringID(command.OrganizationID)).Count(&targetAthlete).Error; err != nil {
		return err
	}
	if sourceAthlete > 0 && targetAthlete > 0 {
		return people.ErrMergeNotAllowed
	}
	if err := tx.Exec(`DELETE FROM person_unit_relationships source
        USING person_unit_relationships target
        WHERE source.organization_id = ? AND source.person_id = ?
          AND target.organization_id = source.organization_id
          AND target.person_id = ? AND target.unit_id = source.unit_id`, stringID(command.OrganizationID), stringID(command.SourcePersonID), stringID(command.TargetPersonID)).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.PersonUnitRelationshipModel{}).Where("organization_id = ? AND person_id = ?", stringID(command.OrganizationID), stringID(command.SourcePersonID)).Update("person_id", stringID(command.TargetPersonID)).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.StudentModel{}).Where("person_id = ?", stringID(command.SourcePersonID)).Updates(map[string]any{"person_id": stringID(command.TargetPersonID), "updated_at": command.CreatedAt}).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.PersonExternalIdentityModel{}).Where("organization_id = ? AND person_id = ?", stringID(command.OrganizationID), stringID(command.SourcePersonID)).Updates(map[string]any{"person_id": stringID(command.TargetPersonID), "updated_at": command.CreatedAt}).Error; err != nil {
		return err
	}
	if sourceAthlete > 0 {
		if err := tx.Model(&models.PersonAthleteAccountModel{}).Where("organization_id = ? AND person_id = ?", stringID(command.OrganizationID), stringID(command.SourcePersonID)).Update("person_id", stringID(command.TargetPersonID)).Error; err != nil {
			return err
		}
	}
	if sourceAddressCount > 0 {
		if err := tx.Model(&models.PersonAddressModel{}).Where("organization_id = ? AND person_id = ?", stringID(command.OrganizationID), stringID(command.SourcePersonID)).Update("person_id", stringID(command.TargetPersonID)).Error; err != nil {
			return err
		}
	}
	if err := tx.Model(&models.PersonModel{}).Where("id = ? AND organization_id = ?", stringID(command.SourcePersonID), stringID(command.OrganizationID)).Updates(map[string]any{"profile_status": "archived", "merged_into_person_id": stringID(command.TargetPersonID), "revision": gorm.Expr("revision + 1"), "updated_at": command.CreatedAt}).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.PersonModel{}).Where("id = ? AND organization_id = ?", stringID(command.TargetPersonID), stringID(command.OrganizationID)).Updates(map[string]any{"revision": gorm.Expr("revision + 1"), "updated_at": command.CreatedAt}).Error; err != nil {
		return err
	}
	return appendPeopleAudit(tx, command.OrganizationID, command.UnitID, command.ActorUserID, "person.merged", command.TargetPersonID, "", map[string]any{"source_person_id": command.SourcePersonID, "review_id": command.ReviewID}, command.CreatedAt)
}

func appendPeopleAudit(tx *gorm.DB, organizationID, unitID, actorUserID domain.ID, action string, personID, studentID domain.ID, metadata map[string]any, createdAt time.Time) error {
	payload, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	var student *string
	if studentID != "" {
		value := stringID(studentID)
		student = &value
	}
	return tx.Create(&models.PeopleAuditEventModel{ID: uuid.NewString(), OrganizationID: stringID(organizationID), UnitID: stringID(unitID), ActorUserID: stringID(actorUserID), Action: action, PersonID: stringID(personID), StudentID: student, Metadata: payload, CreatedAt: createdAt}).Error
}

func profileAddressToDomain(address models.PersonAddressModel) domain.PersonAddress {
	return domain.PersonAddress{ID: domainID(address.ID), PersonID: domainID(address.PersonID), OrganizationID: domainID(address.OrganizationID), AddressType: address.AddressType, PostalCode: optionalString(address.PostalCode), Street: optionalString(address.Street), Number: optionalString(address.Number), Complement: optionalString(address.Complement), District: optionalString(address.District), City: optionalString(address.City), State: optionalString(address.State), Country: address.Country, CreatedByUserID: domainID(address.CreatedByUserID), UpdatedByUserID: domainID(address.UpdatedByUserID), CreatedAt: address.CreatedAt, UpdatedAt: address.UpdatedAt}
}

func externalIdentityToDomain(identity models.PersonExternalIdentityModel) domain.PersonExternalIdentity {
	return domain.PersonExternalIdentity{ID: domainID(identity.ID), OrganizationID: domainID(identity.OrganizationID), PersonID: domainID(identity.PersonID), UnitID: optionalID(identity.UnitID), Provider: identity.Provider, ExternalID: identity.ExternalID, VerifiedAt: identity.VerifiedAt, CreatedAt: identity.CreatedAt, UpdatedAt: identity.UpdatedAt}
}

func reviewToDomain(review models.PersonIdentityReviewModel) domain.PersonIdentityReview {
	result := domain.PersonIdentityReview{ID: domainID(review.ID), OrganizationID: domainID(review.OrganizationID), UnitID: domainID(review.UnitID), TargetPersonID: domainID(review.TargetPersonID), SourcePersonID: domainID(review.SourcePersonID), Reason: review.Reason, Provider: optionalString(review.Provider), Status: review.Status, Resolution: optionalString(review.Resolution), CreatedAt: review.CreatedAt, ResolvedAt: review.ResolvedAt}
	return result
}

func nullableSQLString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func mapIdentityWriteError(err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23505" && postgresError.ConstraintName == "person_external_identities_provider_external_key" {
		return people.ErrExternalIdentityConflict
	}
	return err
}
