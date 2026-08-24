package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type ContractKind string
type ContractTemplateStatus string
type ContractTemplateVersionStatus string
type ContractInstanceStatus string
type ContractAcceptanceSessionStatus string
type ContractRetentionClass string
type ContractActorType string

const (
	ContractKindMembershipTerms ContractKind = "membership_terms"

	ContractTemplateStatusActive   ContractTemplateStatus = "active"
	ContractTemplateStatusArchived ContractTemplateStatus = "archived"

	ContractTemplateVersionStatusDraft     ContractTemplateVersionStatus = "draft"
	ContractTemplateVersionStatusPublished ContractTemplateVersionStatus = "published"
	ContractTemplateVersionStatusRetired   ContractTemplateVersionStatus = "retired"

	ContractInstanceStatusDraft             ContractInstanceStatus = "draft"
	ContractInstanceStatusPendingAcceptance ContractInstanceStatus = "pending_acceptance"
	ContractInstanceStatusAccepted          ContractInstanceStatus = "accepted"
	ContractInstanceStatusDiscarded         ContractInstanceStatus = "discarded"

	ContractAcceptanceSessionStatusActive     ContractAcceptanceSessionStatus = "active"
	ContractAcceptanceSessionStatusUsed       ContractAcceptanceSessionStatus = "used"
	ContractAcceptanceSessionStatusExpired    ContractAcceptanceSessionStatus = "expired"
	ContractAcceptanceSessionStatusSuperseded ContractAcceptanceSessionStatus = "superseded"

	ContractRetentionClassLocalSimulator     ContractRetentionClass = "local_simulator"
	ContractRetentionClassProductionContract ContractRetentionClass = "production_contract"

	ContractActorOwner             ContractActorType = "owner"
	ContractActorAcceptanceSubject ContractActorType = "acceptance_subject"
	ContractActorSystem            ContractActorType = "system"

	PlaceholderSchemaVersion1 = 1
	RendererVersionMarkdownV1 = "markdown-v1"
	SnapshotSchemaVersion1    = 1
	EvidenceSchemaVersion1    = 1
	AdapterLocalSimulator     = "local_simulator"
)

var (
	ErrInvalidContractTemplate = errors.New("invalid contract template")
	ErrUnknownPlaceholder      = errors.New("unknown placeholder in contract template")
	ErrUnclosedPlaceholder     = errors.New("unclosed placeholder in contract template")
	ErrDisallowedMarkdownTag   = errors.New("disallowed tag or element in contract markdown")
	ErrInvalidContractState    = errors.New("invalid contract state transition")
)

var AllowedPlaceholders = map[string]string{
	"{{organization_name}}": "Nome da organização",
	"{{unit_name}}":         "Nome da unidade",
	"{{member_name}}":       "Nome da pessoa matriculada",
	"{{plan_name}}":         "Nome do plano",
	"{{plan_version}}":      "Versão do plano",
	"{{billing_cycle}}":     "Ciclo de cobrança",
	"{{base_price}}":        "Preço base do plano",
	"{{addons_summary}}":    "Resumo dos adicionais",
	"{{addons_price}}":      "Preço dos adicionais",
	"{{total_price}}":       "Preço total",
	"{{starts_on}}":         "Data de início",
	"{{valid_until}}":       "Data de validade",
	"{{access_summary}}":    "Resumo de acesso",
}

