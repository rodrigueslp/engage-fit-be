package repositories

import (
	"context"
	"fmt"

	"boxengage/backend/internal/domain"
	"gorm.io/gorm"
)

type MembershipAccessUsageGormAdapter struct{ db *gorm.DB }

func NewMembershipAccessUsageGormAdapter(db *gorm.DB) MembershipAccessUsageGormAdapter {
	return MembershipAccessUsageGormAdapter{db: db}
}

func (r MembershipAccessUsageGormAdapter) LockMembershipAccessUsage(ctx context.Context, organizationID, personID, entitlementID domain.ID) error {
	return databaseForContext(r.db, ctx).Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", fmt.Sprintf("membership-access:%s:%s:%s", organizationID, personID, entitlementID)).Error
}

func (r MembershipAccessUsageGormAdapter) CountMembershipAccessUsage(ctx context.Context, query domain.MembershipAccessUsageQuery) (int, error) {
	var count int
	err := databaseForContext(r.db, ctx).Raw(`
		SELECT COUNT(*) FROM (
			SELECT r.id::text AS usage_key
			FROM reservations r
			JOIN scheduling_sessions ss ON ss.id = r.session_id AND ss.organization_id = r.organization_id
			WHERE r.organization_id = ? AND r.person_id = ? AND r.enrollment_id = ? AND r.entitlement_id = ?
			  AND r.status IN ('confirmed', 'no_show') AND ss.starts_at >= ? AND ss.starts_at < ?
			UNION ALL
			SELECT ar.id::text
			FROM attendance_records ar
			JOIN attendance_access_decisions ad ON ad.id = ar.decision_id AND ad.organization_id = ar.organization_id
			WHERE ar.organization_id = ? AND ar.person_id = ? AND ad.enrollment_id = ? AND ad.entitlement_id = ?
			  AND ar.session_id IS NULL AND ar.occurred_at >= ? AND ar.occurred_at < ?
		) usage`, stringID(query.OrganizationID), stringID(query.PersonID), stringID(query.EnrollmentID), stringID(query.EntitlementID), query.From, query.Until, stringID(query.OrganizationID), stringID(query.PersonID), stringID(query.EnrollmentID), stringID(query.EntitlementID), query.From, query.Until).Scan(&count).Error
	return count, err
}
