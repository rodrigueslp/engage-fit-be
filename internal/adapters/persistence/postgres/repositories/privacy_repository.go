package repositories

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type privacyCommunicationRow struct {
	Channel      string
	CampaignID   string
	Destination  string
	Status       string
	ErrorMessage string
	SentAt       *time.Time
	CreatedAt    time.Time
}

type contactConsentRow struct {
	Action         string
	Source         string
	Phone          string
	ConsentVersion string
	ConsentText    string
	CreatedAt      time.Time
}

func (r PrivacyGormRepository) ExportStudent(ctx context.Context, boxID, studentID domain.ID) (*domain.StudentPrivacyExport, error) {
	studentRepository := NewStudentGormRepository(r.db)
	student, err := studentRepository.FindByID(ctx, boxID, studentID)
	if err != nil {
		return nil, err
	}
	person, err := r.exportPerson(ctx, boxID, *student)
	if err != nil {
		return nil, err
	}
	checkins, err := NewCheckinGormRepository(r.db).ListByStudent(ctx, boxID, studentID)
	if err != nil {
		return nil, err
	}
	var progressModels []models.CampaignProgressModel
	if err := databaseForContext(r.db, ctx).Where("student_id = ?", stringID(studentID)).Order("updated_at DESC").Find(&progressModels).Error; err != nil {
		return nil, err
	}
	progress := make([]domain.CampaignProgress, 0, len(progressModels))
	for _, item := range progressModels {
		progress = append(progress, campaignProgressToDomain(item))
	}
	var rows []privacyCommunicationRow
	query := `
		SELECT 'whatsapp' AS channel, mr.message_campaign_id::text AS campaign_id, mr.phone AS destination, mr.status, mr.error_message, mr.sent_at, mr.created_at
		FROM message_recipients mr JOIN message_campaigns mc ON mc.id = mr.message_campaign_id
		WHERE mc.box_id = ? AND mr.student_id = ?
		UNION ALL
		SELECT 'email', er.email_campaign_id::text, er.email, er.status, er.error_message, er.sent_at, er.created_at
		FROM email_recipients er JOIN email_campaigns ec ON ec.id = er.email_campaign_id
		WHERE ec.box_id = ? AND er.student_id = ?
		UNION ALL
		SELECT 'workout_whatsapp', wr.workout_message_draft_id::text, wr.phone, wr.status, wr.error_message, wr.sent_at, wr.created_at
		FROM workout_message_recipients wr JOIN workout_message_drafts wd ON wd.id = wr.workout_message_draft_id
		WHERE wd.box_id = ? AND wr.student_id = ?
		ORDER BY created_at DESC`
	if err := databaseForContext(r.db, ctx).Raw(query, stringID(boxID), stringID(studentID), stringID(boxID), stringID(studentID), stringID(boxID), stringID(studentID)).Scan(&rows).Error; err != nil {
		return nil, err
	}
	communications := make([]domain.PrivacyCommunication, 0, len(rows))
	for _, row := range rows {
		communications = append(communications, domain.PrivacyCommunication{Channel: row.Channel, CampaignID: domain.ID(row.CampaignID), Destination: row.Destination, Status: row.Status, ErrorMessage: row.ErrorMessage, SentAt: row.SentAt, CreatedAt: row.CreatedAt})
	}
	interventions, err := NewRetentionGormRepository(r.db).ListInterventions(ctx, boxID, studentID)
	if err != nil {
		return nil, err
	}
	var consentRows []contactConsentRow
	if err := databaseForContext(r.db, ctx).Table("contact_consent_events").
		Select("action, source, phone, consent_version, consent_text, created_at").
		Where("box_id = ? AND student_id = ?", stringID(boxID), stringID(studentID)).
		Order("created_at DESC").
		Find(&consentRows).Error; err != nil {
		return nil, err
	}
	consents := make([]domain.ContactConsentEvent, 0, len(consentRows))
	for _, row := range consentRows {
		consents = append(consents, domain.ContactConsentEvent{
			Action: row.Action, Source: row.Source, Phone: row.Phone,
			ConsentVersion: row.ConsentVersion, ConsentText: row.ConsentText, CreatedAt: row.CreatedAt,
		})
	}
	attendance := []domain.AttendanceAdmission{}
	announcements := []domain.AnnouncementPrivacyView{}
	if student.PersonID != "" {
		var attendanceRows []attendanceAdmissionRow
		if err := databaseForContext(r.db, ctx).Raw(attendanceAdmissionSelect()+` WHERE d.unit_id = ? AND d.person_id = ? ORDER BY d.occurred_at`, stringID(boxID), stringID(student.PersonID)).Scan(&attendanceRows).Error; err != nil {
			return nil, err
		}
		for _, row := range attendanceRows {
			attendance = append(attendance, admissionFromRow(row))
		}
		if err := databaseForContext(r.db, ctx).Raw(`SELECT a.id AS announcement_id, a.title, a.priority, a.published_at, ar.first_viewed_at, ar.last_viewed_at, ar.view_count FROM announcement_recipients ar JOIN announcements a ON a.id = ar.announcement_id WHERE ar.unit_id = ? AND ar.person_id = ? ORDER BY a.publish_at DESC`, stringID(boxID), stringID(student.PersonID)).Scan(&announcements).Error; err != nil {
			return nil, err
		}
	}
	return &domain.StudentPrivacyExport{Student: *student, Person: person, Checkins: checkins, Progress: progress, Communications: communications, ContactConsents: consents, RetentionInterventions: interventions, Attendance: attendance, Announcements: announcements, ExportedAt: time.Now().UTC()}, nil
}

