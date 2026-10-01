package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (r SchedulingGormRepository) LockSeries(ctx context.Context, organizationID, seriesID domain.ID) error {
	return databaseForContext(r.db, ctx).Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", fmt.Sprintf("scheduling-series:%s:%s", organizationID, seriesID)).Error
}

func (r SchedulingGormRepository) LockResources(ctx context.Context, organizationID domain.ID, professionalIDs, spaceIDs []domain.ID) error {
	keys := []string{}
	for _, id := range professionalIDs {
		if id != "" {
			keys = append(keys, "professional:"+string(id))
		}
	}
	for _, id := range spaceIDs {
		if id != "" {
			keys = append(keys, "space:"+string(id))
		}
	}
	sort.Strings(keys)
	db := databaseForContext(r.db, ctx)
	for _, key := range keys {
		if err := db.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", fmt.Sprintf("scheduling-resource:%s:%s", organizationID, key)).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r SchedulingGormRepository) ListSeries(ctx context.Context, organizationID, unitID domain.ID, includeArchived bool) ([]domain.ScheduleSeries, error) {
	q := databaseForContext(r.db, ctx).Where("organization_id = ? AND unit_id = ?", stringID(organizationID), stringID(unitID))
	if !includeArchived {
		q = q.Where("status <> ?", string(domain.ScheduleSeriesArchived))
	}
	var rows []models.SchedulingSeriesModel
	if err := q.Order("CASE status WHEN 'draft' THEN 0 WHEN 'published' THEN 1 WHEN 'cancelled' THEN 2 ELSE 3 END, local_start_time, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	return r.seriesRows(ctx, rows)
}

func (r SchedulingGormRepository) ListPublishedSeries(ctx context.Context) ([]domain.ScheduleSeries, error) {
	var rows []models.SchedulingSeriesModel
	if err := databaseForContext(r.db, ctx).Where("status = ?", string(domain.ScheduleSeriesPublished)).Order("organization_id, unit_id, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	return r.seriesRows(ctx, rows)
}

func (r SchedulingGormRepository) ListSeriesAcrossOrganization(ctx context.Context, organizationID domain.ID) ([]domain.ScheduleSeries, error) {
	var rows []models.SchedulingSeriesModel
	if err := databaseForContext(r.db, ctx).Where("organization_id = ? AND status <> 'archived'", stringID(organizationID)).Order("unit_id,local_start_time,id").Find(&rows).Error; err != nil {
		return nil, err
	}
	return r.seriesRows(ctx, rows)
}

func (r SchedulingGormRepository) seriesRows(ctx context.Context, rows []models.SchedulingSeriesModel) ([]domain.ScheduleSeries, error) {
	result := make([]domain.ScheduleSeries, 0, len(rows))
	for _, row := range rows {
		weekdays, err := r.weekdays(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, seriesModelToDomain(row, weekdays))
	}
	return result, nil
}

func (r SchedulingGormRepository) weekdays(ctx context.Context, seriesID string) ([]int, error) {
	var rows []models.SchedulingSeriesWeekdayModel
	if err := databaseForContext(r.db, ctx).Where("series_id = ?", seriesID).Order("weekday").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]int, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.Weekday)
	}
	return result, nil
}

