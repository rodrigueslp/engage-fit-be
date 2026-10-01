package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/scheduling"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type SchedulingHandler struct {
	service *scheduling.Service
}

func NewSchedulingHandler(service *scheduling.Service) SchedulingHandler {
	return SchedulingHandler{service: service}
}

func (h SchedulingHandler) List(c *gin.Context) {
	unitID, actorID, ok := schedulingContext(c)
	if !ok {
		return
	}
	includeArchived := c.Query("include_archived") == "true"
	query := c.Query("query")

	list, err := h.service.ListProfessionals(c.Request.Context(), actorID, unitID, includeArchived, query)
	if err != nil {
		respondSchedulingError(c, err)
		return
	}

	response := make([]dto.ProfessionalResponse, 0, len(list))
	for _, prof := range list {
		response = append(response, professionalResponse(prof))
	}
	c.JSON(http.StatusOK, response)
}

func (h SchedulingHandler) ListCandidates(c *gin.Context) {
	unitID, actorID, ok := schedulingContext(c)
	if !ok {
		return
	}
	candidates, err := h.service.ListCandidates(c.Request.Context(), actorID, unitID)
	if err != nil {
		respondSchedulingError(c, err)
		return
	}

	response := make([]dto.ProfessionalAccountCandidateResponse, 0, len(candidates))
	for _, cand := range candidates {
		response = append(response, dto.ProfessionalAccountCandidateResponse{
			UserID:           string(cand.UserID),
			DisplayName:      cand.DisplayName,
			Role:             cand.Role,
			Active:           cand.Active,
			AuthorizedInUnit: cand.AuthorizedInUnit,
			IsLinked:         cand.IsLinked,
		})
	}
	c.JSON(http.StatusOK, response)
}

func (h SchedulingHandler) Create(c *gin.Context) {
	unitID, actorID, ok := schedulingContext(c)
	if !ok {
		return
	}
	var req dto.CreateProfessionalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondSchedulingError(c, scheduling.ErrSchedulingProfessionalInvalid)
		return
	}

	var linkedUserID *domain.ID
	if req.LinkedUserID != nil && *req.LinkedUserID != "" {
		id := domain.ID(*req.LinkedUserID)
		linkedUserID = &id
	}

	input := scheduling.CreateInput{
		DisplayName:  req.DisplayName,
		LinkedUserID: linkedUserID,
	}

	item, isNew, err := h.service.Create(c.Request.Context(), actorID, unitID, c.GetHeader("Idempotency-Key"), input)
	if err != nil {
		respondSchedulingError(c, err)
		return
	}

	if !isNew {
		c.Header("Idempotency-Replayed", "true")
		c.JSON(http.StatusOK, professionalResponse(*item))
		return
	}

	c.JSON(http.StatusCreated, professionalResponse(*item))
}

type updatePayload struct {
	DisplayName      *string         `json:"display_name"`
	LinkedUserID     json.RawMessage `json:"linked_user_id"`
	ExpectedRevision int             `json:"expected_revision"`
}

func (h SchedulingHandler) Update(c *gin.Context) {
	unitID, actorID, ok := schedulingContext(c)
	if !ok {
		return
	}
	var raw updatePayload
	if err := c.ShouldBindJSON(&raw); err != nil {
		respondSchedulingError(c, scheduling.ErrSchedulingProfessionalInvalid)
		return
	}

	input := scheduling.UpdateInput{
		DisplayName:      raw.DisplayName,
		ExpectedRevision: raw.ExpectedRevision,
	}

	if len(raw.LinkedUserID) > 0 {
		if string(raw.LinkedUserID) == "null" {
			input.ClearLinkedUser = true
		} else {
			var uid string
			if err := json.Unmarshal(raw.LinkedUserID, &uid); err == nil && uid != "" {
				id := domain.ID(uid)
				input.LinkedUserID = &id
			} else {
				input.ClearLinkedUser = true
			}
		}
	}

	profID := domain.ID(c.Param("professional_id"))
	item, isNew, err := h.service.Update(c.Request.Context(), actorID, unitID, profID, c.GetHeader("Idempotency-Key"), input)
	if err != nil {
		respondSchedulingError(c, err)
		return
	}

	if !isNew {
		c.Header("Idempotency-Replayed", "true")
	}

	c.JSON(http.StatusOK, professionalResponse(*item))
}

func (h SchedulingHandler) Archive(c *gin.Context) {
	unitID, actorID, ok := schedulingContext(c)
	if !ok {
		return
	}
	var req dto.ArchiveProfessionalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondSchedulingError(c, scheduling.ErrSchedulingProfessionalInvalid)
		return
	}

	profID := domain.ID(c.Param("professional_id"))
	input := scheduling.ArchiveInput{
		ExpectedRevision: req.ExpectedRevision,
		Reason:           req.Reason,
	}

	item, isNew, err := h.service.Archive(c.Request.Context(), actorID, unitID, profID, c.GetHeader("Idempotency-Key"), input)
	if err != nil {
		respondSchedulingError(c, err)
		return
	}

	if !isNew {
		c.Header("Idempotency-Replayed", "true")
	}

	c.JSON(http.StatusOK, professionalResponse(*item))
}

