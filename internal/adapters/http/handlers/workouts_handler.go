package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/workouts"
	"boxengage/backend/internal/domain"
)

type WorkoutsHandler struct {
	listWorkouts   workouts.ListWorkoutsUseCase
	createWorkout  workouts.CreateWorkoutUseCase
	getWorkout     workouts.GetWorkoutUseCase
	updateWorkout  workouts.UpdateWorkoutUseCase
	deleteWorkout  workouts.DeleteWorkoutUseCase
	listDrafts     workouts.ListWorkoutDraftsUseCase
	generateDraft  workouts.GenerateWorkoutDraftUseCase
	getDraft       workouts.GetWorkoutDraftUseCase
	updateDraft    workouts.UpdateWorkoutDraftUseCase
	approveDraft   workouts.ApproveWorkoutDraftUseCase
	sendDraft      workouts.SendWorkoutDraftUseCase
	listRecipients workouts.ListWorkoutRecipientsUseCase
}

func NewWorkoutsHandler(listWorkouts workouts.ListWorkoutsUseCase, createWorkout workouts.CreateWorkoutUseCase, getWorkout workouts.GetWorkoutUseCase, updateWorkout workouts.UpdateWorkoutUseCase, deleteWorkout workouts.DeleteWorkoutUseCase, listDrafts workouts.ListWorkoutDraftsUseCase, generateDraft workouts.GenerateWorkoutDraftUseCase, getDraft workouts.GetWorkoutDraftUseCase, updateDraft workouts.UpdateWorkoutDraftUseCase, approveDraft workouts.ApproveWorkoutDraftUseCase, sendDraft workouts.SendWorkoutDraftUseCase, listRecipients workouts.ListWorkoutRecipientsUseCase) WorkoutsHandler {
	return WorkoutsHandler{listWorkouts: listWorkouts, createWorkout: createWorkout, getWorkout: getWorkout, updateWorkout: updateWorkout, deleteWorkout: deleteWorkout, listDrafts: listDrafts, generateDraft: generateDraft, getDraft: getDraft, updateDraft: updateDraft, approveDraft: approveDraft, sendDraft: sendDraft, listRecipients: listRecipients}
}

func (h WorkoutsHandler) List(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	result, err := h.listWorkouts.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}
	response := make([]dto.WorkoutResponse, 0, len(result))
	for _, workout := range result {
		response = append(response, workoutResponse(workout))
	}
	c.JSON(http.StatusOK, response)
}

func (h WorkoutsHandler) Create(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.WorkoutRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	workoutDate, err := time.Parse("2006-01-02", request.WorkoutDate)
	if err != nil {
		respondBadRequest(c)
		return
	}
	status := domain.WorkoutStatus(request.Status)
	if status == "" {
		status = domain.WorkoutStatusDraft
	}
	workout := domain.Workout{BoxID: boxID, WorkoutDate: workoutDate, Title: request.Title, Goal: request.Goal, Movements: request.Movements, CoachNotes: request.CoachNotes, Status: status}
	if err := h.createWorkout.Execute(c.Request.Context(), &workout); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, workoutResponse(workout))
}

func (h WorkoutsHandler) Get(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	workout, err := h.getWorkout.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, workoutResponse(*workout))
}

func (h WorkoutsHandler) Update(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	workout, err := h.getWorkout.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}
	var request dto.WorkoutRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	workoutDate, err := time.Parse("2006-01-02", request.WorkoutDate)
	if err != nil {
		respondBadRequest(c)
		return
	}
	workout.WorkoutDate = workoutDate
	workout.Title = request.Title
	workout.Goal = request.Goal
	workout.Movements = request.Movements
	workout.CoachNotes = request.CoachNotes
	if request.Status != "" {
		workout.Status = domain.WorkoutStatus(request.Status)
	}
	if err := h.updateWorkout.Execute(c.Request.Context(), *workout); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, workoutResponse(*workout))
}

