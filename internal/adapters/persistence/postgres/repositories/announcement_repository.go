package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"gorm.io/gorm"
)

type AnnouncementGormRepository struct{ db *gorm.DB }

func NewAnnouncementGormRepository(db *gorm.DB) AnnouncementGormRepository {
	return AnnouncementGormRepository{db: db}
}

type announcementRow struct {
	ID, OrganizationID, UnitID, Title, Body, Priority, Status, AudienceKind, AudienceValue, CreatedByUserID string
	PublishAt, PublishedAt, ArchivedAt                                                                      *time.Time
	RecipientCount, ViewedCount                                                                             int
	CreatedAt, UpdatedAt                                                                                    time.Time
}

func (r AnnouncementGormRepository) UnitOrganization(ctx context.Context, unitID domain.ID) (domain.ID, error) {
	var organizationID string
	result := databaseForContext(r.db, ctx).Raw(`SELECT organization_id FROM boxes WHERE id = ? AND status = 'active'`, stringID(unitID)).Scan(&organizationID)
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected == 0 || organizationID == "" {
		return "", portrepo.ErrAnnouncementNotFound
	}
	return domain.ID(organizationID), nil
}

func (r AnnouncementGormRepository) List(ctx context.Context, organizationID, unitID domain.ID) ([]domain.Announcement, error) {
	db := databaseForContext(r.db, ctx)
	if err := db.Exec(`UPDATE announcements SET status = 'published', published_at = publish_at, updated_at = NOW() WHERE organization_id = ? AND unit_id = ? AND status = 'scheduled' AND publish_at <= NOW()`, stringID(organizationID), stringID(unitID)).Error; err != nil {
		return nil, err
	}
	var rows []announcementRow
	err := db.Raw(announcementSelect+` WHERE a.organization_id = ? AND a.unit_id = ? GROUP BY a.id ORDER BY a.created_at DESC`, stringID(organizationID), stringID(unitID)).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	values := make([]domain.Announcement, 0, len(rows))
	for _, row := range rows {
		values = append(values, mapAnnouncement(row))
	}
	return values, nil
}

func (r AnnouncementGormRepository) Get(ctx context.Context, organizationID, unitID, announcementID domain.ID) (*domain.Announcement, error) {
	var row announcementRow
	result := databaseForContext(r.db, ctx).Raw(announcementSelect+` WHERE a.organization_id = ? AND a.unit_id = ? AND a.id = ? GROUP BY a.id`, stringID(organizationID), stringID(unitID), stringID(announcementID)).Scan(&row)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 || row.ID == "" {
		return nil, portrepo.ErrAnnouncementNotFound
	}
	value := mapAnnouncement(row)
	return &value, nil
}

func (r AnnouncementGormRepository) Create(ctx context.Context, value *domain.Announcement) error {
	return databaseForContext(r.db, ctx).Exec(`INSERT INTO announcements (id, organization_id, unit_id, title, body, priority, status, audience_kind, audience_value, created_by_user_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, 'draft', ?, NULLIF(?, ''), ?, ?, ?)`, stringID(value.ID), stringID(value.OrganizationID), stringID(value.UnitID), value.Title, value.Body, value.Priority, value.AudienceKind, value.AudienceValue, stringID(value.CreatedByUserID), value.CreatedAt, value.UpdatedAt).Error
}

func (r AnnouncementGormRepository) UpdateDraft(ctx context.Context, value *domain.Announcement) error {
	result := databaseForContext(r.db, ctx).Exec(`UPDATE announcements SET title = ?, body = ?, priority = ?, audience_kind = ?, audience_value = NULLIF(?, ''), updated_at = ? WHERE id = ? AND organization_id = ? AND unit_id = ? AND status = 'draft'`, value.Title, value.Body, value.Priority, value.AudienceKind, value.AudienceValue, value.UpdatedAt, stringID(value.ID), stringID(value.OrganizationID), stringID(value.UnitID))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return portrepo.ErrAnnouncementConflict
	}
	return nil
}

func (r AnnouncementGormRepository) Publish(ctx context.Context, value *domain.Announcement) error {
	db := databaseForContext(r.db, ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		if value.AudienceKind == domain.AnnouncementAudiencePlan {
			var count int64
			if err := tx.Table("membership_plans").Where("id = ? AND organization_id = ? AND home_unit_id = ? AND status = 'active'", value.AudienceValue, stringID(value.OrganizationID), stringID(value.UnitID)).Count(&count).Error; err != nil {
				return err
			}
			if count != 1 {
				return portrepo.ErrAnnouncementConflict
			}
		}
		result := tx.Exec(`UPDATE announcements SET status = ?, publish_at = ?, published_at = ?, updated_at = ? WHERE id = ? AND organization_id = ? AND unit_id = ? AND status = 'draft'`, value.Status, value.PublishAt, value.PublishedAt, value.UpdatedAt, stringID(value.ID), stringID(value.OrganizationID), stringID(value.UnitID))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return portrepo.ErrAnnouncementConflict
		}
		return tx.Exec(`
			INSERT INTO announcement_recipients (id, announcement_id, organization_id, unit_id, athlete_account_id, athlete_membership_id, person_id, created_at)
			SELECT gen_random_uuid(), ?, ?, ?, abm.athlete_account_id, abm.id, s.person_id, ?
			FROM athlete_box_memberships abm
			JOIN athlete_student_links asl ON asl.membership_id = abm.id
			JOIN students s ON s.id = asl.student_id AND s.box_id = abm.box_id AND s.anonymized_at IS NULL
			WHERE abm.box_id = ? AND abm.status = 'active' AND s.person_id IS NOT NULL
			  AND (? = 'all' OR (? = 'source' AND s.source = ?) OR (? = 'plan' AND EXISTS (
				SELECT 1 FROM enrollments e JOIN membership_plan_versions pv ON pv.id = e.plan_version_id AND pv.organization_id = e.organization_id
				WHERE e.organization_id = ? AND e.home_unit_id = ? AND e.person_id = s.person_id AND e.lifecycle_status = 'active' AND pv.plan_id::text = ?
			  )))
			ON CONFLICT (announcement_id, athlete_account_id, athlete_membership_id) DO NOTHING`,
			stringID(value.ID), stringID(value.OrganizationID), stringID(value.UnitID), value.UpdatedAt,
			stringID(value.UnitID), value.AudienceKind, value.AudienceKind, value.AudienceValue,
			value.AudienceKind, stringID(value.OrganizationID), stringID(value.UnitID), value.AudienceValue).Error
	})
}

