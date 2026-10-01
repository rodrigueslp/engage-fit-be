package handlers

import (
	"errors"
	"net/http"

	"boxengage/backend/internal/adapters/http/apiresponse"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/healthforms"
	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"github.com/gin-gonic/gin"
)

type HealthFormHandler struct{ service *healthforms.Service }

func NewHealthFormHandler(service *healthforms.Service) HealthFormHandler {
	return HealthFormHandler{service: service}
}

func (h HealthFormHandler) List(c *gin.Context) {
	organizationID, unitID, _, ok := healthFormContext(c)
	if !ok {
		return
	}
	if !sameRouteUnit(c, unitID) {
		return
	}
	items, err := h.service.ListForEnrollment(c.Request.Context(), organizationID, unitID, domain.ID(c.Param("enrollment_id")))
	if err != nil {
		respondHealthFormError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h HealthFormHandler) Submit(c *gin.Context) {
	organizationID, unitID, actorID, ok := healthFormContext(c)
	if !ok {
		return
	}
	if !sameRouteUnit(c, unitID) {
		return
	}
	var input healthforms.SubmitInput
	if err := c.ShouldBindJSON(&input); err != nil {
		respondHealthFormError(c, healthforms.ErrInvalidHealthFormInput)
		return
	}
	item, err := h.service.Submit(c.Request.Context(), organizationID, unitID, domain.ID(c.Param("enrollment_id")), actorID, input)
	if err != nil {
		respondHealthFormError(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h HealthFormHandler) CreateSession(c *gin.Context) {
	organizationID, unitID, actorID, ok := healthFormContext(c)
	if !ok {
		return
	}
	if !sameRouteUnit(c, unitID) {
		return
	}
	session, err := h.service.CreatePublicSession(c.Request.Context(), organizationID, unitID, domain.ID(c.Param("enrollment_id")), actorID)
	if err != nil {
		respondHealthFormError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, session)
}

func (h HealthFormHandler) GetPublicSession(c *gin.Context) {
	session, err := h.service.GetPublicSession(c.Request.Context(), c.Param("token"))
	if err != nil {
		respondHealthFormError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, session)
}

func (h HealthFormHandler) SubmitPublic(c *gin.Context) {
	var input healthforms.SubmitInput
	if err := c.ShouldBindJSON(&input); err != nil {
		respondHealthFormError(c, healthforms.ErrInvalidHealthFormInput)
		return
	}
	item, err := h.service.SubmitPublic(c.Request.Context(), c.Param("token"), input)
	if err != nil {
		respondHealthFormError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, item)
}

func (h HealthFormHandler) Get(c *gin.Context) {
	organizationID, unitID, actorID, ok := healthFormContext(c)
	if !ok {
		return
	}
	if !sameRouteUnit(c, unitID) {
		return
	}
	item, err := h.service.Get(c.Request.Context(), organizationID, unitID, domain.ID(c.Param("form_id")), actorID)
	if err != nil {
		respondHealthFormError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, item)
}

func (h HealthFormHandler) Revoke(c *gin.Context) {
	organizationID, unitID, actorID, ok := healthFormContext(c)
	if !ok {
		return
	}
	if !sameRouteUnit(c, unitID) {
		return
	}
	item, err := h.service.Revoke(c.Request.Context(), organizationID, unitID, domain.ID(c.Param("form_id")), actorID)
	if err != nil {
		respondHealthFormError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func healthFormContext(c *gin.Context) (domain.ID, domain.ID, domain.ID, bool) {
	unitID, err := middleware.BoxID(c)
	if err != nil {
		apiresponse.AbortError(c, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return "", "", "", false
	}
	actorID, err := middleware.UserID(c)
	if err != nil {
		apiresponse.AbortError(c, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return "", "", "", false
	}
	return unitID, unitID, actorID, true
}

func sameRouteUnit(c *gin.Context, unitID domain.ID) bool {
	if c.Param("unit_id") != string(unitID) {
		apiresponse.AbortError(c, http.StatusNotFound, "health_form_not_found", "health form not found")
		return false
	}
	return true
}

func respondHealthFormError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, healthforms.ErrInvalidHealthFormInput):
		apiresponse.AbortError(c, http.StatusBadRequest, "invalid_health_form_input", "invalid health form input")
	case errors.Is(err, healthforms.ErrHealthFormUnavailable):
		apiresponse.AbortError(c, http.StatusUnprocessableEntity, "health_form_enrollment_unavailable", "enrollment is unavailable")
	case errors.Is(err, repositories.ErrHealthFormAlreadySubmitted):
		apiresponse.AbortError(c, http.StatusConflict, "health_form_already_submitted", "health form already submitted")
	case errors.Is(err, repositories.ErrHealthFormNotFound):
		apiresponse.AbortError(c, http.StatusNotFound, "health_form_not_found", "health form not found")
	case errors.Is(err, repositories.ErrHealthFormSessionInvalid):
		apiresponse.AbortError(c, http.StatusGone, "health_form_session_invalid", "health form session is invalid or expired")
	default:
		apiresponse.AbortError(c, http.StatusInternalServerError, "health_form_internal_error", "health form operation failed")
	}
}
