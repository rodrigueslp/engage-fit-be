package repositories

import (
	"context"
	"database/sql"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	"gorm.io/gorm"
)

func (r RetentionGormRepository) ListMetrics(ctx context.Context, boxID domain.ID, recentStart, previousStart, end time.Time) ([]domain.RetentionMetrics, error) {
	type row struct {
		StudentID                    string
		StudentName                  string
		StudentPhone                 string
		Source                       string
		ContactStatus                string
		FirstCheckin                 *time.Time
		LastCheckin                  *time.Time
		TotalCheckins                int
		RecentCheckins               int
		PreviousCheckins             int
		LastCompletedIntervention    *time.Time
		FirstReturnAfterAction       *time.Time
		LastInterventionID           string
		LastInterventionChannel      string
		LastInterventionStatus       string
		LastInterventionOutcome      string
		LastInterventionPlannedFor   *time.Time
		LastInterventionCreatedAt    *time.Time
		LastInterventionAssigneeID   string
		LastInterventionAssigneeName string
		RetentionMonitoringStatus    string
		RetentionExclusionReason     string
		RetentionExcludedUntil       *time.Time
		RetentionExcludedAt          *time.Time
		RetentionExcludedByUserID    string
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		WITH attendance AS (
			SELECT s.id AS student_id,
			       MIN(c.checkin_date) AS first_checkin,
			       MAX(c.checkin_date) AS last_checkin,
			       COUNT(c.id) AS total_checkins,
			       COUNT(c.id) FILTER (WHERE c.checkin_date >= ?) AS recent_checkins,
			       COUNT(c.id) FILTER (WHERE c.checkin_date >= ? AND c.checkin_date < ?) AS previous_checkins
			FROM students s
			LEFT JOIN checkins c ON c.student_id = s.id AND c.box_id = s.box_id AND c.checkin_date <= ?
			WHERE s.box_id = ? AND s.anonymized_at IS NULL
			GROUP BY s.id
		), last_action AS (
			SELECT DISTINCT ON (student_id) student_id, id, channel, status, outcome,
			       planned_for, completed_at, created_at, assigned_to_user_id
			FROM retention_interventions
			WHERE box_id = ?
			ORDER BY student_id, created_at DESC
		)
		SELECT s.id AS student_id, s.name AS student_name, s.phone AS student_phone,
		       s.source, s.contact_status, a.first_checkin, a.last_checkin,
		       a.total_checkins, a.recent_checkins, a.previous_checkins,
		       la.completed_at AS last_completed_intervention,
		       returns.first_return_after_action,
		       la.id AS last_intervention_id,
		       la.channel AS last_intervention_channel,
		       la.status AS last_intervention_status,
		       la.outcome AS last_intervention_outcome,
		       la.planned_for AS last_intervention_planned_for,
		       la.created_at AS last_intervention_created_at,
		       la.assigned_to_user_id AS last_intervention_assignee_id,
		       assignee.name AS last_intervention_assignee_name
		       , s.retention_monitoring_status, s.retention_exclusion_reason,
		       s.retention_excluded_until, s.retention_excluded_at,
		       s.retention_excluded_by_user_id
		FROM students s
		JOIN attendance a ON a.student_id = s.id
		LEFT JOIN last_action la ON la.student_id = s.id
		LEFT JOIN users assignee ON assignee.id = la.assigned_to_user_id AND assignee.box_id = s.box_id
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
			TotalCheckins: item.TotalCheckins, RecentCheckins: item.RecentCheckins, PreviousCheckins: item.PreviousCheckins,
			LastCompletedIntervention:    item.LastCompletedIntervention,
			FirstReturnAfterAction:       item.FirstReturnAfterAction,
			LastInterventionID:           domainID(item.LastInterventionID),
			LastInterventionChannel:      item.LastInterventionChannel,
			LastInterventionStatus:       item.LastInterventionStatus,
			LastInterventionOutcome:      item.LastInterventionOutcome,
			LastInterventionPlannedFor:   item.LastInterventionPlannedFor,
			LastInterventionCreatedAt:    item.LastInterventionCreatedAt,
			LastInterventionAssigneeID:   domainID(item.LastInterventionAssigneeID),
			LastInterventionAssigneeName: item.LastInterventionAssigneeName,
			RetentionMonitoringStatus:    domain.RetentionMonitoringStatus(item.RetentionMonitoringStatus),
			RetentionExclusionReason:     item.RetentionExclusionReason,
			RetentionExcludedUntil:       item.RetentionExcludedUntil,
			RetentionExcludedAt:          item.RetentionExcludedAt,
			RetentionExcludedByUserID:    domainID(item.RetentionExcludedByUserID),
		})
	}
	return result, nil
}

