package domain

import "time"

type AnnouncementPriority string

const (
	AnnouncementPriorityInfo      AnnouncementPriority = "info"
	AnnouncementPriorityImportant AnnouncementPriority = "important"
	AnnouncementPriorityUrgent    AnnouncementPriority = "urgent"
)

type AnnouncementStatus string

const (
	AnnouncementStatusDraft     AnnouncementStatus = "draft"
	AnnouncementStatusScheduled AnnouncementStatus = "scheduled"
	AnnouncementStatusPublished AnnouncementStatus = "published"
	AnnouncementStatusArchived  AnnouncementStatus = "archived"
)

type AnnouncementAudienceKind string

const (
	AnnouncementAudienceAll    AnnouncementAudienceKind = "all"
	AnnouncementAudienceSource AnnouncementAudienceKind = "source"
	AnnouncementAudiencePlan   AnnouncementAudienceKind = "plan"
)

// Announcement is an in-app publication. Recipients are snapshotted when it
// is published or scheduled so history is stable even if memberships change.
type Announcement struct {
	ID              ID                       `json:"id"`
	OrganizationID  ID                       `json:"-"`
	UnitID          ID                       `json:"unit_id"`
	Title           string                   `json:"title"`
	Body            string                   `json:"body"`
	Priority        AnnouncementPriority     `json:"priority"`
	Status          AnnouncementStatus       `json:"status"`
	AudienceKind    AnnouncementAudienceKind `json:"audience_kind"`
	AudienceValue   string                   `json:"audience_value,omitempty"`
	PublishAt       *time.Time               `json:"publish_at,omitempty"`
	PublishedAt     *time.Time               `json:"published_at,omitempty"`
	ArchivedAt      *time.Time               `json:"archived_at,omitempty"`
	CreatedByUserID ID                       `json:"created_by_user_id"`
	RecipientCount  int                      `json:"recipient_count"`
	ViewedCount     int                      `json:"viewed_count"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
}

type AthleteAnnouncement struct {
	ID           ID                   `json:"id"`
	MembershipID ID                   `json:"membership_id"`
	UnitID       ID                   `json:"unit_id"`
	UnitName     string               `json:"unit_name"`
	Title        string               `json:"title"`
	Body         string               `json:"body"`
	Priority     AnnouncementPriority `json:"priority"`
	PublishedAt  time.Time            `json:"published_at"`
	ViewedAt     *time.Time           `json:"viewed_at,omitempty"`
}
