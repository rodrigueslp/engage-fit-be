package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r CheckinIngestionGormRepository) ListSources(ctx context.Context, boxID domain.ID) ([]domain.CheckinIngestionSource, error) {
	var rows []models.CheckinIngestionSourceModel
	if err := r.db.WithContext(ctx).Where("box_id = ?", stringID(boxID)).Order("name ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.CheckinIngestionSource, 0, len(rows))
	for _, row := range rows {
		result = append(result, ingestionSourceToDomain(row))
	}
	return result, nil
}

func (r CheckinIngestionGormRepository) FindSource(ctx context.Context, id domain.ID) (*domain.CheckinIngestionSource, error) {
	var row models.CheckinIngestionSourceModel
	if err := r.db.WithContext(ctx).Where("id = ?", stringID(id)).First(&row).Error; err != nil {
		return nil, err
	}
	item := ingestionSourceToDomain(row)
	return &item, nil
}

func (r CheckinIngestionGormRepository) FindSourceForBox(ctx context.Context, boxID, id domain.ID) (*domain.CheckinIngestionSource, error) {
	var row models.CheckinIngestionSourceModel
	if err := r.db.WithContext(ctx).Where("box_id = ? AND id = ?", stringID(boxID), stringID(id)).First(&row).Error; err != nil {
		return nil, err
	}
	item := ingestionSourceToDomain(row)
	return &item, nil
}

func (r CheckinIngestionGormRepository) SaveSource(ctx context.Context, item *domain.CheckinIngestionSource) error {
	if err := ensureID(&item.ID); err != nil {
		return err
	}
	row := ingestionSourceToModel(*item)
	return r.db.WithContext(ctx).Create(&row).Error
}

func (r CheckinIngestionGormRepository) UpdateSource(ctx context.Context, item domain.CheckinIngestionSource) error {
	return r.db.WithContext(ctx).Model(&models.CheckinIngestionSourceModel{}).
		Where("box_id = ? AND id = ?", stringID(item.BoxID), stringID(item.ID)).
		Updates(map[string]any{
			"name": item.Name, "token_hash": item.TokenHash, "enabled": item.Enabled, "updated_at": item.UpdatedAt,
		}).Error
}

func (r CheckinIngestionGormRepository) ClaimBatch(ctx context.Context, batch *domain.CheckinIngestionBatch) (bool, *domain.CheckinIngestionBatch, error) {
	if err := ensureID(&batch.ID); err != nil {
		return false, nil, err
	}
	row := ingestionBatchToModel(*batch)
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "source_id"}, {Name: "idempotency_key"}},
		DoNothing: true,
	}).Create(&row)
	if result.Error != nil {
		return false, nil, result.Error
	}
	if result.RowsAffected == 1 {
		return true, batch, nil
	}
	var existing models.CheckinIngestionBatchModel
	if err := r.db.WithContext(ctx).Where("source_id = ? AND idempotency_key = ?", stringID(batch.SourceID), batch.IdempotencyKey).First(&existing).Error; err != nil {
		return false, nil, err
	}
	value := ingestionBatchToDomain(existing)
	return false, &value, nil
}

func (r CheckinIngestionGormRepository) CompleteBatch(ctx context.Context, id domain.ID, status string, importID domain.ID, totalRecords, students, checkins int, completedAt time.Time) error {
	var importHistoryID any
	if importID != "" {
		importHistoryID = stringID(importID)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.CheckinIngestionBatchModel{}).
			Where("id = ?", stringID(id)).
			Updates(map[string]any{
				"status": status, "import_history_id": importHistoryID, "total_records": totalRecords,
				"students_created": students, "checkins_created": checkins, "completed_at": completedAt,
			}).Error; err != nil {
			return err
		}
		if status == "completed" {
			return tx.Model(&models.CheckinIngestionSourceModel{}).
				Where("id = (SELECT source_id FROM checkin_ingestion_batches WHERE id = ?)", stringID(id)).
				Updates(map[string]any{"last_ingested_at": completedAt, "updated_at": completedAt}).Error
		}
		return nil
	})
}

func ingestionSourceToDomain(row models.CheckinIngestionSourceModel) domain.CheckinIngestionSource {
	item := domain.CheckinIngestionSource{
		ID: domainID(row.ID), BoxID: domainID(row.BoxID), Name: row.Name, Source: domain.Source(row.Source),
		TokenHash: row.TokenHash, Enabled: row.Enabled, LastIngestedAt: row.LastIngestedAt,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if row.CreatedByUserID != nil {
		item.CreatedByUserID = domainID(*row.CreatedByUserID)
	}
	return item
}

func ingestionSourceToModel(item domain.CheckinIngestionSource) models.CheckinIngestionSourceModel {
	var userID *string
	if item.CreatedByUserID != "" {
		value := stringID(item.CreatedByUserID)
		userID = &value
	}
	return models.CheckinIngestionSourceModel{
		ID: stringID(item.ID), BoxID: stringID(item.BoxID), CreatedByUserID: userID,
		Name: item.Name, Source: string(item.Source), TokenHash: item.TokenHash, Enabled: item.Enabled,
		LastIngestedAt: item.LastIngestedAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

func ingestionBatchToDomain(row models.CheckinIngestionBatchModel) domain.CheckinIngestionBatch {
	item := domain.CheckinIngestionBatch{
		ID: domainID(row.ID), SourceID: domainID(row.SourceID), BoxID: domainID(row.BoxID),
		IdempotencyKey: row.IdempotencyKey, Status: row.Status, TotalRecords: row.TotalRecords,
		StudentsCreated: row.StudentsCreated, CheckinsCreated: row.CheckinsCreated,
		CreatedAt: row.CreatedAt, CompletedAt: row.CompletedAt,
	}
	if row.ImportHistoryID != nil {
		item.ImportHistoryID = domainID(*row.ImportHistoryID)
	}
	return item
}

func ingestionBatchToModel(item domain.CheckinIngestionBatch) models.CheckinIngestionBatchModel {
	var importID *string
	if item.ImportHistoryID != "" {
		value := stringID(item.ImportHistoryID)
		importID = &value
	}
	return models.CheckinIngestionBatchModel{
		ID: stringID(item.ID), SourceID: stringID(item.SourceID), BoxID: stringID(item.BoxID),
		IdempotencyKey: item.IdempotencyKey, Status: item.Status, ImportHistoryID: importID,
		TotalRecords: item.TotalRecords, StudentsCreated: item.StudentsCreated, CheckinsCreated: item.CheckinsCreated,
		CreatedAt: item.CreatedAt, CompletedAt: item.CompletedAt,
	}
}
