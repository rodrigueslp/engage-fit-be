package handlers

import (
	"net/http"
	"time"

	"boxengage/backend/internal/app/scheduling"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type SchedulingGridHandler struct{ service *scheduling.GridService }

func NewSchedulingGridHandler(service *scheduling.GridService) SchedulingGridHandler {
	return SchedulingGridHandler{service: service}
}

type scheduleTargetRequest struct {
	Kind domain.ScheduleTargetKind `json:"kind"`
	ID   domain.ID                 `json:"id"`
}
type seriesRequest struct {
	Target           scheduleTargetRequest       `json:"target"`
	SpaceID          *domain.ID                  `json:"space_id"`
	ProfessionalID   domain.ID                   `json:"professional_id"`
	DeliveryMode     domain.ScheduleDeliveryMode `json:"delivery_mode"`
	Capacity         int                         `json:"capacity"`
	DurationMinutes  int                         `json:"duration_minutes"`
	Weekdays         []int                       `json:"weekdays"`
	LocalStartTime   string                      `json:"local_start_time"`
	Timezone         string                      `json:"timezone"`
	EffectiveFrom    string                      `json:"effective_from"`
	EffectiveUntil   *string                     `json:"effective_until"`
	ExpectedRevision int                         `json:"expected_revision"`
	SeriesID         *domain.ID                  `json:"series_id"`
	PreviewFrom      string                      `json:"preview_from"`
	PreviewUntil     string                      `json:"preview_until"`
	EffectiveOn      string                      `json:"effective_on"`
}

func (r seriesRequest) input() scheduling.SeriesMutationInput {
	return scheduling.SeriesMutationInput{TargetKind: r.Target.Kind, TargetID: r.Target.ID, ProfessionalID: r.ProfessionalID, SpaceID: r.SpaceID, DeliveryMode: r.DeliveryMode, Capacity: r.Capacity, DurationMinutes: r.DurationMinutes, Weekdays: r.Weekdays, LocalStartTime: r.LocalStartTime, Timezone: r.Timezone, EffectiveFrom: r.EffectiveFrom, EffectiveUntil: r.EffectiveUntil, ExpectedRevision: r.ExpectedRevision}
}

type revisionReasonRequest struct {
	ExpectedRevision int    `json:"expected_revision"`
	Reason           string `json:"reason"`
	EffectiveOn      string `json:"effective_on"`
}
type sessionRequest struct {
	Target           scheduleTargetRequest       `json:"target"`
	SpaceID          *domain.ID                  `json:"space_id"`
	ProfessionalID   domain.ID                   `json:"professional_id"`
	DeliveryMode     domain.ScheduleDeliveryMode `json:"delivery_mode"`
	Capacity         int                         `json:"capacity"`
	StartsAt         time.Time                   `json:"starts_at"`
	DurationMinutes  int                         `json:"duration_minutes"`
	Timezone         string                      `json:"timezone"`
	ExpectedRevision int                         `json:"expected_revision"`
}

func (r sessionRequest) input() scheduling.SessionMutationInput {
	return scheduling.SessionMutationInput{TargetKind: r.Target.Kind, TargetID: r.Target.ID, ProfessionalID: r.ProfessionalID, SpaceID: r.SpaceID, DeliveryMode: r.DeliveryMode, Capacity: r.Capacity, StartsAt: r.StartsAt, DurationMinutes: r.DurationMinutes, Timezone: r.Timezone, ExpectedRevision: r.ExpectedRevision}
}

func (h SchedulingGridHandler) ListSeries(c *gin.Context) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	list, err := h.service.ListSeries(c.Request.Context(), actor, unit, c.Query("include_archived") == "true")
	if err != nil {
		respondSchedulingError(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}
func (h SchedulingGridHandler) GetSeries(c *gin.Context) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	value, err := h.service.GetSeries(c.Request.Context(), actor, unit, domain.ID(c.Param("series_id")))
	if err != nil {
		respondSchedulingError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}
func (h SchedulingGridHandler) CreateSeries(c *gin.Context) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	var req seriesRequest
	if c.ShouldBindJSON(&req) != nil {
		respondSchedulingError(c, scheduling.ErrScheduleSeriesInvalid)
		return
	}
	value, fresh, err := h.service.CreateSeries(c.Request.Context(), actor, unit, c.GetHeader("Idempotency-Key"), req.input())
	if err != nil {
		respondSchedulingError(c, err)
		return
	}
	if !fresh {
		c.Header("Idempotency-Replayed", "true")
		c.JSON(http.StatusOK, value)
		return
	}
	c.JSON(http.StatusCreated, value)
}
func (h SchedulingGridHandler) UpdateSeries(c *gin.Context) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	var req seriesRequest
	if c.ShouldBindJSON(&req) != nil {
		respondSchedulingError(c, scheduling.ErrScheduleSeriesInvalid)
		return
	}
	value, fresh, err := h.service.UpdateSeries(c.Request.Context(), actor, unit, domain.ID(c.Param("series_id")), c.GetHeader("Idempotency-Key"), req.input())
	if err != nil {
		respondSchedulingError(c, err)
		return
	}
	if !fresh {
		c.Header("Idempotency-Replayed", "true")
	}
	c.JSON(http.StatusOK, value)
}
func (h SchedulingGridHandler) PreviewCandidate(c *gin.Context) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	var req seriesRequest
	if c.ShouldBindJSON(&req) != nil {
		respondSchedulingError(c, scheduling.ErrScheduleSeriesInvalid)
		return
	}
	value, err := h.service.PreviewCandidate(c.Request.Context(), actor, unit, scheduling.PreviewInput{SeriesMutationInput: req.input(), SeriesID: req.SeriesID, PreviewFrom: req.PreviewFrom, PreviewUntil: req.PreviewUntil})
	if err != nil {
		respondSchedulingError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}
