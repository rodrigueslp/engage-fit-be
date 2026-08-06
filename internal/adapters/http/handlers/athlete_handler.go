package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/athlete"
	"boxengage/backend/internal/domain"
)

type AthleteHandler struct {
	service *athlete.Service
	session middleware.SessionConfig
}

func NewAthleteHandler(service *athlete.Service, session middleware.SessionConfig) AthleteHandler {
	return AthleteHandler{service: service, session: session}
}

func (h AthleteHandler) CreateInvitation(c *gin.Context) {
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
	output, err := h.service.CreateInvitation(c.Request.Context(), boxID, domain.ID(c.Param("id")), userID)
	if errors.Is(err, athlete.ErrInvalidInput) {
		respondBadRequest(c)
		return
	}
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.CreateAthleteInvitationResponse{Token: output.Token, ExpiresAt: output.ExpiresAt.Format(time.RFC3339)})
}

func (h AthleteHandler) PreviewInvitation(c *gin.Context) {
	invitation, err := h.service.PreviewInvitation(c.Request.Context(), c.Param("token"))
	if err != nil {
		respondPublicError(c, http.StatusNotFound, "athlete_invitation_unavailable", "convite inválido ou expirado")
		return
	}
	c.JSON(http.StatusOK, dto.AthleteInvitationResponse{BoxName: invitation.BoxName, StudentName: invitation.StudentName, ExpiresAt: invitation.ExpiresAt.Format(time.RFC3339)})
}

func (h AthleteHandler) ClaimInvitation(c *gin.Context) {
	var request dto.ClaimAthleteInvitationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	output, err := h.service.ClaimInvitation(c.Request.Context(), athlete.ClaimInput{Token: c.Param("token"), Name: request.Name, Email: request.Email, Password: request.Password})
	switch {
	case errors.Is(err, athlete.ErrInvalidInput):
		respondPublicError(c, http.StatusBadRequest, "athlete_input_invalid", "preencha nome, e-mail válido e uma senha de ao menos 12 caracteres")
		return
	case errors.Is(err, athlete.ErrInvalidCredentials):
		respondPublicError(c, http.StatusUnauthorized, "athlete_credentials_invalid", "este e-mail já possui conta; informe a senha correta")
		return
	case errors.Is(err, athlete.ErrInvitationExpired):
		respondPublicError(c, http.StatusNotFound, "athlete_invitation_unavailable", "convite inválido ou expirado")
		return
	case err != nil:
		respondError(c, err)
		return
	}
	if err := middleware.SetSession(c, h.session, output.Token); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, athleteMeResponse(output.Context))
}

func (h AthleteHandler) Login(c *gin.Context) {
	var request dto.AthleteLoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	output, err := h.service.Login(c.Request.Context(), athlete.LoginInput{Email: request.Email, Password: request.Password})
	if err != nil {
		respondPublicError(c, http.StatusUnauthorized, "athlete_credentials_invalid", "e-mail ou senha inválidos")
		return
	}
	if err := middleware.SetSession(c, h.session, output.Token); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, athleteMeResponse(output.Context))
}

