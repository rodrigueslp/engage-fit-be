package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/automation"
	"boxengage/backend/internal/domain"
)

type AutomationHandler struct {
	listRuns        automation.ListRunsUseCase
	getRun          automation.GetRunUseCase
	createRun       automation.CreateRunUseCase
	updateRun       automation.UpdateRunUseCase
	listSchedules   automation.ListSchedulesUseCase
	getSchedule     automation.GetScheduleUseCase
	createSchedule  automation.CreateScheduleUseCase
	updateSchedule  automation.UpdateScheduleUseCase
	deleteSchedule  automation.DeleteScheduleUseCase
	executeSchedule automation.ExecuteScheduleUseCase
}

func NewAutomationHandler(listRuns automation.ListRunsUseCase, getRun automation.GetRunUseCase, createRun automation.CreateRunUseCase, updateRun automation.UpdateRunUseCase, listSchedules automation.ListSchedulesUseCase, getSchedule automation.GetScheduleUseCase, createSchedule automation.CreateScheduleUseCase, updateSchedule automation.UpdateScheduleUseCase, deleteSchedule automation.DeleteScheduleUseCase, executeSchedule automation.ExecuteScheduleUseCase) AutomationHandler {
	return AutomationHandler{listRuns: listRuns, getRun: getRun, createRun: createRun, updateRun: updateRun, listSchedules: listSchedules, getSchedule: getSchedule, createSchedule: createSchedule, updateSchedule: updateSchedule, deleteSchedule: deleteSchedule, executeSchedule: executeSchedule}
}

func (h AutomationHandler) ListRuns(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	runs, err := h.listRuns.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}
	response := make([]dto.AutomationRunResponse, 0, len(runs))
	for _, run := range runs {
		response = append(response, automationRunResponse(run))
	}
	c.JSON(http.StatusOK, response)
}

func (h AutomationHandler) GetRun(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	run, err := h.getRun.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, automationRunResponse(*run))
}

func (h AutomationHandler) CreateRun(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.AutomationRunRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	run := domain.AutomationRun{BoxID: boxID, ExecutionKey: c.GetHeader("Idempotency-Key"), Status: "running", Source: request.Source, Filename: request.Filename, Imported: request.Imported, RecalculatedCampaigns: request.RecalculatedCampaigns, SkippedMessageCampaigns: request.SkippedMessageCampaigns, SentMessages: request.SentMessages, FailedMessages: request.FailedMessages, ErrorMessage: request.ErrorMessage, StartedAt: time.Now()}
	saved, existing, err := h.createRun.Execute(c.Request.Context(), &run)
	if err != nil {
		if errors.Is(err, automation.ErrInvalidIdempotencyKey) {
			respondBadRequest(c)
			return
		}
		respondError(c, err)
		return
	}
	status := http.StatusCreated
	if existing {
		status = http.StatusOK
	}
	response := automationRunResponse(*saved)
	response.IdempotentReplay = existing
	c.JSON(status, response)
}

func (h AutomationHandler) UpdateRun(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	run, err := h.getRun.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}
	var request dto.AutomationRunUpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	run.Status = request.Status
	if run.Status == "" {
		run.Status = "success"
	}
	run.Imported = request.Imported
	run.RecalculatedCampaigns = request.RecalculatedCampaigns
	run.SkippedMessageCampaigns = request.SkippedMessageCampaigns
	run.SentMessages = request.SentMessages
	run.FailedMessages = request.FailedMessages
	run.ErrorMessage = request.ErrorMessage
	finishedAt := time.Now()
	run.FinishedAt = &finishedAt
	if err := h.updateRun.Execute(c.Request.Context(), *run); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, automationRunResponse(*run))
}

func (h AutomationHandler) ListSchedules(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	schedules, err := h.listSchedules.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}
	response := make([]dto.AutomationScheduleResponse, 0, len(schedules))
	for _, schedule := range schedules {
		response = append(response, automationScheduleResponse(schedule))
	}
	c.JSON(http.StatusOK, response)
}

func (h AutomationHandler) CreateSchedule(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.AutomationScheduleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	now := time.Now()
	schedule := domain.AutomationSchedule{BoxID: boxID, Name: request.Name, Mode: request.Mode, RunTime: request.RunTime, Timezone: request.Timezone, DaysOfWeek: request.DaysOfWeek, AllowResend: request.AllowResend, Enabled: request.Enabled, CreatedAt: now, UpdatedAt: now}
	if err := h.createSchedule.Execute(c.Request.Context(), &schedule); err != nil {
		if errors.Is(err, automation.ErrInvalidSchedule) {
			respondBadRequest(c)
			return
		}
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, automationScheduleResponse(schedule))
}

func (h AutomationHandler) UpdateSchedule(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	schedule, err := h.getSchedule.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}
	var request dto.AutomationScheduleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	schedule.Name = request.Name
	schedule.Mode = request.Mode
	schedule.RunTime = request.RunTime
	schedule.Timezone = request.Timezone
	schedule.DaysOfWeek = request.DaysOfWeek
	schedule.AllowResend = request.AllowResend
	schedule.Enabled = request.Enabled
	schedule.UpdatedAt = time.Now()
	if err := h.updateSchedule.Execute(c.Request.Context(), *schedule); err != nil {
		if errors.Is(err, automation.ErrInvalidSchedule) {
			respondBadRequest(c)
			return
		}
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, automationScheduleResponse(*schedule))
}

func (h AutomationHandler) DeleteSchedule(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	if err := h.deleteSchedule.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id"))); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h AutomationHandler) RunScheduleNow(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	run, err := h.executeSchedule.ExecuteWithKey(c.Request.Context(), boxID, domain.ID(c.Param("id")), c.GetHeader("Idempotency-Key"))
	if err != nil {
		if errors.Is(err, automation.ErrInvalidIdempotencyKey) {
			respondBadRequest(c)
			return
		}
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, automationRunResponse(*run))
}

func automationRunResponse(run domain.AutomationRun) dto.AutomationRunResponse {
	response := dto.AutomationRunResponse{ID: string(run.ID), Status: run.Status, Source: run.Source, Filename: run.Filename, Imported: run.Imported, RecalculatedCampaigns: run.RecalculatedCampaigns, SkippedMessageCampaigns: run.SkippedMessageCampaigns, SentMessages: run.SentMessages, FailedMessages: run.FailedMessages, ErrorMessage: run.ErrorMessage, StartedAt: run.StartedAt.Format("2006-01-02T15:04:05Z07:00")}
	if run.FinishedAt != nil {
		response.FinishedAt = run.FinishedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return response
}

func automationScheduleResponse(schedule domain.AutomationSchedule) dto.AutomationScheduleResponse {
	response := dto.AutomationScheduleResponse{ID: string(schedule.ID), Name: schedule.Name, Mode: schedule.Mode, RunTime: schedule.RunTime, Timezone: schedule.Timezone, DaysOfWeek: schedule.DaysOfWeek, AllowResend: schedule.AllowResend, Enabled: schedule.Enabled, CreatedAt: schedule.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), UpdatedAt: schedule.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")}
	if schedule.LastRunAt != nil {
		response.LastRunAt = schedule.LastRunAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return response
}
