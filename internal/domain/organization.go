package domain

import "time"

type Organization struct {
	ID              ID
	Name            string
	Status          OrganizationStatus
	StatusReason    string
	StatusChangedAt *time.Time
	StatusChangedBy ID
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (o Organization) EffectiveStatus() OrganizationStatus {
	if o.Status == "" {
		return OrganizationStatusActive
	}
	return o.Status
}

func (o Organization) IsActive() bool {
	return o.EffectiveStatus() == OrganizationStatusActive
}
