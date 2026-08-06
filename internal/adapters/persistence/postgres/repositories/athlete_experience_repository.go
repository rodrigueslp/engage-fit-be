package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"boxengage/backend/internal/domain"
)

type athleteWorkoutResultModel struct {
	ID               string `gorm:"primaryKey"`
	AthleteAccountID string
	WorkoutID        string
	MembershipID     string
	Scale            string
	Entries          []byte `gorm:"type:jsonb"`
	RPE              *int
	Notes            string
	PerformedAt      time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (athleteWorkoutResultModel) TableName() string { return "athlete_workout_results" }

type athletePersonalRecordModel struct {
	ID               string `gorm:"primaryKey"`
	AthleteAccountID string
	MovementKey      string
	MovementName     string
	Metric           string
	BestValue        float64
	Unit             string
	Status           string
	SourceResultID   string
	AchievedAt       time.Time
	ConfirmedAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (athletePersonalRecordModel) TableName() string { return "athlete_personal_records" }

type athleteAccountTokenModel struct {
	ID               string `gorm:"primaryKey"`
	AthleteAccountID string
	Purpose          string
	TokenHash        string
	ExpiresAt        time.Time
	UsedAt           *time.Time
	CreatedAt        time.Time
}

func (athleteAccountTokenModel) TableName() string { return "athlete_account_tokens" }

type athleteWorkoutInsightModel struct {
	ID               string `gorm:"primaryKey"`
	AthleteAccountID string
	WorkoutID        string
	InputHash        string
	Provider         string
	Model            string
	Body             string
	CreatedAt        time.Time
}

func (athleteWorkoutInsightModel) TableName() string { return "athlete_workout_insights" }

func (r AthleteGormRepository) FindPublishedWorkout(ctx context.Context, athleteID, workoutID domain.ID) (*domain.AthleteWorkout, error) {
	items, err := r.ListPublishedWorkouts(ctx, athleteID)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ID == workoutID {
			return &items[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r AthleteGormRepository) UpsertWorkoutResult(ctx context.Context, result *domain.AthleteWorkoutResult) ([]domain.AthletePersonalRecord, error) {
	entries, err := json.Marshal(result.Entries)
	if err != nil {
		return nil, err
	}
	if result.ID == "" {
		result.ID = domain.ID(uuid.NewString())
	}
	now := result.UpdatedAt
	if now.IsZero() {
		now = time.Now()
	}
	var rpe *int
	if result.RPE > 0 {
		value := result.RPE
		rpe = &value
	}
	model := athleteWorkoutResultModel{ID: string(result.ID), AthleteAccountID: string(result.AthleteID), WorkoutID: string(result.WorkoutID), MembershipID: string(result.MembershipID), Scale: result.Scale, Entries: entries, RPE: rpe, Notes: result.Notes, PerformedAt: result.PerformedAt, CreatedAt: result.CreatedAt, UpdatedAt: now}
	newRecords := []domain.AthletePersonalRecord{}
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing athleteWorkoutResultModel
		findErr := tx.Where("athlete_account_id = ? AND workout_id = ?", model.AthleteAccountID, model.WorkoutID).Take(&existing).Error
		if findErr == nil {
			model.ID = existing.ID
			model.CreatedAt = existing.CreatedAt
		}
		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}
		if model.CreatedAt.IsZero() {
			model.CreatedAt = now
		}
		if model.PerformedAt.IsZero() {
			model.PerformedAt = now
		}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "athlete_account_id"}, {Name: "workout_id"}}, DoUpdates: clause.AssignmentColumns([]string{"membership_id", "scale", "entries", "rpe", "notes", "performed_at", "updated_at"})}).Create(&model).Error; err != nil {
			return err
		}
		result.ID = domain.ID(model.ID)
		entriesToRecompute := append([]domain.AthleteResultEntry{}, result.Entries...)
		if existing.ID != "" {
			previous, parseErr := workoutResultToDomain(existing)
			if parseErr != nil {
				return parseErr
			}
			entriesToRecompute = append(entriesToRecompute, previous.Entries...)
		}
		affected := map[string]struct{}{}
		for _, entry := range entriesToRecompute {
			metric, _, _ := recordCandidate(entry)
			if metric == "" || strings.TrimSpace(entry.Movement) == "" {
				continue
			}
			key := movementKey(entry.Movement)
			affected[key+"\x00"+metric] = struct{}{}
		}
		var allResults []athleteWorkoutResultModel
		if err := tx.Where("athlete_account_id = ?", model.AthleteAccountID).Find(&allResults).Error; err != nil {
			return err
		}
		for composite := range affected {
			parts := strings.SplitN(composite, "\x00", 2)
			key, metric := parts[0], parts[1]
			bestValue, bestUnit, bestName, bestResultID, bestAt := 0.0, "", "", "", time.Time{}
			for _, candidateResult := range allResults {
				parsed, parseErr := workoutResultToDomain(candidateResult)
				if parseErr != nil {
					return parseErr
				}
				for _, candidateEntry := range parsed.Entries {
					candidateMetric, value, unit := recordCandidate(candidateEntry)
					if candidateMetric != metric || movementKey(candidateEntry.Movement) != key {
						continue
					}
					if bestValue == 0 || (metric == "time" && value < bestValue) || (metric != "time" && value > bestValue) {
						bestValue, bestUnit, bestName, bestResultID, bestAt = value, unit, strings.TrimSpace(candidateEntry.Movement), candidateResult.ID, candidateResult.PerformedAt
					}
				}
			}
			if bestValue == 0 {
				if err := tx.Where("athlete_account_id = ? AND movement_key = ? AND metric = ?", model.AthleteAccountID, key, metric).Delete(&athletePersonalRecordModel{}).Error; err != nil {
					return err
				}
				continue
			}
			var current athletePersonalRecordModel
			findErr := tx.Where("athlete_account_id = ? AND movement_key = ? AND metric = ?", model.AthleteAccountID, key, metric).Take(&current).Error
			changed := errors.Is(findErr, gorm.ErrRecordNotFound) || current.BestValue != bestValue || current.SourceResultID != bestResultID
			status := "estimated"
			var confirmedAt *time.Time
			if findErr == nil && !changed {
				status, confirmedAt = current.Status, current.ConfirmedAt
			}
			record := athletePersonalRecordModel{ID: uuid.NewString(), AthleteAccountID: model.AthleteAccountID, MovementKey: key, MovementName: bestName, Metric: metric, BestValue: bestValue, Unit: bestUnit, Status: status, SourceResultID: bestResultID, AchievedAt: bestAt, ConfirmedAt: confirmedAt, CreatedAt: now, UpdatedAt: now}
			if findErr == nil {
				record.ID = current.ID
				record.CreatedAt = current.CreatedAt
			}
			if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
				return findErr
			}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "athlete_account_id"}, {Name: "movement_key"}, {Name: "metric"}}, DoUpdates: clause.AssignmentColumns([]string{"movement_name", "best_value", "unit", "status", "source_result_id", "achieved_at", "confirmed_at", "updated_at"})}).Create(&record).Error; err != nil {
				return err
			}
			if changed {
				newRecords = append(newRecords, personalRecordToDomain(record))
			}
		}
		return nil
	})
	return newRecords, err
}

