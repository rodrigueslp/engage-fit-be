package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
)

type MembershipAccessRepository interface {
	ResolveMembershipAccessCandidates(context.Context, domain.MembershipAccessRequest) ([]domain.MembershipAccessCandidate, error)
	HasActiveEnrollmentInUnit(context.Context, domain.ID, domain.ID, domain.ID, time.Time) (bool, error)
	HasEnrollmentInUnit(context.Context, domain.ID, domain.ID, domain.ID) (bool, error)
	HasEnrollmentOutsideRequestedUnit(context.Context, domain.ID, domain.ID, domain.ID, time.Time) (bool, error)
}

type LegacyMembershipReader interface {
	HasLegacyMembership(context.Context, domain.ID, domain.ID, domain.ID) (bool, error)
}

type MembershipAccessUsageReader interface {
	LockMembershipAccessUsage(context.Context, domain.ID, domain.ID, domain.ID) error
	CountMembershipAccessUsage(context.Context, domain.MembershipAccessUsageQuery) (int, error)
}

type MembershipAccessPrerequisiteReader interface {
	ResolveMembershipAccessPrerequisites(context.Context, domain.ID, domain.ID, domain.ID, domain.ID, time.Time) (domain.MembershipAccessPrerequisites, error)
}
