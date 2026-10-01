package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r HealthFormGormRepository) Create(ctx context.Context, form *domain.HealthFormSubmission, answers domain.PARQAnswers) error {
	plain, err := json.Marshal(answers)
	if err != nil {
		return err
	}
	encrypted, err := r.cipher.Encrypt(string(plain), domain.HealthFormAnswersAAD(form.OrganizationID, form.ID))
	if err != nil {
		return err
	}
	row := healthFormToModel(*form, encrypted)
	err = databaseForContext(r.db, ctx).Create(&row).Error
	if err != nil && isUniqueViolation(err) {
		return portrepo.ErrHealthFormAlreadySubmitted
	}
	return err
}

func (r HealthFormGormRepository) ListForPerson(ctx context.Context, organizationID, unitID, personID domain.ID) ([]domain.HealthFormSubmission, error) {
	var rows []models.HealthFormSubmissionModel
	if err := databaseForContext(r.db, ctx).Where("organization_id = ? AND unit_id = ? AND person_id = ?", stringID(organizationID), stringID(unitID), stringID(personID)).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.HealthFormSubmission, 0, len(rows))
	for _, row := range rows {
		result = append(result, healthFormFromModel(row, nil))
	}
	return result, nil
}

func (r HealthFormGormRepository) Find(ctx context.Context, organizationID, unitID, formID domain.ID, includeAnswers bool, actorUserID domain.ID) (*domain.HealthFormSubmission, error) {
	db := databaseForContext(r.db, ctx)
	var result *domain.HealthFormSubmission
	err := db.Transaction(func(tx *gorm.DB) error {
		var row models.HealthFormSubmissionModel
		if err := tx.Where("organization_id = ? AND unit_id = ? AND id = ?", stringID(organizationID), stringID(unitID), stringID(formID)).Take(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return portrepo.ErrHealthFormNotFound
			}
			return err
		}
		var answers *domain.PARQAnswers
		if includeAnswers {
			plain, err := r.cipher.Decrypt(row.AnswersEncrypted, domain.HealthFormAnswersAAD(organizationID, formID))
			if err != nil {
				return err
			}
			var parsed domain.PARQAnswers
			if err := json.Unmarshal([]byte(plain), &parsed); err != nil {
				return err
			}
			answers = &parsed
			event := models.HealthFormAccessEventModel{ID: uuid.NewString(), OrganizationID: row.OrganizationID, UnitID: row.UnitID, SubmissionID: row.ID, ActorUserID: stringID(actorUserID), Action: "answers_viewed", CreatedAt: time.Now().UTC()}
			if err := tx.Create(&event).Error; err != nil {
				return err
			}
		}
		value := healthFormFromModel(row, answers)
		result = &value
		return nil
	})
	return result, err
}

func (r HealthFormGormRepository) RevokeConsent(ctx context.Context, organizationID, unitID, formID, actorUserID domain.ID, at time.Time) (*domain.HealthFormSubmission, error) {
	db := databaseForContext(r.db, ctx)
	var result *domain.HealthFormSubmission
	err := db.Transaction(func(tx *gorm.DB) error {
		var row models.HealthFormSubmissionModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id = ? AND unit_id = ? AND id = ?", stringID(organizationID), stringID(unitID), stringID(formID)).Take(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return portrepo.ErrHealthFormNotFound
			}
			return err
		}
		if row.Status == string(domain.HealthFormStatusSubmitted) {
			row.Status = string(domain.HealthFormStatusRevoked)
			row.ConsentRevokedAt = &at
			row.UpdatedAt = at
			if err := tx.Save(&row).Error; err != nil {
				return err
			}
			event := models.HealthFormAccessEventModel{ID: uuid.NewString(), OrganizationID: row.OrganizationID, UnitID: row.UnitID, SubmissionID: row.ID, ActorUserID: stringID(actorUserID), Action: "consent_revoked", CreatedAt: at}
			if err := tx.Create(&event).Error; err != nil {
				return err
			}
		}
		value := healthFormFromModel(row, nil)
		result = &value
		return nil
	})
	return result, err
}

func (r HealthFormGormRepository) ListPrivacy(ctx context.Context, organizationID, personID domain.ID) ([]domain.HealthFormPrivacyExport, error) {
	var rows []models.HealthFormSubmissionModel
	if err := databaseForContext(r.db, ctx).Where("organization_id = ? AND person_id = ?", stringID(organizationID), stringID(personID)).Order("created_at").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.HealthFormPrivacyExport, 0, len(rows))
	for _, row := range rows {
		plain, err := r.cipher.Decrypt(row.AnswersEncrypted, domain.HealthFormAnswersAAD(organizationID, domainID(row.ID)))
		if err != nil {
			return nil, err
		}
		var answers domain.PARQAnswers
		if err := json.Unmarshal([]byte(plain), &answers); err != nil {
			return nil, err
		}
		result = append(result, domain.HealthFormPrivacyExport{Submission: healthFormFromModel(row, nil), Answers: answers})
	}
	return result, nil
}

func (r HealthFormGormRepository) PurgeExpired(ctx context.Context, now time.Time, apply bool) (int64, error) {
	db := databaseForContext(r.db, ctx)
	query := db.Model(&models.HealthFormSubmissionModel{}).Where("retention_due_at <= ?", now)
	if !apply {
		var count int64
		return count, query.Count(&count).Error
	}
	result := query.Delete(&models.HealthFormSubmissionModel{})
	return result.RowsAffected, result.Error
}

