package handlers

import (
	"errors"
	"net/http"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/app/memberbilling"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type MemberBillingHandler struct{ service *memberbilling.Service }

func NewMemberBillingHandler(service *memberbilling.Service) MemberBillingHandler {
	return MemberBillingHandler{service: service}
}
func (h MemberBillingHandler) Snapshot(c *gin.Context) {
	unit, _, ok := enrollmentContext(c)
	if !ok {
		return
	}
	result, err := h.service.Snapshot(c, unit, domain.ID(c.Param("enrollment_id")))
	if err != nil {
		respondMemberBillingError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
func (h MemberBillingHandler) Start(c *gin.Context) {
	unit, actor, ok := enrollmentContext(c)
	if !ok {
		return
	}
	var input dto.StartMemberBillingRequest
	if c.ShouldBindJSON(&input) != nil {
		respondMemberBillingError(c, memberbilling.ErrInvalidRequest)
		return
	}
	result, replayed, err := h.service.Start(c, unit, domain.ID(c.Param("enrollment_id")), actor, c.GetHeader("Idempotency-Key"), input)
	if err != nil {
		respondMemberBillingError(c, err)
		return
	}
	if replayed {
		c.Header("Idempotency-Replayed", "true")
		c.JSON(http.StatusOK, result)
		return
	}
	c.JSON(http.StatusCreated, result)
}
func (h MemberBillingHandler) Materialize(c *gin.Context) {
	unit, actor, ok := enrollmentContext(c)
	if !ok {
		return
	}
	result, replayed, err := h.service.MaterializeNext(c, unit, domain.ID(c.Param("contract_id")), actor, c.GetHeader("Idempotency-Key"))
	respondMemberBillingResult(c, result, replayed, err)
}
func (h MemberBillingHandler) Manual(c *gin.Context) {
	unit, actor, ok := enrollmentContext(c)
	if !ok {
		return
	}
	var input dto.ManualMemberPaymentRequest
	if c.ShouldBindJSON(&input) != nil {
		respondMemberBillingError(c, memberbilling.ErrInvalidRequest)
		return
	}
	result, replayed, err := h.service.RecordManual(c, unit, domain.ID(c.Param("charge_id")), actor, c.GetHeader("Idempotency-Key"), input)
	respondMemberBillingResult(c, result, replayed, err)
}
func (h MemberBillingHandler) Reconcile(c *gin.Context) {
	unit, actor, ok := enrollmentContext(c)
	if !ok {
		return
	}
	var input dto.ReconcileMemberPaymentRequest
	if c.ShouldBindJSON(&input) != nil {
		respondMemberBillingError(c, memberbilling.ErrInvalidRequest)
		return
	}
	result, replayed, err := h.service.Reconcile(c, unit, domain.ID(c.Param("charge_id")), actor, c.GetHeader("Idempotency-Key"), input)
	respondMemberBillingResult(c, result, replayed, err)
}
func (h MemberBillingHandler) Refund(c *gin.Context) {
	unit, actor, ok := enrollmentContext(c)
	if !ok {
		return
	}
	var input dto.RefundMemberPaymentRequest
	if c.ShouldBindJSON(&input) != nil {
		respondMemberBillingError(c, memberbilling.ErrInvalidRequest)
		return
	}
	result, replayed, err := h.service.Refund(c, unit, domain.ID(c.Param("charge_id")), actor, c.GetHeader("Idempotency-Key"), input)
	respondMemberBillingResult(c, result, replayed, err)
}
func respondMemberBillingResult(c *gin.Context, result any, replayed bool, err error) {
	if err != nil {
		respondMemberBillingError(c, err)
		return
	}
	if replayed {
		c.Header("Idempotency-Replayed", "true")
	}
	c.JSON(http.StatusOK, result)
}
func respondMemberBillingError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, memberbilling.ErrInvalidRequest):
		respondPublicError(c, http.StatusUnprocessableEntity, "invalid_member_billing_request", "dados financeiros inválidos")
	case errors.Is(err, memberbilling.ErrNotFound):
		respondPublicError(c, http.StatusNotFound, "member_billing_not_found", "cobrança não encontrada")
	case errors.Is(err, memberbilling.ErrConflict):
		respondPublicError(c, http.StatusConflict, "member_billing_conflict", "a operação financeira conflita com o estado atual")
	case errors.Is(err, memberbilling.ErrIdempotencyReused):
		respondPublicError(c, http.StatusConflict, "idempotency_key_reused", "chave de idempotência reutilizada")
	case errors.Is(err, memberbilling.ErrInvalidTransition):
		respondPublicError(c, http.StatusConflict, "invalid_member_billing_transition", "esta operação financeira não é permitida no estado atual")
	default:
		respondError(c, err)
	}
}
