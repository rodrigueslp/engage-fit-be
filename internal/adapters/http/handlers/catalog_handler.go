package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/catalog"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type CatalogHandler struct{ service *catalog.Service }

func NewCatalogHandler(service *catalog.Service) CatalogHandler {
	return CatalogHandler{service: service}
}

func (h CatalogHandler) Organization(c *gin.Context) {
	unitID, ok := catalogUnitID(c)
	if !ok {
		return
	}
	current, err := h.service.Organization(c.Request.Context(), unitID)
	if err != nil {
		respondCatalogError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.OrganizationResponse{ID: string(current.Organization.ID), Name: current.Organization.Name, Status: string(current.Organization.EffectiveStatus()), CurrentUnit: dto.OrganizationUnitResponse{ID: string(current.Unit.ID), Name: current.Unit.Name, Status: string(current.Unit.EffectiveStatus())}})
}

func (h CatalogHandler) ListModalities(c *gin.Context) {
	unitID, ok := catalogUnitID(c)
	if !ok {
		return
	}
	includeArchived, valid := catalogBoolQuery(c, "include_archived")
	if !valid {
		respondCatalogError(c, catalog.ErrInvalidInput)
		return
	}
	requestedUnitID := domain.ID(c.Query("unit_id"))
	items, err := h.service.ListModalities(c.Request.Context(), unitID, requestedUnitID, includeArchived)
	if err != nil {
		respondCatalogError(c, err)
		return
	}
	response := make([]dto.ModalityResponse, 0, len(items))
	for _, item := range items {
		response = append(response, modalityResponse(item))
	}
	c.JSON(http.StatusOK, response)
}

func (h CatalogHandler) CreateModality(c *gin.Context) {
	unitID, actorID, ok := catalogMutationContext(c)
	if !ok {
		return
	}
	var request dto.CreateModalityRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondCatalogError(c, catalog.ErrInvalidInput)
		return
	}
	item, err := h.service.CreateModality(c.Request.Context(), unitID, actorID, request.Name, request.Color)
	if err != nil {
		respondCatalogError(c, err)
		return
	}
	c.JSON(http.StatusCreated, modalityResponse(*item))
}

func (h CatalogHandler) UpdateModality(c *gin.Context) {
	unitID, actorID, ok := catalogMutationContext(c)
	if !ok {
		return
	}
	var request dto.UpdateModalityRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondCatalogError(c, catalog.ErrInvalidInput)
		return
	}
	item, err := h.service.UpdateModality(c.Request.Context(), unitID, actorID, domain.ID(c.Param("id")), catalog.ModalityChanges{Name: request.Name, Color: request.Color, Status: request.Status})
	if err != nil {
		respondCatalogError(c, err)
		return
	}
	c.JSON(http.StatusOK, modalityResponse(*item))
}

func (h CatalogHandler) SetModalityActive(c *gin.Context) {
	unitID, actorID, ok := catalogMutationContext(c)
	if !ok {
		return
	}
	var request dto.SetUnitModalityRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.Active == nil {
		respondCatalogError(c, catalog.ErrInvalidInput)
		return
	}
	item, err := h.service.SetModalityActive(c.Request.Context(), unitID, domain.ID(c.Param("unit_id")), actorID, domain.ID(c.Param("modality_id")), *request.Active)
	if err != nil {
		respondCatalogError(c, err)
		return
	}
	c.JSON(http.StatusOK, modalityResponse(*item))
}

func (h CatalogHandler) ListSpaces(c *gin.Context) {
	unitID, ok := catalogUnitID(c)
	if !ok {
		return
	}
	includeArchived, valid := catalogBoolQuery(c, "include_archived")
	if !valid {
		respondCatalogError(c, catalog.ErrInvalidInput)
		return
	}
	items, err := h.service.ListSpaces(c.Request.Context(), unitID, domain.ID(c.Param("unit_id")), includeArchived)
	if err != nil {
		respondCatalogError(c, err)
		return
	}
	response := make([]dto.SpaceResponse, 0, len(items))
	for _, item := range items {
		response = append(response, spaceResponse(item))
	}
	c.JSON(http.StatusOK, response)
}