func (h AthleteHandler) Me(c *gin.Context) {
	athleteContext, err := middleware.AthleteContext(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	c.JSON(http.StatusOK, athleteMeResponse(athleteContext))
}

func (h AthleteHandler) Workouts(c *gin.Context) {
	athleteID, err := middleware.AthleteID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	items, err := h.service.Workouts(c.Request.Context(), athleteID)
	if err != nil {
		respondError(c, err)
		return
	}
	response := make([]dto.AthleteWorkoutResponse, 0, len(items))
	for _, item := range items {
		response = append(response, dto.AthleteWorkoutResponse{WorkoutResponse: workoutResponse(item.Workout), BoxName: item.BoxName, MembershipID: string(item.MembershipID), Result: item.Result, Personalization: item.Personalization})
	}
	c.JSON(http.StatusOK, response)
}

func (h AthleteHandler) SaveResult(c *gin.Context) {
	athleteID, err := middleware.AthleteID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.SaveAthleteWorkoutResultRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	performedAt := time.Time{}
	if request.PerformedAt != "" {
		performedAt, err = time.Parse(time.RFC3339, request.PerformedAt)
		if err != nil {
			respondBadRequest(c)
			return
		}
	}
	entries := make([]domain.AthleteResultEntry, 0, len(request.Entries))
	for _, item := range request.Entries {
		entries = append(entries, domain.AthleteResultEntry{SectionIndex: item.SectionIndex, SectionType: item.SectionType, Movement: item.Movement, ScoreType: item.ScoreType, TimeSeconds: item.TimeSeconds, Rounds: item.Rounds, Repetitions: item.Repetitions, LoadKG: item.LoadKG, DistanceM: item.DistanceMeters, Calories: item.Calories, Completed: item.Completed})
	}
	output, err := h.service.SaveResult(c.Request.Context(), athleteID, athlete.ResultInput{WorkoutID: domain.ID(c.Param("id")), Scale: request.Scale, Entries: entries, RPE: request.RPE, Notes: request.Notes, PerformedAt: performedAt})
	if errors.Is(err, athlete.ErrInvalidInput) {
		respondPublicError(c, http.StatusBadRequest, "athlete_result_invalid", "revise a escala e os valores informados")
		return
	}
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": output.Result, "possible_records": output.PossibleRecords})
}

func (h AthleteHandler) Results(c *gin.Context) {
	athleteID, err := middleware.AthleteID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	items, err := h.service.Results(c.Request.Context(), athleteID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}
func (h AthleteHandler) PersonalRecords(c *gin.Context) {
	athleteID, err := middleware.AthleteID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	items, err := h.service.PersonalRecords(c.Request.Context(), athleteID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}
func (h AthleteHandler) ExplainWorkout(c *gin.Context) {
	athleteID, err := middleware.AthleteID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	item, err := h.service.ExplainWorkout(c.Request.Context(), athleteID, domain.ID(c.Param("id")))
	if errors.Is(err, athlete.ErrInvalidInput) {
		respondPublicError(c, http.StatusNotFound, "athlete_workout_not_found", "treino não encontrado")
		return
	}
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}
func (h AthleteHandler) ConfirmPersonalRecord(c *gin.Context) {
	athleteID, err := middleware.AthleteID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	if err := h.service.ConfirmPersonalRecord(c.Request.Context(), athleteID, domain.ID(c.Param("id"))); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h AthleteHandler) RequestPasswordReset(c *gin.Context) {
	var request dto.AthletePasswordResetRequest
	if c.ShouldBindJSON(&request) != nil {
		respondBadRequest(c)
		return
	}
	if err := h.service.RequestPasswordReset(c.Request.Context(), request.Email); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
func (h AthleteHandler) ResetPassword(c *gin.Context) {
	var request dto.AthletePasswordResetConfirmRequest
	if c.ShouldBindJSON(&request) != nil {
		respondBadRequest(c)
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), c.Param("token"), request.Password); err != nil {
		respondPublicError(c, http.StatusBadRequest, "athlete_reset_invalid", "link inválido ou expirado, ou senha fora dos requisitos")
		return
	}
	c.Status(http.StatusNoContent)
}
func (h AthleteHandler) RequestEmailVerification(c *gin.Context) {
	athleteID, err := middleware.AthleteID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	if err := h.service.RequestEmailVerification(c.Request.Context(), athleteID); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
func (h AthleteHandler) VerifyEmail(c *gin.Context) {
	if err := h.service.VerifyEmail(c.Request.Context(), c.Param("token")); err != nil {
		respondPublicError(c, http.StatusBadRequest, "athlete_verification_invalid", "link inválido ou expirado")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h AthleteHandler) Logout(c *gin.Context) {
	token, err := c.Cookie(h.session.CookieName)
	if err == nil && token != "" {
		_ = h.service.Logout(c.Request.Context(), token)
	}
	middleware.ClearSession(c, h.session)
	c.Status(http.StatusNoContent)
}

func athleteMeResponse(athleteContext domain.AthleteContext) dto.AthleteMeResponse {
	memberships := make([]dto.AthleteMembershipResponse, 0, len(athleteContext.Memberships))
	for _, membership := range athleteContext.Memberships {
		memberships = append(memberships, dto.AthleteMembershipResponse{ID: string(membership.ID), BoxID: string(membership.BoxID), BoxName: membership.BoxName, JoinedAt: membership.JoinedAt.Format(time.RFC3339)})
	}
	return dto.AthleteMeResponse{ID: string(athleteContext.Account.ID), Name: athleteContext.Account.Name, Email: athleteContext.Account.Email, Memberships: memberships, EmailVerified: athleteContext.Account.EmailVerifiedAt != nil}
}