func (h SchedulingHandler) Reactivate(c *gin.Context) {
	unitID, actorID, ok := schedulingContext(c)
	if !ok {
		return
	}
	var req dto.ReactivateProfessionalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondSchedulingError(c, scheduling.ErrSchedulingProfessionalInvalid)
		return
	}

	profID := domain.ID(c.Param("professional_id"))
	input := scheduling.ReactivateInput{
		ExpectedRevision: req.ExpectedRevision,
		Reason:           req.Reason,
	}

	item, isNew, err := h.service.Reactivate(c.Request.Context(), actorID, unitID, profID, c.GetHeader("Idempotency-Key"), input)
	if err != nil {
		respondSchedulingError(c, err)
		return
	}

	if !isNew {
		c.Header("Idempotency-Replayed", "true")
	}

	c.JSON(http.StatusOK, professionalResponse(*item))
}

func (h SchedulingHandler) PrivacyExport(c *gin.Context) {
	unitID, actorID, ok := schedulingContext(c)
	if !ok {
		return
	}
	profID := domain.ID(c.Param("professional_id"))
	export, err := h.service.ExportPrivacy(c.Request.Context(), actorID, unitID, profID)
	if err != nil {
		respondSchedulingError(c, err)
		return
	}

	events := make([]dto.SchedulingAuditEventResponse, 0, len(export.Events))
	for _, ev := range export.Events {
		events = append(events, dto.SchedulingAuditEventResponse{
			Action:        ev.Action,
			EntityType:    ev.EntityType,
			EntityID:      string(ev.EntityID),
			Revision:      ev.Revision,
			ChangedFields: ev.ChangedFields,
			BeforeData:    ev.BeforeData,
			AfterData:     ev.AfterData,
			Reason:        ev.Reason,
			CreatedAt:     ev.CreatedAt.Format(time.RFC3339),
		})
	}

	response := dto.SchedulingPrivacyExportResponse{
		Professional:           professionalResponse(export.Professional),
		Events:                 events,
		DraftSeriesAssignments: export.DraftSeriesAssignments,
		ExportedAt:             export.ExportedAt.Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}

func (h SchedulingHandler) Anonymize(c *gin.Context) {
	unitID, actorID, ok := schedulingContext(c)
	if !ok {
		return
	}
	var req dto.AnonymizeProfessionalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondSchedulingError(c, scheduling.ErrSchedulingProfessionalInvalid)
		return
	}

	profID := domain.ID(c.Param("professional_id"))
	input := scheduling.AnonymizeInput{
		ExpectedRevision: req.ExpectedRevision,
		Reason:           req.Reason,
	}

	item, isNew, err := h.service.Anonymize(c.Request.Context(), actorID, unitID, profID, c.GetHeader("Idempotency-Key"), input)
	if err != nil {
		respondSchedulingError(c, err)
		return
	}

	if !isNew {
		c.Header("Idempotency-Replayed", "true")
	}

	c.JSON(http.StatusOK, professionalResponse(*item))
}

func schedulingContext(c *gin.Context) (domain.ID, domain.ID, bool) {
	unitID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return "", "", false
	}
	if c.Param("unit_id") == "" || domain.ID(c.Param("unit_id")) != unitID {
		respondPublicError(c, http.StatusNotFound, "unit_not_found", "unidade não encontrada")
		return "", "", false
	}
	actorID, err := middleware.UserID(c)
	if err != nil {
		respondUnauthorized(c)
		return "", "", false
	}
	return unitID, actorID, true
}

func professionalResponse(item domain.Professional) dto.ProfessionalResponse {
	var linkedUserID *string
	if item.LinkedUserID != nil && *item.LinkedUserID != "" {
		s := string(*item.LinkedUserID)
		linkedUserID = &s
	}
	var anonymizedAt *string
	if item.AnonymizedAt != nil {
		s := item.AnonymizedAt.Format(time.RFC3339)
		anonymizedAt = &s
	}
	return dto.ProfessionalResponse{
		ID:           string(item.ID),
		DisplayName:  item.DisplayName,
		LinkedUserID: linkedUserID,
		Status:       string(item.Status),
		Revision:     item.Revision,
		AnonymizedAt: anonymizedAt,
		ActiveInUnit: item.ActiveInUnit,
		CreatedAt:    item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    item.UpdatedAt.Format(time.RFC3339),
	}
}

