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

func (h RetentionHandler) Summary(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	start, end := today.AddDate(0, 0, -29), today
	if value := c.Query("start_date"); value != "" {
		start, err = time.Parse("2006-01-02", value)
		if err != nil {
			respondBadRequest(c)
			return
		}
	}
	if value := c.Query("end_date"); value != "" {
		end, err = time.Parse("2006-01-02", value)
		if err != nil {
			respondBadRequest(c)
			return
		}
	}
	result, err := h.service.Summary(c.Request.Context(), boxID, start, end)
	if err != nil {
		if errors.Is(err, retention.ErrInvalidIntervention) {
			respondBadRequest(c)
			return
		}
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, retentionSummaryResponse(*result))
}

func (h RetentionHandler) OnboardingJourney(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	items, err := h.service.ListOnboardingJourney(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}
	response := make([]dto.OnboardingJourneyResponse, 0, len(items))
	for _, item := range items {
		response = append(response, onboardingJourneyResponse(item))
	}
	c.JSON(http.StatusOK, response)
}

func (h RetentionHandler) UpdateMembershipStart(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.UpdateMembershipStartRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	startedAt, err := time.Parse("2006-01-02", request.StartedAt)
	if err != nil {
		respondBadRequest(c)
		return
	}
	if err := h.service.UpdateMembershipStart(c.Request.Context(), boxID, domain.ID(c.Param("id")), startedAt); err != nil {
		if errors.Is(err, retention.ErrInvalidIntervention) {
			respondBadRequest(c)
			return
		}
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
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
	item := domain.RetentionIntervention{BoxID: boxID, StudentID: domain.ID(c.Param("id")), CreatedByUserID: userID, AssignedToUserID: domain.ID(request.AssignedToUserID), Channel: request.Channel, Status: request.Status, Outcome: request.Outcome, ReasonCode: request.ReasonCode, PlannedFor: plannedFor, Notes: request.Notes}
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
	item, err := h.service.UpdateIntervention(c.Request.Context(), boxID, domain.ID(c.Param("id")), request.Status, request.Outcome, request.ReasonCode, request.Notes, plannedFor, domain.ID(request.AssignedToUserID))
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
		LastInterventionOutcome:      item.LastInterventionOutcome,
		LastInterventionAssigneeID:   string(item.LastInterventionAssigneeID),
		LastInterventionAssigneeName: item.LastInterventionAssigneeName,
		Recommendation:               dto.RetentionRecommendationResponse{Code: item.Recommendation.Code, Title: item.Recommendation.Title, Message: item.Recommendation.Message},
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
	response := dto.RetentionInterventionResponse{ID: string(item.ID), StudentID: string(item.StudentID), CreatedByUserID: string(item.CreatedByUserID), AssignedToUserID: string(item.AssignedToUserID), AssignedToUserName: item.AssignedToUserName, Channel: item.Channel, Status: item.Status, Outcome: item.Outcome, ReasonCode: item.ReasonCode, Notes: item.Notes, CreatedAt: item.CreatedAt.Format(time.RFC3339), UpdatedAt: item.UpdatedAt.Format(time.RFC3339)}
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

func retentionSummaryResponse(item domain.RetentionSummary) dto.RetentionSummaryResponse {
	response := dto.RetentionSummaryResponse{
		PeriodStart: item.PeriodStart.Format("2006-01-02"), PeriodEnd: item.PeriodEnd.Format("2006-01-02"),
		NeedsAction: item.NeedsAction, WaitingReturn: item.WaitingReturn, FollowUpDue: item.FollowUpDue,
		Recovered: item.Recovered, CompletedInterventions: item.CompletedInterventions,
		ReturnWithin3Days: item.ReturnWithin3Days, ReturnWithin7Days: item.ReturnWithin7Days,
		ReturnWithin14Days: item.ReturnWithin14Days, MedianDaysToReturn: item.MedianDaysToReturn,
		Reasons: []dto.RetentionBreakdownResponse{}, Channels: []dto.RetentionBreakdownResponse{}, Outcomes: []dto.RetentionBreakdownResponse{},
	}
	for _, value := range item.Reasons {
		response.Reasons = append(response.Reasons, dto.RetentionBreakdownResponse{Code: value.Code, Count: value.Count})
	}
	for _, value := range item.Channels {
		response.Channels = append(response.Channels, dto.RetentionBreakdownResponse{Code: value.Code, Count: value.Count})
	}
	for _, value := range item.Outcomes {
		response.Outcomes = append(response.Outcomes, dto.RetentionBreakdownResponse{Code: value.Code, Count: value.Count})
	}
	return response
}

func onboardingJourneyResponse(item domain.OnboardingJourneyItem) dto.OnboardingJourneyResponse {
	response := dto.OnboardingJourneyResponse{
		StudentID: string(item.StudentID), StudentName: item.StudentName, StudentPhone: item.StudentPhone,
		Source: string(item.Source), ContactStatus: string(item.ContactStatus),
		MembershipStartedAt: item.MembershipStartedAt.Format("2006-01-02"), MembershipStartedSource: item.MembershipStartedSource,
		Day: item.Day, DaysSinceCheckin: item.DaysSinceCheckin, CheckinsFirst7Days: item.CheckinsFirst7Days,
		CheckinsFirst14Days: item.CheckinsFirst14Days, CheckinsFirst30Days: item.CheckinsFirst30Days,
		Status: item.Status, StatusMessage: item.StatusMessage,
		Recommendation: dto.RetentionRecommendationResponse{Code: item.Recommendation.Code, Title: item.Recommendation.Title, Message: item.Recommendation.Message},
	}
	if item.FirstCheckin != nil {
		value := item.FirstCheckin.Format("2006-01-02")
		response.FirstCheckin = &value
	}
	if item.SecondCheckin != nil {
		value := item.SecondCheckin.Format("2006-01-02")
		response.SecondCheckin = &value
	}
	if item.LastCheckin != nil {
		value := item.LastCheckin.Format("2006-01-02")
		response.LastCheckin = &value
	}
	return response
}
