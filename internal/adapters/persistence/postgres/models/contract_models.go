package models

import "time"

type ContractTemplateModel struct {
	ID             string `gorm:"primaryKey"`
	OrganizationID string `gorm:"index"`
	HomeUnitID     string `gorm:"index"`
	Code           string
	Name           string
	NameKey        string
	ContractKind   string `gorm:"default:membership_terms"`
	Status         string `gorm:"default:active"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (ContractTemplateModel) TableName() string {
	return "contract_templates"
}

type ContractTemplateVersionModel struct {
	ID                       string `gorm:"primaryKey"`
	OrganizationID           string `gorm:"index"`
	TemplateID               string `gorm:"index"`
	VersionNumber            int
	Revision                 int    `gorm:"default:1"`
	Status                   string `gorm:"default:draft"`
	Title                    string
	BodyMarkdown             string
	AcceptanceStatement      string
	PlaceholderSchemaVersion int    `gorm:"default:1"`
	RendererVersion          string `gorm:"default:markdown-v1"`
	ContentSHA256            *string
	PublishedAt              *time.Time
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

func (ContractTemplateVersionModel) TableName() string {
	return "contract_template_versions"
}

type ContractInstanceModel struct {
	ID                         string `gorm:"primaryKey"`
	OrganizationID             string `gorm:"index"`
	UnitID                     string `gorm:"index"`
	PersonID                   string `gorm:"index"`
	EnrollmentID               string `gorm:"index"`
	TemplateVersionID          string `gorm:"index"`
	ContractKind               string `gorm:"default:membership_terms"`
	Status                     string `gorm:"default:draft"`
	Revision                   int    `gorm:"default:1"`
	SnapshotSchemaVersion      int    `gorm:"default:1"`
	CommercialSnapshot         string `gorm:"type:jsonb"`
	CommercialSnapshotSHA256   string
	PartySnapshot              string `gorm:"type:jsonb"`
	DocumentTitle              string
	DocumentBodyMarkdown       string
	DocumentSHA256             string
	AcceptanceStatement        string
	AcceptanceStatementVersion int `gorm:"default:1"`
	CreatedByUserID            string
	IssuedByUserID             *string
	DiscardedByUserID          *string
	IssuedAt                   *time.Time
	AcceptedAt                 *time.Time
	DiscardedAt                *time.Time
	RetentionClass             string `gorm:"default:local_simulator"`
	RetentionDueAt             *time.Time
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

func (ContractInstanceModel) TableName() string {
	return "contract_instances"
}

type ContractAcceptanceSessionModel struct {
	ID                 string `gorm:"primaryKey"`
	OrganizationID     string `gorm:"index"`
	ContractInstanceID string `gorm:"index"`
	Adapter            string `gorm:"default:local_simulator"`
	AdapterReference   string
	TokenHash          string `gorm:"uniqueIndex"`
	Status             string `gorm:"default:active"`
	ExpiresAt          time.Time
	UsedAt             *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (ContractAcceptanceSessionModel) TableName() string {
	return "contract_acceptance_sessions"
}

type ContractAcceptanceModel struct {
	ID                         string `gorm:"primaryKey"`
	OrganizationID             string `gorm:"index"`
	ContractInstanceID         string `gorm:"uniqueIndex"`
	AcceptanceSessionID        string `gorm:"uniqueIndex"`
	Adapter                    string `gorm:"default:local_simulator"`
	AdapterReference           string
	EvidenceSchemaVersion      int    `gorm:"default:1"`
	PersonID                   string `gorm:"index"`
	DocumentSHA256             string
	AcceptanceStatementVersion int    `gorm:"default:1"`
	Evidence                   string `gorm:"type:jsonb"`
	AcceptedAt                 time.Time
	CreatedAt                  time.Time
}

func (ContractAcceptanceModel) TableName() string {
	return "contract_acceptances"
}

type ContractCommandRequestModel struct {
	ID                 string `gorm:"primaryKey"`
	OrganizationID     string `gorm:"index"`
	Command            string
	IdempotencyKey     string
	RequestFingerprint string
	ResultID           *string
	ResponseData       string `gorm:"type:jsonb"`
	CreatedAt          time.Time
}

func (ContractCommandRequestModel) TableName() string {
	return "contract_command_requests"
}

type ContractEventModel struct {
	ID                 string `gorm:"primaryKey"`
	OrganizationID     string `gorm:"index"`
	UnitID             string `gorm:"index"`
	PersonID           string `gorm:"index"`
	EnrollmentID       string `gorm:"index"`
	ContractInstanceID string `gorm:"index"`
	ActorType          string
	ActorUserID        *string
	Action             string
	BeforeData         string `gorm:"type:jsonb"`
	AfterData          string `gorm:"type:jsonb"`
	CreatedAt          time.Time
}

func (ContractEventModel) TableName() string {
	return "contract_events"
}

type ContractTemplateAuditEventModel struct {
	ID                string `gorm:"primaryKey"`
	OrganizationID    string `gorm:"index"`
	UnitID            string `gorm:"index"`
	TemplateID        string `gorm:"index"`
	TemplateVersionID *string
	ActorUserID       string
	Action            string
	BeforeData        string `gorm:"type:jsonb"`
	AfterData         string `gorm:"type:jsonb"`
	CreatedAt         time.Time
}

func (ContractTemplateAuditEventModel) TableName() string {
	return "contract_template_audit_events"
}