func (r HealthFormGormRepository) CreateSession(ctx context.Context, session *domain.HealthFormSession) error {
	db := databaseForContext(r.db, ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.HealthFormSessionModel{}).
			Where("organization_id = ? AND enrollment_id = ? AND schema_version = ? AND status = 'active'", stringID(session.OrganizationID), stringID(session.EnrollmentID), session.SchemaVersion).
			Updates(map[string]any{"status": "superseded", "updated_at": session.CreatedAt}).Error; err != nil {
			return err
		}
		row := healthFormSessionToModel(*session)
		return tx.Create(&row).Error
	})
}

func (r HealthFormGormRepository) FindSessionByTokenHash(ctx context.Context, tokenHash string) (*domain.HealthFormSession, error) {
	var row models.HealthFormSessionModel
	if err := databaseForContext(r.db, ctx).Where("token_hash = ?", tokenHash).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, portrepo.ErrHealthFormSessionInvalid
		}
		return nil, err
	}
	value := healthFormSessionFromModel(row)
	return &value, nil
}

func (r HealthFormGormRepository) CreateFromSession(ctx context.Context, sessionID domain.ID, form *domain.HealthFormSubmission, answers domain.PARQAnswers, at time.Time) error {
	plain, err := json.Marshal(answers)
	if err != nil {
		return err
	}
	encrypted, err := r.cipher.Encrypt(string(plain), domain.HealthFormAnswersAAD(form.OrganizationID, form.ID))
	if err != nil {
		return err
	}
	db := databaseForContext(r.db, ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		var session models.HealthFormSessionModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", stringID(sessionID)).Take(&session).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return portrepo.ErrHealthFormSessionInvalid
			}
			return err
		}
		if session.Status != "active" || !at.Before(session.ExpiresAt) || session.OrganizationID != stringID(form.OrganizationID) || session.EnrollmentID != stringID(form.EnrollmentID) {
			return portrepo.ErrHealthFormSessionInvalid
		}
		row := healthFormToModel(*form, encrypted)
		if err := tx.Create(&row).Error; err != nil {
			if isUniqueViolation(err) {
				return portrepo.ErrHealthFormAlreadySubmitted
			}
			return err
		}
		return tx.Model(&session).Updates(map[string]any{"status": "submitted", "submitted_at": at, "updated_at": at}).Error
	})
}

func healthFormToModel(form domain.HealthFormSubmission, encrypted string) models.HealthFormSubmissionModel {
	return models.HealthFormSubmissionModel{ID: stringID(form.ID), OrganizationID: stringID(form.OrganizationID), UnitID: stringID(form.UnitID), PersonID: stringID(form.PersonID), EnrollmentID: stringID(form.EnrollmentID), SchemaVersion: form.SchemaVersion, Status: string(form.Status), ConsentVersion: form.ConsentVersion, ConsentText: form.ConsentText, ConsentAcceptedAt: form.ConsentAcceptedAt, ConsentRevokedAt: form.ConsentRevokedAt, RequiresReview: form.RequiresReview, AnswersEncrypted: encrypted, RetentionDueAt: form.RetentionDueAt, CreatedByUserID: stringID(form.CreatedByUserID), CreatedAt: form.CreatedAt, UpdatedAt: form.UpdatedAt}
}

func healthFormFromModel(row models.HealthFormSubmissionModel, answers *domain.PARQAnswers) domain.HealthFormSubmission {
	return domain.HealthFormSubmission{ID: domainID(row.ID), OrganizationID: domainID(row.OrganizationID), UnitID: domainID(row.UnitID), PersonID: domainID(row.PersonID), EnrollmentID: domainID(row.EnrollmentID), SchemaVersion: row.SchemaVersion, Status: domain.HealthFormStatus(row.Status), ConsentVersion: row.ConsentVersion, ConsentText: row.ConsentText, ConsentAcceptedAt: row.ConsentAcceptedAt, ConsentRevokedAt: row.ConsentRevokedAt, RequiresReview: row.RequiresReview, Answers: answers, RetentionDueAt: row.RetentionDueAt, CreatedByUserID: domainID(row.CreatedByUserID), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func healthFormSessionToModel(session domain.HealthFormSession) models.HealthFormSessionModel {
	return models.HealthFormSessionModel{ID: stringID(session.ID), OrganizationID: stringID(session.OrganizationID), UnitID: stringID(session.UnitID), PersonID: stringID(session.PersonID), EnrollmentID: stringID(session.EnrollmentID), SchemaVersion: session.SchemaVersion, TokenHash: session.TokenHash, Status: session.Status, ExpiresAt: session.ExpiresAt, SubmittedAt: session.SubmittedAt, CreatedByUserID: stringID(session.CreatedByUserID), CreatedAt: session.CreatedAt, UpdatedAt: session.UpdatedAt}
}

func healthFormSessionFromModel(row models.HealthFormSessionModel) domain.HealthFormSession {
	return domain.HealthFormSession{ID: domainID(row.ID), OrganizationID: domainID(row.OrganizationID), UnitID: domainID(row.UnitID), PersonID: domainID(row.PersonID), EnrollmentID: domainID(row.EnrollmentID), SchemaVersion: row.SchemaVersion, TokenHash: row.TokenHash, Status: row.Status, ExpiresAt: row.ExpiresAt, SubmittedAt: row.SubmittedAt, CreatedByUserID: domainID(row.CreatedByUserID), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
