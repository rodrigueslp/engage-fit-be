package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	contractsapp "boxengage/backend/internal/app/contracts"
	"boxengage/backend/internal/app/memberships"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"github.com/gin-gonic/gin"
)

type ContractHandler struct {
	service *contractsapp.Service
}

func NewContractHandler(service *contractsapp.Service) ContractHandler {
	return ContractHandler{service: service}
}

// Templates

func (h ContractHandler) ListTemplates(c *gin.Context) {
	unitID, _, ok := contractContext(c)
	if !ok {
		return
	}
	includeArchived := false
	if raw, exists := c.GetQuery("include_archived"); exists {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			respondContractError(c, contractsapp.ErrInvalidContractInput)
			return
		}
		includeArchived = parsed
	}

	templates, err := h.service.ListTemplates(c.Request.Context(), unitID, unitID, includeArchived)
	if err != nil {
		respondContractError(c, err)
		return
	}

	response := make([]dto.ContractTemplateResponse, 0, len(templates))
	for _, t := range templates {
		response = append(response, templateToDTO(t))
	}
	c.JSON(http.StatusOK, response)
}

func (h ContractHandler) GetTemplate(c *gin.Context) {
	unitID, _, ok := contractContext(c)
	if !ok {
		return
	}
	templateID := domain.ID(c.Param("template_id"))
	tmpl, err := h.service.GetTemplate(c.Request.Context(), unitID, unitID, templateID)
	if err != nil {
		respondContractError(c, err)
		return
	}
	c.JSON(http.StatusOK, templateToDTO(*tmpl))
}

func (h ContractHandler) CreateTemplate(c *gin.Context) {
	unitID, actorID, ok := contractContext(c)
	if !ok {
		return
	}
	var req dto.CreateContractTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondContractError(c, contractsapp.ErrInvalidContractInput)
		return
	}
	idempotencyKey := c.GetHeader("Idempotency-Key")

	tmpl, replayed, err := h.service.CreateTemplate(c.Request.Context(), unitID, unitID, actorID, idempotencyKey, contractsapp.CreateTemplateInput{
		Name:                req.Name,
		Title:               req.Title,
		BodyMarkdown:        req.BodyMarkdown,
		AcceptanceStatement: req.AcceptanceStatement,
	})
	if err != nil {
		respondContractError(c, err)
		return
	}
	if replayed {
		c.Header("Idempotency-Replayed", "true")
		c.JSON(http.StatusOK, templateToDTO(*tmpl))
		return
	}
	c.JSON(http.StatusCreated, templateToDTO(*tmpl))
}

func (h ContractHandler) UpdateTemplateDraft(c *gin.Context) {
	unitID, actorID, ok := contractContext(c)
	if !ok {
		return
	}
	templateID := domain.ID(c.Param("template_id"))
	var req dto.UpdateContractTemplateDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondContractError(c, contractsapp.ErrInvalidContractInput)
		return
	}

	tmpl, err := h.service.UpdateTemplateDraft(c.Request.Context(), unitID, unitID, templateID, actorID, contractsapp.UpdateTemplateDraftInput{
		Name:                req.Name,
		Title:               req.Title,
		BodyMarkdown:        req.BodyMarkdown,
		AcceptanceStatement: req.AcceptanceStatement,
		ExpectedRevision:    req.ExpectedRevision,
	})
	if err != nil {
		respondContractError(c, err)
		return
	}
	c.JSON(http.StatusOK, templateToDTO(*tmpl))
}

func (h ContractHandler) PublishTemplate(c *gin.Context) {
	unitID, actorID, ok := contractContext(c)
	if !ok {
		return
	}
	templateID := domain.ID(c.Param("template_id"))
	var req dto.PublishContractTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondContractError(c, contractsapp.ErrInvalidContractInput)
		return
	}
	idempotencyKey := c.GetHeader("Idempotency-Key")

	tmpl, replayed, err := h.service.PublishTemplate(c.Request.Context(), unitID, unitID, templateID, actorID, idempotencyKey, contractsapp.PublishTemplateInput{
		DraftRevision: req.DraftRevision,
	})
	if err != nil {
		respondContractError(c, err)
		return
	}
	if replayed {
		c.Header("Idempotency-Replayed", "true")
	}
	c.JSON(http.StatusOK, templateToDTO(*tmpl))
}

