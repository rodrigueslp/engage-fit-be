package repositories

import (
	"context"
	"errors"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	portrepositories "boxengage/backend/internal/ports/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type contactActivationRow struct {
	ID                string
	BoxID             string
	StudentID         *string
	StudentName       string `gorm:"->"`
	ClaimedName       string
	Source            string
	RecentCheckinDate *time.Time
	IsNewStudent      bool
	SenderPhone       string
	Phone             string
	TokenHash         string
	MatchStrategy     string
	Status            string
	ConsentVersion    string
	ConsentText       string
	ConsentedAt       *time.Time
	ExpiresAt         time.Time
	ResolvedAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (r ContactActivationGormRepository) FindPublicBox(ctx context.Context, activationCode string) (domain.ID, string, error) {
	var row struct {
		ID   string
		Name string
	}
	err := r.db.WithContext(ctx).Table("boxes").
		Select("id, name").
		Where("contact_activation_code::text = ? AND status = ?", strings.TrimSpace(activationCode), string(domain.BoxStatusActive)).
		Take(&row).Error
	return domain.ID(row.ID), row.Name, err
}

func (r ContactActivationGormRepository) ActivationCode(ctx context.Context, boxID domain.ID) (string, error) {
	var code string
	err := r.db.WithContext(ctx).Table("boxes").
		Select("contact_activation_code::text").
		Where("id = ?", stringID(boxID)).
		Scan(&code).Error
	if err == nil && code == "" {
		err = gorm.ErrRecordNotFound
	}
	return code, err
}

func (r ContactActivationGormRepository) FindActivationMatchData(ctx context.Context, boxID domain.ID, source domain.Source, recentCheckinDate time.Time) (domain.ContactActivationMatchData, error) {
	var rows []struct {
		ID               string
		Name             string
		HasRecentCheckin bool
	}
	err := r.db.WithContext(ctx).Table("students").
		Select(`students.id, students.name, EXISTS (
			SELECT 1 FROM checkins
			WHERE checkins.box_id = students.box_id
			  AND checkins.student_id = students.id
			  AND checkins.source = students.source
			  AND checkins.checkin_date = ?
		) AS has_recent_checkin`, recentCheckinDate.Format("2006-01-02")).
		Where("students.box_id = ? AND students.source = ? AND students.anonymized_at IS NULL", stringID(boxID), string(source)).
		Find(&rows).Error
	if err != nil {
		return domain.ContactActivationMatchData{}, err
	}
	result := domain.ContactActivationMatchData{Candidates: make([]domain.ContactActivationCandidate, 0, len(rows))}
	for _, row := range rows {
		result.Candidates = append(result.Candidates, domain.ContactActivationCandidate{
			Student:          domain.Student{ID: domain.ID(row.ID), BoxID: boxID, Name: row.Name, Source: source},
			HasRecentCheckin: row.HasRecentCheckin,
		})
	}
	var latest struct{ Date *time.Time }
	if err := r.db.WithContext(ctx).Table("checkins").
		Select("MAX(checkin_date) AS date").
		Where("box_id = ? AND source = ?", stringID(boxID), string(source)).
		Scan(&latest).Error; err != nil {
		return domain.ContactActivationMatchData{}, err
	}
	result.LatestCheckinDate = latest.Date
	return result, nil
}

func (r ContactActivationGormRepository) CreateActivation(ctx context.Context, activation *domain.ContactActivationRequest) error {
	if activation.ID == "" {
		activation.ID = domain.ID(uuid.NewString())
	}
	row := activationToRow(*activation)
	return r.db.WithContext(ctx).Table("contact_activation_requests").Create(&row).Error
}

func (r ContactActivationGormRepository) FindActivationByTokenHash(ctx context.Context, tokenHash string) (*domain.ContactActivationRequest, error) {
	var row contactActivationRow
	err := activationQuery(r.db.WithContext(ctx)).
		Where("contact_activation_requests.token_hash = ?", tokenHash).
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	activation := activationToDomain(row)
	return &activation, nil
}

func (r ContactActivationGormRepository) FindActivationsByPhoneAndSender(ctx context.Context, phone, senderPhone string) ([]domain.ContactActivationRequest, error) {
	var rows []contactActivationRow
	err := activationQuery(r.db.WithContext(ctx)).
		Where("contact_activation_requests.phone = ? AND contact_activation_requests.sender_phone = ?", phone, senderPhone).
		Where("contact_activation_requests.status IN ?", []string{string(domain.ContactActivationConfirmed), string(domain.ContactActivationNeedsReview), string(domain.ContactActivationCancelled)}).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]domain.ContactActivationRequest, 0, len(rows))
	for _, row := range rows {
		result = append(result, activationToDomain(row))
	}
	return result, nil
}

