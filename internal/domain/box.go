package domain

import "time"

type AccessControlMode string

const (
	AccessControlDisabled   AccessControlMode = "disabled"
	AccessControlManual     AccessControlMode = "manual"
	AccessControlIntegrated AccessControlMode = "integrated"
)

type Box struct {
	ID                      ID
	OrganizationID          ID
	Name                    string
	NameKey                 string
	Timezone                string
	Status                  BoxStatus
	StatusReason            string
	StatusChangedAt         *time.Time
	StatusChangedBy         ID
	BillingAccessBlocked    bool
	BillingAccessReason     string
	BillingAccessChangedAt  *time.Time
	RiskInactiveDays        int
	RiskMessageCooldownDays int
	AccessControlMode       AccessControlMode
	RetentionBaselineAt     *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func (b Box) EffectiveAccessControlMode() AccessControlMode {
	if b.AccessControlMode == "" {
		return AccessControlDisabled
	}
	return b.AccessControlMode
}

func (b Box) EffectiveStatus() BoxStatus {
	if b.Status == "" {
		return BoxStatusActive
	}
	return b.Status
}

func (b Box) IsActive() bool {
	return b.EffectiveStatus() == BoxStatusActive
}

func (b Box) IsOperational() bool {
	return b.IsActive() && !b.BillingAccessBlocked
}
