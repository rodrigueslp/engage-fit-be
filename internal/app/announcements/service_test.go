package announcements

import (
	"context"
	"testing"
	"time"

	"boxengage/backend/internal/domain"
)

type announcementRepositoryStub struct {
	item       *domain.Announcement
	recipients int
}

func (r *announcementRepositoryStub) UnitOrganization(context.Context, domain.ID) (domain.ID, error) {
	return "org-1", nil
}
func (r *announcementRepositoryStub) List(context.Context, domain.ID, domain.ID) ([]domain.Announcement, error) {
	if r.item == nil {
		return []domain.Announcement{}, nil
	}
	return []domain.Announcement{*r.item}, nil
}
func (r *announcementRepositoryStub) Get(context.Context, domain.ID, domain.ID, domain.ID) (*domain.Announcement, error) {
	value := *r.item
	value.RecipientCount = r.recipients
	return &value, nil
}
func (r *announcementRepositoryStub) Create(_ context.Context, value *domain.Announcement) error {
	copy := *value
	r.item = &copy
	return nil
}
func (r *announcementRepositoryStub) UpdateDraft(_ context.Context, value *domain.Announcement) error {
	copy := *value
	r.item = &copy
	return nil
}
func (r *announcementRepositoryStub) Publish(_ context.Context, value *domain.Announcement) error {
	copy := *value
	r.item = &copy
	r.recipients = 3
	return nil
}
func (r *announcementRepositoryStub) Archive(context.Context, domain.ID, domain.ID, domain.ID, time.Time) error {
	r.item.Status = domain.AnnouncementStatusArchived
	return nil
}
func (r *announcementRepositoryStub) ListForAthlete(context.Context, domain.ID, time.Time) ([]domain.AthleteAnnouncement, error) {
	return []domain.AthleteAnnouncement{}, nil
}
func (r *announcementRepositoryStub) MarkViewed(context.Context, domain.ID, domain.ID, time.Time) error {
	return nil
}

func TestCreateAndPublishAnnouncement(t *testing.T) {
	repository := &announcementRepositoryStub{}
	service := NewService(repository)
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	item, err := service.Create(context.Background(), "unit-1", "owner-1", SaveInput{Title: "Aula especial", Body: "Neste sábado teremos um horário especial.", Priority: "important", AudienceKind: "source", AudienceValue: "box_member"})
	if err != nil {
		t.Fatal(err)
	}
	if item.Status != domain.AnnouncementStatusDraft || item.OrganizationID != "org-1" {
		t.Fatalf("unexpected draft: %+v", item)
	}
	published, err := service.Publish(context.Background(), "unit-1", item.ID, PublishInput{})
	if err != nil {
		t.Fatal(err)
	}
	if published.Status != domain.AnnouncementStatusPublished || published.PublishedAt == nil || published.RecipientCount != 3 {
		t.Fatalf("unexpected publication: %+v", published)
	}
}

func TestAnnouncementValidationAndScheduling(t *testing.T) {
	repository := &announcementRepositoryStub{}
	service := NewService(repository)
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	if _, err := service.Create(context.Background(), "unit-1", "owner-1", SaveInput{Title: "Oi", Body: "curto", Priority: "info", AudienceKind: "all"}); err != ErrInvalidInput {
		t.Fatalf("expected invalid input, got %v", err)
	}
	item, err := service.Create(context.Background(), "unit-1", "owner-1", SaveInput{Title: "Mudança de horário", Body: "Confira o novo horário da próxima semana.", Priority: "urgent", AudienceKind: "all"})
	if err != nil {
		t.Fatal(err)
	}
	future := now.Add(24 * time.Hour)
	scheduled, err := service.Publish(context.Background(), "unit-1", item.ID, PublishInput{PublishAt: &future})
	if err != nil {
		t.Fatal(err)
	}
	if scheduled.Status != domain.AnnouncementStatusScheduled || scheduled.PublishAt == nil || scheduled.PublishedAt != nil {
		t.Fatalf("unexpected schedule: %+v", scheduled)
	}
}
