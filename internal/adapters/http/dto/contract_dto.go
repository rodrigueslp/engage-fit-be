package dto

import (
	"time"

	"boxengage/backend/internal/domain"
)

type CreateContractTemplateRequest struct {
	Name                string `json:"name"`
	Title               string `json:"title"`
	BodyMarkdown        string `json:"body_markdown"`
	AcceptanceStatement string `json:"acceptance_statement"`
}

type UpdateContractTemplateDraftRequest struct {
	Name                string `json:"name"`
	Title               string `json:"title"`
	BodyMarkdown        string `json:"body_markdown"`
	AcceptanceStatement string `json:"acceptance_statement"`
	ExpectedRevision    int    `json:"expected_revision"`
}

type PublishContractTemplateRequest struct {
	DraftRevision int `json:"draft_revision"`
}

type CreateContractTemplateVersionRequest struct {
	Title               string `json:"title"`
	BodyMarkdown        string `json:"body_markdown"`
	AcceptanceStatement string `json:"acceptance_statement"`
}

type ContractTemplateVersionResponse struct {
	ID                       string     `json:"id"`
	OrganizationID           string     `json:"organization_id"`
	TemplateID               string     `json:"template_id"`
	VersionNumber            int        `json:"version_number"`
	Revision                 int        `json:"revision"`
	Status                   string     `json:"status"`
	Title                    string     `json:"title"`
	BodyMarkdown             string     `json:"body_markdown"`
	AcceptanceStatement      string     `json:"acceptance_statement"`
	PlaceholderSchemaVersion int        `json:"placeholder_schema_version"`
	RendererVersion          string     `json:"renderer_version"`
	ContentSHA256            string     `json:"content_sha256,omitempty"`
	PublishedAt              *time.Time `json:"published_at,omitempty"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

type ContractTemplateResponse struct {
	ID             string                            `json:"id"`
	OrganizationID string                            `json:"organization_id"`
	HomeUnitID     string                            `json:"home_unit_id"`
	Code           string                            `json:"code"`
	Name           string                            `json:"name"`
	NameKey        string                            `json:"name_key"`
	ContractKind   string                            `json:"contract_kind"`
	Status         string                            `json:"status"`
	CurrentVersion *ContractTemplateVersionResponse  `json:"current_version,omitempty"`
	CurrentDraft   *ContractTemplateVersionResponse  `json:"current_draft,omitempty"`
	Versions       []ContractTemplateVersionResponse `json:"versions,omitempty"`
	CreatedAt      time.Time                         `json:"created_at"`
	UpdatedAt      time.Time                         `json:"updated_at"`
}

type ContractPreviewRequest struct {
	TemplateID        string `json:"template_id"`
	TemplateVersionID string `json:"template_version_id"`
}

type ContractPreviewResponse struct {
	TemplateID           string                            `json:"template_id"`
	TemplateVersionID    string                            `json:"template_version_id"`
	EnrollmentID         string                            `json:"enrollment_id"`
	DocumentTitle        string                            `json:"document_title"`
	DocumentBodyMarkdown string                            `json:"document_body_markdown"`
	AcceptanceStatement  string                            `json:"acceptance_statement"`
	CommercialSnapshot   domain.ContractCommercialSnapshot `json:"commercial_snapshot"`
	PreviewSHA256        string                            `json:"preview_sha256"`
	Warnings             []string                          `json:"warnings"`
}

type CreateContractInstanceRequest struct {
	TemplateID            string `json:"template_id"`
	TemplateVersionID     string `json:"template_version_id"`
	ExpectedPreviewSHA256 string `json:"expected_preview_sha256"`
}

type IssueContractInstanceRequest struct {
	ExpectedRevision int `json:"expected_revision"`
}

type DiscardContractInstanceRequest struct {
	ExpectedRevision int `json:"expected_revision"`
}

type ContractAcceptanceSessionResponse struct {
	ID                 string     `json:"id"`
	OrganizationID     string     `json:"organization_id"`
	ContractInstanceID string     `json:"contract_instance_id"`
	Adapter            string     `json:"adapter"`
	AdapterReference   string     `json:"adapter_reference"`
	Status             string     `json:"status"`
	ExpiresAt          time.Time  `json:"expires_at"`
	UsedAt             *time.Time `json:"used_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

type ContractAcceptanceResponse struct {
	ID                         string         `json:"id"`
	OrganizationID             string         `json:"organization_id"`
	ContractInstanceID         string         `json:"contract_instance_id"`
	AcceptanceSessionID        string         `json:"acceptance_session_id"`
	Adapter                    string         `json:"adapter"`
	AdapterReference           string         `json:"adapter_reference"`
	EvidenceSchemaVersion      int            `json:"evidence_schema_version"`
	PersonID                   string         `json:"person_id"`
	DocumentSHA256             string         `json:"document_sha256"`
	AcceptanceStatementVersion int            `json:"acceptance_statement_version"`
	Evidence                   map[string]any `json:"evidence"`
	AcceptedAt                 time.Time      `json:"accepted_at"`
	CreatedAt                  time.Time      `json:"created_at"`
}

type ContractEventResponse struct {
	ID                 string         `json:"id"`
	OrganizationID     string         `json:"organization_id"`
	UnitID             string         `json:"unit_id"`
	PersonID           string         `json:"person_id"`
	EnrollmentID       string         `json:"enrollment_id"`
	ContractInstanceID string         `json:"contract_instance_id"`
	ActorType          string         `json:"actor_type"`
	ActorUserID        *string        `json:"actor_user_id,omitempty"`
	Action             string         `json:"action"`
	BeforeData         map[string]any `json:"before_data,omitempty"`
	AfterData          map[string]any `json:"after_data,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
}

type ContractInstanceResponse struct {
	ID                         string                             `json:"id"`
	OrganizationID             string                             `json:"organization_id"`
	UnitID                     string                             `json:"unit_id"`
	PersonID                   string                             `json:"person_id"`
	EnrollmentID               string                             `json:"enrollment_id"`
	TemplateVersionID          string                             `json:"template_version_id"`
	ContractKind               string                             `json:"contract_kind"`
	Status                     string                             `json:"status"`
	Revision                   int                                `json:"revision"`
	SnapshotSchemaVersion      int                                `json:"snapshot_schema_version"`
	CommercialSnapshot         domain.ContractCommercialSnapshot  `json:"commercial_snapshot"`
	CommercialSnapshotSHA256   string                             `json:"commercial_snapshot_sha256"`
	PartySnapshot              domain.ContractPartySnapshot       `json:"party_snapshot"`
	DocumentTitle              string                             `json:"document_title"`
	DocumentBodyMarkdown       string                             `json:"document_body_markdown"`
	DocumentSHA256             string                             `json:"document_sha256"`
	AcceptanceStatement        string                             `json:"acceptance_statement"`
	AcceptanceStatementVersion int                                `json:"acceptance_statement_version"`
	CreatedByUserID            string                             `json:"created_by_user_id"`
	IssuedByUserID             *string                            `json:"issued_by_user_id,omitempty"`
	DiscardedByUserID          *string                            `json:"discarded_by_user_id,omitempty"`
	IssuedAt                   *time.Time                         `json:"issued_at,omitempty"`
	AcceptedAt                 *time.Time                         `json:"accepted_at,omitempty"`
	DiscardedAt                *time.Time                         `json:"discarded_at,omitempty"`
	RetentionClass             string                             `json:"retention_class"`
	TemplateVersion            *ContractTemplateVersionResponse   `json:"template_version,omitempty"`
	ActiveSession              *ContractAcceptanceSessionResponse `json:"active_session,omitempty"`
	Acceptance                 *ContractAcceptanceResponse        `json:"acceptance,omitempty"`
	Events                     []ContractEventResponse            `json:"events,omitempty"`
	CreatedAt                  time.Time                          `json:"created_at"`
	UpdatedAt                  time.Time                          `json:"updated_at"`
}

type ContractAcceptanceSessionLaunchResponse struct {
	AdapterReference string    `json:"adapter_reference"`
	LaunchToken      string    `json:"launch_token"`
	LaunchPath       string    `json:"launch_path"`
	ExpiresAt        time.Time `json:"expires_at"`
}

type PublicContractAcceptanceSessionResponse struct {
	Token                      string                            `json:"token"`
	Status                     string                            `json:"status"`
	DocumentTitle              string                            `json:"document_title"`
	DocumentBodyMarkdown       string                            `json:"document_body_markdown"`
	AcceptanceStatement        string                            `json:"acceptance_statement"`
	AcceptanceStatementVersion int                               `json:"acceptance_statement_version"`
	DocumentSHA256             string                            `json:"document_sha256"`
	CommercialSnapshot         domain.ContractCommercialSnapshot `json:"commercial_snapshot"`
	ExpiresAt                  time.Time                         `json:"expires_at"`
	IsSimulator                bool                              `json:"is_simulator"`
	Warning                    string                            `json:"warning"`
}

type AcceptContractRequest struct {
	DocumentSHA256             string `json:"document_sha256"`
	AcceptanceStatementVersion int    `json:"acceptance_statement_version"`
	Accepted                   bool   `json:"accepted"`
}
