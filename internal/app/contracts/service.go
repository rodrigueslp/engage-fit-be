package contracts

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	portservices "boxengage/backend/internal/ports/services"
	"github.com/google/uuid"
)

var (
	ErrInvalidContractInput              = errors.New("invalid contract input")
	ErrContractTemplateNotFound          = portrepo.ErrContractTemplateNotFound
	ErrContractTemplateNameConflict      = portrepo.ErrContractTemplateNameConflict
	ErrContractTemplateDraftExists       = portrepo.ErrContractTemplateDraftExists
	ErrContractTemplateRevisionConflict  = portrepo.ErrContractTemplateRevisionConflict
	ErrContractInstanceNotFound          = portrepo.ErrContractInstanceNotFound
	ErrContractInstanceExists            = portrepo.ErrContractInstanceExists
	ErrContractRevisionConflict          = portrepo.ErrContractRevisionConflict
	ErrContractAlreadyIssued             = portrepo.ErrContractAlreadyIssued
	ErrContractAcceptanceSessionInvalid  = portrepo.ErrContractAcceptanceSessionInvalid
	ErrContractCommandIdempotencyReused  = portrepo.ErrContractCommandIdempotencyReused
	ErrContractEnrollmentUnavailable     = errors.New("contract enrollment unavailable")
	ErrContractPartyIneligible           = errors.New("contract party ineligible")
	ErrContractPreviewChanged            = errors.New("contract preview changed")
	ErrContractAcceptanceDocumentChanged = errors.New("contract acceptance document changed")
	ErrContractAcceptanceUnavailable     = errors.New("contract acceptance unavailable")
)

var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)

type CreateTemplateInput struct {
	Name                string `json:"name"`
	Title               string `json:"title"`
	BodyMarkdown        string `json:"body_markdown"`
	AcceptanceStatement string `json:"acceptance_statement"`
}

type UpdateTemplateDraftInput struct {
	Name                string `json:"name"`
	Title               string `json:"title"`
	BodyMarkdown        string `json:"body_markdown"`
	AcceptanceStatement string `json:"acceptance_statement"`
	ExpectedRevision    int    `json:"expected_revision"`
}

type PublishTemplateInput struct {
	DraftRevision int `json:"draft_revision"`
}

type CreateNewVersionInput struct {
	Title               string `json:"title"`
	BodyMarkdown        string `json:"body_markdown"`
	AcceptanceStatement string `json:"acceptance_statement"`
}

type PreviewContractInput struct {
	TemplateID        string `json:"template_id"`
	TemplateVersionID string `json:"template_version_id"`
}

type CreateInstanceDraftInput struct {
	TemplateID            string `json:"template_id"`
	TemplateVersionID     string `json:"template_version_id"`
	ExpectedPreviewSHA256 string `json:"expected_preview_sha256"`
}

type IssueInstanceInput struct {
	ExpectedRevision int `json:"expected_revision"`
}

type DiscardInstanceInput struct {
	ExpectedRevision int `json:"expected_revision"`
}

type CreateAcceptanceSessionInput struct{}

type AcceptContractInput struct {
	DocumentSHA256             string `json:"document_sha256"`
	AcceptanceStatementVersion int    `json:"acceptance_statement_version"`
	Accepted                   bool   `json:"accepted"`
}

type ServiceOption func(*Service)

func WithContractClock(now func() time.Time) ServiceOption {
	return func(s *Service) { s.now = now }
}

func WithSessionTTL(ttl time.Duration) ServiceOption {
	return func(s *Service) { s.sessionTTL = ttl }
}

type Service struct {
	repository         portrepo.ContractRepository
	enrollmentResolver portrepo.ContractEnrollmentSnapshotResolver
	partyResolver      portrepo.ContractPartyResolver
	boxes              portrepo.BoxRepository
	acceptancePort     portservices.ContractAcceptancePort
	transactions       portrepo.TransactionManager
	sessionTTL         time.Duration
	now                func() time.Time
}

func NewService(
	repository portrepo.ContractRepository,
	enrollmentResolver portrepo.ContractEnrollmentSnapshotResolver,
	partyResolver portrepo.ContractPartyResolver,
	boxes portrepo.BoxRepository,
	acceptancePort portservices.ContractAcceptancePort,
	transactions portrepo.TransactionManager,
	options ...ServiceOption,
) *Service {
	s := &Service{
		repository:         repository,
		enrollmentResolver: enrollmentResolver,
		partyResolver:      partyResolver,
		boxes:              boxes,
		acceptancePort:     acceptancePort,
		transactions:       transactions,
		sessionTTL:         30 * time.Minute,
		now:                func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range options {
		opt(s)
	}
	return s
}

// Templates

func (s *Service) CreateTemplate(ctx context.Context, organizationID, homeUnitID, actorUserID domain.ID, idempotencyKey string, input CreateTemplateInput) (*domain.ContractTemplate, bool, error) {
	started := time.Now()
	if !idempotencyKeyPattern.MatchString(idempotencyKey) || !validID(organizationID) || !validID(homeUnitID) || !validID(actorUserID) {
		return nil, false, ErrInvalidContractInput
	}
	name := strings.TrimSpace(input.Name)
	title := strings.TrimSpace(input.Title)
	body := strings.TrimSpace(input.BodyMarkdown)
	statement := strings.TrimSpace(input.AcceptanceStatement)

	if err := domain.ValidateTemplateTextLimits(name, title, body, statement); err != nil {
		return nil, false, fmt.Errorf("%w: %s", ErrInvalidContractInput, err.Error())
	}
	if err := domain.ValidateMarkdownContent(body); err != nil {
		return nil, false, fmt.Errorf("%w: %s", ErrInvalidContractInput, err.Error())
	}
	if err := domain.ValidatePlaceholders(title, body, statement); err != nil {
		return nil, false, fmt.Errorf("%w: %s", ErrInvalidContractInput, err.Error())
	}

	fingerprint := commandFingerprint(struct {
		OrgID     domain.ID `json:"organization_id"`
		UnitID    domain.ID `json:"home_unit_id"`
		Name      string    `json:"name"`
		Title     string    `json:"title"`
		Body      string    `json:"body_markdown"`
		Statement string    `json:"acceptance_statement"`
	}{organizationID, homeUnitID, name, title, body, statement})

	var result *domain.ContractTemplate
	var replayed bool
	err := s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		prev, err := s.repository.FindCommandRequest(txCtx, organizationID, "template_create", idempotencyKey)
		if err != nil {
			return err
		}
		if prev != nil {
			if !fingerprintsEqual(prev.Fingerprint, fingerprint) {
				return ErrContractCommandIdempotencyReused
			}
			if prev.ResultID != nil {
				result, err = s.repository.FindTemplateByID(txCtx, organizationID, homeUnitID, *prev.ResultID)
				if err != nil {
					return err
				}
				replayed = true
				return nil
			}
		}

		// Check name key conflict
		nameKey := domain.NormalizeTemplateName(name)
		existing, err := s.repository.FindTemplateByNameKey(txCtx, organizationID, nameKey)
		if err != nil {
			return err
		}
		if existing != nil {
			return ErrContractTemplateNameConflict
		}

		now := s.now().UTC()
		templateID := domain.ID(uuid.NewString())
		draftID := domain.ID(uuid.NewString())
		code := fmt.Sprintf("tmpl_%s", strings.ReplaceAll(uuid.NewString(), "-", "")[:16])

		tmpl := &domain.ContractTemplate{
			ID:             templateID,
			OrganizationID: organizationID,
			HomeUnitID:     homeUnitID,
			Code:           code,
			Name:           name,
			NameKey:        nameKey,
			ContractKind:   domain.ContractKindMembershipTerms,
			Status:         domain.ContractTemplateStatusActive,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		draft := &domain.ContractTemplateVersion{
			ID:                       draftID,
			OrganizationID:           organizationID,
			TemplateID:               templateID,
			VersionNumber:            1,
			Revision:                 1,
			Status:                   domain.ContractTemplateVersionStatusDraft,
			Title:                    title,
			BodyMarkdown:             body,
			AcceptanceStatement:      statement,
			PlaceholderSchemaVersion: domain.PlaceholderSchemaVersion1,
			RendererVersion:          domain.RendererVersionMarkdownV1,
			CreatedAt:                now,
			UpdatedAt:                now,
		}

		if err := s.repository.SaveTemplateDraft(txCtx, tmpl, draft); err != nil {
			return err
		}

		auditEvent := &domain.ContractTemplateAuditEvent{
			ID:                domain.ID(uuid.NewString()),
			OrganizationID:    organizationID,
			UnitID:            homeUnitID,
			TemplateID:        templateID,
			TemplateVersionID: &draftID,
			ActorUserID:       actorUserID,
			Action:            "template.draft_created",
			AfterData: map[string]any{
				"name":           name,
				"version_number": 1,
				"status":         "draft",
			},
			CreatedAt: now,
		}
		if err := s.repository.RecordTemplateAuditEvent(txCtx, auditEvent); err != nil {
			return err
		}

		cmdReq := &domain.ContractCommandRequest{
			ID:             domain.ID(uuid.NewString()),
			OrganizationID: organizationID,
			Command:        "template_create",
			IdempotencyKey: idempotencyKey,
			Fingerprint:    fingerprint,
			ResultID:       &templateID,
			ResponseData:   map[string]any{"template_id": templateID},
			CreatedAt:      now,
		}
		if err := s.repository.SaveCommandRequest(txCtx, cmdReq); err != nil {
			return err
		}

		result, err = s.repository.FindTemplateByID(txCtx, organizationID, homeUnitID, templateID)
		return err
	})

	s.logCommand("template_create", organizationID, homeUnitID, actorUserID, resultID(result), replayed, time.Since(started), err)
	return result, replayed, err
}