func (r ContactActivationGormRepository) ConfirmActivation(ctx context.Context, activationID domain.ID, phone string, confirmedAt time.Time) (*domain.ContactActivationRequest, error) {
	var result domain.ContactActivationRequest
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row contactActivationRow
		if err := activationQuery(tx).Clauses(activationLock()).Where("contact_activation_requests.id = ?", stringID(activationID)).Take(&row).Error; err != nil {
			return err
		}
		if row.Status != string(domain.ContactActivationAwaitingMessage) {
			result = activationToDomain(row)
			return nil
		}
		status := domain.ContactActivationNeedsReview
		if row.StudentID == nil && row.MatchStrategy == "awaiting_source_sync" {
			status = domain.ContactActivationPendingSync
		}
		if row.StudentID == nil && row.IsNewStudent {
			studentID, createErr := findOrCreateSelfRegisteredStudent(tx, row, phone, confirmedAt)
			if createErr != nil {
				return createErr
			}
			if studentID != "" {
				row.StudentID = pointerString(studentID)
				row.StudentName = row.ClaimedName
			}
		}
		if row.StudentID != nil {
			status = domain.ContactActivationConfirmed
			var phoneConflicts int64
			if err := tx.Table("students").
				Where("box_id = ? AND id <> ? AND phone = ? AND anonymized_at IS NULL", row.BoxID, *row.StudentID, phone).
				Count(&phoneConflicts).Error; err != nil {
				return err
			}
			if phoneConflicts > 0 {
				status = domain.ContactActivationNeedsReview
			}
		}
		updates := map[string]any{
			"phone":        phone,
			"status":       string(status),
			"consented_at": confirmedAt,
			"updated_at":   confirmedAt,
		}
		if row.StudentID != nil {
			updates["student_id"] = *row.StudentID
		}
		if status == domain.ContactActivationConfirmed {
			updates["resolved_at"] = confirmedAt
		}
		if err := tx.Table("contact_activation_requests").Where("id = ?", row.ID).Updates(updates).Error; err != nil {
			return err
		}
		if status == domain.ContactActivationConfirmed && row.StudentID != nil {
			if err := activateStudent(tx, row.BoxID, *row.StudentID, phone, confirmedAt); err != nil {
				return err
			}
		}
		consentRow := row
		if status != domain.ContactActivationConfirmed {
			consentRow.StudentID = nil
		}
		if err := createConsentEvent(tx, consentRow, phone, "opted_in", "whatsapp_self_activation", confirmedAt); err != nil {
			return err
		}
		row.Phone = phone
		row.Status = string(status)
		row.ConsentedAt = &confirmedAt
		result = activationToDomain(row)
		return nil
	})
	return &result, err
}