func (r SchedulingGormRepository) GetSeries(ctx context.Context, organizationID, unitID, seriesID domain.ID, forUpdate bool) (*domain.ScheduleSeries, error) {
	db := databaseForContext(r.db, ctx)
	var row models.SchedulingSeriesModel
	q := "SELECT * FROM scheduling_series WHERE id = ? AND organization_id = ? AND unit_id = ?"
	if forUpdate {
		q += " FOR UPDATE"
	}
	if err := db.Raw(q, stringID(seriesID), stringID(organizationID), stringID(unitID)).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == "" {
		return nil, portrepo.ErrScheduleSeriesNotFound
	}
	weekdays, err := r.weekdays(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	value := seriesModelToDomain(row, weekdays)
	return &value, nil
}

func (r SchedulingGormRepository) CreateSeries(ctx context.Context, series *domain.ScheduleSeries) error {
	if err := ensureID(&series.ID); err != nil {
		return err
	}
	db := databaseForContext(r.db, ctx)
	row, err := seriesDomainToModel(*series)
	if err != nil {
		return err
	}
	if err := db.Create(&row).Error; err != nil {
		return schedulingGridWriteError(err)
	}
	return r.replaceWeekdays(ctx, *series)
}

func (r SchedulingGormRepository) UpdateSeries(ctx context.Context, series *domain.ScheduleSeries) error {
	db := databaseForContext(r.db, ctx)
	row, err := seriesDomainToModel(*series)
	if err != nil {
		return err
	}
	if err := db.Save(&row).Error; err != nil {
		return schedulingGridWriteError(err)
	}
	return r.replaceWeekdays(ctx, *series)
}

func (r SchedulingGormRepository) replaceWeekdays(ctx context.Context, series domain.ScheduleSeries) error {
	db := databaseForContext(r.db, ctx)
	if err := db.Where("series_id = ?", stringID(series.ID)).Delete(&models.SchedulingSeriesWeekdayModel{}).Error; err != nil {
		return err
	}
	for _, weekday := range series.Weekdays {
		row := models.SchedulingSeriesWeekdayModel{OrganizationID: stringID(series.OrganizationID), UnitID: stringID(series.UnitID), SeriesID: stringID(series.ID), Weekday: weekday}
		if err := db.Create(&row).Error; err != nil {
			return schedulingGridWriteError(err)
		}
	}
	return nil
}

func (r SchedulingGormRepository) CountActiveSeriesForProfessional(ctx context.Context, organizationID, professionalID domain.ID) (int64, error) {
	var count int64
	err := databaseForContext(r.db, ctx).Model(&models.SchedulingSeriesModel{}).Where("organization_id = ? AND professional_id = ? AND status IN ('draft','published')", stringID(organizationID), stringID(professionalID)).Count(&count).Error
	return count, err
}

func (r SchedulingGormRepository) ListSessions(ctx context.Context, organizationID, unitID domain.ID, from, until time.Time, includeCancelled bool) ([]domain.ScheduleSession, error) {
	q := databaseForContext(r.db, ctx).Where("organization_id = ? AND unit_id = ? AND starts_at < ? AND ends_at > ?", stringID(organizationID), stringID(unitID), until, from)
	if !includeCancelled {
		q = q.Where("status = 'scheduled'")
	}
	var rows []models.SchedulingSessionModel
	if err := q.Order("starts_at, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ScheduleSession, 0, len(rows))
	for _, row := range rows {
		result = append(result, sessionModelToDomain(row))
	}
	return result, nil
}

func (r SchedulingGormRepository) ListSessionsAcrossOrganization(ctx context.Context, organizationID domain.ID, from, until time.Time) ([]domain.ScheduleSession, error) {
	var rows []models.SchedulingSessionModel
	if err := databaseForContext(r.db, ctx).Where("organization_id = ? AND status = 'scheduled' AND starts_at < ? AND ends_at > ?", stringID(organizationID), until, from).Order("starts_at,id").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ScheduleSession, 0, len(rows))
	for _, row := range rows {
		result = append(result, sessionModelToDomain(row))
	}
	return result, nil
}

func (r SchedulingGormRepository) GetSession(ctx context.Context, organizationID, unitID, sessionID domain.ID, forUpdate bool) (*domain.ScheduleSession, error) {
	var row models.SchedulingSessionModel
	q := "SELECT * FROM scheduling_sessions WHERE id = ? AND organization_id = ? AND unit_id = ?"
	if forUpdate {
		q += " FOR UPDATE"
	}
	if err := databaseForContext(r.db, ctx).Raw(q, stringID(sessionID), stringID(organizationID), stringID(unitID)).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == "" {
		return nil, portrepo.ErrScheduleSessionNotFound
	}
	value := sessionModelToDomain(row)
	return &value, nil
}

func (r SchedulingGormRepository) CreateSession(ctx context.Context, session *domain.ScheduleSession) error {
	if err := ensureID(&session.ID); err != nil {
		return err
	}
	row, err := sessionDomainToModel(*session)
	if err != nil {
		return err
	}
	return schedulingGridWriteError(databaseForContext(r.db, ctx).Create(&row).Error)
}
func (r SchedulingGormRepository) UpdateSession(ctx context.Context, session *domain.ScheduleSession) error {
	row, err := sessionDomainToModel(*session)
	if err != nil {
		return err
	}
	return schedulingGridWriteError(databaseForContext(r.db, ctx).Save(&row).Error)
}

func (r SchedulingGormRepository) CreateSessionEvent(ctx context.Context, event *domain.SessionEventV1) error {
	if err := ensureID(&event.ID); err != nil {
		return err
	}
	before, _ := json.Marshal(nonNilAuditData(event.BeforeState))
	after, _ := json.Marshal(nonNilAuditData(event.AfterState))
	row := models.SchedulingSessionEventModel{ID: stringID(event.ID), OrganizationID: stringID(event.OrganizationID), UnitID: stringID(event.UnitID), SessionID: stringID(event.SessionID), SessionRevision: event.SessionRevision, EventType: event.EventType, BeforeState: before, AfterState: after, OccurredAt: event.OccurredAt, SchemaVersion: 1}
	return databaseForContext(r.db, ctx).Create(&row).Error
}
func (r SchedulingGormRepository) ListSessionEvents(ctx context.Context, organizationID, unitID domain.ID, afterSequence int64, limit int) ([]domain.SessionEventV1, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	var rows []models.SchedulingSessionEventModel
	if err := databaseForContext(r.db, ctx).Where("organization_id = ? AND unit_id = ? AND sequence > ?", stringID(organizationID), stringID(unitID), afterSequence).Order("sequence").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.SessionEventV1, 0, len(rows))
	for _, row := range rows {
		before, after := map[string]any{}, map[string]any{}
		_ = json.Unmarshal(row.BeforeState, &before)
		_ = json.Unmarshal(row.AfterState, &after)
		out = append(out, domain.SessionEventV1{Sequence: row.Sequence, ID: domainID(row.ID), OrganizationID: domainID(row.OrganizationID), UnitID: domainID(row.UnitID), SessionID: domainID(row.SessionID), SessionRevision: row.SessionRevision, EventType: row.EventType, BeforeState: before, AfterState: after, OccurredAt: row.OccurredAt, SchemaVersion: row.SchemaVersion})
	}
	return out, nil
}