func (h SchedulingGridHandler) PreviewGrid(c *gin.Context) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	value, err := h.service.PreviewGrid(c.Request.Context(), actor, unit, c.Query("from"), c.Query("until"))
	if err != nil {
		respondSchedulingError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}
func (h SchedulingGridHandler) ArchiveSeries(c *gin.Context)    { h.lifecycle(c, true) }
func (h SchedulingGridHandler) ReactivateSeries(c *gin.Context) { h.lifecycle(c, false) }
func (h SchedulingGridHandler) lifecycle(c *gin.Context, archive bool) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	var req revisionReasonRequest
	if c.ShouldBindJSON(&req) != nil {
		respondSchedulingError(c, scheduling.ErrScheduleSeriesInvalid)
		return
	}
	input := scheduling.RevisionReasonInput{ExpectedRevision: req.ExpectedRevision, Reason: req.Reason}
	var value *domain.ScheduleSeries
	var fresh bool
	var err error
	if archive {
		value, fresh, err = h.service.ArchiveSeries(c.Request.Context(), actor, unit, domain.ID(c.Param("series_id")), c.GetHeader("Idempotency-Key"), input)
	} else {
		value, fresh, err = h.service.ReactivateSeries(c.Request.Context(), actor, unit, domain.ID(c.Param("series_id")), c.GetHeader("Idempotency-Key"), input)
	}
	if err != nil {
		respondSchedulingError(c, err)
		return
	}
	if !fresh {
		c.Header("Idempotency-Replayed", "true")
	}
	c.JSON(http.StatusOK, value)
}
func (h SchedulingGridHandler) PublishSeries(c *gin.Context) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	var req revisionReasonRequest
	if c.ShouldBindJSON(&req) != nil {
		respondSchedulingError(c, scheduling.ErrScheduleSeriesInvalid)
		return
	}
	value, fresh, err := h.service.PublishSeries(c.Request.Context(), actor, unit, domain.ID(c.Param("series_id")), c.GetHeader("Idempotency-Key"), req.ExpectedRevision)
	if err != nil {
		respondSchedulingError(c, err)
		return
	}
	if !fresh {
		c.Header("Idempotency-Replayed", "true")
	}
	c.JSON(http.StatusOK, value)
}
func (h SchedulingGridHandler) PreviewSeriesChange(c *gin.Context) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	var req seriesRequest
	if c.ShouldBindJSON(&req) != nil {
		respondSchedulingError(c, scheduling.ErrScheduleSeriesInvalid)
		return
	}
	value, err := h.service.PreviewSeriesChange(c.Request.Context(), actor, unit, domain.ID(c.Param("series_id")), scheduling.SeriesChangeInput{SeriesMutationInput: req.input(), EffectiveOn: req.EffectiveOn})
	if err != nil {
		respondSchedulingError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}