func (h ContractHandler) CreateNewVersion(c *gin.Context) {
	unitID, actorID, ok := contractContext(c)
	if !ok {
		return
	}
	templateID := domain.ID(c.Param("template_id"))
	var req dto.CreateContractTemplateVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondContractError(c, contractsapp.ErrInvalidContractInput)
		return
	}
	idempotencyKey := c.GetHeader("Idempotency-Key")

	tmpl, replayed, err := h.service.CreateNewVersion(c.Request.Context(), unitID, unitID, templateID, actorID, idempotencyKey, contractsapp.CreateNewVersionInput{
		Title:               req.Title,
		BodyMarkdown:        req.BodyMarkdown,
		AcceptanceStatement: req.AcceptanceStatement,
	})
	if err != nil {
		respondContractError(c, err)
		return
	}
	if replayed {
		c.Header("Idempotency-Replayed", "true")
		c.JSON(http.StatusOK, templateToDTO(*tmpl))
		return
	}
	c.JSON(http.StatusCreated, templateToDTO(*tmpl))
}

func (h ContractHandler) ArchiveTemplate(c *gin.Context) {
	unitID, actorID, ok := contractContext(c)
	if !ok {
		return
	}
	templateID := domain.ID(c.Param("template_id"))
	tmpl, err := h.service.ArchiveTemplate(c.Request.Context(), unitID, unitID, templateID, actorID)
	if err != nil {
		respondContractError(c, err)
		return
	}
	c.JSON(http.StatusOK, templateToDTO(*tmpl))
}

func (h ContractHandler) ReactivateTemplate(c *gin.Context) {
	unitID, actorID, ok := contractContext(c)
	if !ok {
		return
	}
	templateID := domain.ID(c.Param("template_id"))
	tmpl, err := h.service.ReactivateTemplate(c.Request.Context(), unitID, unitID, templateID, actorID)
	if err != nil {
		respondContractError(c, err)
		return
	}
	c.JSON(http.StatusOK, templateToDTO(*tmpl))
}

// Instances & Enrollment

func (h ContractHandler) ListForEnrollment(c *gin.Context) {
	unitID, _, ok := contractContext(c)
	if !ok {
		return
	}
	if _, ok := validateUnitRoute(c, unitID); !ok {
		return
	}
	enrollmentID := domain.ID(c.Param("enrollment_id"))

	instances, err := h.service.ListForEnrollment(c.Request.Context(), unitID, unitID, enrollmentID)
	if err != nil {
		respondContractError(c, err)
		return
	}

	response := make([]dto.ContractInstanceResponse, 0, len(instances))
	for _, inst := range instances {
		response = append(response, instanceToDTO(inst))
	}
	c.JSON(http.StatusOK, response)
}

func (h ContractHandler) PreviewContract(c *gin.Context) {
	unitID, _, ok := contractContext(c)
	if !ok {
		return
	}
	if _, ok := validateUnitRoute(c, unitID); !ok {
		return
	}
	enrollmentID := domain.ID(c.Param("enrollment_id"))
	var req dto.ContractPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondContractError(c, contractsapp.ErrInvalidContractInput)
		return
	}

	preview, err := h.service.PreviewEnrollmentContract(c.Request.Context(), unitID, unitID, enrollmentID, contractsapp.PreviewContractInput{
		TemplateID:        req.TemplateID,
		TemplateVersionID: req.TemplateVersionID,
	})
	if err != nil {
		respondContractError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ContractPreviewResponse{
		TemplateID:           string(preview.TemplateID),
		TemplateVersionID:    string(preview.TemplateVersionID),
		EnrollmentID:         string(preview.EnrollmentID),
		DocumentTitle:        preview.DocumentTitle,
		DocumentBodyMarkdown: preview.DocumentBodyMarkdown,
		AcceptanceStatement:  preview.AcceptanceStatement,
		CommercialSnapshot:   preview.CommercialSnapshot,
		PreviewSHA256:        preview.PreviewSHA256,
		Warnings:             preview.Warnings,
	})
}