func (r SchedulingGormRepository) ClaimMaterializationRun(ctx context.Context, executionKey, windowFrom, windowUntil string, startedAt time.Time) (domain.ID, bool, error) {
	requestedID := uuid.NewString()
	var claimed struct {
		ID string
	}
	err := databaseForContext(r.db, ctx).Raw(`
		INSERT INTO scheduling_materialization_runs
			(id, execution_key, status, window_from, window_until, started_at, created_at)
		VALUES (?, ?, 'running', ?::date, ?::date, ?, ?)
		ON CONFLICT (execution_key) DO UPDATE SET
			status = 'running',
			window_from = EXCLUDED.window_from,
			window_until = EXCLUDED.window_until,
			series_read = 0,
			sessions_created = 0,
			sessions_kept = 0,
			conflicts = 0,
			error_code = NULL,
			started_at = EXCLUDED.started_at,
			finished_at = NULL
		WHERE scheduling_materialization_runs.status = 'failed'
		   OR (scheduling_materialization_runs.status = 'running'
		       AND scheduling_materialization_runs.started_at < EXCLUDED.started_at - INTERVAL '2 hours')
		RETURNING id`, requestedID, executionKey, windowFrom, windowUntil, startedAt, startedAt).Scan(&claimed).Error
	if err != nil {
		return "", false, err
	}
	if claimed.ID == "" {
		return "", false, nil
	}
	return domainID(claimed.ID), true, nil
}

func (r SchedulingGormRepository) CompleteMaterializationRun(ctx context.Context, runID domain.ID, status string, seriesRead, sessionsCreated, sessionsKept, conflicts int, errorCode string, finishedAt time.Time) error {
	values := map[string]any{"status": status, "series_read": seriesRead, "sessions_created": sessionsCreated, "sessions_kept": sessionsKept, "conflicts": conflicts, "finished_at": finishedAt}
	if errorCode != "" {
		values["error_code"] = errorCode
	}
	return databaseForContext(r.db, ctx).Table("scheduling_materialization_runs").Where("id = ?", stringID(runID)).Updates(values).Error
}