func (r AthleteGormRepository) ListWorkoutResults(ctx context.Context, athleteID domain.ID) ([]domain.AthleteWorkoutResult, error) {
	var rows []athleteWorkoutResultModel
	if err := r.db.WithContext(ctx).Where("athlete_account_id = ?", string(athleteID)).Order("performed_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.AthleteWorkoutResult, 0, len(rows))
	for _, row := range rows {
		item, err := workoutResultToDomain(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r AthleteGormRepository) ListPersonalRecords(ctx context.Context, athleteID domain.ID) ([]domain.AthletePersonalRecord, error) {
	var rows []athletePersonalRecordModel
	if err := r.db.WithContext(ctx).Where("athlete_account_id = ?", string(athleteID)).Order("movement_name, metric").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.AthletePersonalRecord, 0, len(rows))
	for _, row := range rows {
		items = append(items, personalRecordToDomain(row))
	}
	return items, nil
}

func (r AthleteGormRepository) ConfirmPersonalRecord(ctx context.Context, athleteID, recordID domain.ID, now time.Time) error {
	result := r.db.WithContext(ctx).Model(&athletePersonalRecordModel{}).Where("id = ? AND athlete_account_id = ?", string(recordID), string(athleteID)).Updates(map[string]any{"status": "confirmed", "confirmed_at": now, "updated_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r AthleteGormRepository) SaveAccountToken(ctx context.Context, token *domain.AthleteAccountToken) error {
	if token.ID == "" {
		token.ID = domain.ID(uuid.NewString())
	}
	return r.db.WithContext(ctx).Create(&athleteAccountTokenModel{ID: string(token.ID), AthleteAccountID: string(token.AthleteID), Purpose: token.Purpose, TokenHash: token.TokenHash, ExpiresAt: token.ExpiresAt, CreatedAt: token.CreatedAt}).Error
}

func (r AthleteGormRepository) ConsumeAccountToken(ctx context.Context, tokenHash, purpose string, now time.Time) (*domain.AthleteAccount, error) {
	var account *domain.AthleteAccount
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var token athleteAccountTokenModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("token_hash = ? AND purpose = ? AND used_at IS NULL AND expires_at > ?", tokenHash, purpose, now).Take(&token).Error; err != nil {
			return err
		}
		if err := tx.Model(&token).Update("used_at", now).Error; err != nil {
			return err
		}
		var row athleteAccountModel
		if err := tx.Where("id = ?", token.AthleteAccountID).Take(&row).Error; err != nil {
			return err
		}
		account = athleteAccountToDomain(row)
		return nil
	})
	return account, err
}

func (r AthleteGormRepository) UpdateAthletePassword(ctx context.Context, athleteID domain.ID, passwordHash string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&athleteAccountModel{}).Where("id = ?", string(athleteID)).Updates(map[string]any{"password_hash": passwordHash, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Model(&athleteSessionModel{}).Where("athlete_account_id = ? AND revoked_at IS NULL", string(athleteID)).Update("revoked_at", now).Error
	})
}
func (r AthleteGormRepository) VerifyAthleteEmail(ctx context.Context, athleteID domain.ID, now time.Time) error {
	return r.db.WithContext(ctx).Model(&athleteAccountModel{}).Where("id = ?", string(athleteID)).Updates(map[string]any{"email_verified_at": now, "updated_at": now}).Error
}

func (r AthleteGormRepository) FindWorkoutInsight(ctx context.Context, athleteID, workoutID domain.ID, inputHash string) (*domain.AthleteWorkoutInsight, error) {
	var row athleteWorkoutInsightModel
	if err := r.db.WithContext(ctx).Where("athlete_account_id = ? AND workout_id = ? AND input_hash = ?", string(athleteID), string(workoutID), inputHash).Take(&row).Error; err != nil {
		return nil, err
	}
	return &domain.AthleteWorkoutInsight{ID: domain.ID(row.ID), AthleteID: domain.ID(row.AthleteAccountID), WorkoutID: domain.ID(row.WorkoutID), InputHash: row.InputHash, Provider: row.Provider, Model: row.Model, Body: row.Body, CreatedAt: row.CreatedAt}, nil
}

func (r AthleteGormRepository) SaveWorkoutInsight(ctx context.Context, item *domain.AthleteWorkoutInsight) error {
	if item.ID == "" {
		item.ID = domain.ID(uuid.NewString())
	}
	row := athleteWorkoutInsightModel{ID: string(item.ID), AthleteAccountID: string(item.AthleteID), WorkoutID: string(item.WorkoutID), InputHash: item.InputHash, Provider: item.Provider, Model: item.Model, Body: item.Body, CreatedAt: item.CreatedAt}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
}

func workoutResultToDomain(row athleteWorkoutResultModel) (domain.AthleteWorkoutResult, error) {
	var entries []domain.AthleteResultEntry
	if err := json.Unmarshal(row.Entries, &entries); err != nil {
		return domain.AthleteWorkoutResult{}, err
	}
	rpe := 0
	if row.RPE != nil {
		rpe = *row.RPE
	}
	return domain.AthleteWorkoutResult{ID: domain.ID(row.ID), AthleteID: domain.ID(row.AthleteAccountID), WorkoutID: domain.ID(row.WorkoutID), MembershipID: domain.ID(row.MembershipID), Scale: row.Scale, Entries: entries, RPE: rpe, Notes: row.Notes, PerformedAt: row.PerformedAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}
func personalRecordToDomain(row athletePersonalRecordModel) domain.AthletePersonalRecord {
	return domain.AthletePersonalRecord{ID: domain.ID(row.ID), AthleteID: domain.ID(row.AthleteAccountID), MovementKey: row.MovementKey, MovementName: row.MovementName, Metric: row.Metric, BestValue: row.BestValue, Unit: row.Unit, Status: row.Status, SourceResultID: domain.ID(row.SourceResultID), AchievedAt: row.AchievedAt, ConfirmedAt: row.ConfirmedAt}
}
func recordCandidate(entry domain.AthleteResultEntry) (string, float64, string) {
	switch entry.ScoreType {
	case "load":
		if entry.LoadKG > 0 {
			return "load", entry.LoadKG, "kg"
		}
	case "reps":
		if entry.Repetitions > 0 {
			return "reps", float64(entry.Repetitions), "reps"
		}
	case "time":
		if entry.TimeSeconds > 0 {
			return "time", float64(entry.TimeSeconds), "seconds"
		}
	}
	return "", 0, ""
}

var nonMovement = regexp.MustCompile(`[^a-z0-9]+`)

func movementKey(value string) string {
	decomposed := norm.NFD.String(strings.ToLower(strings.TrimSpace(value)))
	var b strings.Builder
	for _, r := range decomposed {
		if r >= 0x300 && r <= 0x36f {
			continue
		}
		b.WriteRune(r)
	}
	return strings.Trim(nonMovement.ReplaceAllString(b.String(), "-"), "-")
}