func (r RetentionGormRepository) ListInterventions(ctx context.Context, boxID, studentID domain.ID) ([]domain.RetentionIntervention, error) {
	var rows []models.RetentionInterventionModel
	if err := r.db.WithContext(ctx).Table("retention_interventions i").
		Select("i.*, COALESCE(u.name, '') AS assigned_to_user_name").
		Joins("LEFT JOIN users u ON u.id = i.assigned_to_user_id AND u.box_id = i.box_id").
		Where("i.box_id = ? AND i.student_id = ?", stringID(boxID), stringID(studentID)).
		Order("i.created_at DESC").Limit(100).Scan(&rows).Error; err != nil {
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
			"reason_code":         nullableString(item.ReasonCode),
			"assigned_to_user_id": nullableID(item.AssignedToUserID),
			"planned_for":         item.PlannedFor, "completed_at": item.CompletedAt,
			"notes": nullableString(item.Notes), "updated_at": item.UpdatedAt,
		}).Error
}

func (r RetentionGormRepository) SummarizeInterventions(ctx context.Context, boxID domain.ID, start, end time.Time) (*domain.RetentionInterventionSummary, error) {
	type totalsRow struct {
		CompletedInterventions int
		ReturnWithin3Days      int
		ReturnWithin7Days      int
		ReturnWithin14Days     int
		MedianDaysToReturn     sql.NullFloat64
	}
	var totals totalsRow
	err := r.db.WithContext(ctx).Raw(`
		WITH completed_actions AS (
			SELECT i.id, i.student_id, i.completed_at,
			       returns.first_return,
			       CASE WHEN returns.first_return IS NOT NULL
			            THEN returns.first_return::date - i.completed_at::date
			       END AS days_to_return
			FROM retention_interventions i
			LEFT JOIN LATERAL (
				SELECT MIN(c.checkin_date) AS first_return
				FROM checkins c
				WHERE c.box_id = i.box_id
				  AND c.student_id = i.student_id
				  AND c.checkin_date > i.completed_at::date
			) returns ON TRUE
			WHERE i.box_id = ?
			  AND i.status = 'completed'
			  AND i.completed_at >= ?
			  AND i.completed_at < ?
		)
		SELECT COUNT(*) AS completed_interventions,
		       COUNT(*) FILTER (WHERE days_to_return BETWEEN 0 AND 3) AS return_within3_days,
		       COUNT(*) FILTER (WHERE days_to_return BETWEEN 0 AND 7) AS return_within7_days,
		       COUNT(*) FILTER (WHERE days_to_return BETWEEN 0 AND 14) AS return_within14_days,
		       percentile_cont(0.5) WITHIN GROUP (ORDER BY days_to_return)
		           FILTER (WHERE days_to_return BETWEEN 0 AND 14) AS median_days_to_return
		FROM completed_actions
	`, stringID(boxID), start, end).Scan(&totals).Error
	if err != nil {
		return nil, err
	}

	loadBreakdown := func(column string) ([]domain.RetentionBreakdown, error) {
		type row struct {
			Code  string
			Count int
		}
		var rows []row
		query := `
			SELECT ` + column + ` AS code, COUNT(*) AS count
			FROM retention_interventions
			WHERE box_id = ?
			  AND status = 'completed'
			  AND completed_at >= ?
			  AND completed_at < ?
			  AND ` + column + ` IS NOT NULL
			  AND ` + column + ` <> ''
			GROUP BY ` + column + `
			ORDER BY count DESC, code ASC
		`
		if err := r.db.WithContext(ctx).Raw(query, stringID(boxID), start, end).Scan(&rows).Error; err != nil {
			return nil, err
		}
		result := make([]domain.RetentionBreakdown, 0, len(rows))
		for _, item := range rows {
			result = append(result, domain.RetentionBreakdown{Code: item.Code, Count: item.Count})
		}
		return result, nil
	}

	reasons, err := loadBreakdown("reason_code")
	if err != nil {
		return nil, err
	}
	channels, err := loadBreakdown("channel")
	if err != nil {
		return nil, err
	}
	outcomes, err := loadBreakdown("outcome")
	if err != nil {
		return nil, err
	}
	var medianDaysToReturn *float64
	if totals.MedianDaysToReturn.Valid {
		value := totals.MedianDaysToReturn.Float64
		medianDaysToReturn = &value
	}
	return &domain.RetentionInterventionSummary{
		CompletedInterventions: totals.CompletedInterventions,
		ReturnWithin3Days:      totals.ReturnWithin3Days,
		ReturnWithin7Days:      totals.ReturnWithin7Days,
		ReturnWithin14Days:     totals.ReturnWithin14Days,
		MedianDaysToReturn:     medianDaysToReturn,
		Reasons:                reasons,
		Channels:               channels,
		Outcomes:               outcomes,
	}, nil
}