func (h ContractHandler) CreateInstanceDraft(c *gin.Context) {
	unitID, actorID, ok := contractContext(c)
	if !ok {
		return
	}
	if _, ok := validateUnitRoute(c, unitID); !ok {
		return
	}
	enrollmentID := domain.ID(c.Param("enrollment_id"))
	var req dto.CreateContractInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondContractError(c, contractsapp.ErrInvalidContractInput)
		return
	}
	idempotencyKey := c.GetHeader("Idempotency-Key")

	inst, replayed, err := h.service.CreateInstanceDraft(c.Request.Context(), unitID, unitID, enrollmentID, actorID, idempotencyKey, contractsapp.CreateInstanceDraftInput{
		TemplateID:            req.TemplateID,
		TemplateVersionID:     req.TemplateVersionID,
		ExpectedPreviewSHA256: req.ExpectedPreviewSHA256,
	})
	if err != nil {
		respondContractError(c, err)
		return
	}
	if replayed {
		c.Header("Idempotency-Replayed", "true")
		c.JSON(http.StatusOK, instanceToDTO(*inst))
		return
	}
	c.JSON(http.StatusCreated, instanceToDTO(*inst))
}

func (h ContractHandler) GetInstance(c *gin.Context) {
	unitID, _, ok := contractContext(c)
	if !ok {
		return
	}
	if _, ok := validateUnitRoute(c, unitID); !ok {
		return
	}
	contractID := domain.ID(c.Param("contract_id"))

	inst, err := h.service.GetInstance(c.Request.Context(), unitID, unitID, contractID)
	if err != nil {
		respondContractError(c, err)
		return
	}
	c.JSON(http.StatusOK, instanceToDTO(*inst))
}

func (h ContractHandler) IssueInstance(c *gin.Context) {
	unitID, actorID, ok := contractContext(c)
	if !ok {
		return
	}
	if _, ok := validateUnitRoute(c, unitID); !ok {
		return
	}
	contractID := domain.ID(c.Param("contract_id"))
	var req dto.IssueContractInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondContractError(c, contractsapp.ErrInvalidContractInput)
		return
	}
	idempotencyKey := c.GetHeader("Idempotency-Key")

	inst, replayed, err := h.service.IssueInstance(c.Request.Context(), unitID, unitID, contractID, actorID, idempotencyKey, contractsapp.IssueInstanceInput{
		ExpectedRevision: req.ExpectedRevision,
	})
	if err != nil {
		respondContractError(c, err)
		return
	}
	if replayed {
		c.Header("Idempotency-Replayed", "true")
	}
	c.JSON(http.StatusOK, instanceToDTO(*inst))
}

func (h ContractHandler) DiscardInstance(c *gin.Context) {
	unitID, actorID, ok := contractContext(c)
	if !ok {
		return
	}
	if _, ok := validateUnitRoute(c, unitID); !ok {
		return
	}
	contractID := domain.ID(c.Param("contract_id"))
	var req dto.DiscardContractInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondContractError(c, contractsapp.ErrInvalidContractInput)
		return
	}
	idempotencyKey := c.GetHeader("Idempotency-Key")

	inst, replayed, err := h.service.DiscardInstance(c.Request.Context(), unitID, unitID, contractID, actorID, idempotencyKey, contractsapp.DiscardInstanceInput{
		ExpectedRevision: req.ExpectedRevision,
	})
	if err != nil {
		respondContractError(c, err)
		return
	}
	if replayed {
		c.Header("Idempotency-Replayed", "true")
	}
	c.JSON(http.StatusOK, instanceToDTO(*inst))
}

func (h ContractHandler) CreateAcceptanceSession(c *gin.Context) {
	unitID, actorID, ok := contractContext(c)
	if !ok {
		return
	}
	if _, ok := validateUnitRoute(c, unitID); !ok {
		return
	}
	contractID := domain.ID(c.Param("contract_id"))
	idempotencyKey := c.GetHeader("Idempotency-Key")

	launch, replayed, err := h.service.CreateAcceptanceSession(c.Request.Context(), unitID, unitID, contractID, actorID, idempotencyKey)
	if err != nil {
		respondContractError(c, err)
		return
	}
	if replayed {
		c.Header("Idempotency-Replayed", "true")
		c.JSON(http.StatusOK, dto.ContractAcceptanceSessionLaunchResponse{
			AdapterReference: launch.AdapterReference,
			LaunchToken:      launch.LaunchToken,
			LaunchPath:       launch.LaunchPath,
			ExpiresAt:        launch.ExpiresAt,
		})
		return
	}
	c.JSON(http.StatusCreated, dto.ContractAcceptanceSessionLaunchResponse{
		AdapterReference: launch.AdapterReference,
		LaunchToken:      launch.LaunchToken,
		LaunchPath:       launch.LaunchPath,
		ExpiresAt:        launch.ExpiresAt,
	})
}