func (r ContactActivationGormRepository) OptOutActivations(ctx context.Context, activationIDs []domain.ID, phone string, optedOutAt time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, activationID := range activationIDs {
			var row contactActivationRow
			if err := activationQuery(tx).Clauses(activationLock()).Where("contact_activation_requests.id = ?", stringID(activationID)).Take(&row).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				return err
			}
			if err := tx.Table("contact_activation_requests").Where("id = ?", row.ID).Updates(map[string]any{
				"status":     string(domain.ContactActivationCancelled),
				"updated_at": optedOutAt,
			}).Error; err != nil {
				return err
			}
			if row.StudentID != nil {
				if err := tx.Table("students").Where("box_id = ? AND id = ?", row.BoxID, *row.StudentID).Updates(map[string]any{
					"contact_status":            string(domain.ContactStatusOptedOut),
					"contact_status_source":     "whatsapp_keyword",
					"contact_status_updated_at": optedOutAt,
					"updated_at":                optedOutAt,
				}).Error; err != nil {
					return err
				}
			}
			if err := createConsentEvent(tx, row, phone, "opted_out", "whatsapp_keyword", optedOutAt); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r ContactActivationGormRepository) ListActivations(ctx context.Context, boxID domain.ID) ([]domain.ContactActivationRequest, error) {
	var rows []contactActivationRow
	err := activationQuery(r.db.WithContext(ctx)).
		Where("contact_activation_requests.box_id = ?", stringID(boxID)).
		Order("contact_activation_requests.created_at DESC").
		Limit(200).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]domain.ContactActivationRequest, 0, len(rows))
	for _, row := range rows {
		result = append(result, activationToDomain(row))
	}
	return result, nil
}

func (r ContactActivationGormRepository) ListPendingSyncActivations(ctx context.Context, boxID domain.ID, source domain.Source) ([]domain.ContactActivationRequest, error) {
	var rows []contactActivationRow
	err := activationQuery(r.db.WithContext(ctx)).
		Where("contact_activation_requests.box_id = ? AND contact_activation_requests.source = ? AND contact_activation_requests.status = ?", stringID(boxID), string(source), string(domain.ContactActivationPendingSync)).
		Order("contact_activation_requests.created_at ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]domain.ContactActivationRequest, 0, len(rows))
	for _, row := range rows {
		result = append(result, activationToDomain(row))
	}
	return result, nil
}

func (r ContactActivationGormRepository) ResolveActivation(ctx context.Context, boxID, activationID, studentID domain.ID, matchStrategy string, resolvedAt time.Time) (*domain.ContactActivationRequest, error) {
	var result domain.ContactActivationRequest
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row contactActivationRow
		if err := activationQuery(tx).Clauses(activationLock()).
			Where("contact_activation_requests.box_id = ? AND contact_activation_requests.id = ?", stringID(boxID), stringID(activationID)).
			Take(&row).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Table("students").Where("box_id = ? AND id = ? AND source = ? AND anonymized_at IS NULL", stringID(boxID), stringID(studentID), row.Source).Count(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			return gorm.ErrRecordNotFound
		}
		status := domain.ContactActivationAwaitingMessage
		if row.Phone != "" {
			var phoneConflicts int64
			if err := tx.Table("students").
				Where("box_id = ? AND id <> ? AND phone = ? AND anonymized_at IS NULL", row.BoxID, stringID(studentID), row.Phone).
				Count(&phoneConflicts).Error; err != nil {
				return err
			}
			status = domain.ContactActivationNeedsReview
			if phoneConflicts == 0 {
				status = domain.ContactActivationConfirmed
				if err := activateStudent(tx, row.BoxID, stringID(studentID), row.Phone, resolvedAt); err != nil {
					return err
				}
			}
		}
		updates := map[string]any{
			"student_id":     stringID(studentID),
			"status":         string(status),
			"match_strategy": matchStrategy,
			"updated_at":     resolvedAt,
		}
		if status == domain.ContactActivationConfirmed || row.Phone == "" {
			updates["resolved_at"] = resolvedAt
		}
		if err := tx.Table("contact_activation_requests").Where("id = ?", row.ID).Updates(updates).Error; err != nil {
			return err
		}
		if row.Phone != "" && status == domain.ContactActivationConfirmed {
			if err := tx.Table("contact_consent_events").
				Where("activation_request_id = ? AND student_id IS NULL", row.ID).
				Update("student_id", stringID(studentID)).Error; err != nil {
				return err
			}
		}
		row.StudentID = pointerString(stringID(studentID))
		row.Status = string(status)
		row.MatchStrategy = matchStrategy
		if status == domain.ContactActivationConfirmed || row.Phone == "" {
			row.ResolvedAt = &resolvedAt
		}
		result = activationToDomain(row)
		return nil
	})
	return &result, err
}