func seriesDomainToModel(v domain.ScheduleSeries) (models.SchedulingSeriesModel, error) {
	from, err := time.Parse("2006-01-02", v.EffectiveFrom)
	if err != nil {
		return models.SchedulingSeriesModel{}, err
	}
	var until *time.Time
	if v.EffectiveUntil != nil {
		x, e := time.Parse("2006-01-02", *v.EffectiveUntil)
		if e != nil {
			return models.SchedulingSeriesModel{}, e
		}
		until = &x
	}
	return models.SchedulingSeriesModel{ID: stringID(v.ID), OrganizationID: stringID(v.OrganizationID), UnitID: stringID(v.UnitID), Status: string(v.Status), TargetKind: string(v.TargetKind), TargetID: stringID(v.TargetID), TargetNameSnapshot: scheduleNullableString(v.TargetNameSnapshot), ProfessionalID: stringID(v.ProfessionalID), ProfessionalNameSnapshot: scheduleNullableString(v.ProfessionalNameSnapshot), SpaceID: optionalDomainIDStringPtr(v.SpaceID), SpaceNameSnapshot: scheduleNullableString(v.SpaceNameSnapshot), DeliveryMode: string(v.DeliveryMode), Capacity: v.Capacity, DurationMinutes: v.DurationMinutes, LocalStartTime: v.LocalStartTime, Timezone: v.Timezone, EffectiveFrom: from, EffectiveUntil: until, Revision: v.Revision, PublishedAt: v.PublishedAt, PublishedByUserID: optionalDomainIDStringPtr(v.PublishedByUserID), CancelledAt: v.CancelledAt, CancelledByUserID: optionalDomainIDStringPtr(v.CancelledByUserID), CancelledReason: scheduleNullableString(v.CancelledReason), ArchivedAt: v.ArchivedAt, ArchivedByUserID: optionalDomainIDStringPtr(v.ArchivedByUserID), CreatedByUserID: stringID(v.CreatedByUserID), UpdatedByUserID: stringID(v.UpdatedByUserID), CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}, nil
}
func seriesModelToDomain(v models.SchedulingSeriesModel, weekdays []int) domain.ScheduleSeries {
	return domain.ScheduleSeries{ID: domainID(v.ID), OrganizationID: domainID(v.OrganizationID), UnitID: domainID(v.UnitID), Status: domain.ScheduleSeriesStatus(v.Status), TargetKind: domain.ScheduleTargetKind(v.TargetKind), TargetID: domainID(v.TargetID), TargetNameSnapshot: valueString(v.TargetNameSnapshot), ProfessionalID: domainID(v.ProfessionalID), ProfessionalNameSnapshot: valueString(v.ProfessionalNameSnapshot), SpaceID: optionalStringID(v.SpaceID), SpaceNameSnapshot: valueString(v.SpaceNameSnapshot), DeliveryMode: domain.ScheduleDeliveryMode(v.DeliveryMode), Capacity: v.Capacity, DurationMinutes: v.DurationMinutes, Weekdays: weekdays, LocalStartTime: v.LocalStartTime[:5], Timezone: v.Timezone, EffectiveFrom: v.EffectiveFrom.Format("2006-01-02"), EffectiveUntil: optionalDate(v.EffectiveUntil), Revision: v.Revision, PublishedAt: v.PublishedAt, PublishedByUserID: optionalStringID(v.PublishedByUserID), CancelledAt: v.CancelledAt, CancelledByUserID: optionalStringID(v.CancelledByUserID), CancelledReason: valueString(v.CancelledReason), ArchivedAt: v.ArchivedAt, ArchivedByUserID: optionalStringID(v.ArchivedByUserID), CreatedByUserID: domainID(v.CreatedByUserID), UpdatedByUserID: domainID(v.UpdatedByUserID), CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
}
func sessionDomainToModel(v domain.ScheduleSession) (models.SchedulingSessionModel, error) {
	localDate, err := time.Parse("2006-01-02", v.LocalDate)
	if err != nil {
		return models.SchedulingSessionModel{}, err
	}
	var occurrence *time.Time
	if v.OccurrenceOn != nil {
		x, e := time.Parse("2006-01-02", *v.OccurrenceOn)
		if e != nil {
			return models.SchedulingSessionModel{}, e
		}
		occurrence = &x
	}
	return models.SchedulingSessionModel{ID: stringID(v.ID), OrganizationID: stringID(v.OrganizationID), UnitID: stringID(v.UnitID), SeriesID: optionalDomainIDStringPtr(v.SeriesID), Origin: v.Origin, OccurrenceOn: occurrence, Status: string(v.Status), TargetKind: string(v.TargetKind), TargetID: stringID(v.TargetID), TargetNameSnapshot: v.TargetName, ProfessionalID: stringID(v.ProfessionalID), ProfessionalNameSnapshot: v.ProfessionalName, SpaceID: optionalDomainIDStringPtr(v.SpaceID), SpaceNameSnapshot: scheduleNullableString(v.SpaceName), DeliveryMode: string(v.DeliveryMode), Capacity: v.Capacity, StartsAt: v.StartsAt, EndsAt: v.EndsAt, LocalDate: localDate, LocalStartTime: v.LocalStartTime, Timezone: v.Timezone, SourceSeriesRevision: v.SourceSeriesRevision, ManualOverride: v.ManualOverride, Revision: v.Revision, PublishedAt: v.PublishedAt, CancelledAt: v.CancelledAt, CancelledByUserID: optionalDomainIDStringPtr(v.CancelledByUserID), CancelledReason: scheduleNullableString(v.CancelledReason), CreatedByUserID: stringID(v.CreatedByUserID), UpdatedByUserID: stringID(v.UpdatedByUserID), CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}, nil
}
func sessionModelToDomain(v models.SchedulingSessionModel) domain.ScheduleSession {
	return domain.ScheduleSession{ID: domainID(v.ID), OrganizationID: domainID(v.OrganizationID), UnitID: domainID(v.UnitID), SeriesID: optionalStringID(v.SeriesID), Origin: v.Origin, OccurrenceOn: optionalDate(v.OccurrenceOn), Status: domain.ScheduleSessionStatus(v.Status), TargetKind: domain.ScheduleTargetKind(v.TargetKind), TargetID: domainID(v.TargetID), TargetName: v.TargetNameSnapshot, ProfessionalID: domainID(v.ProfessionalID), ProfessionalName: v.ProfessionalNameSnapshot, SpaceID: optionalStringID(v.SpaceID), SpaceName: valueString(v.SpaceNameSnapshot), DeliveryMode: domain.ScheduleDeliveryMode(v.DeliveryMode), Capacity: v.Capacity, StartsAt: v.StartsAt, EndsAt: v.EndsAt, LocalDate: v.LocalDate.Format("2006-01-02"), LocalStartTime: v.LocalStartTime[:5], Timezone: v.Timezone, SourceSeriesRevision: v.SourceSeriesRevision, ManualOverride: v.ManualOverride, Revision: v.Revision, PublishedAt: v.PublishedAt, CancelledAt: v.CancelledAt, CancelledByUserID: optionalStringID(v.CancelledByUserID), CancelledReason: valueString(v.CancelledReason), CreatedByUserID: domainID(v.CreatedByUserID), UpdatedByUserID: domainID(v.UpdatedByUserID), CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
}
func scheduleNullableString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
func valueString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func optionalStringID(v *string) *domain.ID {
	if v == nil || *v == "" {
		return nil
	}
	x := domainID(*v)
	return &x
}
func optionalDomainIDStringPtr(v *domain.ID) *string {
	if v == nil || *v == "" {
		return nil
	}
	x := stringID(*v)
	return &x
}
func optionalDate(v *time.Time) *string {
	if v == nil {
		return nil
	}
	x := v.Format("2006-01-02")
	return &x
}
func schedulingGridWriteError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return portrepo.ErrScheduleConflict
	}
	return schedulingWriteError(err)
}
