package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"gorm.io/gorm"
)

type FeatureFlagsRepository struct{ db *gorm.DB }

func NewFeatureFlagsRepository(db *gorm.DB) *FeatureFlagsRepository {
	return &FeatureFlagsRepository{db: db}
}

func (r *FeatureFlagsRepository) List(ctx context.Context) (map[string]bool, error) {
	var rows []struct {
		FeatureKey string
		Enabled    bool
	}
	if err := r.db.WithContext(ctx).Raw(`SELECT feature_key, enabled FROM platform_feature_overrides`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(rows))
	for _, row := range rows {
		result[row.FeatureKey] = row.Enabled
	}
	return result, nil
}

func (r *FeatureFlagsRepository) Set(ctx context.Context, key string, enabled bool, reason, adminID, ip string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Serialize admin changes and keep the audit entry in the same transaction.
		if err := tx.Exec(`SELECT pg_advisory_xact_lock(731947522)`).Error; err != nil {
			return err
		}
		var previous bool
		var found bool
		row := tx.Raw(`SELECT enabled FROM platform_feature_overrides WHERE feature_key = ?`, key).Row()
		if err := row.Scan(&previous); err == nil {
			found = true
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err := tx.Exec(`INSERT INTO platform_feature_overrides (feature_key, enabled, updated_by, updated_at)
			VALUES (?, ?, ?, NOW()) ON CONFLICT (feature_key) DO UPDATE SET enabled = EXCLUDED.enabled,
			updated_by = EXCLUDED.updated_by, updated_at = NOW()`, key, enabled, adminID).Error; err != nil {
			return err
		}
		var before any
		if found {
			data, _ := json.Marshal(map[string]bool{"enabled": previous})
			before = string(data)
		}
		after, _ := json.Marshal(map[string]bool{"enabled": enabled})
		return tx.Exec(`INSERT INTO admin_audit_logs (id, admin_user_id, action, target_type, target_id, before_data, after_data, reason, ip_address)
			VALUES (gen_random_uuid(), ?, 'feature_flag.updated', 'feature_flag', ?, CAST(? AS jsonb), CAST(? AS jsonb), ?, ?)`, adminID, key, before, string(after), reason, ip).Error
	})
}