func (s *Service) UpdateTemplateDraft(ctx context.Context, organizationID, homeUnitID, templateID, actorUserID domain.ID, input UpdateTemplateDraftInput) (*domain.ContractTemplate, error) {
	started := time.Now()
	if !validID(organizationID) || !validID(homeUnitID) || !validID(templateID) || !validID(actorUserID) || input.ExpectedRevision < 1 {
		return nil, ErrInvalidContractInput
	}
	title := strings.TrimSpace(input.Title)
	body := strings.TrimSpace(input.BodyMarkdown)
	statement := strings.TrimSpace(input.AcceptanceStatement)

	name := "Draft"
	if input.Name != "" {
		name = strings.TrimSpace(input.Name)
	}

	if err := domain.ValidateTemplateTextLimits(name, title, body, statement); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidContractInput, err.Error())
	}
	if err := domain.ValidateMarkdownContent(body); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidContractInput, err.Error())
	}
	if err := domain.ValidatePlaceholders(title, body, statement); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidContractInput, err.Error())
	}

	var result *domain.ContractTemplate
	err := s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		tmpl, err := s.repository.FindTemplateByID(txCtx, organizationID, homeUnitID, templateID)
		if err != nil {
			return err
		}
		if tmpl.Status == domain.ContractTemplateStatusArchived {
			return ErrContractTemplateNotFound
		}

		draft := &domain.ContractTemplateVersion{
			Title:               title,
			BodyMarkdown:        body,
			AcceptanceStatement: statement,
		}
		if err := s.repository.UpdateTemplateDraft(txCtx, organizationID, homeUnitID, templateID, draft, input.ExpectedRevision); err != nil {
			return err
		}

		now := s.now().UTC()
		auditEvent := &domain.ContractTemplateAuditEvent{
			ID:             domain.ID(uuid.NewString()),
			OrganizationID: organizationID,
			UnitID:         homeUnitID,
			TemplateID:     templateID,
			ActorUserID:    actorUserID,
			Action:         "template.draft_updated",
			BeforeData: map[string]any{
				"revision": input.ExpectedRevision,
			},
			AfterData: map[string]any{
				"revision": input.ExpectedRevision + 1,
			},
			CreatedAt: now,
		}
		if err := s.repository.RecordTemplateAuditEvent(txCtx, auditEvent); err != nil {
			return err
		}

		result, err = s.repository.FindTemplateByID(txCtx, organizationID, homeUnitID, templateID)
		return err
	})

	s.logCommand("template_draft_update", organizationID, homeUnitID, actorUserID, templateID, false, time.Since(started), err)
	return result, err
}