func (h WorkoutsHandler) Delete(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	if err := h.deleteWorkout.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id"))); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h WorkoutsHandler) ListDrafts(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	result, err := h.listDrafts.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}
	response := make([]dto.WorkoutDraftResponse, 0, len(result))
	for _, draft := range result {
		response = append(response, workoutDraftResponse(draft))
	}
	c.JSON(http.StatusOK, response)
}

func (h WorkoutsHandler) GenerateDraft(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.GenerateWorkoutDraftRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	studentIDs := make([]domain.ID, 0, len(request.StudentIDs))
	for _, studentID := range request.StudentIDs {
		studentIDs = append(studentIDs, domain.ID(studentID))
	}
	draft, err := h.generateDraft.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")), domain.MessageAudience(request.Audience), domain.ID(request.CampaignID), studentIDs)
	if err != nil {
		log.Printf("workout draft generation failed: box_id=%s workout_id=%s error=%v", boxID, c.Param("id"), err)
		c.JSON(http.StatusBadGateway, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, workoutDraftResponse(*draft))
}

func (h WorkoutsHandler) UpdateDraft(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.UpdateWorkoutDraftRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	draft, err := h.updateDraft.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")), request.ApprovedBody)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, workoutDraftResponse(*draft))
}

func (h WorkoutsHandler) ApproveDraft(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.UpdateWorkoutDraftRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		request.ApprovedBody = ""
	}
	draft, err := h.approveDraft.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")), request.ApprovedBody)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, workoutDraftResponse(*draft))
}

func (h WorkoutsHandler) SendDraft(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	output, err := h.sendDraft.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		log.Printf("workout draft send failed: box_id=%s draft_id=%s error=%v", boxID, c.Param("id"), err)
		c.JSON(http.StatusBadGateway, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.SendWorkoutDraftResponse{Total: output.Total, Sent: output.Sent, Failed: output.Failed})
}

func (h WorkoutsHandler) ListRecipients(c *gin.Context) {
	result, err := h.listRecipients.Execute(c.Request.Context(), domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}
	response := make([]dto.WorkoutRecipientResponse, 0, len(result))
	for _, recipient := range result {
		item := dto.WorkoutRecipientResponse{ID: string(recipient.ID), WorkoutMessageDraftID: string(recipient.WorkoutMessageDraftID), StudentID: string(recipient.StudentID), Phone: recipient.Phone, Status: string(recipient.Status), ErrorMessage: recipient.ErrorMessage, CreatedAt: recipient.CreatedAt.Format("2006-01-02T15:04:05Z07:00")}
		if recipient.SentAt != nil {
			item.SentAt = recipient.SentAt.Format("2006-01-02T15:04:05Z07:00")
		}
		response = append(response, item)
	}
	c.JSON(http.StatusOK, response)
}

func workoutResponse(workout domain.Workout) dto.WorkoutResponse {
	return dto.WorkoutResponse{ID: string(workout.ID), WorkoutDate: workout.WorkoutDate.Format("2006-01-02"), Title: workout.Title, Goal: workout.Goal, Movements: workout.Movements, CoachNotes: workout.CoachNotes, Status: string(workout.Status), CreatedAt: workout.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), UpdatedAt: workout.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")}
}

func workoutDraftResponse(draft domain.WorkoutMessageDraft) dto.WorkoutDraftResponse {
	response := dto.WorkoutDraftResponse{ID: string(draft.ID), WorkoutID: string(draft.WorkoutID), CampaignID: string(draft.CampaignID), Audience: string(draft.Audience), GeneratedBody: draft.GeneratedBody, ApprovedBody: draft.ApprovedBody, Status: string(draft.Status), TotalRecipients: draft.TotalRecipients, SentRecipients: draft.SentRecipients, FailedRecipients: draft.FailedRecipients, GeneratedAt: draft.GeneratedAt.Format("2006-01-02T15:04:05Z07:00")}
	if draft.ApprovedAt != nil {
		response.ApprovedAt = draft.ApprovedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if draft.SentAt != nil {
		response.SentAt = draft.SentAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return response
}