func (h SchedulingGridHandler) ChangeSeries(c *gin.Context) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	var req seriesRequest
	if c.ShouldBindJSON(&req) != nil {
		respondSchedulingError(c, scheduling.ErrScheduleSeriesInvalid)
		return
	}
	value, fresh, err := h.service.ChangeSeries(c.Request.Context(), actor, unit, domain.ID(c.Param("series_id")), c.GetHeader("Idempotency-Key"), scheduling.SeriesChangeInput{SeriesMutationInput: req.input(), EffectiveOn: req.EffectiveOn})
	if err != nil {
		respondSchedulingError(c, err)
		return
	}
	if !fresh {
		c.Header("Idempotency-Replayed", "true")
	}
	c.JSON(http.StatusOK, value)
}
func (h SchedulingGridHandler) CancelSeries(c *gin.Context) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	var req revisionReasonRequest
	if c.ShouldBindJSON(&req) != nil {
		respondSchedulingError(c, scheduling.ErrScheduleSeriesInvalid)
		return
	}
	value, fresh, err := h.service.CancelSeries(c.Request.Context(), actor, unit, domain.ID(c.Param("series_id")), c.GetHeader("Idempotency-Key"), scheduling.SeriesCancelInput{ExpectedRevision: req.ExpectedRevision, EffectiveOn: req.EffectiveOn, Reason: req.Reason})
	if err != nil {
		respondSchedulingError(c, err)
		return
	}
	if !fresh {
		c.Header("Idempotency-Replayed", "true")
	}
	c.JSON(http.StatusOK, value)
}
func (h SchedulingGridHandler) ListSessions(c *gin.Context) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	from, err := time.Parse(time.RFC3339, c.Query("from"))
	if err != nil {
		respondSchedulingError(c, scheduling.ErrSchedulePreviewRangeInvalid)
		return
	}
	until, err := time.Parse(time.RFC3339, c.Query("to"))
	if err != nil {
		respondSchedulingError(c, scheduling.ErrSchedulePreviewRangeInvalid)
		return
	}
	list, err := h.service.ListSessions(c.Request.Context(), actor, unit, from, until, c.Query("status") == "cancelled" || c.Query("include_cancelled") == "true")
	if err != nil {
		respondSchedulingError(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}
func (h SchedulingGridHandler) GetSession(c *gin.Context) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	value, err := h.service.GetSession(c.Request.Context(), actor, unit, domain.ID(c.Param("session_id")))
	if err != nil {
		respondSchedulingError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}
func (h SchedulingGridHandler) CreateSession(c *gin.Context) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	var req sessionRequest
	if c.ShouldBindJSON(&req) != nil {
		respondSchedulingError(c, scheduling.ErrScheduleSeriesInvalid)
		return
	}
	value, fresh, err := h.service.CreateExtraSession(c.Request.Context(), actor, unit, c.GetHeader("Idempotency-Key"), req.input())
	if err != nil {
		respondSchedulingError(c, err)
		return
	}
	if !fresh {
		c.Header("Idempotency-Replayed", "true")
		c.JSON(http.StatusOK, value)
		return
	}
	c.JSON(http.StatusCreated, value)
}
func (h SchedulingGridHandler) ChangeSession(c *gin.Context) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	var req sessionRequest
	if c.ShouldBindJSON(&req) != nil {
		respondSchedulingError(c, scheduling.ErrScheduleSeriesInvalid)
		return
	}
	value, fresh, err := h.service.ChangeSession(c.Request.Context(), actor, unit, domain.ID(c.Param("session_id")), c.GetHeader("Idempotency-Key"), req.input())
	if err != nil {
		respondSchedulingError(c, err)
		return
	}
	if !fresh {
		c.Header("Idempotency-Replayed", "true")
	}
	c.JSON(http.StatusOK, value)
}
func (h SchedulingGridHandler) CancelSession(c *gin.Context) {
	unit, actor, ok := schedulingContext(c)
	if !ok {
		return
	}
	var req revisionReasonRequest
	if c.ShouldBindJSON(&req) != nil {
		respondSchedulingError(c, scheduling.ErrScheduleSeriesInvalid)
		return
	}
	value, fresh, err := h.service.CancelSession(c.Request.Context(), actor, unit, domain.ID(c.Param("session_id")), c.GetHeader("Idempotency-Key"), scheduling.RevisionReasonInput{ExpectedRevision: req.ExpectedRevision, Reason: req.Reason})
	if err != nil {
		respondSchedulingError(c, err)
		return
	}
	if !fresh {
		c.Header("Idempotency-Replayed", "true")
	}
	c.JSON(http.StatusOK, value)
}
