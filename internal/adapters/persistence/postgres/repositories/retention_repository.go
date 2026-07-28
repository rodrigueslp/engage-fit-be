package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
)

func (r RetentionGormRepository) ListMetrics(ctx context.Context, boxID domain.ID, recentStart, previousStart, end time.Time) ([]domain.RetentionMetrics, error) {
	type row struct {
		StudentID                  string
		StudentName                string
		StudentPhone               string
		Source                     string
		ContactStatus              string
		FirstCheckin               *time.Time
		LastCheckin                *time.Time
		RecentCheckins             int
		PreviousCheckins           int
		LastCompletedIntervention  *time.Time
		FirstReturnAfterAction     *time.Time
		LastInterventionID         string
		LastInterventionChannel    string
		LastInterventionStatus     string
		LastInterventionOutcome    string
		LastInterventionPlannedFor *time.Time
		LastInterventionCreatedAt  *time.Time
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		WITH attendance AS (
			SELECT s.id AS student_id,
			       MIN(c.checkin_date) AS first_checkin,
			       MAX(c.checkin_date) AS last_checkin,
			       COUNT(c.id) FILTER (WHERE c.checkin_date >= ?) AS recent_checkins,
			       COUNT(c.id) FILTER (WHERE c.checkin_date >= ? AND c.checkin_date < ?) AS previous_checkins
			FROM students s
			LEFT JOIN checkins c ON c.student_id = s.id AND c.box_id = s.box_id AND c.checkin_date <= ?
			WHERE s.box_id = ? AND s.anonymized_at IS NULL
			GROUP BY s.id
		), last_action AS (
			SELECT DISTINCT ON (student_id) student_id, id, channel, status, outcome,
			       planned_for, completed_at, created_at
			FROM retention_interventions
			WHERE box_id = ?
			ORDER BY student_id, created_at DESC
		)
		SELECT s.id AS student_id, s.name AS student_name, s.phone AS student_phone,
		       s.source, s.contact_status, a.first_checkin, a.last_checkin,
		       a.recent_checkins, a.previous_checkins,
		       la.completed_at AS last_completed_intervention,
		       returns.first_return_after_action,
		       la.id AS last_intervention_id,
		       la.channel AS last_intervention_channel,
		       la.status AS last_intervention_status,
		       la.outcome AS last_intervention_outcome,
		       la.planned_for AS last_intervention_planned_for,
		       la.created_at AS last_intervention_created_at
		FROM students s
		JOIN attendance a ON a.student_id = s.id
		LEFT JOIN last_action la ON la.student_id = s.id
		LEFT JOIN LATERAL (
			SELECT MIN(c.checkin_date) AS first_return_after_action
			FROM checkins c
			WHERE la.completed_at IS NOT NULL
			  AND c.box_id = s.box_id AND c.student_id = s.id
			  AND c.checkin_date > la.completed_at::date
		) returns ON TRUE
		WHERE s.box_id = ? AND s.anonymized_at IS NULL
		ORDER BY s.name ASC
	`, recentStart, previousStart, recentStart, end, stringID(boxID), stringID(boxID), stringID(boxID)).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]domain.RetentionMetrics, 0, len(rows))
	for _, item := range rows {
		result = append(result, domain.RetentionMetrics{
			StudentID: domainID(item.StudentID), StudentName: item.StudentName,
			StudentPhone: item.StudentPhone, Source: domain.Source(item.Source),
			ContactStatus: domain.ContactStatus(item.ContactStatus),
			FirstCheckin:  item.FirstCheckin, LastCheckin: item.LastCheckin,
			RecentCheckins: item.RecentCheckins, PreviousCheckins: item.PreviousCheckins,
			LastCompletedIntervention:  item.LastCompletedIntervention,
			FirstReturnAfterAction:     item.FirstReturnAfterAction,
			LastInterventionID:         domainID(item.LastInterventionID),
			LastInterventionChannel:    item.LastInterventionChannel,
			LastInterventionStatus:     item.LastInterventionStatus,
			LastInterventionOutcome:    item.LastInterventionOutcome,
			LastInterventionPlannedFor: item.LastInterventionPlannedFor,
			LastInterventionCreatedAt:  item.LastInterventionCreatedAt,
		})
	}
	return result, nil
}

func (r RetentionGormRepository) ListInterventions(ctx context.Context, boxID, studentID domain.ID) ([]domain.RetentionIntervention, error) {
	var rows []models.RetentionInterventionModel
	if err := r.db.WithContext(ctx).Where("box_id = ? AND student_id = ?", stringID(boxID), stringID(studentID)).Order("created_at DESC").Limit(100).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.RetentionIntervention, 0, len(rows))
	for _, row := range rows {
		result = append(result, retentionInterventionToDomain(row))
	}
	return result, nil
}

func (r RetentionGormRepository) FindIntervention(ctx context.Context, boxID, id domain.ID) (*domain.RetentionIntervention, error) {
	var row models.RetentionInterventionModel
	if err := r.db.WithContext(ctx).Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).First(&row).Error; err != nil {
		return nil, err
	}
	item := retentionInterventionToDomain(row)
	return &item, nil
}

func (r RetentionGormRepository) SaveIntervention(ctx context.Context, item *domain.RetentionIntervention) error {
	if err := ensureID(&item.ID); err != nil {
		return err
	}
	row := retentionInterventionToModel(*item)
	return r.db.WithContext(ctx).Create(&row).Error
}

func (r RetentionGormRepository) UpdateIntervention(ctx context.Context, item domain.RetentionIntervention) error {
	return r.db.WithContext(ctx).Model(&models.RetentionInterventionModel{}).
		Where("box_id = ? AND id = ?", stringID(item.BoxID), stringID(item.ID)).
		Updates(map[string]any{
			"status": item.Status, "outcome": nullableString(item.Outcome),
			"planned_for": item.PlannedFor, "completed_at": item.CompletedAt,
			"notes": nullableString(item.Notes), "updated_at": item.UpdatedAt,
		}).Error
}

func retentionInterventionToDomain(row models.RetentionInterventionModel) domain.RetentionIntervention {
	item := domain.RetentionIntervention{
		ID: domainID(row.ID), BoxID: domainID(row.BoxID), StudentID: domainID(row.StudentID),
		Channel: row.Channel, Status: row.Status, PlannedFor: row.PlannedFor,
		CompletedAt: row.CompletedAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if row.CreatedByUserID != nil {
		item.CreatedByUserID = domainID(*row.CreatedByUserID)
	}
	if row.Outcome != nil {
		item.Outcome = *row.Outcome
	}
	if row.Notes != nil {
		item.Notes = *row.Notes
	}
	return item
}

func retentionInterventionToModel(item domain.RetentionIntervention) models.RetentionInterventionModel {
	var userID *string
	if item.CreatedByUserID != "" {
		value := stringID(item.CreatedByUserID)
		userID = &value
	}
	var outcome, notes *string
	if item.Outcome != "" {
		value := item.Outcome
		outcome = &value
	}
	if item.Notes != "" {
		value := item.Notes
		notes = &value
	}
	return models.RetentionInterventionModel{
		ID: stringID(item.ID), BoxID: stringID(item.BoxID), StudentID: stringID(item.StudentID),
		CreatedByUserID: userID, Channel: item.Channel, Status: item.Status, Outcome: outcome,
		PlannedFor: item.PlannedFor, CompletedAt: item.CompletedAt, Notes: notes,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
