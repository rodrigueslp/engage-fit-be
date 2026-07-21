package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
)

type StudentFilters struct {
	Source          *domain.Source
	Search          string
	CampaignID      *domain.ID
	Achieved        *bool
	NearGoal        *bool
	Inactive        *bool
	ContactableOnly bool
	Page            int
	Limit           int
}

type StudentRepository interface {
	FindByID(ctx context.Context, boxID, id domain.ID) (*domain.Student, error)
	FindByExternalID(ctx context.Context, boxID domain.ID, source domain.Source, externalID string) (*domain.Student, error)
	List(ctx context.Context, boxID domain.ID, filters StudentFilters) ([]domain.Student, error)
	Save(ctx context.Context, student *domain.Student) error
	UpdateRiskStatus(ctx context.Context, boxID, id domain.ID, status domain.StudentRiskStatus) error
	MarkRiskMessageSent(ctx context.Context, boxID, id domain.ID, sentAt time.Time) error
	UpdateContactPreference(ctx context.Context, boxID, id domain.ID, status domain.ContactStatus, source string, updatedAt time.Time) error
}