func (s *Service) PublishTemplate(ctx context.Context, organizationID, homeUnitID, templateID, actorUserID domain.ID, idempotencyKey string, input PublishTemplateInput) (*domain.ContractTemplate, bool, error) {
	started := time.Now()
	if !idempotencyKeyPattern.MatchString(idempotencyKey) || !validID(organizationID) || !validID(homeUnitID) || !validID(templateID) || !validID(actorUserID) || input.DraftRevision < 1 {
		return nil, false, ErrInvalidContractInput
	}

	fingerprint := commandFingerprint(struct {
		OrgID         domain.ID `json:"organization_id"`
		UnitID        domain.ID `json:"home_unit_id"`
		TemplateID    domain.ID `json:"template_id"`
		DraftRevision int       `json:"draft_revision"`
	}{organizationID, homeUnitID, templateID, input.DraftRevision})

	var result *domain.ContractTemplate
	var replayed bool
	err := s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		prev, err := s.repository.FindCommandRequest(txCtx, organizationID, "template_publish", idempotencyKey)
		if err != nil {
			return err
		}
		if prev != nil {
			if !fingerprintsEqual(prev.Fingerprint, fingerprint) {
				return ErrContractCommandIdempotencyReused
			}
			if prev.ResultID != nil {
				result, err = s.repository.FindTemplateByID(txCtx, organizationID, homeUnitID, *prev.ResultID)
				if err != nil {
					return err
				}
				replayed = true
				return nil
			}
		}

		tmpl, err := s.repository.FindTemplateByID(txCtx, organizationID, homeUnitID, templateID)
		if err != nil {
			return err
		}
		if tmpl.Status != domain.ContractTemplateStatusActive || tmpl.CurrentDraft == nil {
			return ErrContractTemplateNotFound
		}
		draft := tmpl.CurrentDraft
		if draft.Revision != input.DraftRevision {
			return ErrContractTemplateRevisionConflict
		}

		if err := domain.ValidateTemplateTextLimits(tmpl.Name, draft.Title, draft.BodyMarkdown, draft.AcceptanceStatement); err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidContractInput, err.Error())
		}
		if err := domain.ValidateMarkdownContent(draft.BodyMarkdown); err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidContractInput, err.Error())
		}
		if err := domain.ValidatePlaceholders(draft.Title, draft.BodyMarkdown, draft.AcceptanceStatement); err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidContractInput, err.Error())
		}

		contentSHA256, err := domain.CalculateTemplateVersionContentSHA256(draft.Title, draft.BodyMarkdown, draft.AcceptanceStatement)
		if err != nil {
			return err
		}

		now := s.now().UTC()
		if err := s.repository.PublishTemplateVersion(txCtx, organizationID, homeUnitID, templateID, draft.ID, input.DraftRevision, now, contentSHA256); err != nil {
			return err
		}

		auditEvent := &domain.ContractTemplateAuditEvent{
			ID:                domain.ID(uuid.NewString()),
			OrganizationID:    organizationID,
			UnitID:            homeUnitID,
			TemplateID:        templateID,
			TemplateVersionID: &draft.ID,
			ActorUserID:       actorUserID,
			Action:            "template.published",
			AfterData: map[string]any{
				"version_number": draft.VersionNumber,
				"content_sha256": contentSHA256,
			},
			CreatedAt: now,
		}
		if err := s.repository.RecordTemplateAuditEvent(txCtx, auditEvent); err != nil {
			return err
		}

		cmdReq := &domain.ContractCommandRequest{
			ID:             domain.ID(uuid.NewString()),
			OrganizationID: organizationID,
			Command:        "template_publish",
			IdempotencyKey: idempotencyKey,
			Fingerprint:    fingerprint,
			ResultID:       &templateID,
			ResponseData:   map[string]any{"template_id": templateID, "version_number": draft.VersionNumber},
			CreatedAt:      now,
		}
		if err := s.repository.SaveCommandRequest(txCtx, cmdReq); err != nil {
			return err
		}

		result, err = s.repository.FindTemplateByID(txCtx, organizationID, homeUnitID, templateID)
		return err
	})

	s.logCommand("template_publish", organizationID, homeUnitID, actorUserID, templateID, replayed, time.Since(started), err)
	return result, replayed, err
}

func (s *Service) CreateNewVersion(ctx context.Context, organizationID, homeUnitID, templateID, actorUserID domain.ID, idempotencyKey string, input CreateNewVersionInput) (*domain.ContractTemplate, bool, error) {
	started := time.Now()
	if !idempotencyKeyPattern.MatchString(idempotencyKey) || !validID(organizationID) || !validID(homeUnitID) || !validID(templateID) || !validID(actorUserID) {
		return nil, false, ErrInvalidContractInput
	}
	title := strings.TrimSpace(input.Title)
	body := strings.TrimSpace(input.BodyMarkdown)
	statement := strings.TrimSpace(input.AcceptanceStatement)

	fingerprint := commandFingerprint(struct {
		OrgID      domain.ID `json:"organization_id"`
		UnitID     domain.ID `json:"home_unit_id"`
		TemplateID domain.ID `json:"template_id"`
		Title      string    `json:"title"`
		Body       string    `json:"body_markdown"`
		Statement  string    `json:"acceptance_statement"`
	}{organizationID, homeUnitID, templateID, title, body, statement})

	var result *domain.ContractTemplate
	var replayed bool
	err := s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		prev, err := s.repository.FindCommandRequest(txCtx, organizationID, "template_new_version", idempotencyKey)
		if err != nil {
			return err
		}
		if prev != nil {
			if !fingerprintsEqual(prev.Fingerprint, fingerprint) {
				return ErrContractCommandIdempotencyReused
			}
			if prev.ResultID != nil {
				result, err = s.repository.FindTemplateByID(txCtx, organizationID, homeUnitID, *prev.ResultID)
				if err != nil {
					return err
				}
				replayed = true
				return nil
			}
		}

		tmpl, err := s.repository.FindTemplateByID(txCtx, organizationID, homeUnitID, templateID)
		if err != nil {
			return err
		}
		if tmpl.Status != domain.ContractTemplateStatusActive {
			return ErrContractTemplateNotFound
		}
		if tmpl.CurrentDraft != nil {
			return ErrContractTemplateDraftExists
		}

		// If input fields are empty, inherit from current published version
		if title == "" && tmpl.CurrentVersion != nil {
			title = tmpl.CurrentVersion.Title
		}
		if body == "" && tmpl.CurrentVersion != nil {
			body = tmpl.CurrentVersion.BodyMarkdown
		}
		if statement == "" && tmpl.CurrentVersion != nil {
			statement = tmpl.CurrentVersion.AcceptanceStatement
		}

		if err := domain.ValidateTemplateTextLimits(tmpl.Name, title, body, statement); err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidContractInput, err.Error())
		}
		if err := domain.ValidateMarkdownContent(body); err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidContractInput, err.Error())
		}
		if err := domain.ValidatePlaceholders(title, body, statement); err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidContractInput, err.Error())
		}

		now := s.now().UTC()
		newDraftID := domain.ID(uuid.NewString())
		newDraft := &domain.ContractTemplateVersion{
			ID:                       newDraftID,
			Title:                    title,
			BodyMarkdown:             body,
			AcceptanceStatement:      statement,
			PlaceholderSchemaVersion: domain.PlaceholderSchemaVersion1,
			RendererVersion:          domain.RendererVersionMarkdownV1,
			CreatedAt:                now,
			UpdatedAt:                now,
		}

		if err := s.repository.CreateNewTemplateVersion(txCtx, organizationID, homeUnitID, templateID, newDraft); err != nil {
			return err
		}

		auditEvent := &domain.ContractTemplateAuditEvent{
			ID:                domain.ID(uuid.NewString()),
			OrganizationID:    organizationID,
			UnitID:            homeUnitID,
			TemplateID:        templateID,
			TemplateVersionID: &newDraftID,
			ActorUserID:       actorUserID,
			Action:            "template.new_version_created",
			AfterData: map[string]any{
				"version_number": newDraft.VersionNumber,
				"status":         "draft",
			},
			CreatedAt: now,
		}
		if err := s.repository.RecordTemplateAuditEvent(txCtx, auditEvent); err != nil {
			return err
		}

		cmdReq := &domain.ContractCommandRequest{
			ID:             domain.ID(uuid.NewString()),
			OrganizationID: organizationID,
			Command:        "template_new_version",
			IdempotencyKey: idempotencyKey,
			Fingerprint:    fingerprint,
			ResultID:       &templateID,
			ResponseData:   map[string]any{"template_id": templateID, "draft_id": newDraftID},
			CreatedAt:      now,
		}
		if err := s.repository.SaveCommandRequest(txCtx, cmdReq); err != nil {
			return err
		}

		result, err = s.repository.FindTemplateByID(txCtx, organizationID, homeUnitID, templateID)
		return err
	})

	s.logCommand("template_new_version", organizationID, homeUnitID, actorUserID, templateID, replayed, time.Since(started), err)
	return result, replayed, err
}