func (r ContactActivationGormRepository) CreateStudentFromReview(ctx context.Context, boxID, activationID domain.ID, resolvedAt time.Time) (*domain.ContactActivationRequest, error) {
	var result domain.ContactActivationRequest
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row contactActivationRow
		if err := activationQuery(tx).Clauses(activationLock()).
			Where("contact_activation_requests.box_id = ? AND contact_activation_requests.id = ?", stringID(boxID), stringID(activationID)).
			Take(&row).Error; err != nil {
			return err
		}
		if row.Status != string(domain.ContactActivationNeedsReview) || row.Source != string(domain.SourceBoxMember) || row.StudentID != nil || strings.TrimSpace(row.Phone) == "" {
			return gorm.ErrRecordNotFound
		}
		var conflicts int64
		if err := tx.Table("students").
			Where("box_id = ? AND source = ? AND anonymized_at IS NULL", row.BoxID, row.Source).
			Where("phone = ? OR LOWER(REGEXP_REPLACE(BTRIM(name), '[[:space:]]+', ' ', 'g')) = ?", row.Phone, normalizeActivationName(row.ClaimedName)).
			Count(&conflicts).Error; err != nil {
			return err
		}
		if conflicts > 0 {
			return portrepositories.ErrContactActivationConflict
		}
		studentID := uuid.NewString()
		membershipStartedAt := resolvedAt
		if row.RecentCheckinDate != nil {
			membershipStartedAt = *row.RecentCheckinDate
		}
		if err := tx.Table("students").Create(map[string]any{
			"id": studentID, "box_id": row.BoxID, "name": row.ClaimedName, "phone": row.Phone,
			"source": row.Source, "external_id": "self-registration:" + studentID,
			"risk_status": string(domain.StudentRiskStatusActive), "contact_status": string(domain.ContactStatusOptedIn),
			"contact_status_source": "whatsapp_self_activation", "contact_status_updated_at": resolvedAt,
			"membership_started_at": membershipStartedAt, "membership_started_source": "self_registration",
			"created_at": resolvedAt, "updated_at": resolvedAt,
		}).Error; err != nil {
			return err
		}
		if err := tx.Table("contact_activation_requests").Where("id = ?", row.ID).Updates(map[string]any{
			"student_id": studentID, "status": string(domain.ContactActivationConfirmed),
			"match_strategy": "manual_create_box_member", "resolved_at": resolvedAt, "updated_at": resolvedAt,
		}).Error; err != nil {
			return err
		}
		if err := tx.Table("contact_consent_events").
			Where("activation_request_id = ? AND student_id IS NULL", row.ID).
			Update("student_id", studentID).Error; err != nil {
			return err
		}
		row.StudentID = pointerString(studentID)
		row.StudentName = row.ClaimedName
		row.Status = string(domain.ContactActivationConfirmed)
		row.MatchStrategy = "manual_create_box_member"
		row.ResolvedAt = &resolvedAt
		row.UpdatedAt = resolvedAt
		result = activationToDomain(row)
		return nil
	})
	return &result, err
}

