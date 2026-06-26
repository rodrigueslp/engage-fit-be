package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/students"
	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

type StudentsHandler struct {
	listStudents     students.ListStudentsUseCase
	getStudent       students.GetStudentUseCase
	listCheckins     students.ListStudentCheckinsUseCase
	updateRiskStatus students.UpdateStudentRiskStatusUseCase
}

func NewStudentsHandler(listStudents students.ListStudentsUseCase, getStudent students.GetStudentUseCase, listCheckins students.ListStudentCheckinsUseCase, updateRiskStatus students.UpdateStudentRiskStatusUseCase) StudentsHandler {
	return StudentsHandler{listStudents: listStudents, getStudent: getStudent, listCheckins: listCheckins, updateRiskStatus: updateRiskStatus}
}

func (h StudentsHandler) List(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	filters := repositories.StudentFilters{
		Search: c.Query("search"),
		Page:   intQuery(c, "page"),
		Limit:  intQuery(c, "limit"),
	}
	if source := c.Query("source"); source != "" {
		value := domain.Source(source)
		filters.Source = &value
	}
	if campaignID := c.Query("campaign_id"); campaignID != "" {
		value := domain.ID(campaignID)
		filters.CampaignID = &value
	}
	if achieved := boolQuery(c, "achieved"); achieved != nil {
		filters.Achieved = achieved
	}
	if nearGoal := boolQuery(c, "near_goal"); nearGoal != nil {
		filters.NearGoal = nearGoal
	}

	result, err := h.listStudents.Execute(c.Request.Context(), boxID, filters)
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

func (h StudentsHandler) Get(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	student, err := h.getStudent.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, studentResponse(*student))
}

func (h StudentsHandler) Checkins(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	result, err := h.listCheckins.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}

	response := make([]dto.CheckinResponse, 0, len(result))
	for _, checkin := range result {
		item := dto.CheckinResponse{
			ID:          string(checkin.ID),
			StudentID:   string(checkin.StudentID),
			CheckinDate: checkin.CheckinDate.Format("2006-01-02"),
			Source:      string(checkin.Source),
		}
		if checkin.CheckinTime != nil {
			item.CheckinTime = checkin.CheckinTime.Format("15:04:05")
		}
		response = append(response, item)
	}
	c.JSON(http.StatusOK, response)
}

func (h StudentsHandler) UpdateRiskStatus(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	var request dto.UpdateStudentRiskStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}

	if err := h.updateRiskStatus.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")), domain.StudentRiskStatus(request.RiskStatus)); err != nil {
		respondBadRequest(c)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h StudentsHandler) CampaignProgress(c *gin.Context) {
	notImplemented(c, "students.campaign_progress")
}

func studentResponse(student domain.Student) dto.StudentResponse {
	riskStatus := student.RiskStatus
	if riskStatus == "" {
		riskStatus = domain.StudentRiskStatusActive
	}
	response := dto.StudentResponse{
		ID:         string(student.ID),
		Name:       student.Name,
		Email:      student.Email,
		Phone:      student.Phone,
		Source:     string(student.Source),
		ExternalID: student.ExternalID,
		RiskStatus: string(riskStatus),
	}
	if student.RiskLastMessageAt != nil {
		response.RiskLastMessageAt = student.RiskLastMessageAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return response
}

func intQuery(c *gin.Context, key string) int {
	value, _ := strconv.Atoi(c.Query(key))
	return value
}

func boolQuery(c *gin.Context, key string) *bool {
	value := c.Query(key)
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil
	}
	return &parsed
}
