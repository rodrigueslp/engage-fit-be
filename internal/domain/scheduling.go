package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

type ProfessionalStatus string

const (
	ProfessionalStatusActive   ProfessionalStatus = "active"
	ProfessionalStatusArchived ProfessionalStatus = "archived"

	AnonymizedProfessionalName = "Profissional anonimizado"
	AnonymizedProfessionalKey  = "profissional anonimizado"
)

type Professional struct {
	ID               ID                 `json:"id"`
	OrganizationID   ID                 `json:"organization_id"`
	DisplayName      string             `json:"display_name"`
	DisplayNameKey   string             `json:"display_name_key"`
	LinkedUserID     *ID                `json:"linked_user_id,omitempty"`
	Status           ProfessionalStatus `json:"status"`
	Revision         int                `json:"revision"`
	AnonymizedAt     *time.Time         `json:"anonymized_at,omitempty"`
	CreatedByUserID  ID                 `json:"created_by_user_id"`
	UpdatedByUserID  ID                 `json:"updated_by_user_id"`
	ArchivedAt       *time.Time         `json:"archived_at,omitempty"`
	ArchivedByUserID *ID                `json:"archived_by_user_id,omitempty"`
	ActiveInUnit     bool               `json:"active_in_unit"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

type ProfessionalAccountCandidate struct {
	UserID           ID     `json:"user_id"`
	DisplayName      string `json:"display_name"`
	Role             string `json:"role"`
	Active           bool   `json:"active"`
	AuthorizedInUnit bool   `json:"authorized_in_unit"`
	IsLinked         bool   `json:"is_linked"`
}

type SchedulingUnitContext struct {
	OrganizationID ID     `json:"organization_id"`
	UnitID         ID     `json:"unit_id"`
	UnitName       string `json:"unit_name"`
	Role           string `json:"role"`
	Authorized     bool   `json:"authorized"`
}

type SchedulingAuditEvent struct {
	ID             ID             `json:"id"`
	OrganizationID ID             `json:"organization_id"`
	UnitID         ID             `json:"unit_id"`
	ActorUserID    ID             `json:"actor_user_id"`
	Action         string         `json:"action"`
	EntityType     string         `json:"entity_type"`
	EntityID       ID             `json:"entity_id"`
	Revision       int            `json:"revision"`
	ChangedFields  []string       `json:"changed_fields"`
	BeforeData     map[string]any `json:"before_data"`
	AfterData      map[string]any `json:"after_data"`
	Reason         string         `json:"reason,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

type SchedulingCommandRequest struct {
	ID             ID             `json:"id"`
	OrganizationID ID             `json:"organization_id"`
	UnitID         ID             `json:"unit_id"`
	Command        string         `json:"command"`
	IdempotencyKey string         `json:"idempotency_key"`
	Fingerprint    string         `json:"fingerprint"`
	ProfessionalID ID             `json:"professional_id"`
	ResponseData   map[string]any `json:"response_data"`
	CreatedAt      time.Time      `json:"created_at"`
}

type ProfessionalPrivacyExport struct {
	Professional Professional           `json:"professional"`
	Events       []SchedulingAuditEvent `json:"events"`
	ExportedAt   time.Time              `json:"exported_at"`
}

func NormalizeProfessionalName(value string) (name, key string, valid bool) {
	name = strings.Join(strings.Fields(value), " ")
	runeLen := utf8.RuneCountInString(name)
	if runeLen < 1 || runeLen > 100 {
		return "", "", false
	}
	return name, strings.ToLower(name), true
}

func NormalizeSchedulingReason(value string, minLen, maxLen int) (string, bool) {
	reason := strings.Join(strings.Fields(value), " ")
	runeLen := utf8.RuneCountInString(reason)
	if runeLen < minLen || runeLen > maxLen {
		return "", false
	}
	return reason, true
}

func ValidProfessionalStatus(status ProfessionalStatus) bool {
	return status == ProfessionalStatusActive || status == ProfessionalStatusArchived
}