type ContractTemplate struct {
	ID             ID
	OrganizationID ID
	HomeUnitID     ID
	Code           string
	Name           string
	NameKey        string
	ContractKind   ContractKind
	Status         ContractTemplateStatus
	CurrentDraft   *ContractTemplateVersion
	CurrentVersion *ContractTemplateVersion
	Versions       []ContractTemplateVersion
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ContractTemplateVersion struct {
	ID                       ID
	OrganizationID           ID
	TemplateID               ID
	VersionNumber            int
	Revision                 int
	Status                   ContractTemplateVersionStatus
	Title                    string
	BodyMarkdown             string
	AcceptanceStatement      string
	PlaceholderSchemaVersion int
	RendererVersion          string
	ContentSHA256            string
	PublishedAt              *time.Time
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type ContractCommercialAddonSnapshot struct {
	PlanVersionAddonID ID     `json:"plan_version_addon_id"`
	Name               string `json:"name"`
	PriceDeltaCents    int    `json:"price_delta_cents"`
	Summary            string `json:"summary"`
}

type ContractCommercialSnapshot struct {
	SchemaVersion      int                               `json:"schema_version"`
	EnrollmentID       ID                                `json:"enrollment_id"`
	EnrollmentRevision int                               `json:"enrollment_revision"`
	PlanID             ID                                `json:"plan_id"`
	PlanVersionID      ID                                `json:"plan_version_id"`
	PlanVersionNumber  int                               `json:"plan_version_number"`
	PlanName           string                            `json:"plan_name"`
	BillingCycle       MembershipBillingCycle            `json:"billing_cycle"`
	Currency           string                            `json:"currency"`
	BasePriceCents     int                               `json:"base_price_cents"`
	AddonsPriceCents   int                               `json:"addons_price_cents"`
	TotalPriceCents    int                               `json:"total_price_cents"`
	StartsOn           string                            `json:"starts_on"`
	ValidUntil         *string                           `json:"valid_until"`
	BaseSummary        string                            `json:"base_summary"`
	Addons             []ContractCommercialAddonSnapshot `json:"addons"`
}

type ContractPartySnapshot struct {
	OrganizationID   ID     `json:"organization_id"`
	OrganizationName string `json:"organization_name"`
	UnitID           ID     `json:"unit_id"`
	UnitName         string `json:"unit_name"`
	PersonID         ID     `json:"person_id"`
	DisplayName      string `json:"display_name"`
}

type ContractInstance struct {
	ID                         ID
	OrganizationID             ID
	UnitID                     ID
	PersonID                   ID
	EnrollmentID               ID
	TemplateVersionID          ID
	ContractKind               ContractKind
	Status                     ContractInstanceStatus
	Revision                   int
	SnapshotSchemaVersion      int
	CommercialSnapshot         ContractCommercialSnapshot
	CommercialSnapshotSHA256   string
	PartySnapshot              ContractPartySnapshot
	DocumentTitle              string
	DocumentBodyMarkdown       string
	DocumentSHA256             string
	AcceptanceStatement        string
	AcceptanceStatementVersion int
	CreatedByUserID            ID
	IssuedByUserID             *ID
	DiscardedByUserID          *ID
	IssuedAt                   *time.Time
	AcceptedAt                 *time.Time
	DiscardedAt                *time.Time
	RetentionClass             ContractRetentionClass
	RetentionDueAt             *time.Time
	CreatedAt                  time.Time
	UpdatedAt                  time.Time

	TemplateVersion *ContractTemplateVersion
	ActiveSession   *ContractAcceptanceSession
	Acceptance      *ContractAcceptance
	Events          []ContractEvent
}

type ContractAcceptanceSession struct {
	ID                 ID
	OrganizationID     ID
	ContractInstanceID ID
	Adapter            string
	AdapterReference   string
	TokenHash          string
	Status             ContractAcceptanceSessionStatus
	ExpiresAt          time.Time
	UsedAt             *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type ContractAcceptance struct {
	ID                         ID
	OrganizationID             ID
	ContractInstanceID         ID
	AcceptanceSessionID        ID
	Adapter                    string
	AdapterReference           string
	EvidenceSchemaVersion      int
	PersonID                   ID
	DocumentSHA256             string
	AcceptanceStatementVersion int
	Evidence                   map[string]any
	AcceptedAt                 time.Time
	CreatedAt                  time.Time
}

type ContractCommandRequest struct {
	ID             ID
	OrganizationID ID
	Command        string
	IdempotencyKey string
	Fingerprint    string
	ResultID       *ID
	ResponseData   map[string]any
	CreatedAt      time.Time
}

type ContractEvent struct {
	ID                 ID
	OrganizationID     ID
	UnitID             ID
	PersonID           ID
	EnrollmentID       ID
	ContractInstanceID ID
	ActorType          ContractActorType
	ActorUserID        *ID
	Action             string
	BeforeData         map[string]any
	AfterData          map[string]any
	CreatedAt          time.Time
}

type ContractTemplateAuditEvent struct {
	ID                ID
	OrganizationID    ID
	UnitID            ID
	TemplateID        ID
	TemplateVersionID *ID
	ActorUserID       ID
	Action            string
	BeforeData        map[string]any
	AfterData         map[string]any
	CreatedAt         time.Time
}

type ContractPreview struct {
	TemplateID           ID                         `json:"template_id"`
	TemplateVersionID    ID                         `json:"template_version_id"`
	EnrollmentID         ID                         `json:"enrollment_id"`
	DocumentTitle        string                     `json:"document_title"`
	DocumentBodyMarkdown string                     `json:"document_body_markdown"`
	AcceptanceStatement  string                     `json:"acceptance_statement"`
	CommercialSnapshot   ContractCommercialSnapshot `json:"commercial_snapshot"`
	PreviewSHA256        string                     `json:"preview_sha256"`
	Warnings             []string                   `json:"warnings"`
}

type ContractPrivacyExport struct {
	Instance   ContractInstance
	Template   ContractTemplateVersion
	Acceptance *ContractAcceptance
	Sessions   []ContractAcceptanceSession
	Events     []ContractEvent
}

type ContractEnrollmentSnapshot struct {
	PersonID           ID
	EnrollmentRevision int
	EffectiveStatus    EnrollmentEffectiveStatus
	PlanID             ID
	PlanVersionID      ID
	PlanVersionNumber  int
	PlanName           string
	BillingCycle       MembershipBillingCycle
	Currency           string
	BasePriceCents     int
	AddonsPriceCents   int
	TotalPriceCents    int
	StartsOn           time.Time
	ValidUntil         *time.Time
	BaseSummary        string
	Addons             []ContractEnrollmentAddonSnapshot
}

type ContractEnrollmentAddonSnapshot struct {
	PlanVersionAddonID ID
	Name               string
	PriceDeltaCents    int
	Summary            string
}

type ContractPartyResolution struct {
	PersonID      ID
	DisplayName   string
	ProfileStatus PersonProfileStatus
	Eligible      bool
}

type AcceptanceSessionLaunch struct {
	AdapterReference string
	LaunchToken      string
	LaunchPath       string
	ExpiresAt        time.Time
}

type AcceptanceSubmission struct {
	DocumentSHA256             string
	AcceptanceStatementVersion int
	Accepted                   bool
}

type AcceptanceVerificationResult struct {
	Verified           bool
	AcceptedAt         time.Time
	NormalizedEvidence map[string]any
}

type PublicAcceptanceSessionData struct {
	Token                      string                     `json:"token"`
	Status                     string                     `json:"status"`
	ContractID                 ID                         `json:"contract_id"`
	DocumentTitle              string                     `json:"document_title"`
	DocumentBodyMarkdown       string                     `json:"document_body_markdown"`
	AcceptanceStatement        string                     `json:"acceptance_statement"`
	AcceptanceStatementVersion int                        `json:"acceptance_statement_version"`
	DocumentSHA256             string                     `json:"document_sha256"`
	CommercialSnapshot         ContractCommercialSnapshot `json:"commercial_snapshot"`
	PartySnapshot              ContractPartySnapshot      `json:"party_snapshot"`
	ExpiresAt                  time.Time                  `json:"expires_at"`
	IsSimulator                bool                       `json:"is_simulator"`
	Warning                    string                     `json:"warning"`
}

type ContractRenderContext struct {
	OrganizationName string
	UnitName         string
	MemberName       string
	PlanName         string
	PlanVersion      string
	BillingCycle     string
	BasePrice        string
	AddonsSummary    string
	AddonsPrice      string
	TotalPrice       string
	StartsOn         string
	ValidUntil       string
	AccessSummary    string
}

func NormalizeTemplateName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(name, " ")
}

func ValidateTemplateTextLimits(name, title, body, acceptanceStatement string) error {
	nameLen := utf8.RuneCountInString(strings.TrimSpace(name))
	if nameLen < 1 || nameLen > 160 {
		return fmt.Errorf("%w: name length must be between 1 and 160 chars (got %d)", ErrInvalidContractTemplate, nameLen)
	}
	titleLen := utf8.RuneCountInString(strings.TrimSpace(title))
	if titleLen < 1 || titleLen > 160 {
		return fmt.Errorf("%w: title length must be between 1 and 160 chars (got %d)", ErrInvalidContractTemplate, titleLen)
	}
	bodyLen := utf8.RuneCountInString(strings.TrimSpace(body))
	if bodyLen < 1 || bodyLen > 50000 {
		return fmt.Errorf("%w: body length must be between 1 and 50000 chars (got %d)", ErrInvalidContractTemplate, bodyLen)
	}
	stmtLen := utf8.RuneCountInString(strings.TrimSpace(acceptanceStatement))
	if stmtLen < 10 || stmtLen > 1000 {
		return fmt.Errorf("%w: acceptance statement length must be between 10 and 1000 chars (got %d)", ErrInvalidContractTemplate, stmtLen)
	}
	return nil
}

var (
	htmlTagRegex      = regexp.MustCompile(`(?i)<\s*/?\s*(?:script|iframe|object|embed|applet|form|input|button|select|textarea|img|svg|link|style|meta|base|html|body|head|frame|frameset|a\b)[^>]*>`)
	genericHTMLRegex  = regexp.MustCompile(`(?i)<[^>]+>`)
	markdownImgRegex  = regexp.MustCompile(`!\[[^\]]*\]\([^\)]*\)`)
	unsafeSchemeRegex = regexp.MustCompile(`(?i)\[[^\]]*\]\(\s*(?:javascript:|data:|vbscript:)[^\)]*\)`)
	allBracesRegex    = regexp.MustCompile(`\{+[^{}]*\}+`)
)

func ValidateMarkdownContent(content string) error {
	if htmlTagRegex.MatchString(content) || genericHTMLRegex.MatchString(content) {
		return fmt.Errorf("%w: HTML tags are not allowed", ErrDisallowedMarkdownTag)
	}
	if markdownImgRegex.MatchString(content) {
		return fmt.Errorf("%w: remote/inline images are not allowed", ErrDisallowedMarkdownTag)
	}
	if unsafeSchemeRegex.MatchString(content) {
		return fmt.Errorf("%w: unsafe URI schemes (javascript/data/vbscript) are not allowed", ErrDisallowedMarkdownTag)
	}
	return nil
}

func ValidatePlaceholders(texts ...string) error {
	for _, text := range texts {
		openCount := strings.Count(text, "{")
		closeCount := strings.Count(text, "}")
		if openCount != closeCount || openCount%2 != 0 {
			return fmt.Errorf("%w: unbalanced braces (found %d '{' and %d '}')", ErrUnclosedPlaceholder, openCount, closeCount)
		}

		i := 0
		for i < len(text) {
			start := strings.Index(text[i:], "{{")
			if start == -1 {
				// check if there's any stray '{' or '}'
				if strings.Contains(text[i:], "{") || strings.Contains(text[i:], "}") {
					return fmt.Errorf("%w: stray brace found", ErrUnclosedPlaceholder)
				}
				break
			}
			startIdx := i + start
			end := strings.Index(text[startIdx+2:], "}}")
			if end == -1 {
				return fmt.Errorf("%w: unclosed placeholder starting at %d", ErrUnclosedPlaceholder, startIdx)
			}
			endIdx := startIdx + 2 + end + 2
			token := text[startIdx:endIdx]

			// Check inside the token for nested braces
			inner := text[startIdx+2 : startIdx+2+end]
			if strings.Contains(inner, "{") || strings.Contains(inner, "}") {
				return fmt.Errorf("%w: invalid nested braces in placeholder %s", ErrUnclosedPlaceholder, token)
			}

			if _, ok := AllowedPlaceholders[token]; !ok {
				return fmt.Errorf("%w: %s is not an allowed placeholder", ErrUnknownPlaceholder, token)
			}
			i = endIdx
		}
	}
	return nil
}

func RenderTemplateText(templateText string, ctx ContractRenderContext) string {
	replacements := map[string]string{
		"{{organization_name}}": ctx.OrganizationName,
		"{{unit_name}}":         ctx.UnitName,
		"{{member_name}}":       ctx.MemberName,
		"{{plan_name}}":         ctx.PlanName,
		"{{plan_version}}":      ctx.PlanVersion,
		"{{billing_cycle}}":     ctx.BillingCycle,
		"{{base_price}}":        ctx.BasePrice,
		"{{addons_summary}}":    ctx.AddonsSummary,
		"{{addons_price}}":      ctx.AddonsPrice,
		"{{total_price}}":       ctx.TotalPrice,
		"{{starts_on}}":         ctx.StartsOn,
		"{{valid_until}}":       ctx.ValidUntil,
		"{{access_summary}}":    ctx.AccessSummary,
	}

	result := templateText
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

func FormatBRL(cents int) string {
	negative := cents < 0
	if negative {
		cents = -cents
	}
	reais := cents / 100
	centavos := cents % 100
	reaisStr := fmt.Sprintf("%d", reais)
	var withSeparators strings.Builder
	l := len(reaisStr)
	for i, c := range reaisStr {
		if i > 0 && (l-i)%3 == 0 {
			withSeparators.WriteRune('.')
		}
		withSeparators.WriteRune(c)
	}
	sign := ""
	if negative {
		sign = "-"
	}
	return fmt.Sprintf("%sR$ %s,%02d", sign, withSeparators.String(), centavos)
}

func FormatBillingCycle(cycle MembershipBillingCycle) string {
	switch cycle {
	case MembershipBillingMonthly:
		return "Mensal"
	case MembershipBillingQuarterly:
		return "Trimestral"
	case MembershipBillingSemiannual:
		return "Semestral"
	case MembershipBillingAnnual:
		return "Anual"
	case MembershipBillingOneTime:
		return "Pagamento único"
	default:
		return string(cycle)
	}
}

func CanonicalJSON(v any) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func CalculateSHA256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

type CanonicalTemplateContentPayload struct {
	SchemaVersion       int    `json:"schema_version"`
	RendererVersion     string `json:"renderer_version"`
	Title               string `json:"title"`
	BodyMarkdown        string `json:"body_markdown"`
	AcceptanceStatement string `json:"acceptance_statement"`
}

func CalculateTemplateVersionContentSHA256(title, bodyMarkdown, acceptanceStatement string) (string, error) {
	payload := CanonicalTemplateContentPayload{
		SchemaVersion:       PlaceholderSchemaVersion1,
		RendererVersion:     RendererVersionMarkdownV1,
		Title:               title,
		BodyMarkdown:        bodyMarkdown,
		AcceptanceStatement: acceptanceStatement,
	}
	b, err := CanonicalJSON(payload)
	if err != nil {
		return "", err
	}
	return CalculateSHA256Hex(b), nil
}

func CalculateCommercialSnapshotSHA256(snapshot ContractCommercialSnapshot) (string, error) {
	b, err := CanonicalJSON(snapshot)
	if err != nil {
		return "", err
	}
	return CalculateSHA256Hex(b), nil
}

type CanonicalDocumentPayload struct {
	SchemaVersion            int    `json:"schema_version"`
	RendererVersion          string `json:"renderer_version"`
	DocumentTitle            string `json:"document_title"`
	DocumentBodyMarkdown     string `json:"document_body_markdown"`
	AcceptanceStatement      string `json:"acceptance_statement"`
	CommercialSnapshotSHA256 string `json:"commercial_snapshot_sha256"`
}

func CalculateDocumentSHA256(title, bodyMarkdown, acceptanceStatement, commercialSnapshotSHA256 string) (string, error) {
	payload := CanonicalDocumentPayload{
		SchemaVersion:            SnapshotSchemaVersion1,
		RendererVersion:          RendererVersionMarkdownV1,
		DocumentTitle:            title,
		DocumentBodyMarkdown:     bodyMarkdown,
		AcceptanceStatement:      acceptanceStatement,
		CommercialSnapshotSHA256: commercialSnapshotSHA256,
	}
	b, err := CanonicalJSON(payload)
	if err != nil {
		return "", err
	}
	return CalculateSHA256Hex(b), nil
}
