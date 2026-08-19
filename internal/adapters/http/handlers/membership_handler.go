package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/memberships"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type MembershipHandler struct{ service *memberships.Service }

func NewMembershipHandler(service *memberships.Service) MembershipHandler {
	return MembershipHandler{service: service}
}

func (h MembershipHandler) List(c *gin.Context) {
	unitID, _, ok := membershipContext(c)
	if !ok {
		return
	}
	includeArchived := false
	if raw, exists := c.GetQuery("include_archived"); exists {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			respondMembershipError(c, memberships.ErrInvalid)
			return
		}
		includeArchived = parsed
	}
	items, err := h.service.List(c.Request.Context(), unitID, includeArchived)
	if err != nil {
		respondMembershipError(c, err)
		return
	}
	response := make([]dto.MembershipPlanResponse, 0, len(items))
	for _, item := range items {
		response = append(response, membershipPlanResponse(item, false))
	}
	c.JSON(http.StatusOK, response)
}

func (h MembershipHandler) Get(c *gin.Context) {
	unitID, _, ok := membershipContext(c)
	if !ok {
		return
	}
	item, err := h.service.Get(c.Request.Context(), unitID, domain.ID(c.Param("id")))
	if err != nil {
		respondMembershipError(c, err)
		return
	}
	c.JSON(http.StatusOK, membershipPlanResponse(*item, true))
}

func (h MembershipHandler) Create(c *gin.Context) {
	unitID, actorID, ok := membershipContext(c)
	if !ok {
		return
	}
	var request dto.MembershipPlanRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondMembershipError(c, memberships.ErrInvalid)
		return
	}
	item, _, err := h.service.Create(c.Request.Context(), unitID, actorID, c.GetHeader("Idempotency-Key"), request)
	if err != nil {
		respondMembershipError(c, err)
		return
	}
	c.JSON(http.StatusCreated, membershipPlanResponse(*item, true))
}

func (h MembershipHandler) UpdateDraft(c *gin.Context) {
	unitID, actorID, ok := membershipContext(c)
	if !ok {
		return
	}
	var request dto.MembershipPlanRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondMembershipError(c, memberships.ErrInvalid)
		return
	}
	item, err := h.service.UpdateDraft(c.Request.Context(), unitID, actorID, domain.ID(c.Param("id")), request)
	if err != nil {
		respondMembershipError(c, err)
		return
	}
	c.JSON(http.StatusOK, membershipPlanResponse(*item, true))
}

func (h MembershipHandler) Publish(c *gin.Context) {
	unitID, actorID, ok := membershipContext(c)
	if !ok {
		return
	}
	var request dto.PublishMembershipPlanRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondMembershipError(c, memberships.ErrInvalid)
		return
	}
	item, _, err := h.service.Publish(c.Request.Context(), unitID, actorID, domain.ID(c.Param("id")), c.GetHeader("Idempotency-Key"), request.DraftRevision)
	if err != nil {
		respondMembershipError(c, err)
		return
	}
	c.JSON(http.StatusOK, membershipPlanResponse(*item, true))
}

func (h MembershipHandler) NewVersion(c *gin.Context) {
	unitID, actorID, ok := membershipContext(c)
	if !ok {
		return
	}
	item, _, err := h.service.NewVersion(c.Request.Context(), unitID, actorID, domain.ID(c.Param("id")), c.GetHeader("Idempotency-Key"))
	if err != nil {
		respondMembershipError(c, err)
		return
	}
	c.JSON(http.StatusCreated, membershipPlanResponse(*item, true))
}

func (h MembershipHandler) Archive(c *gin.Context)    { h.setArchived(c, true) }
func (h MembershipHandler) Reactivate(c *gin.Context) { h.setArchived(c, false) }
func (h MembershipHandler) setArchived(c *gin.Context, archived bool) {
	unitID, actorID, ok := membershipContext(c)
	if !ok {
		return
	}
	item, err := h.service.SetArchived(c.Request.Context(), unitID, actorID, domain.ID(c.Param("id")), archived)
	if err != nil {
		respondMembershipError(c, err)
		return
	}
	c.JSON(http.StatusOK, membershipPlanResponse(*item, true))
}

