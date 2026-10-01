package repositories

import (
	"context"
	"errors"
	"time"

	"boxengage/backend/internal/domain"
)

var (
	ErrAnnouncementNotFound = errors.New("announcement not found")
	ErrAnnouncementConflict = errors.New("announcement conflict")
)

type AnnouncementRepository interface {
	UnitOrganization(context.Context, domain.ID) (domain.ID, error)
	List(context.Context, domain.ID, domain.ID) ([]domain.Announcement, error)
	Get(context.Context, domain.ID, domain.ID, domain.ID) (*domain.Announcement, error)
	Create(context.Context, *domain.Announcement) error
	UpdateDraft(context.Context, *domain.Announcement) error
	Publish(context.Context, *domain.Announcement) error
	Archive(context.Context, domain.ID, domain.ID, domain.ID, time.Time) error
	ListForAthlete(context.Context, domain.ID, time.Time) ([]domain.AthleteAnnouncement, error)
	MarkViewed(context.Context, domain.ID, domain.ID, time.Time) error
}