func (r ContactActivationGormRepository) CancelReview(ctx context.Context, boxID, activationID domain.ID, resolvedAt time.Time) (*domain.ContactActivationRequest, error) {
	var result domain.ContactActivationRequest
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row contactActivationRow
		if err := activationQuery(tx).Clauses(activationLock()).
			Where("contact_activation_requests.box_id = ? AND contact_activation_requests.id = ?", stringID(boxID), stringID(activationID)).
			Take(&row).Error; err != nil {
			return err
		}
		if row.Status != string(domain.ContactActivationNeedsReview) {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Table("contact_activation_requests").Where("id = ?", row.ID).Updates(map[string]any{
			"status": string(domain.ContactActivationCancelled), "match_strategy": "manual_discard",
			"resolved_at": resolvedAt, "updated_at": resolvedAt,
		}).Error; err != nil {
			return err
		}
		row.Status = string(domain.ContactActivationCancelled)
		row.MatchStrategy = "manual_discard"
		row.ResolvedAt = &resolvedAt
		row.UpdatedAt = resolvedAt
		result = activationToDomain(row)
		return nil
	})
	return &result, err
}

func (r ContactActivationGormRepository) MarkActivationNeedsReview(ctx context.Context, boxID, activationID domain.ID, matchStrategy string, updatedAt time.Time) error {
	return r.db.WithContext(ctx).Table("contact_activation_requests").
		Where("box_id = ? AND id = ? AND status = ?", stringID(boxID), stringID(activationID), string(domain.ContactActivationPendingSync)).
		Updates(map[string]any{"status": string(domain.ContactActivationNeedsReview), "match_strategy": matchStrategy, "updated_at": updatedAt}).Error
}

