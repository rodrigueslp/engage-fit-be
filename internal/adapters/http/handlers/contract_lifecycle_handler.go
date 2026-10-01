package handlers

import (
	"crypto/subtle"
	"errors"
	"net/http"

	"boxengage/backend/internal/adapters/http/apiresponse"
	"boxengage/backend/internal/app/contractlifecycle"
	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"github.com/gin-gonic/gin"
)

type ContractLifecycleHandler struct {
	service      *contractlifecycle.Service
	webhookToken string
}

func NewContractLifecycleHandler(service *contractlifecycle.Service, webhookToken string) ContractLifecycleHandler {
	return ContractLifecycleHandler{service: service, webhookToken: webhookToken}
}

func (h ContractLifecycleHandler) ReconcileSignature(c *gin.Context) {
	provided := c.GetHeader("X-Contract-Webhook-Token")
	if h.webhookToken == "" || len(provided) != len(h.webhookToken) || subtle.ConstantTimeCompare([]byte(provided), []byte(h.webhookToken)) != 1 {
		apiresponse.AbortError(c, http.StatusUnauthorized, "contract_webhook_unauthorized", "unauthorized webhook")
		return
	}
	var input contractlifecycle.ReconcileSignatureInput
	if c.ShouldBindJSON(&input) != nil {
		respondContractLifecycleError(c, contractlifecycle.ErrInvalidLifecycleInput)
		return
	}
	item, replayed, err := h.service.ReconcileSignature(c.Request.Context(), c.Param("provider"), input)
	if err != nil {
		respondContractLifecycleError(c, err)
		return
	}
	if replayed {
		c.Header("Idempotency-Replayed", "true")
	}
	c.JSON(http.StatusOK, item)
}

func (h ContractLifecycleHandler) CreateSignatureEnvelope(c *gin.Context) {
	org, unit, actor, ok := healthFormContext(c)
	if !ok || !sameRouteUnit(c, unit) {
		return
	}
	var input contractlifecycle.CreateSignatureEnvelopeInput
	if c.ShouldBindJSON(&input) != nil {
		respondContractLifecycleError(c, contractlifecycle.ErrInvalidLifecycleInput)
		return
	}
	item, err := h.service.CreateSignatureEnvelope(c.Request.Context(), org, unit, domain.ID(c.Param("contract_id")), actor, input)
	if err != nil {
		respondContractLifecycleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}
func (h ContractLifecycleHandler) ListSignatureEnvelopes(c *gin.Context) {
	org, unit, _, ok := healthFormContext(c)
	if !ok || !sameRouteUnit(c, unit) {
		return
	}
	items, err := h.service.ListSignatureEnvelopes(c.Request.Context(), org, unit, domain.ID(c.Param("contract_id")))
	if err != nil {
		respondContractLifecycleError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}
func (h ContractLifecycleHandler) CreateLegalAction(c *gin.Context) {
	org, unit, actor, ok := healthFormContext(c)
	if !ok || !sameRouteUnit(c, unit) {
		return
	}
	var input contractlifecycle.CreateLegalActionInput
	if c.ShouldBindJSON(&input) != nil {
		respondContractLifecycleError(c, contractlifecycle.ErrInvalidLifecycleInput)
		return
	}
	item, err := h.service.CreateLegalAction(c.Request.Context(), org, unit, domain.ID(c.Param("enrollment_id")), actor, input)
	if err != nil {
		respondContractLifecycleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}
func (h ContractLifecycleHandler) ListLegalActions(c *gin.Context) {
	org, unit, _, ok := healthFormContext(c)
	if !ok || !sameRouteUnit(c, unit) {
		return
	}
	items, err := h.service.ListLegalActions(c.Request.Context(), org, unit, domain.ID(c.Param("enrollment_id")))
	if err != nil {
		respondContractLifecycleError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func respondContractLifecycleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, contractlifecycle.ErrInvalidLifecycleInput):
		apiresponse.AbortError(c, http.StatusBadRequest, "invalid_contract_lifecycle_input", "invalid contract lifecycle input")
	case errors.Is(err, contractlifecycle.ErrSignatureUnavailable):
		apiresponse.AbortError(c, http.StatusUnprocessableEntity, "contract_signature_unavailable", "signature provider unavailable")
	case errors.Is(err, contractlifecycle.ErrContractNotAccepted):
		apiresponse.AbortError(c, http.StatusUnprocessableEntity, "accepted_contract_required", "accepted contract required")
	case errors.Is(err, repositories.ErrSignatureEnvelopeExists), errors.Is(err, repositories.ErrContractLegalConflict):
		apiresponse.AbortError(c, http.StatusConflict, "contract_lifecycle_conflict", "contract lifecycle conflict")
	case errors.Is(err, repositories.ErrContractInstanceNotFound):
		apiresponse.AbortError(c, http.StatusNotFound, "contract_instance_not_found", "contract not found")
	case errors.Is(err, repositories.ErrSignatureEnvelopeNotFound):
		apiresponse.AbortError(c, http.StatusNotFound, "signature_envelope_not_found", "signature envelope not found")
	default:
		apiresponse.AbortError(c, http.StatusInternalServerError, "contract_lifecycle_internal_error", "contract lifecycle operation failed")
	}
}
