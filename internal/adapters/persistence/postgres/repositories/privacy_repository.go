package repositories

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
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

func (r PrivacyGormRepository) ExportStudent(ctx context.Context, boxID, studentID domain.ID) (*domain.StudentPrivacyExport, error) {
	studentRepository := NewStudentGormRepository(r.db)
	student, err := studentRepository.FindByID(ctx, boxID, studentID)
	if err != nil {
		return nil, err
	}
	checkins, err := NewCheckinGormRepository(r.db).ListByStudent(ctx, boxID, studentID)
	if err != nil {
		return nil, err
	}
	var progressModels []models.CampaignProgressModel
	if err := r.db.WithContext(ctx).Where("student_id = ?", stringID(studentID)).Order("updated_at DESC").Find(&progressModels).Error; err != nil {
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
	if err := r.db.WithContext(ctx).Raw(query, stringID(boxID), stringID(studentID), stringID(boxID), stringID(studentID), stringID(boxID), stringID(studentID)).Scan(&rows).Error; err != nil {
		return nil, err
	}
	communications := make([]domain.PrivacyCommunication, 0, len(rows))
	for _, row := range rows {
		communications = append(communications, domain.PrivacyCommunication{Channel: row.Channel, CampaignID: domain.ID(row.CampaignID), Destination: row.Destination, Status: row.Status, ErrorMessage: row.ErrorMessage, SentAt: row.SentAt, CreatedAt: row.CreatedAt})
	}
	return &domain.StudentPrivacyExport{Student: *student, Checkins: checkins, Progress: progress, Communications: communications, ExportedAt: time.Now().UTC()}, nil
}

func (r PrivacyGormRepository) AnonymizeStudent(ctx context.Context, boxID, studentID, actorUserID domain.ID, reason string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
		return PrivacyGormRepository{db: tx}.recordAudit(ctx, boxID, studentID, actorUserID, "anonymized", reason)
	})
}

func (r PrivacyGormRepository) IsIdentitySuppressed(ctx context.Context, boxID domain.ID, source domain.Source, externalID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("privacy_suppressions").Where("box_id = ? AND source = ? AND external_id_hash = ?", stringID(boxID), string(source), privacyIdentityHash(externalID)).Count(&count).Error
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
	return r.db.WithContext(ctx).Table("privacy_audit_events").Create(map[string]any{"id": stringID(id), "box_id": stringID(boxID), "student_id": nullableID(studentID), "actor_user_id": nullableID(actorUserID), "action": action, "reason": reason, "created_at": time.Now().UTC()}).Error
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
