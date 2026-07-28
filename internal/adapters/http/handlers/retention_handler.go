package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/retention"
	"boxengage/backend/internal/domain"
)

type RetentionHandler struct{ service retention.Service }

func NewRetentionHandler(service retention.Service) RetentionHandler {
	return RetentionHandler{service: service}
}

func (h RetentionHandler) Radar(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	items, err := h.service.ListRadar(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}
	response := make([]dto.RetentionRadarResponse, 0, len(items))
	for _, item := range items {
		response = append(response, retentionRadarResponse(item))
	}
	c.JSON(http.StatusOK, response)
}

func (h RetentionHandler) ListInterventions(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	items, err := h.service.ListInterventions(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}
	response := make([]dto.RetentionInterventionResponse, 0, len(items))
	for _, item := range items {
		response = append(response, retentionInterventionResponse(item))
	}
	c.JSON(http.StatusOK, response)
}

func (h RetentionHandler) CreateIntervention(c *gin.Context) {
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
	var request dto.RetentionInterventionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	plannedFor, ok := parseOptionalTime(request.PlannedFor)
	if !ok {
		respondBadRequest(c)
		return
	}
	item := domain.RetentionIntervention{BoxID: boxID, StudentID: domain.ID(c.Param("id")), CreatedByUserID: userID, Channel: request.Channel, Status: request.Status, Outcome: request.Outcome, PlannedFor: plannedFor, Notes: request.Notes}
	if err := h.service.CreateIntervention(c.Request.Context(), &item); err != nil {
		if errors.Is(err, retention.ErrInvalidIntervention) {
			respondBadRequest(c)
			return
		}
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, retentionInterventionResponse(item))
}

func (h RetentionHandler) UpdateIntervention(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.RetentionInterventionUpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	plannedFor, ok := parseOptionalTime(request.PlannedFor)
	if !ok {
		respondBadRequest(c)
		return
	}
	item, err := h.service.UpdateIntervention(c.Request.Context(), boxID, domain.ID(c.Param("id")), request.Status, request.Outcome, request.Notes, plannedFor)
	if err != nil {
		if errors.Is(err, retention.ErrInvalidIntervention) {
			respondBadRequest(c)
			return
		}
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, retentionInterventionResponse(*item))
}

func parseOptionalTime(value *string) (*time.Time, bool) {
	if value == nil || *value == "" {
		return nil, true
	}
	parsed, err := time.Parse(time.RFC3339, *value)
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

func retentionRadarResponse(item domain.RetentionRadarItem) dto.RetentionRadarResponse {
	response := dto.RetentionRadarResponse{
		StudentID: string(item.StudentID), StudentName: item.StudentName, StudentPhone: item.StudentPhone,
		Source: string(item.Source), ContactStatus: string(item.ContactStatus), Level: string(item.Level),
		DaysSinceCheckin: item.DaysSinceCheckin, RecentCheckins: item.RecentCheckins, PreviousCheckins: item.PreviousCheckins,
		RecentWeeklyAverage: item.RecentWeeklyAverage, PreviousWeeklyAverage: item.PreviousWeeklyAverage,
		DropPercentage: item.DropPercentage, Signals: []dto.RetentionSignalResponse{},
		ReturnWithin3Days: item.ReturnWithin3Days, ReturnWithin7Days: item.ReturnWithin7Days, ReturnWithin14Days: item.ReturnWithin14Days,
		WorkflowStatus: string(item.WorkflowStatus), LastInterventionID: string(item.LastInterventionID),
		LastInterventionChannel: item.LastInterventionChannel, LastInterventionStatus: item.LastInterventionStatus,
		LastInterventionOutcome: item.LastInterventionOutcome,
	}
	if item.FirstCheckin != nil {
		value := item.FirstCheckin.Format("2006-01-02")
		response.FirstCheckin = &value
	}
	if item.LastCheckin != nil {
		value := item.LastCheckin.Format("2006-01-02")
		response.LastCheckin = &value
	}
	if item.LastCompletedIntervention != nil {
		value := item.LastCompletedIntervention.Format(time.RFC3339)
		response.LastCompletedIntervention = &value
	}
	if item.FirstReturnAfterAction != nil {
		value := item.FirstReturnAfterAction.Format("2006-01-02")
		response.FirstReturnAfterAction = &value
	}
	if item.FollowUpDueAt != nil {
		value := item.FollowUpDueAt.Format(time.RFC3339)
		response.FollowUpDueAt = &value
	}
	if item.LastInterventionPlannedFor != nil {
		value := item.LastInterventionPlannedFor.Format(time.RFC3339)
		response.LastInterventionPlannedFor = &value
	}
	if item.LastInterventionCreatedAt != nil {
		value := item.LastInterventionCreatedAt.Format(time.RFC3339)
		response.LastInterventionCreatedAt = &value
	}
	for _, signal := range item.Signals {
		response.Signals = append(response.Signals, dto.RetentionSignalResponse{Code: signal.Code, Message: signal.Message})
	}
	return response
}

func retentionInterventionResponse(item domain.RetentionIntervention) dto.RetentionInterventionResponse {
	response := dto.RetentionInterventionResponse{ID: string(item.ID), StudentID: string(item.StudentID), CreatedByUserID: string(item.CreatedByUserID), Channel: item.Channel, Status: item.Status, Outcome: item.Outcome, Notes: item.Notes, CreatedAt: item.CreatedAt.Format(time.RFC3339), UpdatedAt: item.UpdatedAt.Format(time.RFC3339)}
	if item.PlannedFor != nil {
		value := item.PlannedFor.Format(time.RFC3339)
		response.PlannedFor = &value
	}
	if item.CompletedAt != nil {
		value := item.CompletedAt.Format(time.RFC3339)
		response.CompletedAt = &value
	}
	return response
}