func (h CatalogHandler) CreateSpace(c *gin.Context) {
	unitID, actorID, ok := catalogMutationContext(c)
	if !ok {
		return
	}
	var request dto.CreateSpaceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondCatalogError(c, catalog.ErrInvalidInput)
		return
	}
	capacity, _, valid := parseOptionalCapacity(request.DefaultCapacity)
	if !valid {
		respondCatalogError(c, catalog.ErrInvalidInput)
		return
	}
	item, err := h.service.CreateSpace(c.Request.Context(), unitID, domain.ID(c.Param("unit_id")), actorID, request.Name, capacity)
	if err != nil {
		respondCatalogError(c, err)
		return
	}
	c.JSON(http.StatusCreated, spaceResponse(*item))
}

func (h CatalogHandler) UpdateSpace(c *gin.Context) {
	unitID, actorID, ok := catalogMutationContext(c)
	if !ok {
		return
	}
	var request dto.UpdateSpaceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondCatalogError(c, catalog.ErrInvalidInput)
		return
	}
	capacity, clearCapacity, valid := parseOptionalCapacity(request.DefaultCapacity)
	if !valid {
		respondCatalogError(c, catalog.ErrInvalidInput)
		return
	}
	item, err := h.service.UpdateSpace(c.Request.Context(), unitID, domain.ID(c.Param("unit_id")), actorID, domain.ID(c.Param("id")), catalog.SpaceChanges{Name: request.Name, DefaultCapacity: capacity, ClearDefaultCapacity: clearCapacity, Status: request.Status})
	if err != nil {
		respondCatalogError(c, err)
		return
	}
	c.JSON(http.StatusOK, spaceResponse(*item))
}

func catalogUnitID(c *gin.Context) (domain.ID, bool) {
	id, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return "", false
	}
	return id, true
}

func catalogMutationContext(c *gin.Context) (domain.ID, domain.ID, bool) {
	unitID, ok := catalogUnitID(c)
	if !ok {
		return "", "", false
	}
	actorID, err := middleware.UserID(c)
	if err != nil {
		respondUnauthorized(c)
		return "", "", false
	}
	return unitID, actorID, true
}

func catalogBoolQuery(c *gin.Context, name string) (bool, bool) {
	value, exists := c.GetQuery(name)
	if !exists || value == "" {
		return false, true
	}
	parsed, err := strconv.ParseBool(value)
	return parsed, err == nil
}

func parseOptionalCapacity(raw json.RawMessage) (*int, bool, bool) {
	if len(raw) == 0 {
		return nil, false, true
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, true, true
	}
	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, false, false
	}
	return &value, false, true
}

func modalityResponse(item domain.Modality) dto.ModalityResponse {
	return dto.ModalityResponse{ID: string(item.ID), Name: item.Name, Color: item.Color, Status: string(item.Status), ActiveInUnit: item.ActiveInUnit, CreatedAt: item.CreatedAt.Format(time.RFC3339), UpdatedAt: item.UpdatedAt.Format(time.RFC3339)}
}

func spaceResponse(item domain.Space) dto.SpaceResponse {
	return dto.SpaceResponse{ID: string(item.ID), UnitID: string(item.UnitID), Name: item.Name, DefaultCapacity: item.DefaultCapacity, Status: string(item.Status), CreatedAt: item.CreatedAt.Format(time.RFC3339), UpdatedAt: item.UpdatedAt.Format(time.RFC3339)}
}

func respondCatalogError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, catalog.ErrInvalidInput):
		respondPublicError(c, http.StatusBadRequest, "invalid_catalog_input", "dados do catálogo inválidos")
	case errors.Is(err, catalog.ErrOrganizationNotFound):
		respondPublicError(c, http.StatusNotFound, "organization_not_found", "organização não encontrada")
	case errors.Is(err, catalog.ErrUnitNotFound):
		respondPublicError(c, http.StatusNotFound, "unit_not_found", "unidade não encontrada")
	case errors.Is(err, catalog.ErrModalityNotFound):
		respondPublicError(c, http.StatusNotFound, "modality_not_found", "modalidade não encontrada")
	case errors.Is(err, catalog.ErrSpaceNotFound):
		respondPublicError(c, http.StatusNotFound, "space_not_found", "espaço não encontrado")
	case errors.Is(err, catalog.ErrModalityNameConflict):
		respondPublicError(c, http.StatusConflict, "modality_name_conflict", "já existe uma modalidade com este nome")
	case errors.Is(err, catalog.ErrSpaceNameConflict):
		respondPublicError(c, http.StatusConflict, "space_name_conflict", "já existe um espaço com este nome")
	case errors.Is(err, catalog.ErrModalityArchived):
		respondPublicError(c, http.StatusConflict, "modality_archived", "reative a modalidade antes de ativá-la na unidade")
	default:
		respondError(c, err)
	}
}
