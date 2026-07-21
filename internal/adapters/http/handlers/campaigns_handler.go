package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/campaigns"
	"boxengage/backend/internal/app/students"
	"boxengage/backend/internal/domain"
)

type CampaignsHandler struct {
	listCampaigns  campaigns.ListCampaignsUseCase
	createCampaign campaigns.CreateCampaignUseCase
	getCampaign    campaigns.GetCampaignUseCase
	updateCampaign campaigns.UpdateCampaignUseCase
	closeCampaign  campaigns.CloseCampaignUseCase
	deleteCampaign campaigns.DeleteCampaignUseCase
	listGoals      campaigns.ListCampaignGoalsUseCase
	upsertGoal     campaigns.UpsertCampaignGoalUseCase
	deleteGoal     campaigns.DeleteCampaignGoalUseCase
	listProgress   campaigns.ListCampaignProgressUseCase
	recalculate    campaigns.RecalculateCampaignProgressUseCase
	getStudent     students.GetStudentUseCase
}

func NewCampaignsHandler(listCampaigns campaigns.ListCampaignsUseCase, createCampaign campaigns.CreateCampaignUseCase, getCampaign campaigns.GetCampaignUseCase, updateCampaign campaigns.UpdateCampaignUseCase, closeCampaign campaigns.CloseCampaignUseCase, deleteCampaign campaigns.DeleteCampaignUseCase, listGoals campaigns.ListCampaignGoalsUseCase, upsertGoal campaigns.UpsertCampaignGoalUseCase, deleteGoal campaigns.DeleteCampaignGoalUseCase, listProgress campaigns.ListCampaignProgressUseCase, recalculate campaigns.RecalculateCampaignProgressUseCase, getStudent students.GetStudentUseCase) CampaignsHandler {
	return CampaignsHandler{listCampaigns: listCampaigns, createCampaign: createCampaign, getCampaign: getCampaign, updateCampaign: updateCampaign, closeCampaign: closeCampaign, deleteCampaign: deleteCampaign, listGoals: listGoals, upsertGoal: upsertGoal, deleteGoal: deleteGoal, listProgress: listProgress, recalculate: recalculate, getStudent: getStudent}
}