func (r ContactActivationGormRepository) Summary(ctx context.Context, boxID domain.ID) (domain.ContactActivationSummary, error) {
	var row struct {
		TotalStudents   int64
		WithPhone       int64
		OptedIn         int64
		OptedOut        int64
		PendingReview   int64
		PendingSync     int64
		AwaitingMessage int64
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			COUNT(*) FILTER (WHERE s.anonymized_at IS NULL) AS total_students,
			COUNT(*) FILTER (WHERE s.anonymized_at IS NULL AND s.phone <> '') AS with_phone,
			COUNT(*) FILTER (WHERE s.anonymized_at IS NULL AND s.contact_status = 'opted_in') AS opted_in,
			COUNT(*) FILTER (WHERE s.anonymized_at IS NULL AND s.contact_status = 'opted_out') AS opted_out,
			(SELECT COUNT(*) FROM contact_activation_requests car WHERE car.box_id = ? AND car.status = 'needs_review') AS pending_review,
			(SELECT COUNT(*) FROM contact_activation_requests car WHERE car.box_id = ? AND car.status = 'pending_sync') AS pending_sync,
			(SELECT COUNT(*) FROM contact_activation_requests car WHERE car.box_id = ? AND car.status = 'awaiting_message' AND car.expires_at > NOW()) AS awaiting_message
		FROM students s
		WHERE s.box_id = ?
	`, stringID(boxID), stringID(boxID), stringID(boxID), stringID(boxID)).Scan(&row).Error
	return domain.ContactActivationSummary{
		TotalStudents: row.TotalStudents, WithPhone: row.WithPhone, OptedIn: row.OptedIn,
		OptedOut: row.OptedOut, PendingReview: row.PendingReview, PendingSync: row.PendingSync, AwaitingMessage: row.AwaitingMessage,
	}, err
}

func activationQuery(db *gorm.DB) *gorm.DB {
	return db.Table("contact_activation_requests").
		Select("contact_activation_requests.*, COALESCE(students.name, '') AS student_name").
		Joins("LEFT JOIN students ON students.id = contact_activation_requests.student_id")
}

func activationLock() clause.Locking {
	return clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "contact_activation_requests"}}
}

func activationToRow(value domain.ContactActivationRequest) contactActivationRow {
	var studentID *string
	if value.StudentID != "" {
		studentID = pointerString(stringID(value.StudentID))
	}
	return contactActivationRow{
		ID: stringID(value.ID), BoxID: stringID(value.BoxID), StudentID: studentID,
		ClaimedName: value.ClaimedName, Source: string(value.Source), RecentCheckinDate: value.RecentCheckinDate, IsNewStudent: value.IsNewStudent,
		SenderPhone: value.SenderPhone, Phone: value.Phone, TokenHash: value.TokenHash, Status: string(value.Status),
		MatchStrategy:  value.MatchStrategy,
		ConsentVersion: value.ConsentVersion, ConsentText: value.ConsentText, ConsentedAt: value.ConsentedAt,
		ExpiresAt: value.ExpiresAt, ResolvedAt: value.ResolvedAt, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func activationToDomain(row contactActivationRow) domain.ContactActivationRequest {
	studentID := domain.ID("")
	if row.StudentID != nil {
		studentID = domain.ID(*row.StudentID)
	}
	return domain.ContactActivationRequest{
		ID: domain.ID(row.ID), BoxID: domain.ID(row.BoxID), StudentID: studentID, StudentName: row.StudentName,
		ClaimedName: row.ClaimedName, Source: domain.Source(row.Source), RecentCheckinDate: row.RecentCheckinDate, IsNewStudent: row.IsNewStudent,
		SenderPhone: row.SenderPhone, Phone: row.Phone, TokenHash: row.TokenHash, MatchStrategy: row.MatchStrategy,
		Status: domain.ContactActivationStatus(row.Status), ConsentVersion: row.ConsentVersion,
		ConsentText: row.ConsentText, ConsentedAt: row.ConsentedAt, ExpiresAt: row.ExpiresAt,
		ResolvedAt: row.ResolvedAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func activateStudent(tx *gorm.DB, boxID, studentID, phone string, at time.Time) error {
	return tx.Table("students").Where("box_id = ? AND id = ? AND anonymized_at IS NULL", boxID, studentID).Updates(map[string]any{
		"phone":                     phone,
		"contact_status":            string(domain.ContactStatusOptedIn),
		"contact_status_source":     "whatsapp_self_activation",
		"contact_status_updated_at": at,
		"updated_at":                at,
	}).Error
}

func findOrCreateSelfRegisteredStudent(tx *gorm.DB, row contactActivationRow, phone string, at time.Time) (string, error) {
	var candidates []struct {
		ID    string
		Phone string
	}
	err := tx.Table("students").
		Select("id, phone").
		Where("box_id = ? AND source = ? AND anonymized_at IS NULL", row.BoxID, row.Source).
		Where("phone = ? OR LOWER(REGEXP_REPLACE(BTRIM(name), '[[:space:]]+', ' ', 'g')) = ?", phone, normalizeActivationName(row.ClaimedName)).
		Find(&candidates).Error
	if err != nil {
		return "", err
	}
	if len(candidates) == 1 && candidates[0].Phone == phone {
		return candidates[0].ID, nil
	}
	if len(candidates) > 0 {
		return "", nil
	}
	studentID := uuid.NewString()
	err = tx.Table("students").Create(map[string]any{
		"id": studentID, "box_id": row.BoxID, "name": row.ClaimedName, "phone": phone,
		"source": row.Source, "external_id": "self-registration:" + studentID,
		"risk_status": string(domain.StudentRiskStatusActive), "contact_status": string(domain.ContactStatusUnknown),
		"membership_started_at": at, "membership_started_source": "self_registration",
		"created_at": at, "updated_at": at,
	}).Error
	return studentID, err
}

func createConsentEvent(tx *gorm.DB, row contactActivationRow, phone, action, source string, at time.Time) error {
	return tx.Table("contact_consent_events").Create(map[string]any{
		"id": uuid.NewString(), "box_id": row.BoxID, "student_id": row.StudentID,
		"activation_request_id": row.ID, "phone": phone, "action": action, "source": source,
		"consent_version": row.ConsentVersion, "consent_text": row.ConsentText, "created_at": at,
	}).Error
}

func normalizeActivationName(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func pointerString(value string) *string { return &value }
