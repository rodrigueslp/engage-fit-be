package domain

import "time"

type WhatsappConnectionMode string

const (
	WhatsappConnectionPlatform  WhatsappConnectionMode = "platform"
	WhatsappConnectionDedicated WhatsappConnectionMode = "dedicated"
)

type WhatsappSettings struct {
	ID                ID
	BoxID             ID
	ConnectionMode    WhatsappConnectionMode
	Provider          string
	BaseURL           string
	InstanceName      string
	APIKeyEncrypted   string
	Enabled           bool
	PlatformAvailable bool
	PlatformSender    string
	ContentSIDs       map[MessageTemplateType]string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
