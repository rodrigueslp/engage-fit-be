package handlers

import (
	"errors"
	"net/http"
	"time"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/memberships"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type EnrollmentHandler struct {
	service *memberships.EnrollmentService
}

func NewEnrollmentHandler(service *memberships.EnrollmentService) EnrollmentHandler {
	return EnrollmentHandler{service: service}
}

func (h EnrollmentHandler) ListForStudent(c *gin.Context) {
	unitID, _, ok := enrollmentContext(c)
	if !ok {
		return
	}
	result, err := h.service.ListForStudent(c.Request.Context(), unitID, domain.ID(c.Param("student_id")))
	if err != nil {
		respondEnrollmentError(c, err)
		return
	}
	response := dto.EnrollmentHistoryResponse{History: make([]dto.EnrollmentResponse, 0, len(result.History))}
	if result.Current != nil {
		converted := enrollmentResponse(*result.Current)
		response.Current = &converted
	}
	for _, item := range result.History {
		response.History = append(response.History, enrollmentResponse(item))
	}
	c.JSON(http.StatusOK, response)
}

func (h EnrollmentHandler) Create(c *gin.Context) {
	unitID, actorID, ok := enrollmentContext(c)
	if !ok {
		return
	}
	var request dto.CreateEnrollmentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondEnrollmentError(c, memberships.ErrInvalidEnrollmentRequest)
		return
	}
	item, replayed, err := h.service.Create(c.Request.Context(), unitID, domain.ID(c.Param("student_id")), actorID, c.GetHeader("Idempotency-Key"), request)
	if err != nil {
		respondEnrollmentError(c, err)
		return
	}
	if replayed {
		c.Header("Idempotency-Replayed", "true")
		c.JSON(http.StatusOK, enrollmentResponse(*item))
		return
	}
	c.JSON(http.StatusCreated, enrollmentResponse(*item))
}

func (h EnrollmentHandler) Get(c *gin.Context) {
	unitID, _, ok := enrollmentContext(c)
	if !ok {
		return
	}
	item, err := h.service.Get(c.Request.Context(), unitID, domain.ID(c.Param("enrollment_id")))
	if err != nil {
		respondEnrollmentError(c, err)
		return
	}
	c.JSON(http.StatusOK, enrollmentResponse(*item))
}

func (h EnrollmentHandler) End(c *gin.Context) {
	unitID, actorID, ok := enrollmentContext(c)
	if !ok {
		return
	}
	var request dto.EndEnrollmentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondEnrollmentError(c, memberships.ErrInvalidEnrollmentRequest)
		return
	}
	item, replayed, err := h.service.End(c.Request.Context(), unitID, domain.ID(c.Param("enrollment_id")), actorID, c.GetHeader("Idempotency-Key"), request)
	if err != nil {
		respondEnrollmentError(c, err)
		return
	}
	if replayed {
		c.Header("Idempotency-Replayed", "true")
	}
	c.JSON(http.StatusOK, enrollmentResponse(*item))
}

func enrollmentContext(c *gin.Context) (domain.ID, domain.ID, bool) {
	unitID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return "", "", false
	}
	if c.Param("unit_id") == "" || domain.ID(c.Param("unit_id")) != unitID {
		respondPublicError(c, http.StatusNotFound, "enrollment_not_found", "matrícula não encontrada")
		return "", "", false
	}
	actorID, err := middleware.UserID(c)
	if err != nil {
		respondUnauthorized(c)
		return "", "", false
	}
	return unitID, actorID, true
}

