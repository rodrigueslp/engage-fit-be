package handlers

import (
	"errors"
	"net/http"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/organizations"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type OrganizationHandler struct{ service *organizations.Service }

func NewOrganizationHandler(service *organizations.Service) OrganizationHandler {
	return OrganizationHandler{service: service}
}

func (h OrganizationHandler) ListUnits(c *gin.Context) {
	userID, unitID, ok := organizationContext(c)
	if !ok {
		return
	}
	items, err := h.service.ListUnits(c.Request.Context(), userID, unitID)
	if err != nil {
		respondOrganizationError(c, err)
		return
	}
	response := make([]dto.UnitResponse, 0, len(items))
	for _, item := range items {
		response = append(response, unitResponse(item, unitID))
	}
	c.JSON(http.StatusOK, response)
}

func (h OrganizationHandler) CreateUnit(c *gin.Context) {
	userID, unitID, ok := organizationContext(c)
	if !ok {
		return
	}
	var request dto.CreateUnitRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondOrganizationError(c, organizations.ErrInvalidUnit)
		return
	}
	item, err := h.service.CreateUnit(c.Request.Context(), userID, unitID, request.Name, request.Timezone)
	if err != nil {
		respondOrganizationError(c, err)
		return
	}
	c.JSON(http.StatusCreated, unitResponse(*item, unitID))
}

func (h OrganizationHandler) CreateUnitAsPlatformAdmin(c *gin.Context) {
	adminID, err := middleware.UserID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.CreateUnitRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondOrganizationError(c, organizations.ErrInvalidUnit)
		return
	}
	item, err := h.service.CreateUnitAsPlatformAdmin(c.Request.Context(), adminID, domain.ID(c.Param("organization_id")), request.Name, request.Timezone)
	if err != nil {
		respondOrganizationError(c, err)
		return
	}
	c.JSON(http.StatusCreated, unitResponse(*item, ""))
}

func (h OrganizationHandler) UpdateUnit(c *gin.Context) {
	userID, currentUnitID, ok := organizationContext(c)
	if !ok {
		return
	}
	var request dto.UpdateUnitRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondOrganizationError(c, organizations.ErrInvalidUnit)
		return
	}
	var status *domain.BoxStatus
	if request.Status != nil {
		value := domain.BoxStatus(*request.Status)
		status = &value
	}
	item, err := h.service.UpdateUnit(c.Request.Context(), userID, currentUnitID, domain.ID(c.Param("unit_id")), organizations.UnitChanges{Name: request.Name, Timezone: request.Timezone, Status: status, Reason: request.Reason})
	if err != nil {
		respondOrganizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, unitResponse(*item, currentUnitID))
}

func (h OrganizationHandler) MemberUnitAccess(c *gin.Context) {
	actorID, currentUnitID, ok := organizationContext(c)
	if !ok {
		return
	}
	ids, err := h.service.ListMemberUnitIDs(c.Request.Context(), actorID, currentUnitID, domain.ID(c.Param("id")))
	if err != nil {
		respondOrganizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.MemberUnitAccessResponse{UnitIDs: stringIDs(ids)})
}

func (h OrganizationHandler) ReplaceMemberUnitAccess(c *gin.Context) {
	actorID, currentUnitID, ok := organizationContext(c)
	if !ok {
		return
	}
	var request dto.ReplaceMemberUnitAccessRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondOrganizationError(c, organizations.ErrInvalidMemberUnitAccess)
		return
	}
	unitIDs := make([]domain.ID, 0, len(request.UnitIDs))
	for _, id := range request.UnitIDs {
		unitIDs = append(unitIDs, domain.ID(id))
	}
	ids, err := h.service.ReplaceMemberUnitAccess(c.Request.Context(), actorID, currentUnitID, domain.ID(c.Param("id")), unitIDs)
	if err != nil {
		respondOrganizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.MemberUnitAccessResponse{UnitIDs: stringIDs(ids)})
}

func organizationContext(c *gin.Context) (domain.ID, domain.ID, bool) {
	userID, userErr := middleware.UserID(c)
	unitID, unitErr := middleware.BoxID(c)
	if userErr != nil || unitErr != nil {
		respondUnauthorized(c)
		return "", "", false
	}
	return userID, unitID, true
}

func unitResponse(item domain.Box, currentUnitID domain.ID) dto.UnitResponse {
	return dto.UnitResponse{ID: string(item.ID), Name: item.Name, Timezone: item.Timezone, Status: string(item.EffectiveStatus()), IsCurrent: item.ID == currentUnitID}
}

func stringIDs(ids []domain.ID) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		result = append(result, string(id))
	}
	return result
}

func respondOrganizationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, organizations.ErrInvalidUnit), errors.Is(err, organizations.ErrInvalidMemberUnitAccess):
		respondPublicError(c, http.StatusBadRequest, "invalid_organization_input", "dados da organização inválidos")
	case errors.Is(err, organizations.ErrUnitNotFound), errors.Is(err, organizations.ErrUnitAccessNotFound), errors.Is(err, organizations.ErrMemberNotFound):
		respondPublicError(c, http.StatusNotFound, "unit_not_found", "unidade não encontrada")
	case errors.Is(err, organizations.ErrUnitNameConflict):
		respondPublicError(c, http.StatusConflict, "unit_name_conflict", "já existe uma unidade com este nome")
	case errors.Is(err, organizations.ErrLastOperationalUnit):
		respondPublicError(c, http.StatusConflict, "last_operational_unit", "a organização precisa manter ao menos uma unidade")
	case errors.Is(err, organizations.ErrCurrentUnitArchive):
		respondPublicError(c, http.StatusConflict, "current_unit_archive", "troque de unidade antes de arquivar a unidade atual")
	default:
		respondError(c, err)
	}
}