func (s *Service) ArchiveTemplate(ctx context.Context, organizationID, homeUnitID, templateID, actorUserID domain.ID) (*domain.ContractTemplate, error) {
	started := time.Now()
	if !validID(organizationID) || !validID(homeUnitID) || !validID(templateID) || !validID(actorUserID) {
		return nil, ErrInvalidContractInput
	}

	var result *domain.ContractTemplate
	err := s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.repository.ArchiveTemplate(txCtx, organizationID, homeUnitID, templateID); err != nil {
			return err
		}
		now := s.now().UTC()
		auditEvent := &domain.ContractTemplateAuditEvent{
			ID:             domain.ID(uuid.NewString()),
			OrganizationID: organizationID,
			UnitID:         homeUnitID,
			TemplateID:     templateID,
			ActorUserID:    actorUserID,
			Action:         "template.archived",
			CreatedAt:      now,
		}
		if err := s.repository.RecordTemplateAuditEvent(txCtx, auditEvent); err != nil {
			return err
		}
		var err error
		result, err = s.repository.FindTemplateByID(txCtx, organizationID, homeUnitID, templateID)
		return err
	})

	s.logCommand("template_archive", organizationID, homeUnitID, actorUserID, templateID, false, time.Since(started), err)
	return result, err
}

func (s *Service) ReactivateTemplate(ctx context.Context, organizationID, homeUnitID, templateID, actorUserID domain.ID) (*domain.ContractTemplate, error) {
	started := time.Now()
	if !validID(organizationID) || !validID(homeUnitID) || !validID(templateID) || !validID(actorUserID) {
		return nil, ErrInvalidContractInput
	}

	var result *domain.ContractTemplate
	err := s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.repository.ReactivateTemplate(txCtx, organizationID, homeUnitID, templateID); err != nil {
			return err
		}
		now := s.now().UTC()
		auditEvent := &domain.ContractTemplateAuditEvent{
			ID:             domain.ID(uuid.NewString()),
			OrganizationID: organizationID,
			UnitID:         homeUnitID,
			TemplateID:     templateID,
			ActorUserID:    actorUserID,
			Action:         "template.reactivated",
			CreatedAt:      now,
		}
		if err := s.repository.RecordTemplateAuditEvent(txCtx, auditEvent); err != nil {
			return err
		}
		var err error
		result, err = s.repository.FindTemplateByID(txCtx, organizationID, homeUnitID, templateID)
		return err
	})

	s.logCommand("template_reactivate", organizationID, homeUnitID, actorUserID, templateID, false, time.Since(started), err)
	return result, err
}

func (s *Service) GetTemplate(ctx context.Context, organizationID, homeUnitID, templateID domain.ID) (*domain.ContractTemplate, error) {
	return s.repository.FindTemplateByID(ctx, organizationID, homeUnitID, templateID)
}

func (s *Service) ListTemplates(ctx context.Context, organizationID, homeUnitID domain.ID, includeArchived bool) ([]domain.ContractTemplate, error) {
	return s.repository.ListTemplates(ctx, organizationID, homeUnitID, includeArchived)
}

// Instances & Preview

func (s *Service) PreviewEnrollmentContract(ctx context.Context, organizationID, unitID, enrollmentID domain.ID, input PreviewContractInput) (*domain.ContractPreview, error) {
	if !validID(organizationID) || !validID(unitID) || !validID(enrollmentID) || !validID(domain.ID(input.TemplateID)) || !validID(domain.ID(input.TemplateVersionID)) {
		return nil, ErrInvalidContractInput
	}

	tmpl, err := s.repository.FindTemplateByID(ctx, organizationID, unitID, domain.ID(input.TemplateID))
	if err != nil {
		return nil, err
	}
	if tmpl.Status != domain.ContractTemplateStatusActive {
		return nil, ErrContractTemplateNotFound
	}

	tmplVer, err := s.repository.FindTemplateVersionByID(ctx, organizationID, domain.ID(input.TemplateVersionID))
	if err != nil {
		return nil, err
	}
	if tmplVer.TemplateID != tmpl.ID || tmplVer.Status != domain.ContractTemplateVersionStatusPublished {
		return nil, ErrContractTemplateNotFound
	}

	enrSnapshot, err := s.enrollmentResolver.ResolveContractEnrollmentSnapshot(ctx, organizationID, unitID, enrollmentID, s.now())
	if err != nil {
		return nil, err
	}
	if enrSnapshot == nil {
		return nil, ErrContractEnrollmentUnavailable
	}
	if enrSnapshot.EffectiveStatus != domain.EnrollmentEffectiveScheduled && enrSnapshot.EffectiveStatus != domain.EnrollmentEffectiveActive {
		return nil, fmt.Errorf("%w: status %s is not eligible for contract", ErrContractEnrollmentUnavailable, enrSnapshot.EffectiveStatus)
	}

	party, err := s.partyResolver.ResolveContractParty(ctx, organizationID, enrSnapshot.PersonID)
	if err != nil {
		return nil, err
	}
	if party == nil || !party.Eligible {
		return nil, ErrContractPartyIneligible
	}

	box, err := s.boxes.FindByID(ctx, unitID)
	if err != nil {
		return nil, err
	}
	unitName := box.Name
	orgName := box.Name // in current organization model box represents the organization unit

	// Build commercial snapshot
	commAddons := make([]domain.ContractCommercialAddonSnapshot, 0, len(enrSnapshot.Addons))
	addonSummaries := make([]string, 0, len(enrSnapshot.Addons))
	for _, a := range enrSnapshot.Addons {
		commAddons = append(commAddons, domain.ContractCommercialAddonSnapshot{
			PlanVersionAddonID: a.PlanVersionAddonID,
			Name:               a.Name,
			PriceDeltaCents:    a.PriceDeltaCents,
			Summary:            a.Summary,
		})
		addonSummaries = append(addonSummaries, fmt.Sprintf("%s (%s)", a.Name, domain.FormatBRL(a.PriceDeltaCents)))
	}

	// Deterministic order of addons by ID/position
	sort.SliceStable(commAddons, func(i, j int) bool {
		return commAddons[i].PlanVersionAddonID < commAddons[j].PlanVersionAddonID
	})

	var validUntilStr *string
	if enrSnapshot.ValidUntil != nil {
		s := enrSnapshot.ValidUntil.Format("2006-01-02")
		validUntilStr = &s
	}

	commSnapshot := domain.ContractCommercialSnapshot{
		SchemaVersion:      domain.SnapshotSchemaVersion1,
		EnrollmentID:       enrollmentID,
		EnrollmentRevision: enrSnapshot.EnrollmentRevision,
		PlanID:             enrSnapshot.PlanID,
		PlanVersionID:      enrSnapshot.PlanVersionID,
		PlanVersionNumber:  enrSnapshot.PlanVersionNumber,
		PlanName:           enrSnapshot.PlanName,
		BillingCycle:       enrSnapshot.BillingCycle,
		Currency:           enrSnapshot.Currency,
		BasePriceCents:     enrSnapshot.BasePriceCents,
		AddonsPriceCents:   enrSnapshot.AddonsPriceCents,
		TotalPriceCents:    enrSnapshot.TotalPriceCents,
		StartsOn:           enrSnapshot.StartsOn.Format("2006-01-02"),
		ValidUntil:         validUntilStr,
		BaseSummary:        enrSnapshot.BaseSummary,
		Addons:             commAddons,
	}

	addonsSummaryText := "Sem adicionais"
	if len(addonSummaries) > 0 {
		addonsSummaryText = strings.Join(addonSummaries, ", ")
	}

	validUntilFormatted := "Sem término definido"
	if enrSnapshot.ValidUntil != nil {
		validUntilFormatted = enrSnapshot.ValidUntil.Format("02/01/2006")
	}

	renderCtx := domain.ContractRenderContext{
		OrganizationName: orgName,
		UnitName:         unitName,
		MemberName:       party.DisplayName,
		PlanName:         enrSnapshot.PlanName,
		PlanVersion:      fmt.Sprintf("v%d", enrSnapshot.PlanVersionNumber),
		BillingCycle:     domain.FormatBillingCycle(enrSnapshot.BillingCycle),
		BasePrice:        domain.FormatBRL(enrSnapshot.BasePriceCents),
		AddonsSummary:    addonsSummaryText,
		AddonsPrice:      domain.FormatBRL(enrSnapshot.AddonsPriceCents),
		TotalPrice:       domain.FormatBRL(enrSnapshot.TotalPriceCents),
		StartsOn:         enrSnapshot.StartsOn.Format("02/01/2006"),
		ValidUntil:       validUntilFormatted,
		AccessSummary:    enrSnapshot.BaseSummary,
	}

	renderedTitle := domain.RenderTemplateText(tmplVer.Title, renderCtx)
	renderedBody := domain.RenderTemplateText(tmplVer.BodyMarkdown, renderCtx)
	renderedStatement := domain.RenderTemplateText(tmplVer.AcceptanceStatement, renderCtx)

	commSHA, err := domain.CalculateCommercialSnapshotSHA256(commSnapshot)
	if err != nil {
		return nil, err
	}
	previewSHA, err := domain.CalculateDocumentSHA256(renderedTitle, renderedBody, renderedStatement, commSHA)
	if err != nil {
		return nil, err
	}

	return &domain.ContractPreview{
		TemplateID:           tmpl.ID,
		TemplateVersionID:    tmplVer.ID,
		EnrollmentID:         enrollmentID,
		DocumentTitle:        renderedTitle,
		DocumentBodyMarkdown: renderedBody,
		AcceptanceStatement:  renderedStatement,
		CommercialSnapshot:   commSnapshot,
		PreviewSHA256:        previewSHA,
		Warnings:             []string{"Esta etapa não gera cobrança, não altera a matrícula e não libera acesso."},
	}, nil
}