func enrollmentResponse(item domain.Enrollment) dto.EnrollmentResponse {
	addons := make([]dto.EnrollmentAddonResponse, 0, len(item.Addons))
	for _, a := range item.Addons {
		addons = append(addons, dto.EnrollmentAddonResponse{
			ID:                 string(a.ID),
			PlanVersionAddonID: string(a.PlanVersionAddonID),
			Name:               a.Name,
			PriceDeltaCents:    a.PriceDeltaCents,
			Summary:            a.Summary,
			CreatedAt:          a.CreatedAt.Format(time.RFC3339),
		})
	}
	terms := dto.EnrollmentTermsResponse{
		PlanID:           string(item.Terms.PlanID),
		PlanVersionID:    string(item.Terms.PlanVersionID),
		VersionNumber:    item.Terms.VersionNumber,
		Name:             item.Terms.Name,
		Description:      item.Terms.Description,
		PriceCents:       item.Terms.PriceCents,
		BasePriceCents:   item.Terms.BasePriceCents,
		AddonsPriceCents: item.Terms.AddonsPriceCents,
		TotalPriceCents:  item.Terms.TotalPriceCents,
		Currency:         item.Terms.Currency,
		BillingCycle:     string(item.Terms.BillingCycle),
		Summary:          item.Terms.Summary,
		BaseSummary:      item.Terms.BaseSummary,
	}
	response := dto.EnrollmentResponse{
		ID:               string(item.ID),
		PersonID:         string(item.PersonID),
		LifecycleStatus:  string(item.LifecycleStatus),
		EffectiveStatus:  string(item.EffectiveStatus),
		StartsOn:         item.StartsOn.Format("2006-01-02"),
		EndedReason:      item.EndedReason,
		Revision:         item.Revision,
		BasePriceCents:   item.BasePriceCents,
		AddonsPriceCents: item.AddonsPriceCents,
		TotalPriceCents:  item.TotalPriceCents,
		Addons:           addons,
		Terms:            terms,
		CreatedAt:        item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        item.UpdatedAt.Format(time.RFC3339),
	}
	if item.ValidUntil != nil {
		value := item.ValidUntil.Format("2006-01-02")
		response.ValidUntil = &value
	}
	if item.EndedOn != nil {
		value := item.EndedOn.Format("2006-01-02")
		response.EndedOn = &value
	}
	if item.EndedAt != nil {
		value := item.EndedAt.Format(time.RFC3339)
		response.EndedAt = &value
	}
	return response
}

func respondEnrollmentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, memberships.ErrInvalidEnrollmentRequest):
		respondPublicError(c, http.StatusBadRequest, "invalid_request", "requisição inválida")
	case errors.Is(err, memberships.ErrEnrollmentSubjectNotFound):
		respondPublicError(c, http.StatusNotFound, "enrollment_subject_not_found", "aluno não encontrado para matrícula")
	case errors.Is(err, memberships.ErrEnrollmentNotFound):
		respondPublicError(c, http.StatusNotFound, "enrollment_not_found", "matrícula não encontrada")
	case errors.Is(err, memberships.ErrActiveEnrollmentExists):
		respondPublicError(c, http.StatusConflict, "active_enrollment_exists", "já existe uma matrícula aberta")
	case errors.Is(err, memberships.ErrPlanVersionChanged):
		respondPublicError(c, http.StatusConflict, "plan_version_changed", "a versão do plano mudou; revise os termos novamente")
	case errors.Is(err, memberships.ErrEnrollmentRevisionConflict):
		respondPublicError(c, http.StatusConflict, "enrollment_revision_conflict", "a matrícula foi alterada em outra aba")
	case errors.Is(err, memberships.ErrEnrollmentIdempotencyReused):
		respondPublicError(c, http.StatusConflict, "idempotency_key_reused", "chave de idempotência reutilizada")
	case errors.Is(err, memberships.ErrEnrollmentAlreadyEnded):
		respondPublicError(c, http.StatusConflict, "enrollment_already_ended", "a matrícula já foi encerrada")
	case errors.Is(err, memberships.ErrEnrollmentAddonNotFound):
		respondPublicError(c, http.StatusUnprocessableEntity, "membership_plan_addon_not_found", "adicional não encontrado no plano selecionado")
	case errors.Is(err, memberships.ErrEnrollmentAddonDuplicate):
		respondPublicError(c, http.StatusUnprocessableEntity, "membership_plan_addon_duplicate", "adicional duplicado na seleção")
	case errors.Is(err, memberships.ErrEnrollmentSubjectIneligible):
		respondPublicError(c, http.StatusUnprocessableEntity, "enrollment_subject_ineligible", "a pessoa não está elegível para matrícula")
	case errors.Is(err, memberships.ErrPlanUnavailable):
		respondPublicError(c, http.StatusUnprocessableEntity, "plan_unavailable", "plano indisponível nesta unidade")
	case errors.Is(err, memberships.ErrPlanTargetsInactive):
		respondPublicError(c, http.StatusUnprocessableEntity, "plan_targets_inactive", "modalidade ou serviço do plano está indisponível")
	case errors.Is(err, memberships.ErrInvalidStartDate):
		respondPublicError(c, http.StatusUnprocessableEntity, "invalid_start_date", "data de início fora do período permitido")
	case errors.Is(err, memberships.ErrInvalidEnd):
		respondPublicError(c, http.StatusUnprocessableEntity, "invalid_end", "data ou motivo de encerramento inválido")
	default:
		respondError(c, err)
	}
}