// Public Simulator

func (h ContractHandler) GetPublicAcceptanceSession(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	token := c.Param("token")

	sessionData, err := h.service.GetPublicAcceptanceSession(c.Request.Context(), token)
	if err != nil {
		respondContractError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.PublicContractAcceptanceSessionResponse{
		Token:                      sessionData.Token,
		Status:                     sessionData.Status,
		ContractID:                 string(sessionData.ContractID),
		DocumentTitle:              sessionData.DocumentTitle,
		DocumentBodyMarkdown:       sessionData.DocumentBodyMarkdown,
		AcceptanceStatement:        sessionData.AcceptanceStatement,
		AcceptanceStatementVersion: sessionData.AcceptanceStatementVersion,
		DocumentSHA256:             sessionData.DocumentSHA256,
		CommercialSnapshot:         sessionData.CommercialSnapshot,
		PartySnapshot:              sessionData.PartySnapshot,
		ExpiresAt:                  sessionData.ExpiresAt,
		IsSimulator:                sessionData.IsSimulator,
		Warning:                    sessionData.Warning,
	})
}

func (h ContractHandler) AcceptContract(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	token := c.Param("token")
	var req dto.AcceptContractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondContractError(c, contractsapp.ErrInvalidContractInput)
		return
	}
	idempotencyKey := c.GetHeader("Idempotency-Key")

	inst, replayed, err := h.service.AcceptContract(c.Request.Context(), token, idempotencyKey, contractsapp.AcceptContractInput{
		DocumentSHA256:             req.DocumentSHA256,
		AcceptanceStatementVersion: req.AcceptanceStatementVersion,
		Accepted:                   req.Accepted,
	})
	if err != nil {
		respondContractError(c, err)
		return
	}
	if replayed {
		c.Header("Idempotency-Replayed", "true")
	}
	c.JSON(http.StatusOK, instanceToDTO(*inst))
}

// Helpers & Error Mapping

func contractContext(c *gin.Context) (domain.ID, domain.ID, bool) {
	unitID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return "", "", false
	}
	actorID, err := middleware.UserID(c)
	if err != nil {
		respondUnauthorized(c)
		return "", "", false
	}
	return unitID, actorID, true
}

func validateUnitRoute(c *gin.Context, authenticatedUnitID domain.ID) (domain.ID, bool) {
	paramUnitID := domain.ID(c.Param("unit_id"))
	if paramUnitID != authenticatedUnitID {
		respondPublicError(c, http.StatusNotFound, "contract_not_found", "recurso não encontrado")
		return "", false
	}
	return paramUnitID, true
}

func templateToDTO(t domain.ContractTemplate) dto.ContractTemplateResponse {
	resp := dto.ContractTemplateResponse{
		ID:             string(t.ID),
		OrganizationID: string(t.OrganizationID),
		HomeUnitID:     string(t.HomeUnitID),
		Code:           t.Code,
		Name:           t.Name,
		NameKey:        t.NameKey,
		ContractKind:   string(t.ContractKind),
		Status:         string(t.Status),
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
	}
	if t.CurrentVersion != nil {
		v := templateVersionToDTO(*t.CurrentVersion)
		resp.CurrentVersion = &v
	}
	if t.CurrentDraft != nil {
		v := templateVersionToDTO(*t.CurrentDraft)
		resp.CurrentDraft = &v
	}
	for _, ver := range t.Versions {
		resp.Versions = append(resp.Versions, templateVersionToDTO(ver))
	}
	return resp
}

func templateVersionToDTO(v domain.ContractTemplateVersion) dto.ContractTemplateVersionResponse {
	return dto.ContractTemplateVersionResponse{
		ID:                       string(v.ID),
		OrganizationID:           string(v.OrganizationID),
		TemplateID:               string(v.TemplateID),
		VersionNumber:            v.VersionNumber,
		Revision:                 v.Revision,
		Status:                   string(v.Status),
		Title:                    v.Title,
		BodyMarkdown:             v.BodyMarkdown,
		AcceptanceStatement:      v.AcceptanceStatement,
		PlaceholderSchemaVersion: v.PlaceholderSchemaVersion,
		RendererVersion:          v.RendererVersion,
		ContentSHA256:            v.ContentSHA256,
		PublishedAt:              v.PublishedAt,
		CreatedAt:                v.CreatedAt,
		UpdatedAt:                v.UpdatedAt,
	}
}