func (s *Service) CreateInstanceDraft(ctx context.Context, organizationID, unitID, enrollmentID, actorUserID domain.ID, idempotencyKey string, input CreateInstanceDraftInput) (*domain.ContractInstance, bool, error) {
	started := time.Now()
	if !idempotencyKeyPattern.MatchString(idempotencyKey) || !validID(organizationID) || !validID(unitID) || !validID(enrollmentID) || !validID(actorUserID) || input.ExpectedPreviewSHA256 == "" {
		return nil, false, ErrInvalidContractInput
	}

	preview, err := s.PreviewEnrollmentContract(ctx, organizationID, unitID, enrollmentID, PreviewContractInput{
		TemplateID:        input.TemplateID,
		TemplateVersionID: input.TemplateVersionID,
	})
	if err != nil {
		return nil, false, err
	}

	if preview.PreviewSHA256 != input.ExpectedPreviewSHA256 {
		return nil, false, ErrContractPreviewChanged
	}

	fingerprint := commandFingerprint(struct {
		OrgID             domain.ID `json:"organization_id"`
		UnitID            domain.ID `json:"unit_id"`
		EnrollmentID      domain.ID `json:"enrollment_id"`
		TemplateVersionID string    `json:"template_version_id"`
		ExpectedSHA       string    `json:"expected_preview_sha256"`
	}{organizationID, unitID, enrollmentID, input.TemplateVersionID, input.ExpectedPreviewSHA256})

	var result *domain.ContractInstance
	var replayed bool
	err = s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		prev, err := s.repository.FindCommandRequest(txCtx, organizationID, "instance_create", idempotencyKey)
		if err != nil {
			return err
		}
		if prev != nil {
			if !fingerprintsEqual(prev.Fingerprint, fingerprint) {
				return ErrContractCommandIdempotencyReused
			}
			if prev.ResultID != nil {
				result, err = s.repository.FindInstanceByID(txCtx, organizationID, unitID, *prev.ResultID)
				if err != nil {
					return err
				}
				replayed = true
				return nil
			}
		}

		existing, err := s.repository.FindActiveInstanceByEnrollmentID(txCtx, organizationID, unitID, enrollmentID)
		if err != nil {
			return err
		}
		if existing != nil {
			return ErrContractInstanceExists
		}

		enrSnapshot, err := s.enrollmentResolver.ResolveContractEnrollmentSnapshot(txCtx, organizationID, unitID, enrollmentID, s.now())
		if err != nil {
			return err
		}
		party, err := s.partyResolver.ResolveContractParty(txCtx, organizationID, enrSnapshot.PersonID)
		if err != nil {
			return err
		}
		box, err := s.boxes.FindByID(txCtx, unitID)
		if err != nil {
			return err
		}

		now := s.now().UTC()
		instanceID := domain.ID(uuid.NewString())
		retentionDue := now.Add(30 * 24 * time.Hour)

		commSHA, err := domain.CalculateCommercialSnapshotSHA256(preview.CommercialSnapshot)
		if err != nil {
			return err
		}

		partySnapshot := domain.ContractPartySnapshot{
			OrganizationID:   organizationID,
			OrganizationName: box.Name,
			UnitID:           unitID,
			UnitName:         box.Name,
			PersonID:         party.PersonID,
			DisplayName:      party.DisplayName,
		}

		instance := &domain.ContractInstance{
			ID:                         instanceID,
			OrganizationID:             organizationID,
			UnitID:                     unitID,
			PersonID:                   party.PersonID,
			EnrollmentID:               enrollmentID,
			TemplateVersionID:          preview.TemplateVersionID,
			ContractKind:               domain.ContractKindMembershipTerms,
			Status:                     domain.ContractInstanceStatusDraft,
			Revision:                   1,
			SnapshotSchemaVersion:      domain.SnapshotSchemaVersion1,
			CommercialSnapshot:         preview.CommercialSnapshot,
			CommercialSnapshotSHA256:   commSHA,
			PartySnapshot:              partySnapshot,
			DocumentTitle:              preview.DocumentTitle,
			DocumentBodyMarkdown:       preview.DocumentBodyMarkdown,
			DocumentSHA256:             preview.PreviewSHA256,
			AcceptanceStatement:        preview.AcceptanceStatement,
			AcceptanceStatementVersion: 1,
			CreatedByUserID:            actorUserID,
			RetentionClass:             domain.ContractRetentionClassLocalSimulator,
			RetentionDueAt:             &retentionDue,
			CreatedAt:                  now,
			UpdatedAt:                  now,
		}

		if err := s.repository.SaveInstanceDraft(txCtx, instance); err != nil {
			return err
		}

		event := &domain.ContractEvent{
			ID:                 domain.ID(uuid.NewString()),
			OrganizationID:     organizationID,
			UnitID:             unitID,
			PersonID:           party.PersonID,
			EnrollmentID:       enrollmentID,
			ContractInstanceID: instanceID,
			ActorType:          domain.ContractActorOwner,
			ActorUserID:        &actorUserID,
			Action:             "contract.draft_created",
			AfterData: map[string]any{
				"status":              "draft",
				"document_sha256":     preview.PreviewSHA256,
				"template_version_id": preview.TemplateVersionID,
			},
			CreatedAt: now,
		}
		if err := s.repository.RecordEvent(txCtx, event); err != nil {
			return err
		}

		cmdReq := &domain.ContractCommandRequest{
			ID:             domain.ID(uuid.NewString()),
			OrganizationID: organizationID,
			Command:        "instance_create",
			IdempotencyKey: idempotencyKey,
			Fingerprint:    fingerprint,
			ResultID:       &instanceID,
			ResponseData:   map[string]any{"contract_id": instanceID},
			CreatedAt:      now,
		}
		if err := s.repository.SaveCommandRequest(txCtx, cmdReq); err != nil {
			return err
		}

		result, err = s.repository.FindInstanceByID(txCtx, organizationID, unitID, instanceID)
		return err
	})

	s.logCommand("instance_create", organizationID, unitID, actorUserID, instanceResultID(result), replayed, time.Since(started), err)
	return result, replayed, err
}