func (r RetentionGormRepository) ListOnboardingMetrics(ctx context.Context, boxID domain.ID, today time.Time) ([]domain.OnboardingMetrics, error) {
	type row struct {
		StudentID                  string
		StudentName                string
		StudentPhone               string
		Source                     string
		ContactStatus              string
		MembershipStartedAt        time.Time
		MembershipStartedSource    string
		ObservationDaysBeforeStart int
		FirstCheckin               *time.Time
		SecondCheckin              *time.Time
		LastCheckin                *time.Time
		CheckinsFirst7Days         int
		CheckinsFirst14Days        int
		CheckinsFirst30Days        int
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		WITH source_coverage AS (
			SELECT source, MIN(checkin_date)::date AS coverage_start
			FROM checkins
			WHERE box_id = ? AND checkin_date <= ?::date
			GROUP BY source
		), attendance_days AS (
			SELECT s.id AS student_id,
			       c.checkin_date::date AS checkin_day
			FROM students s
			JOIN checkins c ON c.box_id = s.box_id AND c.student_id = s.id
			WHERE s.box_id = ?
			  AND s.anonymized_at IS NULL
			  AND s.membership_started_at IS NOT NULL
			  AND c.checkin_date::date >= s.membership_started_at
			GROUP BY s.id, c.checkin_date::date
		), ranked_days AS (
			SELECT student_id, checkin_day,
			       ROW_NUMBER() OVER (PARTITION BY student_id ORDER BY checkin_day) AS visit_number
			FROM attendance_days
		)
		SELECT s.id AS student_id, s.name AS student_name, s.phone AS student_phone,
		       s.source, s.contact_status, s.membership_started_at, s.membership_started_source,
		       GREATEST(s.membership_started_at - coverage.coverage_start, 0) AS observation_days_before_start,
		       MIN(r.checkin_day) FILTER (WHERE r.visit_number = 1) AS first_checkin,
		       MIN(r.checkin_day) FILTER (WHERE r.visit_number = 2) AS second_checkin,
		       MAX(r.checkin_day) AS last_checkin,
		       COUNT(r.checkin_day) FILTER (WHERE r.checkin_day < s.membership_started_at + 7) AS checkins_first7_days,
		       COUNT(r.checkin_day) FILTER (WHERE r.checkin_day < s.membership_started_at + 14) AS checkins_first14_days,
		       COUNT(r.checkin_day) FILTER (WHERE r.checkin_day < s.membership_started_at + 30) AS checkins_first30_days
		FROM students s
		LEFT JOIN ranked_days r ON r.student_id = s.id
		LEFT JOIN source_coverage coverage ON coverage.source = s.source
		WHERE s.box_id = ?
		  AND s.anonymized_at IS NULL
		  AND (s.retention_monitoring_status <> 'excluded'
		       OR (s.retention_excluded_until IS NOT NULL AND s.retention_excluded_until < ?::date))
		  AND s.membership_started_at BETWEEN ?::date - 30 AND ?::date
		GROUP BY s.id, coverage.coverage_start
		ORDER BY s.membership_started_at DESC, s.name ASC
	`, stringID(boxID), today, stringID(boxID), stringID(boxID), today, today, today).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]domain.OnboardingMetrics, 0, len(rows))
	for _, item := range rows {
		result = append(result, domain.OnboardingMetrics{
			StudentID: domainID(item.StudentID), StudentName: item.StudentName, StudentPhone: item.StudentPhone,
			Source: domain.Source(item.Source), ContactStatus: domain.ContactStatus(item.ContactStatus),
			MembershipStartedAt: item.MembershipStartedAt, MembershipStartedSource: item.MembershipStartedSource,
			ObservationDaysBeforeStart: item.ObservationDaysBeforeStart,
			FirstCheckin:               item.FirstCheckin, SecondCheckin: item.SecondCheckin, LastCheckin: item.LastCheckin,
			CheckinsFirst7Days: item.CheckinsFirst7Days, CheckinsFirst14Days: item.CheckinsFirst14Days,
			CheckinsFirst30Days: item.CheckinsFirst30Days,
		})
	}
	return result, nil
}

func (r RetentionGormRepository) UpdateMembershipStart(ctx context.Context, boxID, studentID domain.ID, startedAt time.Time, source string, updatedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&models.StudentModel{}).
		Where("box_id = ? AND id = ? AND anonymized_at IS NULL", stringID(boxID), stringID(studentID)).
		Updates(map[string]any{
			"membership_started_at": startedAt, "membership_started_source": source, "updated_at": updatedAt,
		}).Error
}

func (r RetentionGormRepository) UpdateMonitoring(ctx context.Context, boxID, studentID, actorUserID domain.ID, status domain.RetentionMonitoringStatus, reason string, excludedUntil *time.Time, updatedAt time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"retention_monitoring_status": status,
			"updated_at":                  updatedAt,
		}
		if status == domain.RetentionMonitoringExcluded {
			updates["retention_exclusion_reason"] = reason
			updates["retention_excluded_until"] = excludedUntil
			updates["retention_excluded_at"] = updatedAt
			updates["retention_excluded_by_user_id"] = nullableID(actorUserID)
		} else {
			updates["retention_exclusion_reason"] = nil
			updates["retention_excluded_until"] = nil
			updates["retention_excluded_at"] = nil
			updates["retention_excluded_by_user_id"] = nil
		}
		result := tx.WithContext(ctx).Model(&models.StudentModel{}).
			Where("box_id = ? AND id = ? AND anonymized_at IS NULL", stringID(boxID), stringID(studentID)).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		id := domain.ID("")
		if err := ensureID(&id); err != nil {
			return err
		}
		return tx.WithContext(ctx).Table("retention_monitoring_events").Create(map[string]any{
			"id": stringID(id), "box_id": stringID(boxID), "student_id": stringID(studentID),
			"actor_user_id": nullableID(actorUserID), "monitoring_status": string(status),
			"reason": nullableString(reason), "excluded_until": excludedUntil, "created_at": updatedAt,
		}).Error
	})
}

func retentionInterventionToDomain(row models.RetentionInterventionModel) domain.RetentionIntervention {
	item := domain.RetentionIntervention{
		ID: domainID(row.ID), BoxID: domainID(row.BoxID), StudentID: domainID(row.StudentID),
		Channel: row.Channel, Status: row.Status, PlannedFor: row.PlannedFor,
		AssignedToUserName: row.AssignedToUserName,
		CompletedAt:        row.CompletedAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if row.CreatedByUserID != nil {
		item.CreatedByUserID = domainID(*row.CreatedByUserID)
	}
	if row.AssignedToUserID != nil {
		item.AssignedToUserID = domainID(*row.AssignedToUserID)
	}
	if row.Outcome != nil {
		item.Outcome = *row.Outcome
	}
	if row.ReasonCode != nil {
		item.ReasonCode = *row.ReasonCode
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
	var assignedToUserID *string
	if item.AssignedToUserID != "" {
		value := stringID(item.AssignedToUserID)
		assignedToUserID = &value
	}
	var outcome, reasonCode, notes *string
	if item.Outcome != "" {
		value := item.Outcome
		outcome = &value
	}
	if item.ReasonCode != "" {
		value := item.ReasonCode
		reasonCode = &value
	}
	if item.Notes != "" {
		value := item.Notes
		notes = &value
	}
	return models.RetentionInterventionModel{
		ID: stringID(item.ID), BoxID: stringID(item.BoxID), StudentID: stringID(item.StudentID),
		CreatedByUserID: userID, AssignedToUserID: assignedToUserID, Channel: item.Channel, Status: item.Status, Outcome: outcome,
		ReasonCode: reasonCode, PlannedFor: item.PlannedFor, CompletedAt: item.CompletedAt, Notes: notes,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
