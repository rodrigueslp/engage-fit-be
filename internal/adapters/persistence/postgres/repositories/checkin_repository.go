package repositories

import (
	"context"
	"time"

	"gorm.io/gorm/clause"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
)

func (r CheckinGormRepository) ListByStudent(ctx context.Context, boxID, studentID domain.ID) ([]domain.Checkin, error) {
	var modelsList []models.CheckinModel
	if err := r.db.WithContext(ctx).
		Where("box_id = ? AND student_id = ?", stringID(boxID), stringID(studentID)).
		Order("checkin_date DESC, checkin_time DESC").
		Find(&modelsList).Error; err != nil {
		return nil, err
	}

	return checkinsToDomain(modelsList), nil
}

func (r CheckinGormRepository) ListByRange(ctx context.Context, boxID domain.ID, period domain.TimeRange) ([]domain.Checkin, error) {
	var modelsList []models.CheckinModel
	if err := r.db.WithContext(ctx).
		Where("box_id = ? AND checkin_date BETWEEN ? AND ?", stringID(boxID), period.Start, period.End).
		Order("checkin_date DESC, checkin_time DESC").
		Find(&modelsList).Error; err != nil {
		return nil, err
	}

	return checkinsToDomain(modelsList), nil
}

func (r CheckinGormRepository) ListMonthlyFrequency(ctx context.Context, boxID domain.ID, period domain.TimeRange) ([]domain.MonthlyFrequencyReportRow, error) {
	type frequencyRow struct {
		StudentID    string
		StudentName  string
		StudentPhone string
		Source       string
		Checkins     int
		FirstCheckin *time.Time
		LastCheckin  *time.Time
	}

	var rows []frequencyRow
	if err := r.db.WithContext(ctx).
		Model(&models.CheckinModel{}).
		Select(`
			students.id AS student_id,
			students.name AS student_name,
			students.phone AS student_phone,
			students.source AS source,
			COUNT(checkins.id) AS checkins,
			MIN(checkins.checkin_date) AS first_checkin,
			MAX(checkins.checkin_date) AS last_checkin
		`).
		Joins("JOIN students ON students.id = checkins.student_id").
		Where("checkins.box_id = ? AND checkins.checkin_date BETWEEN ? AND ?", stringID(boxID), period.Start, period.End).
		Group("students.id, students.name, students.phone, students.source").
		Order("checkins DESC, students.name ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]domain.MonthlyFrequencyReportRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.MonthlyFrequencyReportRow{
			StudentID:    domainID(row.StudentID),
			StudentName:  row.StudentName,
			StudentPhone: row.StudentPhone,
			Source:       domain.Source(row.Source),
			Checkins:     row.Checkins,
			FirstCheckin: row.FirstCheckin,
			LastCheckin:  row.LastCheckin,
		})
	}
	return result, nil
}

func (r CheckinGormRepository) CountBySource(ctx context.Context, boxID domain.ID, period domain.TimeRange) (map[domain.Source]int, error) {
	type countBySource struct {
		Source string
		Total  int
	}

	var rows []countBySource
	if err := r.db.WithContext(ctx).
		Model(&models.CheckinModel{}).
		Select("source, COUNT(*) AS total").
		Where("box_id = ? AND checkin_date BETWEEN ? AND ?", stringID(boxID), period.Start, period.End).
		Group("source").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := map[domain.Source]int{}
	for _, row := range rows {
		result[domain.Source(row.Source)] = row.Total
	}
	return result, nil
}

func (r CheckinGormRepository) LastCheckinDate(ctx context.Context, boxID, studentID domain.ID) (*time.Time, error) {
	var model models.CheckinModel
	if err := r.db.WithContext(ctx).
		Where("box_id = ? AND student_id = ?", stringID(boxID), stringID(studentID)).
		Order("checkin_date DESC").
		First(&model).Error; err != nil {
		return nil, err
	}

	return &model.CheckinDate, nil
}

func (r CheckinGormRepository) SaveMany(ctx context.Context, checkins []domain.Checkin) (int, error) {
	modelsList := make([]models.CheckinModel, 0, len(checkins))
	for i := range checkins {
		if err := ensureID(&checkins[i].ID); err != nil {
			return 0, err
		}
		modelsList = append(modelsList, checkinToModel(checkins[i]))
	}

	if len(modelsList) == 0 {
		return 0, nil
	}

	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "box_id"},
			{Name: "source"},
			{Name: "student_id"},
			{Name: "checkin_date"},
			{Name: "checkin_time"},
		},
		DoNothing: true,
	}).Create(&modelsList)
	return int(result.RowsAffected), result.Error
}

func checkinsToDomain(modelsList []models.CheckinModel) []domain.Checkin {
	checkins := make([]domain.Checkin, 0, len(modelsList))
	for _, model := range modelsList {
		checkins = append(checkins, checkinToDomain(model))
	}
	return checkins
}
