package domain

import "time"

type ContactStatus string

type RetentionMonitoringStatus string

const (
	RetentionMonitoringMonitored RetentionMonitoringStatus = "monitored"
	RetentionMonitoringExcluded  RetentionMonitoringStatus = "excluded"
)

const (
	ContactStatusUnknown  ContactStatus = "unknown"
	ContactStatusOptedIn  ContactStatus = "opted_in"
	ContactStatusOptedOut ContactStatus = "opted_out"
)

type Student struct {
	ID                        ID
	BoxID                     ID
	Name                      string
	Email                     string
	Phone                     string
	Source                    Source
	ExternalID                string
	RiskStatus                StudentRiskStatus
	RiskLastMessageAt         *time.Time
	ContactStatus             ContactStatus
	ContactStatusUpdatedAt    *time.Time
	ContactStatusSource       string
	MembershipStartedAt       *time.Time
	MembershipStartedSource   string
	RetentionMonitoringStatus RetentionMonitoringStatus
	RetentionExclusionReason  string
	RetentionExcludedUntil    *time.Time
	RetentionExcludedAt       *time.Time
	RetentionExcludedByUserID ID
	AnonymizedAt              *time.Time
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

func (s Student) CanContact() bool {
	return s.AnonymizedAt == nil && s.ContactStatus == ContactStatusOptedIn
}
