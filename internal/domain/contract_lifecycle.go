package domain

import "time"

type ContractSignatureEnvelope struct {
	ID                 ID         `json:"id"`
	OrganizationID     ID         `json:"-"`
	UnitID             ID         `json:"unit_id"`
	ContractInstanceID ID         `json:"contract_instance_id"`
	Provider           string     `json:"provider"`
	ProviderReference  string     `json:"provider_reference"`
	Status             string     `json:"status"`
	ExpiresAt          time.Time  `json:"expires_at"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	CreatedByUserID    ID         `json:"-"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type ContractSignerQualification struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Document string `json:"document"`
}

func ContractSignerAAD(organizationID, envelopeID ID) string {
	return "contracts:" + string(organizationID) + ":" + string(envelopeID) + ":signer:v1"
}

type ContractLegalAction struct {
	ID                       ID             `json:"id"`
	OrganizationID           ID             `json:"-"`
	UnitID                   ID             `json:"unit_id"`
	PersonID                 ID             `json:"person_id"`
	EnrollmentID             ID             `json:"enrollment_id"`
	SourceContractInstanceID ID             `json:"source_contract_instance_id"`
	ActionType               string         `json:"action_type"`
	Status                   string         `json:"status"`
	EffectiveOn              string         `json:"effective_on"`
	EndsOn                   *string        `json:"ends_on,omitempty"`
	Reason                   string         `json:"reason"`
	TermsSnapshot            map[string]any `json:"terms_snapshot"`
	FinancialInstruction     map[string]any `json:"financial_instruction"`
	Revision                 int            `json:"revision"`
	SupersedesActionID       *ID            `json:"supersedes_action_id,omitempty"`
	CreatedByUserID          ID             `json:"-"`
	CreatedAt                time.Time      `json:"created_at"`
	UpdatedAt                time.Time      `json:"updated_at"`
}
