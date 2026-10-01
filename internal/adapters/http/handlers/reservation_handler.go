package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"boxengage/backend/internal/app/reservations"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type ReservationHandler struct{ service *reservations.Service }

func NewReservationHandler(service *reservations.Service) ReservationHandler {
	return ReservationHandler{service: service}
}

func (h ReservationHandler) Book(c *gin.Context) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	var input reservations.BookInput
	if c.ShouldBindJSON(&input) != nil {
		respondReservationError(c, reservations.ErrInvalid)
		return
	}
	value, fresh, err := h.service.Book(c.Request.Context(), actor, unit, c.GetHeader("Idempotency-Key"), input)
	if err != nil {
		respondReservationError(c, err)
		return
	}
	if !fresh {
		c.Header("Idempotency-Replayed", "true")
		c.JSON(http.StatusOK, value)
		return
	}
	status := http.StatusCreated
	if value.Status == domain.ReservationWaitlisted {
		status = http.StatusAccepted
	}
	c.JSON(status, value)
}

func (h ReservationHandler) Cancel(c *gin.Context) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	var input reservations.CancelInput
	if c.ShouldBindJSON(&input) != nil {
		respondReservationError(c, reservations.ErrInvalid)
		return
	}
	value, promoted, fresh, err := h.service.Cancel(c.Request.Context(), actor, unit, domain.ID(c.Param("reservation_id")), c.GetHeader("Idempotency-Key"), input)
	if err != nil {
		respondReservationError(c, err)
		return
	}
	if !fresh {
		c.Header("Idempotency-Replayed", "true")
	}
	c.JSON(http.StatusOK, gin.H{"reservation": value, "promoted": promoted})
}

func (h ReservationHandler) NoShow(c *gin.Context) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	var input struct {
		ExpectedRevision int `json:"expected_revision"`
	}
	if c.ShouldBindJSON(&input) != nil {
		respondReservationError(c, reservations.ErrInvalid)
		return
	}
	value, fresh, err := h.service.MarkNoShow(c.Request.Context(), actor, unit, domain.ID(c.Param("reservation_id")), c.GetHeader("Idempotency-Key"), input.ExpectedRevision)
	if err != nil {
		respondReservationError(c, err)
		return
	}
	if !fresh {
		c.Header("Idempotency-Replayed", "true")
	}
	c.JSON(http.StatusOK, value)
}

func (h ReservationHandler) ListSession(c *gin.Context) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	items, summary, err := h.service.ListSession(c.Request.Context(), actor, unit, domain.ID(c.Param("session_id")))
	if err != nil {
		respondReservationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "summary": summary})
}

func (h ReservationHandler) SessionSummaries(c *gin.Context) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	parts := strings.Split(c.Query("session_ids"), ",")
	ids := make([]domain.ID, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			ids = append(ids, domain.ID(value))
		}
	}
	values, err := h.service.SessionSummaries(c.Request.Context(), actor, unit, ids)
	if err != nil {
		respondReservationError(c, err)
		return
	}
	c.JSON(http.StatusOK, values)
}

func (h ReservationHandler) ListPerson(c *gin.Context) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	from, err := time.Parse(time.RFC3339, c.Query("from"))
	if err != nil {
		respondReservationError(c, reservations.ErrInvalid)
		return
	}
	until, err := time.Parse(time.RFC3339, c.Query("to"))
	if err != nil {
		respondReservationError(c, reservations.ErrInvalid)
		return
	}
	items, err := h.service.ListPerson(c.Request.Context(), actor, unit, domain.ID(c.Param("id")), from, until)
	if err != nil {
		respondReservationError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func respondReservationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, reservations.ErrNotFound):
		respondPublicError(c, http.StatusNotFound, "reservation_not_found", "reserva ou sessão não encontrada")
	case errors.Is(err, reservations.ErrAlreadyExists):
		respondPublicError(c, http.StatusConflict, "reservation_already_exists", "a pessoa já está reservada ou na fila desta sessão")
	case errors.Is(err, reservations.ErrNotEligible):
		respondPublicError(c, http.StatusUnprocessableEntity, "reservation_not_eligible", "a pessoa não possui matrícula e direito válidos para esta atividade")
	case errors.Is(err, reservations.ErrContractPending):
		respondPublicError(c, http.StatusUnprocessableEntity, "reservation_contract_pending", "aceite do contrato pendente para esta matrícula")
	case errors.Is(err, reservations.ErrHealthFormPending):
		respondPublicError(c, http.StatusUnprocessableEntity, "reservation_health_form_pending", "preenchimento do PAR-Q pendente para esta matrícula")
	case errors.Is(err, reservations.ErrPaymentRequired):
		respondPublicError(c, http.StatusPaymentRequired, "reservation_payment_required", "o pagamento inicial da matrícula ainda não foi confirmado")
	case errors.Is(err, reservations.ErrPaymentOverdue):
		respondPublicError(c, http.StatusPaymentRequired, "reservation_payment_overdue", "a matrícula possui pagamento vencido em aberto")
	case errors.Is(err, reservations.ErrLimitReached):
		respondPublicError(c, http.StatusConflict, "reservation_limit_reached", "o limite do direito para este período foi atingido")
	case errors.Is(err, reservations.ErrRevisionConflict):
		respondPublicError(c, http.StatusConflict, "reservation_revision_conflict", "a reserva foi alterada por outro usuário")
	case errors.Is(err, reservations.ErrAlreadyFinal):
		respondPublicError(c, http.StatusConflict, "reservation_transition_invalid", "a reserva não aceita esta alteração")
	case errors.Is(err, reservations.ErrIdempotencyReused):
		respondPublicError(c, http.StatusConflict, "idempotency_key_reused", "chave de idempotência reutilizada com dados diferentes")
	case errors.Is(err, reservations.ErrMakeupInvalid):
		respondPublicError(c, http.StatusUnprocessableEntity, "reservation_makeup_invalid", "a falta não concede uma reposição válida")
	case errors.Is(err, reservations.ErrInvalid):
		respondPublicError(c, http.StatusUnprocessableEntity, "reservation_invalid", "dados da reserva inválidos")
	default:
		respondPublicError(c, http.StatusInternalServerError, "internal_error", "erro interno ao processar reserva")
	}
}