func (r PrivacyGormRepository) exportPerson(ctx context.Context, boxID domain.ID, student domain.Student) (*domain.PersonPrivacyExport, error) {
	if student.PersonID == "" {
		return nil, nil
	}
	db := databaseForContext(r.db, ctx)
	var person models.PersonModel
	err := db.Table("persons p").Select("p.*").
		Joins("JOIN boxes b ON b.organization_id = p.organization_id").
		Where("b.id = ? AND p.id = ?", stringID(boxID), stringID(student.PersonID)).Take(&person).Error
	if err != nil {
		return nil, err
	}
	plainCPF := ""
	if person.CPFEncrypted != nil {
		if r.cipher == nil {
			return nil, errors.New("person CPF cipher unavailable")
		}
		plainCPF, err = r.cipher.Decrypt(*person.CPFEncrypted, domain.PersonCPFAssociatedData(domainID(person.OrganizationID), domainID(person.ID)))
		if err != nil {
			return nil, err
		}
	}
	var addressModel models.PersonAddressModel
	var address *domain.PersonAddress
	err = db.Where("person_id = ? AND organization_id = ? AND address_type = 'home'", person.ID, person.OrganizationID).Take(&addressModel).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil {
		item := addressToDomain(addressModel)
		address = &item
	}
	relationships := []domain.PersonUnitRelationship{}
	var relationshipRows []struct {
		ID, UnitID, UnitName, RelationshipStatus, StatusReason string
		StartedAt                                              time.Time
		EndedAt                                                *time.Time
	}
	if err := db.Table("person_unit_relationships r").Select("r.id, r.unit_id, b.name AS unit_name, r.relationship_status, r.status_reason, r.started_at, r.ended_at").Joins("JOIN boxes b ON b.id = r.unit_id AND b.organization_id = r.organization_id").Where("r.person_id = ? AND r.organization_id = ?", person.ID, person.OrganizationID).Order("r.started_at").Scan(&relationshipRows).Error; err != nil {
		return nil, err
	}
	for _, row := range relationshipRows {
		relationships = append(relationships, domain.PersonUnitRelationship{ID: domainID(row.ID), UnitID: domainID(row.UnitID), UnitName: row.UnitName, RelationshipStatus: domain.PersonRelationshipStatus(row.RelationshipStatus), StartedAt: row.StartedAt, EndedAt: row.EndedAt, StatusReason: row.StatusReason})
	}
	identities := []domain.PersonExternalIdentity{}
	var identityModels []models.PersonExternalIdentityModel
	if err := db.Where("person_id = ? AND organization_id = ?", person.ID, person.OrganizationID).Find(&identityModels).Error; err != nil {
		return nil, err
	}
	for _, identity := range identityModels {
		identities = append(identities, externalIdentityToDomain(identity))
	}
	var athleteAccount *domain.PersonAthleteAccount
	var athlete models.PersonAthleteAccountModel
	if err := db.Where("person_id = ? AND organization_id = ?", person.ID, person.OrganizationID).Take(&athlete).Error; err == nil {
		athleteAccount = &domain.PersonAthleteAccount{AthleteAccountID: domainID(athlete.AthleteAccountID), LinkedAt: athlete.LinkedAt, LinkMethod: athlete.LinkMethod}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &domain.PersonPrivacyExport{ID: domainID(person.ID), Name: person.Name, PhoneE164: optionalString(person.PhoneE164), Email: optionalString(person.Email), CPF: plainCPF, ProfileStatus: domain.PersonProfileStatus(person.ProfileStatus), Address: address, Relationships: relationships, Identities: identities, AthleteAccount: athleteAccount}, nil
}

func (r PrivacyGormRepository) AnonymizeStudent(ctx context.Context, boxID, studentID, actorUserID domain.ID, reason string) error {
	return databaseForContext(r.db, ctx).Transaction(func(tx *gorm.DB) error {
		var student models.StudentModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("box_id = ? AND id = ?", stringID(boxID), stringID(studentID)).First(&student).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		suppressionID, err := newID()
		if err != nil {
			return err
		}
		suppression := map[string]any{"id": stringID(suppressionID), "box_id": stringID(boxID), "source": student.Source, "external_id_hash": privacyIdentityHash(student.ExternalID), "reason": reason, "created_at": now}
		if err := tx.Table("privacy_suppressions").Clauses(clause.OnConflict{DoNothing: true}).Create(suppression).Error; err != nil {
			return err
		}
		anonymousExternalID := "anonymized:" + stringID(studentID)
		if err := tx.Model(&models.StudentModel{}).Where("box_id = ? AND id = ?", stringID(boxID), stringID(studentID)).Updates(map[string]any{"name": "Aluno anonimizado", "email": "", "phone": "", "external_id": anonymousExternalID, "risk_status": string(domain.StudentRiskStatusPaused), "risk_last_message_at": nil, "contact_status": string(domain.ContactStatusOptedOut), "contact_status_source": "privacy_anonymization", "contact_status_updated_at": now, "anonymized_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.MessageRecipientModel{}).Where("student_id = ?", stringID(studentID)).Updates(map[string]any{"phone": "", "error_message": ""}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.EmailRecipientModel{}).Where("student_id = ?", stringID(studentID)).Updates(map[string]any{"email": "", "error_message": ""}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.WorkoutMessageRecipientModel{}).Where("student_id = ?", stringID(studentID)).Updates(map[string]any{"phone": "", "error_message": ""}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.RetentionInterventionModel{}).Where("box_id = ? AND student_id = ?", stringID(boxID), stringID(studentID)).Updates(map[string]any{"notes": nil}).Error; err != nil {
			return err
		}
		if err := tx.Table("contact_activation_requests").Where("box_id = ? AND student_id = ?", stringID(boxID), stringID(studentID)).Updates(map[string]any{"claimed_name": "", "phone": "", "updated_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Table("contact_consent_events").Where("box_id = ? AND student_id = ?", stringID(boxID), stringID(studentID)).Update("phone", "").Error; err != nil {
			return err
		}
		if student.PersonID != nil && *student.PersonID != "" {
			if err := anonymizeLinkedPerson(tx, boxID, studentID, actorUserID, domainID(*student.PersonID), now); err != nil {
				return err
			}
		}
		return PrivacyGormRepository{db: tx}.recordAudit(ctx, boxID, studentID, actorUserID, "anonymized", reason)
	})
}

func anonymizeLinkedPerson(tx *gorm.DB, boxID, studentID, actorUserID, personID domain.ID, now time.Time) error {
	var organizationID string
	if err := tx.Table("boxes").Select("organization_id").Where("id = ?", stringID(boxID)).Scan(&organizationID).Error; err != nil || organizationID == "" {
		if err != nil {
			return err
		}
		return gorm.ErrRecordNotFound
	}
	var otherActive int64
	if err := tx.Model(&models.StudentModel{}).Where("person_id = ? AND id <> ? AND anonymized_at IS NULL", stringID(personID), stringID(studentID)).Count(&otherActive).Error; err != nil {
		return err
	}
	action := "person.privacy_preserved_shared_profile"
	metadata := map[string]any{"student_id": studentID, "pii_removed": false, "shared_profile": true}
	if otherActive == 0 {
		var identities []models.PersonExternalIdentityModel
		if err := tx.Where("person_id = ? AND organization_id = ?", stringID(personID), organizationID).Find(&identities).Error; err != nil {
			return err
		}
		for _, identity := range identities {
			unitID := stringID(boxID)
			if identity.UnitID != nil {
				unitID = *identity.UnitID
			}
			suppression := map[string]any{"id": uuid.NewString(), "box_id": unitID, "source": identity.Provider, "external_id_hash": privacyIdentityHash(identity.ExternalID), "reason": "person_anonymization", "created_at": now}
			if err := tx.Table("privacy_suppressions").Clauses(clause.OnConflict{DoNothing: true}).Create(suppression).Error; err != nil {
				return err
			}
		}
		result := tx.Model(&models.PersonModel{}).Where("id = ? AND organization_id = ?", stringID(personID), organizationID).Updates(map[string]any{"name": "Pessoa anonimizada", "preferred_name": nil, "phone_e164": nil, "email": nil, "cpf_encrypted": nil, "cpf_lookup_hash": nil, "cpf_last2": nil, "profile_status": string(domain.PersonProfileAnonymized), "revision": gorm.Expr("revision + 1"), "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Where("person_id = ? AND organization_id = ?", stringID(personID), organizationID).Delete(&models.PersonAddressModel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("person_id = ? AND organization_id = ?", stringID(personID), organizationID).Delete(&models.PersonExternalIdentityModel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("person_id = ? AND organization_id = ?", stringID(personID), organizationID).Delete(&models.PersonAthleteAccountModel{}).Error; err != nil {
			return err
		}
		action = "person.anonymized"
		metadata = map[string]any{"student_id": studentID, "pii_removed": true, "address_removed": true}
	}
	eventID, err := newID()
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	studentIDValue := stringID(studentID)
	event := models.PeopleAuditEventModel{ID: stringID(eventID), OrganizationID: organizationID, UnitID: stringID(boxID), ActorUserID: stringID(actorUserID), Action: action, PersonID: stringID(personID), StudentID: &studentIDValue, Metadata: encoded, CreatedAt: now}
	return tx.Create(&event).Error
}

func addressToDomain(address models.PersonAddressModel) domain.PersonAddress {
	return domain.PersonAddress{ID: domainID(address.ID), PersonID: domainID(address.PersonID), OrganizationID: domainID(address.OrganizationID), AddressType: address.AddressType, PostalCode: optionalString(address.PostalCode), Street: optionalString(address.Street), Number: optionalString(address.Number), Complement: optionalString(address.Complement), District: optionalString(address.District), City: optionalString(address.City), State: optionalString(address.State), Country: address.Country, CreatedByUserID: domainID(address.CreatedByUserID), UpdatedByUserID: domainID(address.UpdatedByUserID), CreatedAt: address.CreatedAt, UpdatedAt: address.UpdatedAt}
}

func (r PrivacyGormRepository) IsIdentitySuppressed(ctx context.Context, boxID domain.ID, source domain.Source, externalID string) (bool, error) {
	var count int64
	err := databaseForContext(r.db, ctx).Table("privacy_suppressions").Where("box_id = ? AND source = ? AND external_id_hash = ?", stringID(boxID), string(source), privacyIdentityHash(externalID)).Count(&count).Error
	return count > 0, err
}

func (r PrivacyGormRepository) RecordAudit(ctx context.Context, boxID, studentID, actorUserID domain.ID, action, reason string) error {
	return r.recordAudit(ctx, boxID, studentID, actorUserID, action, reason)
}

func (r PrivacyGormRepository) recordAudit(ctx context.Context, boxID, studentID, actorUserID domain.ID, action, reason string) error {
	id, err := newID()
	if err != nil {
		return err
	}
	return databaseForContext(r.db, ctx).Table("privacy_audit_events").Create(map[string]any{"id": stringID(id), "box_id": stringID(boxID), "student_id": nullableID(studentID), "actor_user_id": nullableID(actorUserID), "action": action, "reason": reason, "created_at": time.Now().UTC()}).Error
}

func privacyIdentityHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func nullableID(value domain.ID) any {
	if value == "" {
		return nil
	}
	return stringID(value)
}