func instanceToDTO(inst domain.ContractInstance) dto.ContractInstanceResponse {
	resp := dto.ContractInstanceResponse{
		ID:                         string(inst.ID),
		OrganizationID:             string(inst.OrganizationID),
		UnitID:                     string(inst.UnitID),
		PersonID:                   string(inst.PersonID),
		EnrollmentID:               string(inst.EnrollmentID),
		TemplateVersionID:          string(inst.TemplateVersionID),
		ContractKind:               string(inst.ContractKind),
		Status:                     string(inst.Status),
		Revision:                   inst.Revision,
		SnapshotSchemaVersion:      inst.SnapshotSchemaVersion,
		CommercialSnapshot:         inst.CommercialSnapshot,
		CommercialSnapshotSHA256:   inst.CommercialSnapshotSHA256,
		PartySnapshot:              inst.PartySnapshot,
		DocumentTitle:              inst.DocumentTitle,
		DocumentBodyMarkdown:       inst.DocumentBodyMarkdown,
		DocumentSHA256:             inst.DocumentSHA256,
		AcceptanceStatement:        inst.AcceptanceStatement,
		AcceptanceStatementVersion: inst.AcceptanceStatementVersion,
		CreatedByUserID:            string(inst.CreatedByUserID),
		IssuedAt:                   inst.IssuedAt,
		AcceptedAt:                 inst.AcceptedAt,
		DiscardedAt:                inst.DiscardedAt,
		RetentionClass:             string(inst.RetentionClass),
		CreatedAt:                  inst.CreatedAt,
		UpdatedAt:                  inst.UpdatedAt,
	}
	if inst.IssuedByUserID != nil {
		v := string(*inst.IssuedByUserID)
		resp.IssuedByUserID = &v
	}
	if inst.DiscardedByUserID != nil {
		v := string(*inst.DiscardedByUserID)
		resp.DiscardedByUserID = &v
	}
	if inst.TemplateVersion != nil {
		v := templateVersionToDTO(*inst.TemplateVersion)
		resp.TemplateVersion = &v
	}
	if inst.ActiveSession != nil {
		resp.ActiveSession = &dto.ContractAcceptanceSessionResponse{
			ID:                 string(inst.ActiveSession.ID),
			OrganizationID:     string(inst.ActiveSession.OrganizationID),
			ContractInstanceID: string(inst.ActiveSession.ContractInstanceID),
			Adapter:            inst.ActiveSession.Adapter,
			AdapterReference:   inst.ActiveSession.AdapterReference,
			Status:             string(inst.ActiveSession.Status),
			ExpiresAt:          inst.ActiveSession.ExpiresAt,
			UsedAt:             inst.ActiveSession.UsedAt,
			CreatedAt:          inst.ActiveSession.CreatedAt,
		}
	}
	if inst.Acceptance != nil {
		resp.Acceptance = &dto.ContractAcceptanceResponse{
			ID:                         string(inst.Acceptance.ID),
			OrganizationID:             string(inst.Acceptance.OrganizationID),
			ContractInstanceID:         string(inst.Acceptance.ContractInstanceID),
			AcceptanceSessionID:        string(inst.Acceptance.AcceptanceSessionID),
			Adapter:                    inst.Acceptance.Adapter,
			AdapterReference:           inst.Acceptance.AdapterReference,
			EvidenceSchemaVersion:      inst.Acceptance.EvidenceSchemaVersion,
			PersonID:                   string(inst.Acceptance.PersonID),
			DocumentSHA256:             inst.Acceptance.DocumentSHA256,
			AcceptanceStatementVersion: inst.Acceptance.AcceptanceStatementVersion,
			Evidence:                   inst.Acceptance.Evidence,
			AcceptedAt:                 inst.Acceptance.AcceptedAt,
			CreatedAt:                  inst.Acceptance.CreatedAt,
		}
	}
	for _, e := range inst.Events {
		var actorID *string
		if e.ActorUserID != nil {
			v := string(*e.ActorUserID)
			actorID = &v
		}
		resp.Events = append(resp.Events, dto.ContractEventResponse{
			ID:                 string(e.ID),
			OrganizationID:     string(e.OrganizationID),
			UnitID:             string(e.UnitID),
			PersonID:           string(e.PersonID),
			EnrollmentID:       string(e.EnrollmentID),
			ContractInstanceID: string(e.ContractInstanceID),
			ActorType:          string(e.ActorType),
			ActorUserID:        actorID,
			Action:             e.Action,
			BeforeData:         e.BeforeData,
			AfterData:          e.AfterData,
			CreatedAt:          e.CreatedAt,
		})
	}
	return resp
}

