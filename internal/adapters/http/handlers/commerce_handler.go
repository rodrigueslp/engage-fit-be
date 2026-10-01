package handlers

import (
	"errors"
	"net/http"
	"strconv"

	commerceapp "boxengage/backend/internal/app/commerce"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type CommerceHandler struct{ service *commerceapp.Service }

func NewCommerceHandler(service *commerceapp.Service) CommerceHandler {
	return CommerceHandler{service: service}
}

func (h CommerceHandler) ListProducts(c *gin.Context) {
	unitID, _, ok := enrollmentContext(c)
	if !ok {
		return
	}
	items, err := h.service.ListProducts(c, unitID, c.Query("include_inactive") == "true")
	if err != nil {
		respondCommerceError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h CommerceHandler) CreateProduct(c *gin.Context) {
	unitID, actorID, ok := enrollmentContext(c)
	if !ok {
		return
	}
	var input commerceapp.ProductInput
	if c.ShouldBindJSON(&input) != nil {
		respondCommerceError(c, commerceapp.ErrInvalidInput)
		return
	}
	item, err := h.service.CreateProduct(c, unitID, actorID, input)
	if err != nil {
		respondCommerceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h CommerceHandler) UpdateProduct(c *gin.Context) {
	unitID, _, ok := enrollmentContext(c)
	if !ok {
		return
	}
	var input commerceapp.ProductInput
	if c.ShouldBindJSON(&input) != nil {
		respondCommerceError(c, commerceapp.ErrInvalidInput)
		return
	}
	item, err := h.service.UpdateProduct(c, unitID, domain.ID(c.Param("id")), input)
	if err != nil {
		respondCommerceError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h CommerceHandler) ListSales(c *gin.Context) {
	unitID, _, ok := enrollmentContext(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.service.ListSales(c, unitID, limit)
	if err != nil {
		respondCommerceError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h CommerceHandler) CreateSale(c *gin.Context) {
	unitID, actorID, ok := enrollmentContext(c)
	if !ok {
		return
	}
	var input commerceapp.CreateSaleInput
	if c.ShouldBindJSON(&input) != nil {
		respondCommerceError(c, commerceapp.ErrInvalidInput)
		return
	}
	item, replayed, err := h.service.CreateSale(c, unitID, actorID, c.GetHeader("Idempotency-Key"), input)
	if err != nil {
		respondCommerceError(c, err)
		return
	}
	if replayed {
		c.Header("Idempotency-Replayed", "true")
		c.JSON(http.StatusOK, item)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func respondCommerceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, commerceapp.ErrInvalidInput):
		respondPublicError(c, http.StatusUnprocessableEntity, "commerce_invalid", "revise o produto, itens, pagamento ou data da venda")
	case errors.Is(err, commerceapp.ErrNotFound):
		respondPublicError(c, http.StatusNotFound, "commerce_not_found", "produto, pessoa ou venda não encontrado")
	case errors.Is(err, commerceapp.ErrConflict):
		respondPublicError(c, http.StatusConflict, "commerce_conflict", "a operação conflita com o catálogo atual")
	case errors.Is(err, commerceapp.ErrIdempotencyReused):
		respondPublicError(c, http.StatusConflict, "idempotency_key_reused", "chave de idempotência reutilizada")
	default:
		respondError(c, err)
	}
}
