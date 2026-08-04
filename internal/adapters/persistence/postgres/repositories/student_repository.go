package repositories

import (
	"context"
	"time"

	portrepo "boxengage/backend/internal/ports/repositories"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
)

func (r StudentGormRepository) FindByID(ctx context.Context, boxID, id domain.ID) (*domain.Student, error) {
	var model models.StudentModel
	if err := databaseForContext(r.db, ctx).Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).First(&model).Error; err != nil {
		return nil, err
	}

	student := studentToDomain(model)
	return &student, nil
}

func (r StudentGormRepository) FindByExternalID(ctx context.Context, boxID domain.ID, source domain.Source, externalID string) (*domain.Student, error) {
	var model models.StudentModel
	if err := databaseForContext(r.db, ctx).
		Where("box_id = ? AND source = ? AND external_id = ?", stringID(boxID), string(source), externalID).
		First(&model).Error; err != nil {
		return nil, err
	}

	student := studentToDomain(model)
	return &student, nil
}

func (r StudentGormRepository) List(ctx context.Context, boxID domain.ID, filters portrepo.StudentFilters) ([]domain.Student, error) {
	query := databaseForContext(r.db, ctx).Model(&models.StudentModel{}).Where("box_id = ?", stringID(boxID))
	if filters.ContactableOnly {
		query = query.Where("contact_status = ? AND anonymized_at IS NULL", string(domain.ContactStatusOptedIn))
	}

	if filters.Source != nil {
		query = query.Where("source = ?", string(*filters.Source))
	}
	if filters.Search != "" {
		search := "%" + filters.Search + "%"
		query = query.Where("(name ILIKE ? OR email ILIKE ? OR phone ILIKE ?)", search, search, search)
	}
	if filters.CampaignID != nil {
		query = query.Joins("JOIN campaign_progresses ON campaign_progresses.student_id = students.id AND campaign_progresses.campaign_id = ?", stringID(*filters.CampaignID))
		if filters.Achieved != nil {
			query = query.Where("campaign_progresses.achieved = ?", *filters.Achieved)
		}
		if filters.NearGoal != nil {
			query = query.Where("campaign_progresses.near_goal = ?", *filters.NearGoal)
		}
	}

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Page > 1 && filters.Limit > 0 {
		query = query.Offset((filters.Page - 1) * filters.Limit)
	}

	var modelsList []models.StudentModel
	if err := query.Order("name ASC").Find(&modelsList).Error; err != nil {
		return nil, err
	}

	students := make([]domain.Student, 0, len(modelsList))
	for _, model := range modelsList {
		students = append(students, studentToDomain(model))
	}
	return students, nil
}

func (r StudentGormRepository) UpdateContactPreference(ctx context.Context, boxID, id domain.ID, status domain.ContactStatus, source string, updatedAt time.Time) error {
	return databaseForContext(r.db, ctx).
		Model(&models.StudentModel{}).
		Where("box_id = ? AND id = ? AND anonymized_at IS NULL", stringID(boxID), stringID(id)).
		Updates(map[string]any{
			"contact_status":            string(status),
			"contact_status_source":     source,
			"contact_status_updated_at": updatedAt,
			"updated_at":                updatedAt,
		}).Error
}

func (r StudentGormRepository) UpdateMembershipStart(ctx context.Context, boxID, id domain.ID, startedAt time.Time, source string, updatedAt time.Time) error {
	return databaseForContext(r.db, ctx).
		Model(&models.StudentModel{}).
		Where("box_id = ? AND id = ? AND anonymized_at IS NULL", stringID(boxID), stringID(id)).
		Updates(map[string]any{
			"membership_started_at":     startedAt,
			"membership_started_source": source,
			"updated_at":                updatedAt,
		}).Error
}

func (r StudentGormRepository) SetEarlierInferredMembershipStart(ctx context.Context, boxID, id domain.ID, startedAt time.Time, updatedAt time.Time) error {
	return databaseForContext(r.db, ctx).
		Model(&models.StudentModel{}).
		Where("box_id = ? AND id = ? AND anonymized_at IS NULL", stringID(boxID), stringID(id)).
		Where("membership_started_at IS NULL OR (membership_started_source = ? AND membership_started_at > ?)", "first_checkin_inferred", startedAt).
		Updates(map[string]any{
			"membership_started_at":     startedAt,
			"membership_started_source": "first_checkin_inferred",
			"updated_at":                updatedAt,
		}).Error
}

func (r StudentGormRepository) Save(ctx context.Context, student *domain.Student) error {
	if err := ensureID(&student.ID); err != nil {
		return err
	}
	if student.RiskStatus == "" {
		student.RiskStatus = domain.StudentRiskStatusActive
	}

	model := studentToModel(*student)
	return databaseForContext(r.db, ctx).Save(&model).Error
}

func (r StudentGormRepository) UpdateRiskStatus(ctx context.Context, boxID, id domain.ID, status domain.StudentRiskStatus) error {
	return databaseForContext(r.db, ctx).
		Model(&models.StudentModel{}).
		Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).
		Update("risk_status", string(status)).Error
}

func (r StudentGormRepository) MarkRiskMessageSent(ctx context.Context, boxID, id domain.ID, sentAt time.Time) error {
	return databaseForContext(r.db, ctx).
		Model(&models.StudentModel{}).
		Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).
		Updates(map[string]any{
			"risk_status":          string(domain.StudentRiskStatusObserving),
			"risk_last_message_at": sentAt,
		}).Error
}