func membershipContext(c *gin.Context) (domain.ID, domain.ID, bool) {
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

func membershipPlanResponse(item domain.MembershipPlan, history bool) dto.MembershipPlanResponse {
	response := dto.MembershipPlanResponse{ID: string(item.ID), Name: item.Name, Status: string(item.Status), CreatedAt: item.CreatedAt.Format(time.RFC3339), UpdatedAt: item.UpdatedAt.Format(time.RFC3339)}
	if item.Published != nil {
		converted := membershipVersionResponse(*item.Published)
		response.Published = &converted
	}
	if item.Draft != nil {
		converted := membershipVersionResponse(*item.Draft)
		response.Draft = &converted
	}
	if history {
		response.Versions = make([]dto.MembershipPlanVersionResponse, 0, len(item.Versions))
		for _, version := range item.Versions {
			response.Versions = append(response.Versions, membershipVersionResponse(version))
		}
	}
	return response
}

func membershipVersionResponse(item domain.MembershipPlanVersion) dto.MembershipPlanVersionResponse {
	response := dto.MembershipPlanVersionResponse{ID: string(item.ID), VersionNumber: item.VersionNumber, Revision: item.Revision, Status: string(item.Status), Name: item.Name, Description: item.Description, PriceCents: item.PriceCents, Currency: item.Currency, BillingCycle: string(item.BillingCycle), ValidityCount: item.ValidityCount, Entitlements: make([]dto.MembershipPlanEntitlementResponse, 0, len(item.Entitlements)), Summary: item.Summary, Warnings: item.Warnings, CreatedAt: item.CreatedAt.Format(time.RFC3339), UpdatedAt: item.UpdatedAt.Format(time.RFC3339)}
	if item.ValidityUnit != nil {
		value := string(*item.ValidityUnit)
		response.ValidityUnit = &value
	}
	if item.PublishedAt != nil {
		value := item.PublishedAt.Format(time.RFC3339)
		response.PublishedAt = &value
	}
	for _, entitlement := range item.Entitlements {
		converted := dto.MembershipPlanEntitlementResponse{ID: string(entitlement.ID), TargetName: entitlement.TargetName, TargetType: entitlement.TargetType, AccessType: string(entitlement.AccessType), Quantity: entitlement.Quantity, DailyLimit: entitlement.DailyLimit, BookingRequired: entitlement.BookingRequired, Position: entitlement.Position}
		if entitlement.ModalityID != "" {
			value := string(entitlement.ModalityID)
			converted.ModalityID = &value
		}
		if entitlement.ServiceOfferingID != "" {
			value := string(entitlement.ServiceOfferingID)
			converted.ServiceOfferingID = &value
		}
		if entitlement.PeriodUnit != nil {
			value := string(*entitlement.PeriodUnit)
			converted.PeriodUnit = &value
		}
		if entitlement.DeliveryMode != nil {
			value := string(*entitlement.DeliveryMode)
			converted.DeliveryMode = &value
		}
		response.Entitlements = append(response.Entitlements, converted)
	}
	return response
}

func respondMembershipError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, memberships.ErrPlanNotFound):
		respondPublicError(c, http.StatusNotFound, "membership_plan_not_found", "plano não encontrado")
	case errors.Is(err, memberships.ErrNameConflict):
		respondPublicError(c, http.StatusConflict, "membership_plan_name_conflict", "já existe um plano com este nome")
	case errors.Is(err, memberships.ErrInvalid), errors.Is(err, memberships.ErrInvalidKey):
		respondPublicError(c, http.StatusUnprocessableEntity, "membership_plan_invalid", "dados do plano inválidos")
	case errors.Is(err, memberships.ErrTargetInvalid):
		respondPublicError(c, http.StatusUnprocessableEntity, "membership_plan_target_invalid", "modalidade ou serviço indisponível nesta unidade")
	case errors.Is(err, memberships.ErrDraftNotFound):
		respondPublicError(c, http.StatusConflict, "membership_plan_draft_not_found", "rascunho não encontrado")
	case errors.Is(err, memberships.ErrDraftExists):
		respondPublicError(c, http.StatusConflict, "membership_plan_draft_exists", "já existe um rascunho para este plano")
	case errors.Is(err, memberships.ErrRevisionConflict):
		respondPublicError(c, http.StatusConflict, "membership_plan_revision_conflict", "o rascunho foi alterado em outra aba")
	case errors.Is(err, memberships.ErrAlreadyPublished):
		respondPublicError(c, http.StatusConflict, "membership_plan_already_published", "publicação inválida")
	case errors.Is(err, memberships.ErrPlanArchived):
		respondPublicError(c, http.StatusConflict, "membership_plan_archived", "reative o plano antes de publicar")
	case errors.Is(err, memberships.ErrIdempotencyReused):
		respondPublicError(c, http.StatusConflict, "idempotency_key_reused", "chave de idempotência reutilizada")
	case errors.Is(err, memberships.ErrUnitNotFound):
		respondPublicError(c, http.StatusNotFound, "membership_plan_not_found", "plano não encontrado")
	default:
		respondError(c, err)
	}
}
