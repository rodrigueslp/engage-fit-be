package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
)

type AthletePortalRepository interface {
	LoadAthletePortal(ctx context.Context, athleteID domain.ID, from, until time.Time) ([]domain.AthletePortalUnit, error)
	ResolveAthleteScope(ctx context.Context, athleteID, membershipID domain.ID) (*domain.AthletePortalScope, error)
}
