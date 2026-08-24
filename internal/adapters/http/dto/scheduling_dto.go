package dto

type ProfessionalResponse struct {
	ID           string  `json:"id"`
	DisplayName  string  `json:"display_name"`
	LinkedUserID *string `json:"linked_user_id,omitempty"`
	Status       string  `json:"status"`
	Revision     int     `json:"revision"`
	AnonymizedAt *string `json:"anonymized_at,omitempty"`
	ActiveInUnit bool    `json:"active_in_unit"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type ProfessionalAccountCandidateResponse struct {
	UserID           string `json:"user_id"`
	DisplayName      string `json:"display_name"`
	Role             string `json:"role"`
	Active           bool   `json:"active"`
	AuthorizedInUnit bool   `json:"authorized_in_unit"`
	IsLinked         bool   `json:"is_linked"`
}

type CreateProfessionalRequest struct {
	DisplayName  string  `json:"display_name"`
	LinkedUserID *string `json:"linked_user_id"`
}

type UpdateProfessionalRequest struct {
	DisplayName      *string `json:"display_name"`
	LinkedUserID     *string `json:"linked_user_id"`
	ExpectedRevision int     `json:"expected_revision"`
}

type ArchiveProfessionalRequest struct {
	ExpectedRevision int    `json:"expected_revision"`
	Reason           string `json:"reason"`
}

type ReactivateProfessionalRequest struct {
	ExpectedRevision int    `json:"expected_revision"`
	Reason           string `json:"reason"`
}

type AnonymizeProfessionalRequest struct {
	ExpectedRevision int    `json:"expected_revision"`
	Reason           string `json:"reason"`
}

type SchedulingAuditEventResponse struct {
	Action        string         `json:"action"`
	EntityType    string         `json:"entity_type"`
	EntityID      string         `json:"entity_id"`
	Revision      int            `json:"revision"`
	ChangedFields []string       `json:"changed_fields"`
	BeforeData    map[string]any `json:"before_data"`
	AfterData     map[string]any `json:"after_data"`
	Reason        string         `json:"reason,omitempty"`
	CreatedAt     string         `json:"created_at"`
}

type SchedulingPrivacyExportResponse struct {
	Professional ProfessionalResponse           `json:"professional"`
	Events       []SchedulingAuditEventResponse `json:"events"`
	ExportedAt   string                         `json:"exported_at"`
}
