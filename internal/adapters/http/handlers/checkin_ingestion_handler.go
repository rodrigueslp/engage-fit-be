package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/checkiningestion"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type CheckinIngestionHandler struct {
	service *checkiningestion.Service
}

func NewCheckinIngestionHandler(service *checkiningestion.Service) CheckinIngestionHandler {
	return CheckinIngestionHandler{service: service}
}

func (h CheckinIngestionHandler) ListSources(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	items, err := h.service.ListSources(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}
	response := make([]dto.CheckinIngestionSourceResponse, 0, len(items))
	for _, item := range items {
		response = append(response, checkinIngestionSourceResponse(item, ""))
	}
	c.JSON(http.StatusOK, response)
}

func (h CheckinIngestionHandler) CreateSource(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	userID, err := middleware.UserID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.CreateCheckinIngestionSourceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	result, err := h.service.CreateSource(c.Request.Context(), boxID, userID, request.Name, domain.Source(request.Source))
	if err != nil {
		if errors.Is(err, checkiningestion.ErrInvalidSource) {
			respondBadRequest(c)
			return
		}
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, checkinIngestionSourceResponse(result.Source, result.Token))
}

func (h CheckinIngestionHandler) UpdateSource(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.UpdateCheckinIngestionSourceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	item, err := h.service.UpdateSource(c.Request.Context(), boxID, domain.ID(c.Param("id")), request.Name, request.Enabled)
	if err != nil {
		if errors.Is(err, checkiningestion.ErrInvalidSource) {
			respondBadRequest(c)
			return
		}
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, checkinIngestionSourceResponse(*item, ""))
}

func (h CheckinIngestionHandler) RotateToken(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	result, err := h.service.RotateToken(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, checkinIngestionSourceResponse(result.Source, result.Token))
}

func (h CheckinIngestionHandler) Ingest(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		respondPublicError(c, http.StatusBadRequest, "invalid_file", "invalid check-in file")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		respondPublicError(c, http.StatusBadRequest, "invalid_file", "invalid check-in file")
		return
	}
	defer file.Close()
	result, err := h.service.Ingest(c.Request.Context(), checkiningestion.IngestInput{
		SourceID: domain.ID(c.Param("sourceId")), Token: c.GetHeader("X-Ingestion-Token"),
		IdempotencyKey: strings.TrimSpace(c.GetHeader("Idempotency-Key")), Filename: fileHeader.Filename, File: file,
	})
	if err != nil {
		switch {
		case errors.Is(err, checkiningestion.ErrInvalidCredential):
			respondPublicError(c, http.StatusUnauthorized, "ingestion_unauthorized", "invalid ingestion credential")
		case errors.Is(err, checkiningestion.ErrIngestionDisabled):
			respondPublicError(c, http.StatusForbidden, "ingestion_disabled", "ingestion source disabled")
		case errors.Is(err, checkiningestion.ErrInvalidIdempotency):
			respondPublicError(c, http.StatusBadRequest, "invalid_idempotency_key", "valid Idempotency-Key required")
		default:
			respondError(c, err)
		}
		return
	}
	status := http.StatusCreated
	if result.Replay {
		switch result.Batch.Status {
		case "processing":
			status = http.StatusAccepted
		case "failed":
			status = http.StatusConflict
		default:
			status = http.StatusOK
		}
	}
	c.JSON(status, checkinIngestionBatchResponse(result.Batch, result.Replay))
}

func checkinIngestionSourceResponse(item domain.CheckinIngestionSource, token string) dto.CheckinIngestionSourceResponse {
	response := dto.CheckinIngestionSourceResponse{
		ID: string(item.ID), Name: item.Name, Source: string(item.Source), Enabled: item.Enabled,
		CreatedAt: item.CreatedAt.Format(time.RFC3339), UpdatedAt: item.UpdatedAt.Format(time.RFC3339), Token: token,
	}
	if item.LastIngestedAt != nil {
		value := item.LastIngestedAt.Format(time.RFC3339)
		response.LastIngestedAt = &value
	}
	return response
}

func checkinIngestionBatchResponse(item domain.CheckinIngestionBatch, replay bool) dto.CheckinIngestionBatchResponse {
	response := dto.CheckinIngestionBatchResponse{
		ID: string(item.ID), Status: item.Status, IdempotentReplay: replay,
		ImportHistoryID: string(item.ImportHistoryID), TotalRecords: item.TotalRecords,
		StudentsCreated: item.StudentsCreated, CheckinsCreated: item.CheckinsCreated,
		CreatedAt: item.CreatedAt.Format(time.RFC3339),
	}
	if item.CompletedAt != nil {
		value := item.CompletedAt.Format(time.RFC3339)
		response.CompletedAt = &value
	}
	return response
}
