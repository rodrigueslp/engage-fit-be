package handlers

import (
	"errors"
	"net/http"
	"time"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/attendance"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AttendanceHandler struct {
	service *attendance.Service
	access  *attendance.AccessService
}

func NewAttendanceHandler(service *attendance.Service) AttendanceHandler {
	return AttendanceHandler{service: service}
}

func NewAttendanceAccessHandler(service *attendance.AccessService) AttendanceHandler {
	return AttendanceHandler{access: service}
}

func (h AttendanceHandler) Admit(c *gin.Context) {
	userID, err := middleware.UserID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request attendance.AdmitInput
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	result, err := h.access.Admit(c.Request.Context(), userID, domain.ID(c.Param("unit_id")), c.GetHeader("Idempotency-Key"), request)
	if err != nil {
		respondAttendanceError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h AttendanceHandler) Today(c *gin.Context) {
	userID, err := middleware.UserID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	result, err := h.access.ListToday(c.Request.Context(), userID, domain.ID(c.Param("unit_id")))
	if err != nil {
		respondAttendanceError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h AttendanceHandler) CreateSession(c *gin.Context) {
	boxID, boxErr := middleware.BoxID(c)
	userID, userErr := middleware.UserID(c)
	if boxErr != nil || userErr != nil {
		respondUnauthorized(c)
		return
	}
	result, err := h.service.CreateSession(c.Request.Context(), boxID, userID)
	if err != nil {
		respondAttendanceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.SelfCheckinSessionResponse{Token: result.Token, ExpiresAt: result.ExpiresAt})
}

func (h AttendanceHandler) PublicSession(c *gin.Context) {
	result, err := h.service.PublicSession(c.Request.Context(), c.Param("token"))
	if err != nil {
		respondAttendanceError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SelfCheckinSessionResponse{BoxName: result.BoxName, ExpiresAt: result.ExpiresAt})
}

func (h AttendanceHandler) ManualCheckin(c *gin.Context) {
	boxID, boxErr := middleware.BoxID(c)
	userID, userErr := middleware.UserID(c)
	if boxErr != nil || userErr != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.ManualCheckinRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	date, err := time.Parse("2006-01-02", request.Date)
	if err != nil {
		respondBadRequest(c)
		return
	}
	result, err := h.service.ManualCheckin(c.Request.Context(), attendance.ManualCheckinInput{
		BoxID: boxID, StudentID: domain.ID(c.Param("id")), UserID: userID, Date: date,
	})
	if err != nil {
		respondAttendanceError(c, err)
		return
	}
	c.JSON(http.StatusOK, attendanceResponse(result))
}

func (h AttendanceHandler) SelfCheckin(c *gin.Context) {
	var request dto.SelfCheckinRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	result, err := h.service.SelfCheckin(c.Request.Context(), attendance.SelfCheckinInput{
		Token: c.Param("token"), Name: request.Name, Phone: request.Phone,
	})
	if err != nil {
		respondAttendanceError(c, err)
		return
	}
	c.JSON(http.StatusOK, attendanceResponse(result))
}

func attendanceResponse(result *attendance.CheckinResult) dto.AttendanceCheckinResponse {
	return dto.AttendanceCheckinResponse{
		StudentID: string(result.StudentID), StudentName: result.StudentName,
		CheckinDate: result.CheckinDate.Format("2006-01-02"), AlreadyRecorded: result.AlreadyRecorded,
	}
}

func respondAttendanceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, attendance.ErrInvalidInput):
		respondPublicError(c, http.StatusBadRequest, "invalid_checkin", "Confira os dados informados e tente novamente.")
	case errors.Is(err, attendance.ErrIdempotencyReused):
		respondPublicError(c, http.StatusConflict, "idempotency_key_reused", "Use uma nova chave para uma entrada diferente.")
	case errors.Is(err, attendance.ErrAccessUnavailable):
		respondPublicError(c, http.StatusNotFound, "attendance_unavailable", "Operação de presença indisponível.")
	case errors.Is(err, attendance.ErrSessionUnavailable):
		respondPublicError(c, http.StatusGone, "checkin_session_unavailable", "Este QR Code expirou. Peça à academia para exibir um novo código.")
	case errors.Is(err, attendance.ErrStudentNotFound), errors.Is(err, gorm.ErrRecordNotFound):
		respondPublicError(c, http.StatusNotFound, "box_member_not_found", "Não encontramos um mensalista ativo com esses dados.")
	default:
		respondError(c, err)
	}
}
