package domain

import "time"

type Box struct {
	ID                      ID
	Name                    string
	RiskInactiveDays        int
	RiskMessageCooldownDays int
	CreatedAt               time.Time
	UpdatedAt               time.Time
}
