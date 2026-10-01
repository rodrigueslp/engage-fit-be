package announcements

import (
	"context"
	"errors"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"github.com/google/uuid"
)

var (
	ErrInvalidInput = errors.New("invalid announcement input")
	ErrNotFound     = repositories.ErrAnnouncementNotFound
	ErrConflict     = repositories.ErrAnnouncementConflict
)

type SaveInput struct {
	Title         string `json:"title"`
	Body          string `json:"body"`
	Priority      string `json:"priority"`
	AudienceKind  string `json:"audience_kind"`
	AudienceValue string `json:"audience_value"`
}

type PublishInput struct {
	PublishAt *time.Time `json:"publish_at"`
}

type Service struct {
	repository repositories.AnnouncementRepository
	now        func() time.Time
}

func NewService(repository repositories.AnnouncementRepository) *Service {
	return &Service{repository: repository, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) List(ctx context.Context, unitID domain.ID) ([]domain.Announcement, error) {
	organizationID, err := s.repository.UnitOrganization(ctx, unitID)
	if err != nil {
		return nil, err
	}
	return s.repository.List(ctx, organizationID, unitID)
}

func (s *Service) Create(ctx context.Context, unitID, actorID domain.ID, input SaveInput) (*domain.Announcement, error) {
	organizationID, err := s.repository.UnitOrganization(ctx, unitID)
	if err != nil {
		return nil, err
	}
	value := &domain.Announcement{ID: domain.ID(uuid.NewString()), OrganizationID: organizationID, UnitID: unitID, CreatedByUserID: actorID, Status: domain.AnnouncementStatusDraft, CreatedAt: s.now(), UpdatedAt: s.now()}
	if err := apply(value, input); err != nil {
		return nil, err
	}
	if err := s.repository.Create(ctx, value); err != nil {
		return nil, err
	}
	return value, nil
}

func (s *Service) Update(ctx context.Context, unitID, announcementID domain.ID, input SaveInput) (*domain.Announcement, error) {
	organizationID, err := s.repository.UnitOrganization(ctx, unitID)
	if err != nil {
		return nil, err
	}
	value, err := s.repository.Get(ctx, organizationID, unitID, announcementID)
	if err != nil {
		return nil, err
	}
	if value.Status != domain.AnnouncementStatusDraft {
		return nil, ErrConflict
	}
	if err := apply(value, input); err != nil {
		return nil, err
	}
	value.UpdatedAt = s.now()
	if err := s.repository.UpdateDraft(ctx, value); err != nil {
		return nil, err
	}
	return value, nil
}

func (s *Service) Publish(ctx context.Context, unitID, announcementID domain.ID, input PublishInput) (*domain.Announcement, error) {
	organizationID, err := s.repository.UnitOrganization(ctx, unitID)
	if err != nil {
		return nil, err
	}
	value, err := s.repository.Get(ctx, organizationID, unitID, announcementID)
	if err != nil {
		return nil, err
	}
	if value.Status != domain.AnnouncementStatusDraft {
		return nil, ErrConflict
	}
	now := s.now()
	if input.PublishAt != nil && input.PublishAt.Before(now.Add(-time.Minute)) {
		return nil, ErrInvalidInput
	}
	if input.PublishAt != nil && input.PublishAt.After(now) {
		publishAt := input.PublishAt.UTC()
		value.PublishAt = &publishAt
		value.Status = domain.AnnouncementStatusScheduled
	} else {
		value.PublishAt = &now
		value.PublishedAt = &now
		value.Status = domain.AnnouncementStatusPublished
	}
	value.UpdatedAt = now
	if err := s.repository.Publish(ctx, value); err != nil {
		return nil, err
	}
	return s.repository.Get(ctx, organizationID, unitID, announcementID)
}

func (s *Service) Archive(ctx context.Context, unitID, announcementID domain.ID) error {
	organizationID, err := s.repository.UnitOrganization(ctx, unitID)
	if err != nil {
		return err
	}
	return s.repository.Archive(ctx, organizationID, unitID, announcementID, s.now())
}

func (s *Service) AthleteList(ctx context.Context, athleteID domain.ID) ([]domain.AthleteAnnouncement, error) {
	return s.repository.ListForAthlete(ctx, athleteID, s.now())
}

func (s *Service) MarkViewed(ctx context.Context, athleteID, announcementID domain.ID) error {
	return s.repository.MarkViewed(ctx, athleteID, announcementID, s.now())
}

func apply(value *domain.Announcement, input SaveInput) error {
	value.Title, value.Body = strings.TrimSpace(input.Title), strings.TrimSpace(input.Body)
	value.Priority = domain.AnnouncementPriority(input.Priority)
	value.AudienceKind = domain.AnnouncementAudienceKind(input.AudienceKind)
	value.AudienceValue = strings.TrimSpace(input.AudienceValue)
	if len(value.Title) < 3 || len(value.Title) > 120 || len(value.Body) < 3 || len(value.Body) > 4000 {
		return ErrInvalidInput
	}
	if value.Priority != domain.AnnouncementPriorityInfo && value.Priority != domain.AnnouncementPriorityImportant && value.Priority != domain.AnnouncementPriorityUrgent {
		return ErrInvalidInput
	}
	switch value.AudienceKind {
	case domain.AnnouncementAudienceAll:
		value.AudienceValue = ""
	case domain.AnnouncementAudienceSource:
		if value.AudienceValue != string(domain.SourceBoxMember) && value.AudienceValue != string(domain.SourceWellhub) && value.AudienceValue != string(domain.SourceTotalPass) {
			return ErrInvalidInput
		}
	case domain.AnnouncementAudiencePlan:
		if _, err := uuid.Parse(value.AudienceValue); err != nil {
			return ErrInvalidInput
		}
	default:
		return ErrInvalidInput
	}
	return nil
}
