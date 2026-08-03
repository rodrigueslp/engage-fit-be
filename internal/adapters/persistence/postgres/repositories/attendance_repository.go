package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r AttendanceGormRepository) SaveSelfCheckinSession(ctx context.Context, session *domain.SelfCheckinSession) error {
	if session.ID == "" {
		session.ID = domain.ID(uuid.NewString())
	}
	model := models.SelfCheckinSessionModel{
		ID: stringID(session.ID), BoxID: stringID(session.BoxID),
		CreatedByUserID: stringPointer(stringID(session.CreatedByUserID)), TokenHash: session.TokenHash,
		ExpiresAt: session.ExpiresAt, CreatedAt: session.CreatedAt,
	}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r AttendanceGormRepository) FindValidSelfCheckinSession(ctx context.Context, tokenHash string, now time.Time) (*domain.SelfCheckinSession, string, error) {
	type row struct {
		models.SelfCheckinSessionModel
		BoxName string
	}
	var value row
	err := r.db.WithContext(ctx).Table("self_checkin_sessions").
		Select("self_checkin_sessions.*, boxes.name AS box_name").
		Joins("JOIN boxes ON boxes.id = self_checkin_sessions.box_id").
		Where("self_checkin_sessions.token_hash = ? AND self_checkin_sessions.expires_at > ?", tokenHash, now).
		Where("boxes.status = ? AND boxes.billing_access_blocked = ?", string(domain.BoxStatusActive), false).
		Take(&value).Error
	if err != nil {
		return nil, "", err
	}
	createdBy := ""
	if value.CreatedByUserID != nil {
		createdBy = *value.CreatedByUserID
	}
	return &domain.SelfCheckinSession{
		ID: domainID(value.ID), BoxID: domainID(value.BoxID), CreatedByUserID: domainID(createdBy),
		TokenHash: value.TokenHash, ExpiresAt: value.ExpiresAt, CreatedAt: value.CreatedAt,
	}, value.BoxName, nil
}

func (r AttendanceGormRepository) FindActiveBoxMembersByPhone(ctx context.Context, boxID domain.ID, phone string) ([]domain.Student, error) {
	var items []models.StudentModel
	err := r.db.WithContext(ctx).
		Where("box_id = ? AND source = ? AND contact_status = ? AND anonymized_at IS NULL", stringID(boxID), string(domain.SourceBoxMember), string(domain.ContactStatusOptedIn)).
		Where("REGEXP_REPLACE(phone, '[^0-9]', '', 'g') = ?", phone).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	students := make([]domain.Student, 0, len(items))
	for _, item := range items {
		students = append(students, studentToDomain(item))
	}
	return students, nil
}

func (r AttendanceGormRepository) SaveBoxMemberCheckin(ctx context.Context, checkin *domain.Checkin) (bool, error) {
	if checkin.ID == "" {
		checkin.ID = domain.ID(uuid.NewString())
	}
	model := checkinToModel(*checkin)
	result := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Table("students").
			Where("box_id = ? AND id = ? AND source = ? AND anonymized_at IS NULL", stringID(checkin.BoxID), stringID(checkin.StudentID), string(domain.SourceBoxMember)).
			Count(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			return gorm.ErrRecordNotFound
		}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model).Error
	})
	if result != nil {
		return false, result
	}
	var count int64
	if err := r.db.WithContext(ctx).Table("checkins").Where("id = ?", stringID(checkin.ID)).Count(&count).Error; err != nil {
		return false, err
	}
	return count == 1, nil
}
