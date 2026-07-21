package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

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
	exportData       students.ExportStudentDataUseCase
	updateContact    students.UpdateContactPreferenceUseCase
	anonymize        students.AnonymizeStudentUseCase
}

func NewStudentsHandler(listStudents students.ListStudentsUseCase, getStudent students.GetStudentUseCase, listCheckins students.ListStudentCheckinsUseCase, updateRiskStatus students.UpdateStudentRiskStatusUseCase, exportData students.ExportStudentDataUseCase, updateContact students.UpdateContactPreferenceUseCase, anonymize students.AnonymizeStudentUseCase) StudentsHandler {
	return StudentsHandler{listStudents: listStudents, getStudent: getStudent, listCheckins: listCheckins, updateRiskStatus: updateRiskStatus, exportData: exportData, updateContact: updateContact, anonymize: anonymize}
}

func (h StudentsHandler) ExportData(c *gin.Context) {
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
	result, err := h.exportData.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")), userID)
	if err != nil {
		respondError(c, err)
		return
	}
	response := dto.StudentPrivacyExportResponse{Student: studentResponse(result.Student), ExportedAt: result.ExportedAt.Format(time.RFC3339), Checkins: []dto.CheckinResponse{}, Progress: []dto.CampaignProgressResponse{}, Communications: []dto.PrivacyCommunicationResponse{}}
	for _, checkin := range result.Checkins {
		item := dto.CheckinResponse{ID: string(checkin.ID), StudentID: string(checkin.StudentID), CheckinDate: checkin.CheckinDate.Format("2006-01-02"), Source: string(checkin.Source)}
		if checkin.CheckinTime != nil {
			item.CheckinTime = checkin.CheckinTime.Format("15:04:05")
		}
		response.Checkins = append(response.Checkins, item)
	}
	for _, progress := range result.Progress {
		response.Progress = append(response.Progress, campaignProgressResponse(progress, &result.Student))
	}
	for _, communication := range result.Communications {
		item := dto.PrivacyCommunicationResponse{Channel: communication.Channel, CampaignID: string(communication.CampaignID), Destination: communication.Destination, Status: communication.Status, ErrorMessage: communication.ErrorMessage, CreatedAt: communication.CreatedAt.Format(time.RFC3339)}
		if communication.SentAt != nil {
			item.SentAt = communication.SentAt.Format(time.RFC3339)
		}
		response.Communications = append(response.Communications, item)
	}
	c.Header("Content-Disposition", `attachment; filename="student-data-export.json"`)
	c.JSON(http.StatusOK, response)
}

func (h StudentsHandler) UpdateContactPreference(c *gin.Context) {
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
	var request dto.UpdateContactPreferenceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	if err := h.updateContact.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")), userID, domain.ContactStatus(request.Status), request.Source); err != nil {
		if errors.Is(err, students.ErrInvalidPrivacyRequest) {
			respondBadRequest(c)
			return
		}
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h StudentsHandler) Anonymize(c *gin.Context) {
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
	var request dto.AnonymizeStudentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	if err := h.anonymize.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")), userID, request.Confirmed, request.Reason); err != nil {
		if errors.Is(err, students.ErrInvalidPrivacyRequest) {
			respondBadRequest(c)
			return
		}
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
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
		ID:                  string(student.ID),
		Name:                student.Name,
		Email:               student.Email,
		Phone:               student.Phone,
		Source:              string(student.Source),
		ExternalID:          student.ExternalID,
		RiskStatus:          string(riskStatus),
		ContactStatus:       string(student.ContactStatus),
		ContactStatusSource: student.ContactStatusSource,
	}
	if student.ContactStatusUpdatedAt != nil {
		response.ContactStatusUpdatedAt = student.ContactStatusUpdatedAt.Format(time.RFC3339)
	}
	if student.AnonymizedAt != nil {
		response.AnonymizedAt = student.AnonymizedAt.Format(time.RFC3339)
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
