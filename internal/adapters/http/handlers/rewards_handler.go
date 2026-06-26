package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/rewards"
	"boxengage/backend/internal/domain"
)

type RewardsHandler struct {
	listRewards       rewards.ListRewardsUseCase
	createReward      rewards.CreateRewardUseCase
	getReward         rewards.GetRewardUseCase
	updateReward      rewards.UpdateRewardUseCase
	deleteReward      rewards.DeleteRewardUseCase
	deliveries        rewards.ListRewardDeliveriesUseCase
	pendingDeliveries rewards.ListPendingRewardDeliveriesUseCase
	markDelivered     rewards.MarkRewardDeliveredUseCase
}

func NewRewardsHandler(listRewards rewards.ListRewardsUseCase, createReward rewards.CreateRewardUseCase, getReward rewards.GetRewardUseCase, updateReward rewards.UpdateRewardUseCase, deleteReward rewards.DeleteRewardUseCase, deliveries rewards.ListRewardDeliveriesUseCase, pendingDeliveries rewards.ListPendingRewardDeliveriesUseCase, markDelivered rewards.MarkRewardDeliveredUseCase) RewardsHandler {
	return RewardsHandler{listRewards: listRewards, createReward: createReward, getReward: getReward, updateReward: updateReward, deleteReward: deleteReward, deliveries: deliveries, pendingDeliveries: pendingDeliveries, markDelivered: markDelivered}
}

func (h RewardsHandler) ListByCampaign(c *gin.Context) {
	result, err := h.listRewards.Execute(c.Request.Context(), domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}

	response := make([]dto.RewardResponse, 0, len(result))
	for _, reward := range result {
		response = append(response, rewardResponse(reward))
	}
	c.JSON(http.StatusOK, response)
}

func (h RewardsHandler) Create(c *gin.Context) {
	var request dto.RewardRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}

	reward := domain.Reward{
		CampaignID:  domain.ID(c.Param("id")),
		Name:        request.Name,
		Description: request.Description,
		Quantity:    request.Quantity,
	}
	if err := h.createReward.Execute(c.Request.Context(), &reward); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, rewardResponse(reward))
}

func (h RewardsHandler) Update(c *gin.Context) {
	existing, err := h.getReward.Execute(c.Request.Context(), domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}

	var request dto.RewardRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}

	existing.Name = request.Name
	existing.Description = request.Description
	existing.Quantity = request.Quantity
	if err := h.updateReward.Execute(c.Request.Context(), *existing); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, rewardResponse(*existing))
}

func (h RewardsHandler) Delete(c *gin.Context) {
	if err := h.deleteReward.Execute(c.Request.Context(), domain.ID(c.Param("id"))); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h RewardsHandler) PendingDeliveries(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	result, err := h.pendingDeliveries.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}

	response := make([]dto.RewardDeliveryResponse, 0, len(result))
	for _, delivery := range result {
		response = append(response, rewardDeliveryResponse(delivery))
	}
	c.JSON(http.StatusOK, response)
}

func (h RewardsHandler) Deliveries(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	result, err := h.deliveries.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}

	response := make([]dto.RewardDeliveryResponse, 0, len(result))
	for _, delivery := range result {
		response = append(response, rewardDeliveryResponse(delivery))
	}
	c.JSON(http.StatusOK, response)
}

func (h RewardsHandler) MarkDelivered(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	if err := h.markDelivered.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id"))); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func rewardResponse(reward domain.Reward) dto.RewardResponse {
	return dto.RewardResponse{
		ID:                  string(reward.ID),
		CampaignID:          string(reward.CampaignID),
		Name:                reward.Name,
		Description:         reward.Description,
		Quantity:            reward.Quantity,
		PendingDeliveries:   reward.PendingDeliveries,
		DeliveredDeliveries: reward.DeliveredDeliveries,
		AvailableQuantity:   reward.Quantity - reward.DeliveredDeliveries,
	}
}

func rewardDeliveryResponse(delivery domain.RewardDelivery) dto.RewardDeliveryResponse {
	item := dto.RewardDeliveryResponse{
		ID:           string(delivery.ID),
		CampaignID:   string(delivery.CampaignID),
		CampaignName: delivery.CampaignName,
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
	return item
}
