package domain

import "time"

type WhatsappSettings struct {
	ID              ID
	BoxID           ID
	Provider        string
	BaseURL         string
	InstanceName    string
	APIKeyEncrypted string
	Enabled         bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