func (s *Service) IssueInstance(ctx context.Context, organizationID, unitID, contractID, actorUserID domain.ID, idempotencyKey string, input IssueInstanceInput) (*domain.ContractInstance, bool, error) {
	started := time.Now()
	if !idempotencyKeyPattern.MatchString(idempotencyKey) || !validID(organizationID) || !validID(unitID) || !validID(contractID) || !validID(actorUserID) || input.ExpectedRevision < 1 {
		return nil, false, ErrInvalidContractInput
	}

	fingerprint := commandFingerprint(struct {
		OrgID            domain.ID `json:"organization_id"`
		UnitID           domain.ID `json:"unit_id"`
		ContractID       domain.ID `json:"contract_id"`
		ExpectedRevision int       `json:"expected_revision"`
	}{organizationID, unitID, contractID, input.ExpectedRevision})

	var result *domain.ContractInstance
	var replayed bool
	err := s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		prev, err := s.repository.FindCommandRequest(txCtx, organizationID, "instance_issue", idempotencyKey)
		if err != nil {
			return err
		}
		if prev != nil {
			if !fingerprintsEqual(prev.Fingerprint, fingerprint) {
				return ErrContractCommandIdempotencyReused
			}
			if prev.ResultID != nil {
				result, err = s.repository.FindInstanceByID(txCtx, organizationID, unitID, *prev.ResultID)
				if err != nil {
					return err
				}
				replayed = true
				return nil
			}
		}

		inst, err := s.repository.FindInstanceByID(txCtx, organizationID, unitID, contractID)
		if err != nil {
			return err
		}
		if inst.Status != domain.ContractInstanceStatusDraft {
			return ErrContractAlreadyIssued
		}

		// Revalidate enrollment and person status
		enrSnapshot, err := s.enrollmentResolver.ResolveContractEnrollmentSnapshot(txCtx, organizationID, unitID, inst.EnrollmentID, s.now())
		if err != nil {
			return err
		}
		if enrSnapshot == nil || (enrSnapshot.EffectiveStatus != domain.EnrollmentEffectiveScheduled && enrSnapshot.EffectiveStatus != domain.EnrollmentEffectiveActive) {
			return ErrContractEnrollmentUnavailable
		}
		party, err := s.partyResolver.ResolveContractParty(txCtx, organizationID, inst.PersonID)
		if err != nil {
			return err
		}
		if party == nil || !party.Eligible {
			return ErrContractPartyIneligible
		}

		now := s.now().UTC()
		result, err = s.repository.IssueInstance(txCtx, organizationID, unitID, contractID, actorUserID, input.ExpectedRevision, now)
		if err != nil {
			return err
		}

		event := &domain.ContractEvent{
			ID:                 domain.ID(uuid.NewString()),
			OrganizationID:     organizationID,
			UnitID:             unitID,
			PersonID:           inst.PersonID,
			EnrollmentID:       inst.EnrollmentID,
			ContractInstanceID: contractID,
			ActorType:          domain.ContractActorOwner,
			ActorUserID:        &actorUserID,
			Action:             "contract.issued",
			BeforeData: map[string]any{
				"status":   "draft",
				"revision": input.ExpectedRevision,
			},
			AfterData: map[string]any{
				"status":    "pending_acceptance",
				"revision":  result.Revision,
				"issued_at": now.Format(time.RFC3339),
			},
			CreatedAt: now,
		}
		if err := s.repository.RecordEvent(txCtx, event); err != nil {
			return err
		}

		cmdReq := &domain.ContractCommandRequest{
			ID:             domain.ID(uuid.NewString()),
			OrganizationID: organizationID,
			Command:        "instance_issue",
			IdempotencyKey: idempotencyKey,
			Fingerprint:    fingerprint,
			ResultID:       &contractID,
			ResponseData:   map[string]any{"contract_id": contractID, "status": "pending_acceptance"},
			CreatedAt:      now,
		}
		return s.repository.SaveCommandRequest(txCtx, cmdReq)
	})

	s.logCommand("instance_issue", organizationID, unitID, actorUserID, contractID, replayed, time.Since(started), err)
	return result, replayed, err
}

func (s *Service) DiscardInstance(ctx context.Context, organizationID, unitID, contractID, actorUserID domain.ID, idempotencyKey string, input DiscardInstanceInput) (*domain.ContractInstance, bool, error) {
	started := time.Now()
	if !idempotencyKeyPattern.MatchString(idempotencyKey) || !validID(organizationID) || !validID(unitID) || !validID(contractID) || !validID(actorUserID) || input.ExpectedRevision < 1 {
		return nil, false, ErrInvalidContractInput
	}

	fingerprint := commandFingerprint(struct {
		OrgID            domain.ID `json:"organization_id"`
		UnitID           domain.ID `json:"unit_id"`
		ContractID       domain.ID `json:"contract_id"`
		ExpectedRevision int       `json:"expected_revision"`
	}{organizationID, unitID, contractID, input.ExpectedRevision})

	var result *domain.ContractInstance
	var replayed bool
	err := s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		prev, err := s.repository.FindCommandRequest(txCtx, organizationID, "instance_discard", idempotencyKey)
		if err != nil {
			return err
		}
		if prev != nil {
			if !fingerprintsEqual(prev.Fingerprint, fingerprint) {
				return ErrContractCommandIdempotencyReused
			}
			if prev.ResultID != nil {
				result, err = s.repository.FindInstanceByID(txCtx, organizationID, unitID, *prev.ResultID)
				if err != nil {
					return err
				}
				replayed = true
				return nil
			}
		}

		inst, err := s.repository.FindInstanceByID(txCtx, organizationID, unitID, contractID)
		if err != nil {
			return err
		}
		if inst.Status != domain.ContractInstanceStatusDraft {
			return ErrContractAlreadyIssued
		}

		now := s.now().UTC()
		result, err = s.repository.DiscardInstance(txCtx, organizationID, unitID, contractID, actorUserID, input.ExpectedRevision, now)
		if err != nil {
			return err
		}

		event := &domain.ContractEvent{
			ID:                 domain.ID(uuid.NewString()),
			OrganizationID:     organizationID,
			UnitID:             unitID,
			PersonID:           inst.PersonID,
			EnrollmentID:       inst.EnrollmentID,
			ContractInstanceID: contractID,
			ActorType:          domain.ContractActorOwner,
			ActorUserID:        &actorUserID,
			Action:             "contract.draft_discarded",
			BeforeData: map[string]any{
				"status":   "draft",
				"revision": input.ExpectedRevision,
			},
			AfterData: map[string]any{
				"status":       "discarded",
				"revision":     result.Revision,
				"discarded_at": now.Format(time.RFC3339),
			},
			CreatedAt: now,
		}
		if err := s.repository.RecordEvent(txCtx, event); err != nil {
			return err
		}

		cmdReq := &domain.ContractCommandRequest{
			ID:             domain.ID(uuid.NewString()),
			OrganizationID: organizationID,
			Command:        "instance_discard",
			IdempotencyKey: idempotencyKey,
			Fingerprint:    fingerprint,
			ResultID:       &contractID,
			ResponseData:   map[string]any{"contract_id": contractID, "status": "discarded"},
			CreatedAt:      now,
		}
		return s.repository.SaveCommandRequest(txCtx, cmdReq)
	})

	s.logCommand("instance_discard", organizationID, unitID, actorUserID, contractID, replayed, time.Since(started), err)
	return result, replayed, err
}

