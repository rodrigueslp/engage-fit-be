package domain

import "time"

type Box struct {
	ID                      ID
	Name                    string
	Status                  BoxStatus
	StatusReason            string
	StatusChangedAt         *time.Time
	StatusChangedBy         ID
	BillingAccessBlocked    bool
	BillingAccessReason     string
	BillingAccessChangedAt  *time.Time
	RiskInactiveDays        int
	RiskMessageCooldownDays int
	CreatedAt               time.Time
	UpdatedAt               time.Time
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
