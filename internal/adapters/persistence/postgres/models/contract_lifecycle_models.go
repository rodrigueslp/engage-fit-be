package models

import "time"

type ContractSignatureEnvelopeModel struct {
	ID                           string `gorm:"primaryKey"`
	OrganizationID               string
	UnitID                       string
	ContractInstanceID           string
	Provider                     string
	ProviderReference            *string
	Status                       string
	SignerQualificationEncrypted string
	EvidenceEncrypted            *string
	WebhookEventID               *string
	ExpiresAt                    time.Time
	CompletedAt                  *time.Time
	CreatedByUserID              string
	CreatedAt                    time.Time
	UpdatedAt                    time.Time
}

func (ContractSignatureEnvelopeModel) TableName() string { return "contract_signature_envelopes" }

type ContractSignatureWebhookEventModel struct {
	ID, OrganizationID, UnitID, EnvelopeID, Provider, ProviderEventID, NormalizedStatus, EvidenceEncrypted string
	CreatedAt                                                                                              time.Time
}

func (ContractSignatureWebhookEventModel) TableName() string {
	return "contract_signature_webhook_events"
}

type ContractLegalActionModel struct {
	ID                       string `gorm:"primaryKey"`
	OrganizationID           string
	UnitID                   string
	PersonID                 string
	EnrollmentID             string
	SourceContractInstanceID string
	ActionType               string
	Status                   string
	EffectiveOn              time.Time
	EndsOn                   *time.Time
	Reason                   string
	TermsSnapshot            string `gorm:"type:jsonb"`
	FinancialInstruction     string `gorm:"type:jsonb"`
	Revision                 int
	SupersedesActionID       *string
	CreatedByUserID          string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

func (ContractLegalActionModel) TableName() string { return "contract_legal_actions" }

type ContractLegalEventModel struct {
	ID             string `gorm:"primaryKey"`
	OrganizationID string
	UnitID         string
	LegalActionID  string
	ActorUserID    string
	Action         string
	Data           string `gorm:"type:jsonb"`
	CreatedAt      time.Time
}

func (ContractLegalEventModel) TableName() string { return "contract_legal_events" }