func (s *Service) GetInstance(ctx context.Context, organizationID, unitID, instanceID domain.ID) (*domain.ContractInstance, error) {
	return s.repository.FindInstanceByID(ctx, organizationID, unitID, instanceID)
}

func (s *Service) ListForEnrollment(ctx context.Context, organizationID, unitID, enrollmentID domain.ID) ([]domain.ContractInstance, error) {
	return s.repository.ListInstancesForEnrollment(ctx, organizationID, unitID, enrollmentID)
}

// Acceptance Sessions & Simulator

func (s *Service) CreateAcceptanceSession(ctx context.Context, organizationID, unitID, contractID, actorUserID domain.ID, idempotencyKey string) (*domain.AcceptanceSessionLaunch, bool, error) {
	started := time.Now()
	if s.acceptancePort == nil {
		return nil, false, ErrContractAcceptanceUnavailable
	}
	if !idempotencyKeyPattern.MatchString(idempotencyKey) || !validID(organizationID) || !validID(unitID) || !validID(contractID) || !validID(actorUserID) {
		return nil, false, ErrInvalidContractInput
	}

	fingerprint := commandFingerprint(struct {
		OrgID      domain.ID `json:"organization_id"`
		UnitID     domain.ID `json:"unit_id"`
		ContractID domain.ID `json:"contract_id"`
	}{organizationID, unitID, contractID})

	var launch *domain.AcceptanceSessionLaunch
	var replayed bool
	err := s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		prev, err := s.repository.FindCommandRequest(txCtx, organizationID, "acceptance_session_create", idempotencyKey)
		if err != nil {
			return err
		}
		if prev != nil {
			if !fingerprintsEqual(prev.Fingerprint, fingerprint) {
				return ErrContractCommandIdempotencyReused
			}
			// Replay does not return the secret raw token again; returns session metadata
			if prev.ResultID != nil {
				replayed = true
				launch = &domain.AcceptanceSessionLaunch{
					AdapterReference: "replayed_session",
					LaunchPath:       "",
					ExpiresAt:        time.Time{},
				}
				return nil
			}
		}

		inst, err := s.repository.FindInstanceByID(txCtx, organizationID, unitID, contractID)
		if err != nil {
			return err
		}
		if inst.Status != domain.ContractInstanceStatusPendingAcceptance {
			return ErrContractAlreadyIssued
		}

		now := s.now().UTC()
		expiresAt := now.Add(s.sessionTTL)

		var prepErr error
		launch, prepErr = s.acceptancePort.Prepare(txCtx, contractID, inst.DocumentSHA256, expiresAt)
		if prepErr != nil {
			return prepErr
		}

		tokenHash := sha256Hex(launch.LaunchToken)
		sessionID := domain.ID(uuid.NewString())

		if err := s.repository.SupersedeActiveAcceptanceSessions(txCtx, organizationID, contractID); err != nil {
			return err
		}

		session := &domain.ContractAcceptanceSession{
			ID:                 sessionID,
			OrganizationID:     organizationID,
			ContractInstanceID: contractID,
			Adapter:            domain.AdapterLocalSimulator,
			AdapterReference:   launch.AdapterReference,
			TokenHash:          tokenHash,
			Status:             domain.ContractAcceptanceSessionStatusActive,
			ExpiresAt:          expiresAt,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		if err := s.repository.SaveAcceptanceSession(txCtx, session); err != nil {
			return err
		}

		event := &domain.ContractEvent{
			ID:                 domain.ID(uuid.NewString()),
			OrganizationID:     organizationID,
			UnitID:             unitID,
			PersonID:           inst.PersonID,
			EnrollmentID:       inst.EnrollmentID,
			ContractInstanceID: contractID,
			ActorType:          domain.ContractActorOwner,
			ActorUserID:        &actorUserID,
			Action:             "contract.acceptance_session_created",
			AfterData: map[string]any{
				"session_id":        sessionID,
				"adapter":           domain.AdapterLocalSimulator,
				"adapter_reference": launch.AdapterReference,
				"expires_at":        expiresAt.Format(time.RFC3339),
			},
			CreatedAt: now,
		}
		if err := s.repository.RecordEvent(txCtx, event); err != nil {
			return err
		}

		// Save command request without the raw launch token
		cmdReq := &domain.ContractCommandRequest{
			ID:             domain.ID(uuid.NewString()),
			OrganizationID: organizationID,
			Command:        "acceptance_session_create",
			IdempotencyKey: idempotencyKey,
			Fingerprint:    fingerprint,
			ResultID:       &sessionID,
			ResponseData:   map[string]any{"session_id": sessionID, "adapter": domain.AdapterLocalSimulator},
			CreatedAt:      now,
		}
		return s.repository.SaveCommandRequest(txCtx, cmdReq)
	})

	s.logCommand("acceptance_session_create", organizationID, unitID, actorUserID, contractID, replayed, time.Since(started), err)
	return launch, replayed, err
}

func (s *Service) GetPublicAcceptanceSession(ctx context.Context, token string) (*domain.PublicAcceptanceSessionData, error) {
	if s.acceptancePort == nil || token == "" {
		return nil, ErrContractAcceptanceSessionInvalid
	}

	tokenHash := sha256Hex(token)
	session, instance, err := s.repository.FindAcceptanceSessionByTokenHash(ctx, tokenHash)
	if err != nil || session == nil || instance == nil {
		return nil, ErrContractAcceptanceSessionInvalid
	}

	now := s.now().UTC()
	if session.Status != domain.ContractAcceptanceSessionStatusActive || now.After(session.ExpiresAt) {
		return nil, ErrContractAcceptanceSessionInvalid
	}

	return &domain.PublicAcceptanceSessionData{
		Token:                      token,
		Status:                     string(instance.Status),
		DocumentTitle:              instance.DocumentTitle,
		DocumentBodyMarkdown:       instance.DocumentBodyMarkdown,
		AcceptanceStatement:        instance.AcceptanceStatement,
		AcceptanceStatementVersion: instance.AcceptanceStatementVersion,
		DocumentSHA256:             instance.DocumentSHA256,
		CommercialSnapshot:         instance.CommercialSnapshot,
		ExpiresAt:                  session.ExpiresAt,
		IsSimulator:                true,
		Warning:                    "Simulador local de homologação — sem validade jurídica de assinatura eletrônica.",
	}, nil
}

