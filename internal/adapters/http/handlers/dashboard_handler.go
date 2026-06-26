package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/dashboard"
	"boxengage/backend/internal/app/rewards"
)

type DashboardHandler struct {
	summary         dashboard.GetDashboardSummaryUseCase
	activeCampaigns dashboard.ListActiveCampaignsUseCase
	nearGoal        dashboard.ListNearGoalStudentsUseCase
	atRisk          dashboard.ListAtRiskStudentsUseCase
	pendingRewards  rewards.ListPendingRewardDeliveriesUseCase
}

func NewDashboardHandler(summary dashboard.GetDashboardSummaryUseCase, activeCampaigns dashboard.ListActiveCampaignsUseCase, nearGoal dashboard.ListNearGoalStudentsUseCase, atRisk dashboard.ListAtRiskStudentsUseCase, pendingRewards rewards.ListPendingRewardDeliveriesUseCase) DashboardHandler {
	return DashboardHandler{summary: summary, activeCampaigns: activeCampaigns, nearGoal: nearGoal, atRisk: atRisk, pendingRewards: pendingRewards}
}

func (h DashboardHandler) Summary(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	summary, err := h.summary.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}

	checkinsByPlatform := map[string]int{}
	for source, total := range summary.CheckinsByPlatform {
		checkinsByPlatform[string(source)] = total
	}

	c.JSON(http.StatusOK, dto.DashboardSummaryResponse{
		TotalStudents:      summary.TotalStudents,
		TotalCheckins:      summary.TotalCheckins,
		EligibleStudents:   summary.EligibleStudents,
		NearGoalStudents:   summary.NearGoalStudents,
		AtRiskStudents:     summary.AtRiskStudents,
		PendingRewards:     summary.PendingRewards,
		DeliveredRewards:   summary.DeliveredRewards,
		CheckinsByPlatform: checkinsByPlatform,
	})
}

func (h DashboardHandler) ActiveCampaigns(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	result, err := h.activeCampaigns.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}

	response := make([]dto.CampaignResponse, 0, len(result))
	for _, campaign := range result {
		response = append(response, campaignResponse(campaign))
	}
	c.JSON(http.StatusOK, response)
}

func (h DashboardHandler) NearGoalStudents(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	result, err := h.nearGoal.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}

	response := make([]dto.StudentResponse, 0, len(result))
	for _, student := range result {
		response = append(response, studentResponse(student))
	}
	c.JSON(http.StatusOK, response)
}

func (h DashboardHandler) AtRiskStudents(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	result, err := h.atRisk.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}

	response := make([]dto.StudentResponse, 0, len(result))
	for _, student := range result {
		response = append(response, studentResponse(student))
	}
	c.JSON(http.StatusOK, response)
}

func (h DashboardHandler) PendingRewards(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	result, err := h.pendingRewards.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}

	response := make([]dto.RewardDeliveryResponse, 0, len(result))
	for _, delivery := range result {
		item := dto.RewardDeliveryResponse{
			ID:           string(delivery.ID),
			RewardID:     string(delivery.RewardID),
			RewardName:   delivery.RewardName,
			StudentID:    string(delivery.StudentID),
			StudentName:  delivery.StudentName,
			StudentPhone: delivery.StudentPhone,
			Delivered:    delivery.Delivered,
		}
		if delivery.DeliveredAt != nil {
			item.DeliveredAt = delivery.DeliveredAt.Format("2006-01-02T15:04:05Z07:00")
		}
		response = append(response, item)
	}
	c.JSON(http.StatusOK, response)
}