func respondSchedulingError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, scheduling.ErrSchedulingUnitNotFound):
		respondPublicError(c, http.StatusNotFound, "unit_not_found", "unidade não encontrada")
	case errors.Is(err, scheduling.ErrSchedulingProfessionalNotFound):
		respondPublicError(c, http.StatusNotFound, "scheduling_professional_not_found", "profissional não encontrado")
	case errors.Is(err, scheduling.ErrSchedulingRevisionConflict):
		respondPublicError(c, http.StatusConflict, "scheduling_revision_conflict", "o profissional foi alterado por outro usuário")
	case errors.Is(err, scheduling.ErrSchedulingAccountConflict):
		respondPublicError(c, http.StatusConflict, "scheduling_professional_account_conflict", "esta conta já está vinculada a outro profissional")
	case errors.Is(err, scheduling.ErrSchedulingProfessionalArchived):
		respondPublicError(c, http.StatusConflict, "scheduling_professional_archived", "o profissional está arquivado")
	case errors.Is(err, scheduling.ErrSchedulingProfessionalAnonymized):
		respondPublicError(c, http.StatusConflict, "scheduling_professional_anonymized", "o profissional foi anonimizado e não pode ser alterado")
	case errors.Is(err, scheduling.ErrSchedulingIdempotencyKeyReused):
		respondPublicError(c, http.StatusConflict, "idempotency_key_reused", "chave de idempotência reutilizada com dados diferentes")
	case errors.Is(err, scheduling.ErrSchedulingProfessionalInvalid):
		respondPublicError(c, http.StatusUnprocessableEntity, "scheduling_professional_invalid", "dados do profissional inválidos")
	case errors.Is(err, scheduling.ErrSchedulingAccountUnavailable):
		respondPublicError(c, http.StatusUnprocessableEntity, "scheduling_professional_account_unavailable", "conta indisponível ou fora da unidade")
	case errors.Is(err, scheduling.ErrSchedulingAnonymizationBlocked):
		respondPublicError(c, http.StatusUnprocessableEntity, "scheduling_professional_anonymization_blocked", "o profissional precisa estar arquivado e sem conta vinculada para ser anonimizado")
	case errors.Is(err, scheduling.ErrScheduleSeriesNotFound):
		respondPublicError(c, http.StatusNotFound, "schedule_series_not_found", "série não encontrada")
	case errors.Is(err, scheduling.ErrScheduleSessionNotFound):
		respondPublicError(c, http.StatusNotFound, "schedule_session_not_found", "sessão não encontrada")
	case errors.Is(err, scheduling.ErrScheduleSeriesArchived):
		respondPublicError(c, http.StatusConflict, "schedule_series_archived", "a série não aceita esta alteração")
	case errors.Is(err, scheduling.ErrScheduleSeriesAlreadyPublished):
		respondPublicError(c, http.StatusConflict, "schedule_series_already_published", "transição inválida para a série")
	case errors.Is(err, scheduling.ErrScheduleConflict):
		respondPublicError(c, http.StatusConflict, "scheduling_conflict", "há conflito de profissional ou espaço")
	case errors.Is(err, scheduling.ErrScheduleSessionAlreadyStarted):
		respondPublicError(c, http.StatusConflict, "schedule_session_already_started", "a sessão já começou")
	case errors.Is(err, scheduling.ErrScheduleSessionCancelled):
		respondPublicError(c, http.StatusConflict, "schedule_session_cancelled", "a sessão está cancelada")
	case errors.Is(err, scheduling.ErrScheduleTargetUnavailable):
		respondPublicError(c, http.StatusUnprocessableEntity, "scheduling_target_unavailable", "modalidade ou serviço indisponível")
	case errors.Is(err, scheduling.ErrScheduleSpaceUnavailable):
		respondPublicError(c, http.StatusUnprocessableEntity, "scheduling_space_unavailable", "espaço indisponível")
	case errors.Is(err, scheduling.ErrScheduleProfessionalUnavailable):
		respondPublicError(c, http.StatusUnprocessableEntity, "scheduling_professional_unavailable", "profissional indisponível")
	case errors.Is(err, scheduling.ErrScheduleInvalidLocalTime):
		respondPublicError(c, http.StatusUnprocessableEntity, "invalid_local_time", "horário local inválido ou ambíguo")
	case errors.Is(err, scheduling.ErrSchedulePreviewRangeInvalid):
		respondPublicError(c, http.StatusUnprocessableEntity, "scheduling_preview_range_invalid", "intervalo de consulta inválido")
	case errors.Is(err, scheduling.ErrSchedulePreviewTooLarge):
		respondPublicError(c, http.StatusUnprocessableEntity, "scheduling_preview_too_large", "prévia excede o limite")
	case errors.Is(err, scheduling.ErrScheduleSeriesInvalid):
		respondPublicError(c, http.StatusUnprocessableEntity, "scheduling_series_invalid", "dados da agenda inválidos")
	case errors.Is(err, scheduling.ErrScheduleMaterializerUnavailable):
		respondPublicError(c, http.StatusServiceUnavailable, "scheduling_materializer_unavailable", "publicação temporariamente indisponível")
	default:
		respondPublicError(c, http.StatusInternalServerError, "internal_error", "erro interno ao processar profissional")
	}
}