func (s *Service) AcceptContract(ctx context.Context, token string, idempotencyKey string, input AcceptContractInput) (*domain.ContractInstance, bool, error) {
	started := time.Now()
	if s.acceptancePort == nil || token == "" || !input.Accepted {
		return nil, false, ErrInvalidContractInput
	}
	if !idempotencyKeyPattern.MatchString(idempotencyKey) {
		return nil, false, ErrInvalidContractInput
	}

	tokenHash := sha256Hex(token)
	session, instance, err := s.repository.FindAcceptanceSessionByTokenHash(ctx, tokenHash)
	if err != nil || session == nil || instance == nil {
		return nil, false, ErrContractAcceptanceSessionInvalid
	}

	now := s.now().UTC()
	if session.Status != domain.ContractAcceptanceSessionStatusActive || now.After(session.ExpiresAt) {
		return nil, false, ErrContractAcceptanceSessionInvalid
	}

	if input.DocumentSHA256 != instance.DocumentSHA256 || input.AcceptanceStatementVersion != instance.AcceptanceStatementVersion {
		return nil, false, ErrContractAcceptanceDocumentChanged
	}

	// Revalidate enrollment
	enrSnapshot, err := s.enrollmentResolver.ResolveContractEnrollmentSnapshot(ctx, instance.OrganizationID, instance.UnitID, instance.EnrollmentID, now)
	if err != nil {
		return nil, false, err
	}
	if enrSnapshot == nil || (enrSnapshot.EffectiveStatus != domain.EnrollmentEffectiveScheduled && enrSnapshot.EffectiveStatus != domain.EnrollmentEffectiveActive) {
		return nil, false, ErrContractEnrollmentUnavailable
	}

	fingerprint := commandFingerprint(struct {
		SessionID      domain.ID `json:"session_id"`
		DocumentSHA256 string    `json:"document_sha256"`
		Version        int       `json:"acceptance_statement_version"`
	}{session.ID, input.DocumentSHA256, input.AcceptanceStatementVersion})

	var result *domain.ContractInstance
	var replayed bool
	err = s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		prev, err := s.repository.FindCommandRequest(txCtx, instance.OrganizationID, "accept", idempotencyKey)
		if err != nil {
			return err
		}
		if prev != nil {
			if !fingerprintsEqual(prev.Fingerprint, fingerprint) {
				return ErrContractCommandIdempotencyReused
			}
			if prev.ResultID != nil {
				result, err = s.repository.FindInstanceByID(txCtx, instance.OrganizationID, instance.UnitID, *prev.ResultID)
				if err != nil {
					return err
				}
				replayed = true
				return nil
			}
		}

		verification, verifyErr := s.acceptancePort.Verify(txCtx, session, domain.AcceptanceSubmission{
			DocumentSHA256:             input.DocumentSHA256,
			AcceptanceStatementVersion: input.AcceptanceStatementVersion,
			Accepted:                   input.Accepted,
		})
		if verifyErr != nil {
			return verifyErr
		}
		if !verification.Verified {
			return ErrContractAcceptanceDocumentChanged
		}

		result, err = s.repository.AcceptInstance(
			txCtx,
			instance.OrganizationID,
			instance.UnitID,
			instance.ID,
			session.ID,
			verification.AcceptedAt,
			verification.NormalizedEvidence,
			domain.AdapterLocalSimulator,
			session.AdapterReference,
			instance.DocumentSHA256,
			instance.AcceptanceStatementVersion,
			instance.PersonID,
		)
		if err != nil {
			return err
		}

		event := &domain.ContractEvent{
			ID:                 domain.ID(uuid.NewString()),
			OrganizationID:     instance.OrganizationID,
			UnitID:             instance.UnitID,
			PersonID:           instance.PersonID,
			EnrollmentID:       instance.EnrollmentID,
			ContractInstanceID: instance.ID,
			ActorType:          domain.ContractActorAcceptanceSubject,
			Action:             "contract.accepted",
			BeforeData: map[string]any{
				"status": "pending_acceptance",
			},
			AfterData: map[string]any{
				"status":            "accepted",
				"accepted_at":       verification.AcceptedAt.Format(time.RFC3339),
				"adapter":           domain.AdapterLocalSimulator,
				"adapter_reference": session.AdapterReference,
			},
			CreatedAt: verification.AcceptedAt,
		}
		if err := s.repository.RecordEvent(txCtx, event); err != nil {
			return err
		}

		cmdReq := &domain.ContractCommandRequest{
			ID:             domain.ID(uuid.NewString()),
			OrganizationID: instance.OrganizationID,
			Command:        "accept",
			IdempotencyKey: idempotencyKey,
			Fingerprint:    fingerprint,
			ResultID:       &instance.ID,
			ResponseData:   map[string]any{"contract_id": instance.ID, "status": "accepted"},
			CreatedAt:      verification.AcceptedAt,
		}
		return s.repository.SaveCommandRequest(txCtx, cmdReq)
	})

	s.logCommand("contract_accept", instance.OrganizationID, instance.UnitID, "", instance.ID, replayed, time.Since(started), err)
	return result, replayed, err
}

// Privacy

func (s *Service) ListPrivacy(ctx context.Context, organizationID, personID domain.ID) ([]domain.ContractPrivacyExport, error) {
	return s.repository.ListPrivacyContracts(ctx, organizationID, personID)
}

func (s *Service) PurgeRetention(ctx context.Context, now time.Time, apply bool) (int64, error) {
	return s.repository.PurgeExpiredSimulatorRecords(ctx, now, apply)
}

// Helpers

func validID(id domain.ID) bool {
	_, err := uuid.Parse(string(id))
	return err == nil
}

func commandFingerprint(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func fingerprintsEqual(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func nonNilAuditData(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}

func resultID(t *domain.ContractTemplate) domain.ID {
	if t != nil {
		return t.ID
	}
	return ""
}

func instanceResultID(i *domain.ContractInstance) domain.ID {
	if i != nil {
		return i.ID
	}
	return ""
}

func (s *Service) logCommand(action string, orgID, unitID, actorID, targetID domain.ID, replayed bool, duration time.Duration, err error) {
	result := "success"
	if replayed {
		result = "replay"
	}
	if err != nil {
		result = "error"
	}
	slog.Info("contract_command",
		"action", action,
		"organization_id", orgID,
		"unit_id", unitID,
		"actor_user_id", actorID,
		"target_id", targetID,
		"result", result,
		"duration_ms", duration.Milliseconds(),
		"replayed", replayed,
		"rolled_back", err != nil,
	)
}