func (r AnnouncementGormRepository) Archive(ctx context.Context, organizationID, unitID, announcementID domain.ID, at time.Time) error {
	result := databaseForContext(r.db, ctx).Exec(`UPDATE announcements SET status = 'archived', archived_at = ?, updated_at = ? WHERE id = ? AND organization_id = ? AND unit_id = ? AND status IN ('scheduled','published')`, at, at, stringID(announcementID), stringID(organizationID), stringID(unitID))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return portrepo.ErrAnnouncementConflict
	}
	return nil
}

func (r AnnouncementGormRepository) ListForAthlete(ctx context.Context, athleteID domain.ID, now time.Time) ([]domain.AthleteAnnouncement, error) {
	db := databaseForContext(r.db, ctx)
	if err := db.Exec(`UPDATE announcements SET status = 'published', published_at = publish_at, updated_at = ? WHERE status = 'scheduled' AND publish_at <= ?`, now, now).Error; err != nil {
		return nil, err
	}
	type row struct {
		ID, MembershipID, UnitID, UnitName, Title, Body, Priority string
		PublishedAt                                               time.Time
		ViewedAt                                                  *time.Time
	}
	var rows []row
	err := db.Raw(`
		SELECT a.id, ar.athlete_membership_id AS membership_id, a.unit_id, b.name AS unit_name,
			a.title, a.body, a.priority, a.published_at, ar.first_viewed_at AS viewed_at
		FROM announcement_recipients ar
		JOIN announcements a ON a.id = ar.announcement_id AND a.organization_id = ar.organization_id AND a.unit_id = ar.unit_id
		JOIN boxes b ON b.id = a.unit_id
		WHERE ar.athlete_account_id = ? AND a.status = 'published' AND a.published_at <= ?
		ORDER BY CASE a.priority WHEN 'urgent' THEN 0 WHEN 'important' THEN 1 ELSE 2 END, a.published_at DESC`, stringID(athleteID), now).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	values := make([]domain.AthleteAnnouncement, 0, len(rows))
	for _, row := range rows {
		values = append(values, domain.AthleteAnnouncement{ID: domain.ID(row.ID), MembershipID: domain.ID(row.MembershipID), UnitID: domain.ID(row.UnitID), UnitName: row.UnitName, Title: row.Title, Body: row.Body, Priority: domain.AnnouncementPriority(row.Priority), PublishedAt: row.PublishedAt, ViewedAt: row.ViewedAt})
	}
	return values, nil
}

func (r AnnouncementGormRepository) MarkViewed(ctx context.Context, athleteID, announcementID domain.ID, at time.Time) error {
	result := databaseForContext(r.db, ctx).Exec(`UPDATE announcement_recipients ar SET first_viewed_at = COALESCE(first_viewed_at, ?), last_viewed_at = ?, view_count = view_count + 1 WHERE ar.announcement_id = ? AND ar.athlete_account_id = ? AND EXISTS (SELECT 1 FROM announcements a WHERE a.id = ar.announcement_id AND a.status = 'published' AND a.published_at <= ?)`, at, at, stringID(announcementID), stringID(athleteID), at)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return portrepo.ErrAnnouncementNotFound
	}
	return nil
}

const announcementSelect = `SELECT a.id, a.organization_id, a.unit_id, a.title, a.body, a.priority, a.status, a.audience_kind, COALESCE(a.audience_value, '') AS audience_value, a.publish_at, a.published_at, a.archived_at, a.created_by_user_id, a.created_at, a.updated_at, COUNT(ar.id)::int AS recipient_count, COUNT(ar.first_viewed_at)::int AS viewed_count FROM announcements a LEFT JOIN announcement_recipients ar ON ar.announcement_id = a.id`

func mapAnnouncement(row announcementRow) domain.Announcement {
	return domain.Announcement{ID: domain.ID(row.ID), OrganizationID: domain.ID(row.OrganizationID), UnitID: domain.ID(row.UnitID), Title: row.Title, Body: row.Body, Priority: domain.AnnouncementPriority(row.Priority), Status: domain.AnnouncementStatus(row.Status), AudienceKind: domain.AnnouncementAudienceKind(row.AudienceKind), AudienceValue: row.AudienceValue, PublishAt: row.PublishAt, PublishedAt: row.PublishedAt, ArchivedAt: row.ArchivedAt, CreatedByUserID: domain.ID(row.CreatedByUserID), RecipientCount: row.RecipientCount, ViewedCount: row.ViewedCount, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

var _ portrepo.AnnouncementRepository = AnnouncementGormRepository{}