func respondContractError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidContractTemplate),
		errors.Is(err, domain.ErrUnknownPlaceholder),
		errors.Is(err, domain.ErrUnclosedPlaceholder),
		errors.Is(err, domain.ErrDisallowedMarkdownTag):
		respondPublicError(c, http.StatusUnprocessableEntity, "contract_template_invalid", "conteúdo do modelo de contrato inválido")
	case errors.Is(err, contractsapp.ErrInvalidContractInput):
		respondPublicError(c, http.StatusBadRequest, "invalid_contract_input", "dados de contrato inválidos")
	case errors.Is(err, contractsapp.ErrContractTemplateNotFound):
		respondPublicError(c, http.StatusNotFound, "contract_template_not_found", "modelo de contrato não encontrado")
	case errors.Is(err, contractsapp.ErrContractInstanceNotFound):
		respondPublicError(c, http.StatusNotFound, "contract_instance_not_found", "contrato não encontrado")
	case errors.Is(err, contractsapp.ErrContractAcceptanceSessionInvalid):
		respondPublicError(c, http.StatusNotFound, "contract_acceptance_session_invalid", "sessão de aceite inválida ou expirada")
	case errors.Is(err, contractsapp.ErrContractTemplateNameConflict):
		respondPublicError(c, http.StatusConflict, "contract_template_name_conflict", "já existe um modelo com este nome")
	case errors.Is(err, contractsapp.ErrContractTemplateRevisionConflict):
		respondPublicError(c, http.StatusConflict, "contract_template_revision_conflict", "o modelo foi alterado em outra aba")
	case errors.Is(err, contractsapp.ErrContractTemplateDraftExists):
		respondPublicError(c, http.StatusConflict, "contract_template_draft_exists", "já existe um rascunho para este modelo")
	case errors.Is(err, contractsapp.ErrContractInstanceExists):
		respondPublicError(c, http.StatusConflict, "contract_instance_exists", "já existe um contrato ativo para esta matrícula")
	case errors.Is(err, contractsapp.ErrContractPreviewChanged):
		respondPublicError(c, http.StatusConflict, "contract_preview_changed", "a prévia do contrato foi alterada; revise os termos novamente")
	case errors.Is(err, contractsapp.ErrContractRevisionConflict):
		respondPublicError(c, http.StatusConflict, "contract_revision_conflict", "o contrato foi alterado em outra aba")
	case errors.Is(err, contractsapp.ErrContractAlreadyIssued):
		respondPublicError(c, http.StatusConflict, "contract_already_issued", "o contrato já foi emitido ou aceito")
	case errors.Is(err, contractsapp.ErrContractAcceptanceDocumentChanged):
		respondPublicError(c, http.StatusConflict, "contract_acceptance_document_changed", "o documento contratual foi alterado")
	case errors.Is(err, contractsapp.ErrContractCommandIdempotencyReused):
		respondPublicError(c, http.StatusConflict, "idempotency_key_reused", "chave de idempotência reutilizada")
	case errors.Is(err, contractsapp.ErrContractEnrollmentUnavailable),
		errors.Is(err, portrepo.ErrEnrollmentNotFound),
		errors.Is(err, memberships.ErrEnrollmentNotFound):
		respondPublicError(c, http.StatusUnprocessableEntity, "contract_enrollment_unavailable", "matrícula indisponível para contrato")
	case errors.Is(err, contractsapp.ErrContractPartyIneligible):
		respondPublicError(c, http.StatusUnprocessableEntity, "contract_party_ineligible", "pessoa inelegível para contrato")
	case errors.Is(err, contractsapp.ErrContractAcceptanceUnavailable):
		respondPublicError(c, http.StatusUnprocessableEntity, "contract_acceptance_unavailable", "simulador de aceite indisponível")
	default:
		respondError(c, err)
	}
}