func (h CampaignsHandler) List(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	result, err := h.listCampaigns.Execute(c.Request.Context(), boxID)
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

func (h CampaignsHandler) Create(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	request, ok := campaignRequest(c)
	if !ok {
		return
	}

	campaign, err := h.createCampaign.Execute(c.Request.Context(), campaigns.CreateCampaignInput{
		BoxID:       boxID,
		Name:        request.Name,
		Description: request.Description,
		StartDate:   request.startDate,
		EndDate:     request.endDate,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, campaignResponse(*campaign))
}

func (h CampaignsHandler) Get(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	campaign, err := h.getCampaign.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, campaignResponse(*campaign))
}

func (h CampaignsHandler) Update(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	request, ok := campaignRequest(c)
	if !ok {
		return
	}

	active := true
	if request.Active != nil {
		active = *request.Active
	}

	campaign, err := h.updateCampaign.Execute(c.Request.Context(), campaigns.UpdateCampaignInput{
		BoxID:       boxID,
		CampaignID:  domain.ID(c.Param("id")),
		Name:        request.Name,
		Description: request.Description,
		StartDate:   request.startDate,
		EndDate:     request.endDate,
		Active:      active,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, campaignResponse(*campaign))
}

func (h CampaignsHandler) Close(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	if err := h.closeCampaign.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id"))); err != nil {
		respondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h CampaignsHandler) Delete(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	if err := h.deleteCampaign.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id"))); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h CampaignsHandler) CreateGoal(c *gin.Context) {
	h.upsertGoalRequest(c, "")
}

func (h CampaignsHandler) UpdateGoal(c *gin.Context) {
	h.upsertGoalRequest(c, c.Param("goalId"))
}

func (h CampaignsHandler) DeleteGoal(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	if err := h.deleteGoal.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")), domain.ID(c.Param("goalId"))); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h CampaignsHandler) ListGoals(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	result, err := h.listGoals.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}

	response := make([]dto.CampaignGoalResponse, 0, len(result))
	for _, goal := range result {
		response = append(response, dto.CampaignGoalResponse{
			ID:             string(goal.ID),
			CampaignID:     string(goal.CampaignID),
			Source:         string(goal.Source),
			TargetCheckins: goal.TargetCheckins,
		})
	}
	c.JSON(http.StatusOK, response)
}

func (h CampaignsHandler) Progress(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	result, err := h.listProgress.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}

	response := make([]dto.CampaignProgressResponse, 0, len(result))
	for _, progress := range result {
		student, _ := h.getStudent.Execute(c.Request.Context(), boxID, progress.StudentID)
		response = append(response, campaignProgressResponse(progress, student))
	}
	c.JSON(http.StatusOK, response)
}

func (h CampaignsHandler) RecalculateProgress(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	if err := h.recalculate.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id"))); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
func (h CampaignsHandler) EligibleStudents(c *gin.Context) {
	h.progressFiltered(c, func(progress domain.CampaignProgress) bool { return progress.Achieved })
}
func (h CampaignsHandler) NearGoalStudents(c *gin.Context) {
	h.progressFiltered(c, func(progress domain.CampaignProgress) bool { return progress.NearGoal })
}

func campaignResponse(campaign domain.Campaign) dto.CampaignResponse {
	return dto.CampaignResponse{
		ID:          string(campaign.ID),
		Name:        campaign.Name,
		Description: campaign.Description,
		StartDate:   campaign.StartDate.Format("2006-01-02"),
		EndDate:     campaign.EndDate.Format("2006-01-02"),
		Active:      campaign.Active,
	}
}

type parsedCampaignRequest struct {
	dto.CampaignRequest
	startDate time.Time
	endDate   time.Time
}

func campaignRequest(c *gin.Context) (parsedCampaignRequest, bool) {
	var request dto.CampaignRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return parsedCampaignRequest{}, false
	}
	startDate, err := time.Parse("2006-01-02", request.StartDate)
	if err != nil {
		respondBadRequest(c)
		return parsedCampaignRequest{}, false
	}
	endDate, err := time.Parse("2006-01-02", request.EndDate)
	if err != nil {
		respondBadRequest(c)
		return parsedCampaignRequest{}, false
	}
	return parsedCampaignRequest{CampaignRequest: request, startDate: startDate, endDate: endDate}, true
}

func (h CampaignsHandler) upsertGoalRequest(c *gin.Context, goalID string) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.CampaignGoalRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}

	goal := domain.CampaignGoal{
		ID:             domain.ID(goalID),
		CampaignID:     domain.ID(c.Param("id")),
		Source:         domain.Source(request.Source),
		TargetCheckins: request.TargetCheckins,
	}
	if err := h.upsertGoal.Execute(c.Request.Context(), boxID, &goal); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CampaignGoalResponse{
		ID:             string(goal.ID),
		CampaignID:     string(goal.CampaignID),
		Source:         string(goal.Source),
		TargetCheckins: goal.TargetCheckins,
	})
}

func (h CampaignsHandler) progressFiltered(c *gin.Context, include func(domain.CampaignProgress) bool) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	result, err := h.listProgress.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}
	response := []dto.CampaignProgressResponse{}
	for _, progress := range result {
		if !include(progress) {
			continue
		}
		student, _ := h.getStudent.Execute(c.Request.Context(), boxID, progress.StudentID)
		response = append(response, campaignProgressResponse(progress, student))
	}
	c.JSON(http.StatusOK, response)
}

func campaignProgressResponse(progress domain.CampaignProgress, student *domain.Student) dto.CampaignProgressResponse {
	remaining := progress.TargetCheckins - progress.CurrentCheckins
	if remaining < 0 {
		remaining = 0
	}
	response := dto.CampaignProgressResponse{
		ID:                 string(progress.ID),
		CampaignID:         string(progress.CampaignID),
		StudentID:          string(progress.StudentID),
		CurrentCheckins:    progress.CurrentCheckins,
		TargetCheckins:     progress.TargetCheckins,
		RemainingCheckins:  remaining,
		ProgressPercentage: progress.ProgressPercentage,
		Achieved:           progress.Achieved,
		NearGoal:           progress.NearGoal,
	}
	if student != nil {
		response.StudentName = student.Name
		response.StudentEmail = student.Email
		response.StudentPhone = student.Phone
		response.StudentSource = string(student.Source)
	}
	return response
}
