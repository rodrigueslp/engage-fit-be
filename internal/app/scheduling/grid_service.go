package scheduling

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"github.com/google/uuid"
)

var (
	ErrScheduleSeriesNotFound          = portrepo.ErrScheduleSeriesNotFound
	ErrScheduleSessionNotFound         = portrepo.ErrScheduleSessionNotFound
	ErrScheduleSeriesInvalid           = portrepo.ErrScheduleSeriesInvalid
	ErrScheduleSeriesArchived          = portrepo.ErrScheduleSeriesArchived
	ErrScheduleSeriesAlreadyPublished  = portrepo.ErrScheduleSeriesAlreadyPublished
	ErrScheduleTargetUnavailable       = portrepo.ErrScheduleTargetUnavailable
	ErrScheduleSpaceUnavailable        = portrepo.ErrScheduleSpaceUnavailable
	ErrScheduleProfessionalUnavailable = portrepo.ErrScheduleProfessionalUnavailable
	ErrScheduleConflict                = portrepo.ErrScheduleConflict
	ErrScheduleInvalidLocalTime        = portrepo.ErrScheduleInvalidLocalTime
	ErrSchedulePreviewRangeInvalid     = portrepo.ErrSchedulePreviewRangeInvalid
	ErrSchedulePreviewTooLarge         = portrepo.ErrSchedulePreviewTooLarge
	ErrScheduleSessionAlreadyStarted   = portrepo.ErrScheduleSessionAlreadyStarted
	ErrScheduleSessionCancelled        = portrepo.ErrScheduleSessionCancelled
	ErrScheduleMaterializerUnavailable = portrepo.ErrScheduleMaterializerUnavailable
)

type SeriesMutationInput struct {
	TargetKind       domain.ScheduleTargetKind   `json:"target_kind"`
	TargetID         domain.ID                   `json:"target_id"`
	ProfessionalID   domain.ID                   `json:"professional_id"`
	SpaceID          *domain.ID                  `json:"space_id"`
	DeliveryMode     domain.ScheduleDeliveryMode `json:"delivery_mode"`
	Capacity         int                         `json:"capacity"`
	DurationMinutes  int                         `json:"duration_minutes"`
	Weekdays         []int                       `json:"weekdays"`
	LocalStartTime   string                      `json:"local_start_time"`
	Timezone         string                      `json:"timezone"`
	EffectiveFrom    string                      `json:"effective_from"`
	EffectiveUntil   *string                     `json:"effective_until"`
	ExpectedRevision int                         `json:"expected_revision,omitempty"`
}

type PreviewInput struct {
	SeriesMutationInput
	SeriesID     *domain.ID `json:"series_id,omitempty"`
	PreviewFrom  string     `json:"preview_from"`
	PreviewUntil string     `json:"preview_until"`
}
type RevisionReasonInput struct {
	ExpectedRevision int    `json:"expected_revision"`
	Reason           string `json:"reason"`
}
type PublishResult struct {
	Series   domain.ScheduleSeries    `json:"series"`
	Sessions []domain.ScheduleSession `json:"sessions"`
}
type SeriesChangeInput struct {
	SeriesMutationInput
	EffectiveOn string `json:"effective_on"`
}
type SeriesChangePreview struct {
	Preview   domain.SchedulePreview `json:"preview"`
	Kept      int                    `json:"kept"`
	Changed   int                    `json:"changed"`
	Created   int                    `json:"created"`
	Cancelled int                    `json:"cancelled"`
}
type SeriesCancelInput struct {
	ExpectedRevision int    `json:"expected_revision"`
	EffectiveOn      string `json:"effective_on"`
	Reason           string `json:"reason"`
}
type SessionMutationInput struct {
	TargetKind       domain.ScheduleTargetKind   `json:"target_kind"`
	TargetID         domain.ID                   `json:"target_id"`
	ProfessionalID   domain.ID                   `json:"professional_id"`
	SpaceID          *domain.ID                  `json:"space_id"`
	DeliveryMode     domain.ScheduleDeliveryMode `json:"delivery_mode"`
	Capacity         int                         `json:"capacity"`
	StartsAt         time.Time                   `json:"starts_at"`
	DurationMinutes  int                         `json:"duration_minutes"`
	Timezone         string                      `json:"timezone"`
	ExpectedRevision int                         `json:"expected_revision,omitempty"`
}

type GridService struct {
	repo                    portrepo.SchedulingGridRepository
	contexts                portrepo.SchedulingUnitContextResolver
	catalog                 portrepo.SchedulingCatalogResolver
	tx                      portrepo.TransactionManager
	now                     func() time.Time
	materializerOperational bool
	reservations            portrepo.SessionReservationReconciler
	professionalResolver    portrepo.SchedulingProfessionalResolver
}

func (s *GridService) SetReservationReconciler(value portrepo.SessionReservationReconciler) {
	s.reservations = value
}

type GridOption func(*GridService)

func WithGridClock(now func() time.Time) GridOption {
	return func(service *GridService) { service.now = now }
}

func WithGridMaterializerOperational(enabled bool) GridOption {
	return func(service *GridService) { service.materializerOperational = enabled }
}

func WithGridProfessionalResolver(resolver portrepo.SchedulingProfessionalResolver) GridOption {
	return func(service *GridService) { service.professionalResolver = resolver }
}

func NewGridService(repo portrepo.SchedulingGridRepository, contexts portrepo.SchedulingUnitContextResolver, catalog portrepo.SchedulingCatalogResolver, tx portrepo.TransactionManager, opts ...GridOption) *GridService {
	clock := func() time.Time { return time.Now().UTC() }
	service := &GridService{repo: repo, contexts: contexts, catalog: catalog, tx: tx, now: clock, materializerOperational: true}
	for _, opt := range opts {
		opt(service)
	}
	return service
}

func (s *GridService) ownerContext(ctx context.Context, actor, unit domain.ID) (*domain.SchedulingUnitContext, error) {
	value, err := s.contexts.ResolveSchedulingUnitContext(ctx, actor, unit)
	if err != nil || value == nil || !value.Authorized || value.Role != "OWNER" {
		return nil, ErrSchedulingUnitNotFound
	}
	return value, nil
}
func (s *GridService) readerContext(ctx context.Context, actor, unit domain.ID) (*domain.SchedulingUnitContext, error) {
	value, err := s.contexts.ResolveSchedulingUnitContext(ctx, actor, unit)
	if err != nil || value == nil || !value.Authorized || (value.Role != "OWNER" && value.Role != "COACH") {
		return nil, ErrSchedulingUnitNotFound
	}
	return value, nil
}

func seriesFromInput(input SeriesMutationInput) domain.ScheduleSeries {
	value := domain.ScheduleSeries{TargetKind: input.TargetKind, TargetID: input.TargetID, ProfessionalID: input.ProfessionalID, SpaceID: input.SpaceID, DeliveryMode: input.DeliveryMode, Capacity: input.Capacity, DurationMinutes: input.DurationMinutes, Weekdays: append([]int(nil), input.Weekdays...), LocalStartTime: input.LocalStartTime, Timezone: input.Timezone, EffectiveFrom: input.EffectiveFrom, EffectiveUntil: input.EffectiveUntil}
	domain.NormalizeScheduleSeries(&value)
	return value
}

func (s *GridService) resolve(ctx context.Context, series *domain.ScheduleSeries, command bool) ([]domain.SchedulePreviewIssue, error) {
	refs, err := s.catalog.ResolveSchedulingReferences(ctx, series.OrganizationID, series.UnitID, series.TargetKind, series.TargetID, series.SpaceID)
	if err != nil {
		return nil, err
	}
	prof, err := s.repo.GetByID(ctx, series.OrganizationID, series.UnitID, series.ProfessionalID)
	if err != nil {
		return nil, ErrScheduleProfessionalUnavailable
	}
	health := domain.ScheduleReferenceHealth{TargetAvailable: refs.Target.Status == "active" && refs.Target.EnabledInUnit, ProfessionalAvailable: prof.Status == domain.ProfessionalStatusActive && prof.ActiveInUnit && prof.AnonymizedAt == nil, SpaceAvailable: refs.Space == nil || (refs.Space.Status == "active" && refs.Space.EnabledInUnit), TargetName: refs.Target.DisplayName, ProfessionalName: prof.DisplayName}
	if refs.Space != nil {
		health.SpaceName = refs.Space.DisplayName
	}
	series.ReferenceHealth = health
	issues := []domain.SchedulePreviewIssue{}
	if !health.TargetAvailable {
		issues = append(issues, domain.SchedulePreviewIssue{Code: "target_unavailable", ResourceType: "target", ResourceID: series.TargetID})
	}
	if !health.ProfessionalAvailable {
		issues = append(issues, domain.SchedulePreviewIssue{Code: "professional_unavailable", ResourceType: "professional", ResourceID: series.ProfessionalID})
	}
	if !health.SpaceAvailable && series.SpaceID != nil {
		issues = append(issues, domain.SchedulePreviewIssue{Code: "space_unavailable", ResourceType: "space", ResourceID: *series.SpaceID})
	}
	if command && len(issues) > 0 {
		switch issues[0].Code {
		case "target_unavailable":
			return nil, ErrScheduleTargetUnavailable
		case "space_unavailable":
			return nil, ErrScheduleSpaceUnavailable
		default:
			return nil, ErrScheduleProfessionalUnavailable
		}
	}
	return issues, nil
}

func (s *GridService) ListSeries(ctx context.Context, actor, unit domain.ID, includeArchived bool) ([]domain.ScheduleSeries, error) {
	uc, err := s.ownerContext(ctx, actor, unit)
	if err != nil {
		return nil, err
	}
	list, err := s.repo.ListSeries(ctx, uc.OrganizationID, unit, includeArchived)
	if err != nil {
		return nil, err
	}
	for i := range list {
		_, _ = s.resolve(ctx, &list[i], false)
	}
	return list, nil
}
func (s *GridService) GetSeries(ctx context.Context, actor, unit, id domain.ID) (*domain.ScheduleSeries, error) {
	uc, err := s.ownerContext(ctx, actor, unit)
	if err != nil {
		return nil, err
	}
	value, err := s.repo.GetSeries(ctx, uc.OrganizationID, unit, id, false)
	if err != nil {
		return nil, err
	}
	_, _ = s.resolve(ctx, value, false)
	return value, nil
}

func (s *GridService) CreateSeries(ctx context.Context, actor, unit domain.ID, key string, input SeriesMutationInput) (*domain.ScheduleSeries, bool, error) {
	uc, err := s.ownerContext(ctx, actor, unit)
	if err != nil {
		return nil, false, err
	}
	if !idempotencyKeyPattern.MatchString(key) {
		return nil, false, ErrScheduleSeriesInvalid
	}
	value := seriesFromInput(input)
	value.ID = domain.ID(uuid.NewString())
	value.OrganizationID = uc.OrganizationID
	value.UnitID = unit
	value.Status = domain.ScheduleSeriesDraft
	value.Revision = 1
	now := s.now()
	value.CreatedAt = now
	value.UpdatedAt = now
	value.CreatedByUserID = actor
	value.UpdatedByUserID = actor
	if err := domain.ValidateScheduleSeries(value); err != nil {
		return nil, false, ErrScheduleSeriesInvalid
	}
	if _, err := s.resolve(ctx, &value, true); err != nil {
		return nil, false, err
	}
	fresh, err := s.command(ctx, uc.OrganizationID, unit, "series_create", key, value.ID, "", fingerprintValue(input), &value, func(tx context.Context) error {
		if err := s.repo.CreateSeries(tx, &value); err != nil {
			return err
		}
		return s.audit(tx, actor, value.OrganizationID, unit, "scheduling.series.created", "series", value.ID, 1, nil, map[string]any{"status": "draft", "revision": 1})
	})
	return &value, fresh, err
}

func (s *GridService) UpdateSeries(ctx context.Context, actor, unit, id domain.ID, key string, input SeriesMutationInput) (*domain.ScheduleSeries, bool, error) {
	uc, err := s.ownerContext(ctx, actor, unit)
	if err != nil {
		return nil, false, err
	}
	if !idempotencyKeyPattern.MatchString(key) || input.ExpectedRevision < 1 {
		return nil, false, ErrScheduleSeriesInvalid
	}
	var out domain.ScheduleSeries
	fresh, err := s.command(ctx, uc.OrganizationID, unit, "series_update", key, id, "", fingerprintValue(struct {
		ID    domain.ID
		Input SeriesMutationInput
	}{id, input}), &out, func(tx context.Context) error {
		current, e := s.repo.GetSeries(tx, uc.OrganizationID, unit, id, true)
		if e != nil {
			return e
		}
		if current.Status != domain.ScheduleSeriesDraft {
			return ErrScheduleSeriesArchived
		}
		if current.Revision != input.ExpectedRevision {
			return ErrSchedulingRevisionConflict
		}
		next := seriesFromInput(input)
		next.ID = id
		next.OrganizationID = uc.OrganizationID
		next.UnitID = unit
		next.Status = current.Status
		next.Revision = current.Revision + 1
		next.CreatedAt = current.CreatedAt
		next.CreatedByUserID = current.CreatedByUserID
		next.UpdatedAt = s.now()
		next.UpdatedByUserID = actor
		if e = domain.ValidateScheduleSeries(next); e != nil {
			return ErrScheduleSeriesInvalid
		}
		if _, e = s.resolve(tx, &next, true); e != nil {
			return e
		}
		if e = s.repo.UpdateSeries(tx, &next); e != nil {
			return e
		}
		if e = s.audit(tx, actor, uc.OrganizationID, unit, "scheduling.series.updated", "series", id, next.Revision, map[string]any{"revision": current.Revision}, map[string]any{"revision": next.Revision}); e != nil {
			return e
		}
		out = next
		return nil
	})
	return &out, fresh, err
}

func (s *GridService) ArchiveSeries(ctx context.Context, actor, unit, id domain.ID, key string, input RevisionReasonInput) (*domain.ScheduleSeries, bool, error) {
	return s.seriesLifecycle(ctx, actor, unit, id, key, input, true)
}
func (s *GridService) ReactivateSeries(ctx context.Context, actor, unit, id domain.ID, key string, input RevisionReasonInput) (*domain.ScheduleSeries, bool, error) {
	return s.seriesLifecycle(ctx, actor, unit, id, key, input, false)
}
func (s *GridService) seriesLifecycle(ctx context.Context, actor, unit, id domain.ID, key string, input RevisionReasonInput, archive bool) (*domain.ScheduleSeries, bool, error) {
	uc, err := s.ownerContext(ctx, actor, unit)
	if err != nil {
		return nil, false, err
	}
	reason, ok := domain.NormalizeSchedulingReason(input.Reason, 3, 240)
	if !ok || input.ExpectedRevision < 1 || !idempotencyKeyPattern.MatchString(key) {
		return nil, false, ErrScheduleSeriesInvalid
	}
	command := "series_reactivate"
	action := "scheduling.series.reactivated"
	if archive {
		command = "series_archive"
		action = "scheduling.series.archived"
	}
	var out domain.ScheduleSeries
	fresh, err := s.command(ctx, uc.OrganizationID, unit, command, key, id, "", fingerprintValue(struct {
		ID    domain.ID
		Input RevisionReasonInput
	}{id, input}), &out, func(tx context.Context) error {
		current, e := s.repo.GetSeries(tx, uc.OrganizationID, unit, id, true)
		if e != nil {
			return e
		}
		if current.Revision != input.ExpectedRevision {
			return ErrSchedulingRevisionConflict
		}
		if archive {
			if current.Status != domain.ScheduleSeriesDraft {
				return ErrScheduleSeriesArchived
			}
			current.Status = domain.ScheduleSeriesArchived
			now := s.now()
			current.ArchivedAt = &now
			current.ArchivedByUserID = &actor
		} else {
			if current.Status != domain.ScheduleSeriesArchived {
				return ErrScheduleSeriesArchived
			}
			if _, e = s.resolve(tx, current, true); e != nil {
				return e
			}
			current.Status = domain.ScheduleSeriesDraft
			current.ArchivedAt = nil
			current.ArchivedByUserID = nil
		}
		current.Revision++
		current.UpdatedAt = s.now()
		current.UpdatedByUserID = actor
		if e = s.repo.UpdateSeries(tx, current); e != nil {
			return e
		}
		if e = s.auditReason(tx, actor, uc.OrganizationID, unit, action, "series", id, current.Revision, reason); e != nil {
			return e
		}
		out = *current
		return nil
	})
	return &out, fresh, err
}

func (s *GridService) PreviewCandidate(ctx context.Context, actor, unit domain.ID, input PreviewInput) (*domain.SchedulePreview, error) {
	uc, err := s.ownerContext(ctx, actor, unit)
	if err != nil {
		return nil, err
	}
	candidate := seriesFromInput(input.SeriesMutationInput)
	candidate.OrganizationID = uc.OrganizationID
	candidate.UnitID = unit
	candidate.Status = domain.ScheduleSeriesDraft
	if input.SeriesID != nil {
		candidate.ID = *input.SeriesID
		if _, err = s.repo.GetSeries(ctx, uc.OrganizationID, unit, *input.SeriesID, false); err != nil {
			return nil, err
		}
	}
	return s.preview(ctx, uc.OrganizationID, []domain.ScheduleSeries{candidate}, input.PreviewFrom, input.PreviewUntil, input.SeriesID)
}
func (s *GridService) PreviewGrid(ctx context.Context, actor, unit domain.ID, from, until string) (*domain.SchedulePreview, error) {
	uc, err := s.ownerContext(ctx, actor, unit)
	if err != nil {
		return nil, err
	}
	list, err := s.repo.ListSeries(ctx, uc.OrganizationID, unit, false)
	if err != nil {
		return nil, err
	}
	drafts := []domain.ScheduleSeries{}
	for _, v := range list {
		if v.Status == domain.ScheduleSeriesDraft {
			drafts = append(drafts, v)
		}
	}
	return s.preview(ctx, uc.OrganizationID, drafts, from, until, nil)
}
func (s *GridService) preview(ctx context.Context, org domain.ID, candidates []domain.ScheduleSeries, from, until string, exclude *domain.ID) (*domain.SchedulePreview, error) {
	if !validPreviewRange(from, until) {
		return nil, ErrSchedulePreviewRangeInvalid
	}
	all, err := s.repo.ListSeriesAcrossOrganization(ctx, org)
	if err != nil {
		return nil, err
	}
	combined := []domain.SchedulePreviewOccurrence{}
	issues := []domain.SchedulePreviewIssue{}
	candidateIDs := map[domain.ID]bool{}
	for _, v := range candidates {
		candidateIDs[v.ID] = true
		if domain.ValidateScheduleSeries(v) != nil {
			return nil, ErrScheduleSeriesInvalid
		}
		refIssues, e := s.resolve(ctx, &v, false)
		if e != nil {
			return nil, e
		}
		issues = append(issues, refIssues...)
		occ, expIssues, e := domain.ExpandScheduleSeries(v, from, until)
		if e != nil {
			return nil, ErrSchedulePreviewRangeInvalid
		}
		combined = append(combined, occ...)
		issues = append(issues, expIssues...)
	}
	for _, v := range all {
		if v.Status != domain.ScheduleSeriesDraft || candidateIDs[v.ID] || (exclude != nil && v.ID == *exclude) {
			continue
		}
		_, _ = s.resolve(ctx, &v, false)
		occ, expIssues, e := domain.ExpandScheduleSeries(v, from, until)
		if e == nil {
			combined = append(combined, occ...)
			issues = append(issues, expIssues...)
		}
	}
	issues = append(issues, domain.DetectScheduleConflicts(combined)...)
	if len(combined) > 2000 || len(issues) > 200 {
		return nil, ErrSchedulePreviewTooLarge
	}
	sort.Slice(combined, func(i, j int) bool { return combined[i].StartsAt.Before(combined[j].StartsAt) })
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%v", from, until, combined)))
	return &domain.SchedulePreview{PreviewFrom: from, PreviewUntil: until, GeneratedAt: s.now(), Fingerprint: hex.EncodeToString(sum[:]), ReadyForFuturePublication: len(issues) == 0, Occurrences: combined, Issues: issues}, nil
}

func (s *GridService) PublishSeries(ctx context.Context, actor, unit, id domain.ID, key string, expectedRevision int) (*PublishResult, bool, error) {
	if !s.materializerOperational {
		return nil, false, ErrScheduleMaterializerUnavailable
	}
	uc, err := s.ownerContext(ctx, actor, unit)
	if err != nil {
		return nil, false, err
	}
	if expectedRevision < 1 || !idempotencyKeyPattern.MatchString(key) {
		return nil, false, ErrScheduleSeriesInvalid
	}
	out := PublishResult{}
	fresh, err := s.command(ctx, uc.OrganizationID, unit, "series_publish", key, id, "", fingerprintValue(struct {
		ID       domain.ID
		Revision int
	}{id, expectedRevision}), &out, func(tx context.Context) error {
		series, e := s.repo.GetSeries(tx, uc.OrganizationID, unit, id, true)
		if e != nil {
			return e
		}
		if series.Status != domain.ScheduleSeriesDraft {
			return ErrScheduleSeriesAlreadyPublished
		}
		if series.Revision != expectedRevision {
			return ErrSchedulingRevisionConflict
		}
		if _, e = s.resolve(tx, series, true); e != nil {
			return e
		}
		loc, _ := time.LoadLocation(series.Timezone)
		today := s.now().In(loc).Format("2006-01-02")
		untilDate, _ := time.Parse("2006-01-02", today)
		until := untilDate.AddDate(0, 0, 55).Format("2006-01-02")
		occ, issues, e := domain.ExpandScheduleSeries(*series, today, until)
		if e != nil || len(issues) > 0 || len(occ) == 0 {
			return ErrScheduleInvalidLocalTime
		}
		spaces := []domain.ID{}
		if series.SpaceID != nil {
			spaces = append(spaces, *series.SpaceID)
		}
		if e = s.repo.LockResources(tx, uc.OrganizationID, []domain.ID{series.ProfessionalID}, spaces); e != nil {
			return e
		}
		existing, e := s.repo.ListSessionsAcrossOrganization(tx, uc.OrganizationID, occ[0].StartsAt, occ[len(occ)-1].EndsAt)
		if e != nil {
			return e
		}
		if hasSessionConflict(occ, existing, "") {
			return ErrScheduleConflict
		}
		now := s.now()
		series.Status = domain.ScheduleSeriesPublished
		series.TargetNameSnapshot = series.ReferenceHealth.TargetName
		series.ProfessionalNameSnapshot = series.ReferenceHealth.ProfessionalName
		series.SpaceNameSnapshot = series.ReferenceHealth.SpaceName
		series.Revision++
		series.PublishedAt = &now
		series.PublishedByUserID = &actor
		series.UpdatedAt = now
		series.UpdatedByUserID = actor
		if e = s.repo.UpdateSeries(tx, series); e != nil {
			return e
		}
		sessions := make([]domain.ScheduleSession, 0, len(occ))
		for _, o := range occ {
			session := sessionFromOccurrence(*series, o, actor, now)
			if e = s.repo.CreateSession(tx, &session); e != nil {
				return e
			}
			if e = s.event(tx, "session_published", domain.ScheduleSession{}, session); e != nil {
				return e
			}
			sessions = append(sessions, session)
		}
		if e = s.audit(tx, actor, uc.OrganizationID, unit, "scheduling.series.published", "series", id, series.Revision, map[string]any{"status": "draft"}, map[string]any{"status": "published", "sessions": len(sessions)}); e != nil {
			return e
		}
		out = PublishResult{Series: *series, Sessions: sessions}
		return nil
	})
	return &out, fresh, err
}

func (s *GridService) ListSessions(ctx context.Context, actor, unit domain.ID, from, until time.Time, includeCancelled bool) ([]domain.ScheduleSession, error) {
	uc, err := s.readerContext(ctx, actor, unit)
	if err != nil {
		return nil, err
	}
	if until.Before(from) || until.Sub(from) > 93*24*time.Hour {
		return nil, ErrSchedulePreviewRangeInvalid
	}
	sessions, err := s.repo.ListSessions(ctx, uc.OrganizationID, unit, from, until, includeCancelled)
	if err != nil {
		return nil, err
	}
	if s.professionalResolver == nil {
		return sessions, nil
	}
	professionalID, err := s.professionalResolver.ResolveProfessionalForUser(ctx, uc.OrganizationID, unit, actor)
	if err != nil {
		return nil, err
	}
	if professionalID == nil {
		return sessions, nil
	}
	for index := range sessions {
		sessions[index].AssignedToCurrentUser = sessions[index].ProfessionalID == *professionalID
	}
	return sessions, nil
}
func (s *GridService) GetSession(ctx context.Context, actor, unit, id domain.ID) (*domain.ScheduleSession, error) {
	uc, err := s.readerContext(ctx, actor, unit)
	if err != nil {
		return nil, err
	}
	return s.repo.GetSession(ctx, uc.OrganizationID, unit, id, false)
}

func (s *GridService) CreateExtraSession(ctx context.Context, actor, unit domain.ID, key string, input SessionMutationInput) (*domain.ScheduleSession, bool, error) {
	uc, err := s.ownerContext(ctx, actor, unit)
	if err != nil {
		return nil, false, err
	}
	if !idempotencyKeyPattern.MatchString(key) {
		return nil, false, ErrScheduleSeriesInvalid
	}
	id := domain.ID(uuid.NewString())
	out := domain.ScheduleSession{}
	fresh, err := s.command(ctx, uc.OrganizationID, unit, "session_create", key, "", id, fingerprintValue(input), &out, func(tx context.Context) error {
		session, e := s.buildSession(tx, uc.OrganizationID, unit, id, nil, "extra", nil, actor, input)
		if e != nil {
			return e
		}
		if !session.StartsAt.After(s.now()) {
			return ErrScheduleSessionAlreadyStarted
		}
		if e = s.lockAndCheckSession(tx, *session, ""); e != nil {
			return e
		}
		if e = s.repo.CreateSession(tx, session); e != nil {
			return e
		}
		if e = s.event(tx, "session_published", domain.ScheduleSession{}, *session); e != nil {
			return e
		}
		if e = s.audit(tx, actor, uc.OrganizationID, unit, "scheduling.session.extra_created", "session", id, 1, nil, map[string]any{"status": "scheduled"}); e != nil {
			return e
		}
		out = *session
		return nil
	})
	return &out, fresh, err
}

func (s *GridService) ChangeSession(ctx context.Context, actor, unit, id domain.ID, key string, input SessionMutationInput) (*domain.ScheduleSession, bool, error) {
	uc, err := s.ownerContext(ctx, actor, unit)
	if err != nil {
		return nil, false, err
	}
	if input.ExpectedRevision < 1 || !idempotencyKeyPattern.MatchString(key) {
		return nil, false, ErrScheduleSeriesInvalid
	}
	var out domain.ScheduleSession
	fresh, err := s.command(ctx, uc.OrganizationID, unit, "session_change", key, "", id, fingerprintValue(struct {
		ID    domain.ID
		Input SessionMutationInput
	}{id, input}), &out, func(tx context.Context) error {
		current, e := s.repo.GetSession(tx, uc.OrganizationID, unit, id, true)
		if e != nil {
			return e
		}
		if current.Revision != input.ExpectedRevision {
			return ErrSchedulingRevisionConflict
		}
		if current.Status == domain.ScheduleSessionCancelled {
			return ErrScheduleSessionCancelled
		}
		if !current.StartsAt.After(s.now()) {
			return ErrScheduleSessionAlreadyStarted
		}
		next, e := s.buildSession(tx, uc.OrganizationID, unit, id, current.SeriesID, current.Origin, current.OccurrenceOn, actor, input)
		if e != nil {
			return e
		}
		next.CreatedAt = current.CreatedAt
		next.CreatedByUserID = current.CreatedByUserID
		next.PublishedAt = current.PublishedAt
		next.Revision = current.Revision + 1
		next.ManualOverride = true
		next.SourceSeriesRevision = current.SourceSeriesRevision
		if e = s.lockAndCheckSession(tx, *next, id); e != nil {
			return e
		}
		if e = s.repo.UpdateSession(tx, next); e != nil {
			return e
		}
		if e = s.event(tx, "session_changed", *current, *next); e != nil {
			return e
		}
		if e = s.audit(tx, actor, uc.OrganizationID, unit, "scheduling.session.changed", "session", id, next.Revision, map[string]any{"revision": current.Revision}, map[string]any{"revision": next.Revision, "manual_override": true}); e != nil {
			return e
		}
		out = *next
		return nil
	})
	return &out, fresh, err
}

func (s *GridService) CancelSession(ctx context.Context, actor, unit, id domain.ID, key string, input RevisionReasonInput) (*domain.ScheduleSession, bool, error) {
	uc, err := s.ownerContext(ctx, actor, unit)
	if err != nil {
		return nil, false, err
	}
	reason, ok := domain.NormalizeSchedulingReason(input.Reason, 3, 240)
	if !ok || input.ExpectedRevision < 1 || !idempotencyKeyPattern.MatchString(key) {
		return nil, false, ErrScheduleSeriesInvalid
	}
	var out domain.ScheduleSession
	fresh, err := s.command(ctx, uc.OrganizationID, unit, "session_cancel", key, "", id, fingerprintValue(struct {
		ID    domain.ID
		Input RevisionReasonInput
	}{id, input}), &out, func(tx context.Context) error {
		current, e := s.repo.GetSession(tx, uc.OrganizationID, unit, id, true)
		if e != nil {
			return e
		}
		if current.Revision != input.ExpectedRevision {
			return ErrSchedulingRevisionConflict
		}
		if current.Status == domain.ScheduleSessionCancelled {
			return ErrScheduleSessionCancelled
		}
		if !current.StartsAt.After(s.now()) {
			return ErrScheduleSessionAlreadyStarted
		}
		before := *current
		now := s.now()
		current.Status = domain.ScheduleSessionCancelled
		current.CancelledAt = &now
		current.CancelledByUserID = &actor
		current.CancelledReason = reason
		current.Revision++
		current.UpdatedAt = now
		current.UpdatedByUserID = actor
		if e = s.repo.UpdateSession(tx, current); e != nil {
			return e
		}
		if e = s.event(tx, "session_cancelled", before, *current); e != nil {
			return e
		}
		if e = s.auditReason(tx, actor, uc.OrganizationID, unit, "scheduling.session.cancelled", "session", id, current.Revision, reason); e != nil {
			return e
		}
		if s.reservations != nil {
			if e = s.reservations.CancelSessionReservations(tx, uc.OrganizationID, unit, id, actor, reason); e != nil {
				return e
			}
		}
		out = *current
		return nil
	})
	return &out, fresh, err
}

func (s *GridService) CancelSeries(ctx context.Context, actor, unit, id domain.ID, key string, input SeriesCancelInput) (*PublishResult, bool, error) {
	uc, err := s.ownerContext(ctx, actor, unit)
	if err != nil {
		return nil, false, err
	}
	reason, ok := domain.NormalizeSchedulingReason(input.Reason, 3, 240)
	if !ok || input.ExpectedRevision < 1 || !validDate(input.EffectiveOn) || !idempotencyKeyPattern.MatchString(key) {
		return nil, false, ErrScheduleSeriesInvalid
	}
	out := PublishResult{}
	fresh, err := s.command(ctx, uc.OrganizationID, unit, "series_cancel", key, id, "", fingerprintValue(struct {
		ID    domain.ID
		Input SeriesCancelInput
	}{id, input}), &out, func(tx context.Context) error {
		series, e := s.repo.GetSeries(tx, uc.OrganizationID, unit, id, true)
		if e != nil {
			return e
		}
		if series.Status != domain.ScheduleSeriesPublished {
			return ErrScheduleSeriesAlreadyPublished
		}
		if series.Revision != input.ExpectedRevision {
			return ErrSchedulingRevisionConflict
		}
		from, _ := time.Parse("2006-01-02", input.EffectiveOn)
		sessions, e := s.repo.ListSessions(tx, uc.OrganizationID, unit, from, s.now().AddDate(2, 0, 0), false)
		if e != nil {
			return e
		}
		now := s.now()
		changed := []domain.ScheduleSession{}
		for i := range sessions {
			if sessions[i].SeriesID == nil || *sessions[i].SeriesID != id || sessions[i].LocalDate < input.EffectiveOn || !sessions[i].StartsAt.After(now) {
				continue
			}
			before := sessions[i]
			sessions[i].Status = domain.ScheduleSessionCancelled
			sessions[i].CancelledAt = &now
			sessions[i].CancelledByUserID = &actor
			sessions[i].CancelledReason = reason
			sessions[i].Revision++
			sessions[i].UpdatedAt = now
			sessions[i].UpdatedByUserID = actor
			if e = s.repo.UpdateSession(tx, &sessions[i]); e != nil {
				return e
			}
			if e = s.event(tx, "session_cancelled", before, sessions[i]); e != nil {
				return e
			}
			if s.reservations != nil {
				if e = s.reservations.CancelSessionReservations(tx, uc.OrganizationID, unit, sessions[i].ID, actor, reason); e != nil {
					return e
				}
			}
			changed = append(changed, sessions[i])
		}
		series.Status = domain.ScheduleSeriesCancelled
		series.CancelledAt = &now
		series.CancelledByUserID = &actor
		series.CancelledReason = reason
		series.Revision++
		series.UpdatedAt = now
		series.UpdatedByUserID = actor
		if e = s.repo.UpdateSeries(tx, series); e != nil {
			return e
		}
		if e = s.auditReason(tx, actor, uc.OrganizationID, unit, "scheduling.series.cancelled", "series", id, series.Revision, reason); e != nil {
			return e
		}
		out = PublishResult{Series: *series, Sessions: changed}
		return nil
	})
	return &out, fresh, err
}

func (s *GridService) PreviewSeriesChange(ctx context.Context, actor, unit, id domain.ID, input SeriesChangeInput) (*SeriesChangePreview, error) {
	uc, err := s.ownerContext(ctx, actor, unit)
	if err != nil {
		return nil, err
	}
	current, err := s.repo.GetSeries(ctx, uc.OrganizationID, unit, id, false)
	if err != nil {
		return nil, err
	}
	if current.Status != domain.ScheduleSeriesPublished {
		return nil, ErrScheduleSeriesAlreadyPublished
	}
	candidate := seriesFromInput(input.SeriesMutationInput)
	candidate.ID = id
	candidate.OrganizationID = uc.OrganizationID
	candidate.UnitID = unit
	candidate.Status = domain.ScheduleSeriesPublished
	candidate.ReferenceHealth = current.ReferenceHealth
	loc, err := time.LoadLocation(candidate.Timezone)
	if err != nil {
		return nil, ErrScheduleSeriesInvalid
	}
	until := s.now().In(loc).AddDate(0, 0, 55).Format("2006-01-02")
	preview, err := s.preview(ctx, uc.OrganizationID, []domain.ScheduleSeries{candidate}, input.EffectiveOn, until, &id)
	if err != nil {
		return nil, err
	}
	return &SeriesChangePreview{Preview: *preview, Changed: len(preview.Occurrences)}, nil
}

func (s *GridService) ChangeSeries(ctx context.Context, actor, unit, id domain.ID, key string, input SeriesChangeInput) (*PublishResult, bool, error) {
	uc, err := s.ownerContext(ctx, actor, unit)
	if err != nil {
		return nil, false, err
	}
	if input.ExpectedRevision < 1 || !validDate(input.EffectiveOn) || !idempotencyKeyPattern.MatchString(key) {
		return nil, false, ErrScheduleSeriesInvalid
	}
	out := PublishResult{}
	fresh, err := s.command(ctx, uc.OrganizationID, unit, "series_change", key, id, "", fingerprintValue(struct {
		ID    domain.ID
		Input SeriesChangeInput
	}{id, input}), &out, func(tx context.Context) error {
		current, e := s.repo.GetSeries(tx, uc.OrganizationID, unit, id, true)
		if e != nil {
			return e
		}
		if current.Status != domain.ScheduleSeriesPublished {
			return ErrScheduleSeriesAlreadyPublished
		}
		if current.Revision != input.ExpectedRevision {
			return ErrSchedulingRevisionConflict
		}
		next := seriesFromInput(input.SeriesMutationInput)
		next.ID = id
		next.OrganizationID = uc.OrganizationID
		next.UnitID = unit
		next.Status = current.Status
		next.Revision = current.Revision + 1
		next.PublishedAt = current.PublishedAt
		next.PublishedByUserID = current.PublishedByUserID
		next.CreatedAt = current.CreatedAt
		next.CreatedByUserID = current.CreatedByUserID
		next.UpdatedAt = s.now()
		next.UpdatedByUserID = actor
		if _, e = s.resolve(tx, &next, true); e != nil {
			return e
		}
		next.TargetNameSnapshot = next.ReferenceHealth.TargetName
		next.ProfessionalNameSnapshot = next.ReferenceHealth.ProfessionalName
		next.SpaceNameSnapshot = next.ReferenceHealth.SpaceName
		loc, _ := time.LoadLocation(next.Timezone)
		until := s.now().In(loc).AddDate(0, 0, 55).Format("2006-01-02")
		occ, issues, e := domain.ExpandScheduleSeries(next, input.EffectiveOn, until)
		if e != nil || len(issues) > 0 {
			return ErrScheduleInvalidLocalTime
		}
		spaces := []domain.ID{}
		if next.SpaceID != nil {
			spaces = append(spaces, *next.SpaceID)
		}
		if e = s.repo.LockResources(tx, uc.OrganizationID, []domain.ID{next.ProfessionalID}, spaces); e != nil {
			return e
		}
		existing, e := s.repo.ListSessionsAcrossOrganization(tx, uc.OrganizationID, s.now(), s.now().AddDate(0, 0, 60))
		if e != nil {
			return e
		}
		conflictExisting := make([]domain.ScheduleSession, 0, len(existing))
		for _, session := range existing {
			modifiableOccurrence := session.SeriesID != nil && *session.SeriesID == id &&
				session.LocalDate >= input.EffectiveOn && !session.ManualOverride &&
				session.Status == domain.ScheduleSessionScheduled && session.StartsAt.After(s.now())
			if !modifiableOccurrence {
				conflictExisting = append(conflictExisting, session)
			}
		}
		if hasSessionConflict(occ, conflictExisting, "") {
			return ErrScheduleConflict
		}
		byDate := map[string]domain.SchedulePreviewOccurrence{}
		for _, o := range occ {
			byDate[o.LocalDate] = o
		}
		changed := []domain.ScheduleSession{}
		for i := range existing {
			sess := existing[i]
			if sess.SeriesID == nil || *sess.SeriesID != id || sess.LocalDate < input.EffectiveOn || sess.ManualOverride || sess.Status == domain.ScheduleSessionCancelled || !sess.StartsAt.After(s.now()) {
				continue
			}
			o, keep := byDate[sess.LocalDate]
			before := sess
			if !keep {
				now := s.now()
				sess.Status = domain.ScheduleSessionCancelled
				sess.CancelledAt = &now
				sess.CancelledByUserID = &actor
				sess.CancelledReason = "Alteração da série"
			} else {
				applyOccurrence(&sess, next, o)
				delete(byDate, sess.LocalDate)
			}
			sess.Revision++
			sess.UpdatedAt = s.now()
			sess.UpdatedByUserID = actor
			if e = s.repo.UpdateSession(tx, &sess); e != nil {
				return e
			}
			eventType := "session_changed"
			if sess.Status == domain.ScheduleSessionCancelled {
				eventType = "session_cancelled"
			}
			if e = s.event(tx, eventType, before, sess); e != nil {
				return e
			}
			if eventType == "session_cancelled" && s.reservations != nil {
				if e = s.reservations.CancelSessionReservations(tx, uc.OrganizationID, unit, sess.ID, actor, "Alteração da série"); e != nil {
					return e
				}
			}
			changed = append(changed, sess)
		}
		for _, o := range byDate {
			sess := sessionFromOccurrence(next, o, actor, s.now())
			if e = s.repo.CreateSession(tx, &sess); e != nil {
				return e
			}
			if e = s.event(tx, "session_published", domain.ScheduleSession{}, sess); e != nil {
				return e
			}
			changed = append(changed, sess)
		}
		if e = s.repo.UpdateSeries(tx, &next); e != nil {
			return e
		}
		if e = s.audit(tx, actor, uc.OrganizationID, unit, "scheduling.series.changed", "series", id, next.Revision, map[string]any{"revision": current.Revision}, map[string]any{"revision": next.Revision, "effective_on": input.EffectiveOn}); e != nil {
			return e
		}
		out = PublishResult{Series: next, Sessions: changed}
		return nil
	})
	return &out, fresh, err
}

func (s *GridService) Materialize(ctx context.Context) (int, error) {
	if !s.materializerOperational {
		return 0, ErrScheduleMaterializerUnavailable
	}
	windowFrom := s.now().UTC().Format("2006-01-02")
	windowUntil := s.now().UTC().AddDate(0, 0, 55).Format("2006-01-02")
	runID, claimed, err := s.repo.ClaimMaterializationRun(ctx, "materialize:"+windowFrom, windowFrom, windowUntil, s.now())
	if err != nil {
		return 0, err
	}
	if !claimed {
		return 0, nil
	}
	series, err := s.repo.ListPublishedSeries(ctx)
	if err != nil {
		_ = s.repo.CompleteMaterializationRun(ctx, runID, "failed", 0, 0, 0, 0, "list_series_failed", s.now())
		return 0, err
	}
	created := 0
	kept := 0
	failures := 0
	for _, listed := range series {
		seriesCreated := 0
		seriesKept := 0
		err = s.tx.WithinTransaction(ctx, func(tx context.Context) error {
			if err := s.repo.LockSeries(tx, listed.OrganizationID, listed.ID); err != nil {
				return err
			}
			v, err := s.repo.GetSeries(tx, listed.OrganizationID, listed.UnitID, listed.ID, true)
			if err != nil {
				return err
			}
			if v.Status != domain.ScheduleSeriesPublished {
				return nil
			}
			loc, err := time.LoadLocation(v.Timezone)
			if err != nil {
				return err
			}
			from := s.now().In(loc).Format("2006-01-02")
			until := s.now().In(loc).AddDate(0, 0, 55).Format("2006-01-02")
			occ, issues, err := domain.ExpandScheduleSeries(*v, from, until)
			if err != nil || len(issues) > 0 {
				return ErrScheduleInvalidLocalTime
			}
			if len(occ) == 0 {
				return nil
			}
			spaces := []domain.ID{}
			if v.SpaceID != nil {
				spaces = append(spaces, *v.SpaceID)
			}
			if err := s.repo.LockResources(tx, v.OrganizationID, []domain.ID{v.ProfessionalID}, spaces); err != nil {
				return err
			}
			existing, err := s.repo.ListSessionsAcrossOrganization(tx, v.OrganizationID, occ[0].StartsAt, occ[len(occ)-1].EndsAt)
			if err != nil {
				return err
			}
			keys := map[string]bool{}
			for _, x := range existing {
				if x.SeriesID != nil && *x.SeriesID == v.ID && x.OccurrenceOn != nil {
					keys[*x.OccurrenceOn] = true
					seriesKept++
				}
			}
			missing := make([]domain.SchedulePreviewOccurrence, 0, len(occ))
			for _, o := range occ {
				if keys[o.LocalDate] {
					continue
				}
				missing = append(missing, o)
			}
			if hasSessionConflict(missing, existing, "") {
				return ErrScheduleConflict
			}
			for _, o := range missing {
				sess := sessionFromOccurrence(*v, o, v.UpdatedByUserID, s.now())
				if err = s.repo.CreateSession(tx, &sess); err != nil {
					return err
				}
				if err = s.event(tx, "session_published", domain.ScheduleSession{}, sess); err != nil {
					return err
				}
				seriesCreated++
			}
			return nil
		})
		if err != nil {
			failures++
			continue
		}
		created += seriesCreated
		kept += seriesKept
	}
	status, errorCode := "completed", ""
	if failures > 0 {
		status, errorCode = "failed", "series_failed"
	}
	if err := s.repo.CompleteMaterializationRun(ctx, runID, status, len(series), created, kept, failures, errorCode, s.now()); err != nil {
		return created, err
	}
	if failures > 0 {
		return created, fmt.Errorf("%d scheduling series failed materialization", failures)
	}
	return created, nil
}

func (s *GridService) GetPublishedSession(ctx context.Context, organizationID, unitID, sessionID domain.ID) (*domain.ScheduleSession, error) {
	return s.repo.GetSession(ctx, organizationID, unitID, sessionID, false)
}
func (s *GridService) ListPublishedSessions(ctx context.Context, organizationID, unitID domain.ID, from, until time.Time) ([]domain.ScheduleSession, error) {
	return s.repo.ListSessions(ctx, organizationID, unitID, from, until, false)
}
func (s *GridService) ListSessionEvents(ctx context.Context, organizationID, unitID domain.ID, after int64, limit int) ([]domain.SessionEventV1, error) {
	return s.repo.ListSessionEvents(ctx, organizationID, unitID, after, limit)
}

func (s *GridService) command(ctx context.Context, org, unit domain.ID, command, key string, seriesID, sessionID domain.ID, fingerprint string, out any, run func(context.Context) error) (bool, error) {
	fresh := false
	err := s.tx.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.repo.LockCommand(tx, org, command, key); err != nil {
			return err
		}
		existing, err := s.repo.FindCommand(tx, org, command, key)
		if err != nil {
			return err
		}
		if existing != nil {
			if existing.Fingerprint != fingerprint {
				return ErrSchedulingIdempotencyKeyReused
			}
			b, _ := json.Marshal(existing.ResponseData)
			return json.Unmarshal(b, out)
		}
		if err = run(tx); err != nil {
			return err
		}
		response, err := structToMap(out)
		if err != nil {
			return err
		}
		request := domain.SchedulingCommandRequest{ID: domain.ID(uuid.NewString()), OrganizationID: org, UnitID: unit, Command: command, IdempotencyKey: key, Fingerprint: fingerprint, SeriesID: seriesID, SessionID: sessionID, ResponseData: response, CreatedAt: s.now()}
		if err = s.repo.CreateCommand(tx, &request); err != nil {
			return err
		}
		fresh = true
		return nil
	})
	return fresh, err
}
func fingerprintValue(v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func validDate(v string) bool { _, err := time.Parse("2006-01-02", v); return err == nil }
func validPreviewRange(from, until string) bool {
	f, e := time.Parse("2006-01-02", from)
	if e != nil {
		return false
	}
	u, e := time.Parse("2006-01-02", until)
	return e == nil && !u.Before(f) && int(u.Sub(f).Hours()/24)+1 <= 56
}

func (s *GridService) buildSession(ctx context.Context, org, unit, id domain.ID, seriesID *domain.ID, origin string, occurrence *string, actor domain.ID, input SessionMutationInput) (*domain.ScheduleSession, error) {
	if input.DurationMinutes < 5 || input.DurationMinutes > 1440 || input.StartsAt.IsZero() {
		return nil, ErrScheduleSeriesInvalid
	}
	series := domain.ScheduleSeries{OrganizationID: org, UnitID: unit, TargetKind: input.TargetKind, TargetID: input.TargetID, ProfessionalID: input.ProfessionalID, SpaceID: input.SpaceID, DeliveryMode: input.DeliveryMode, Capacity: input.Capacity, DurationMinutes: input.DurationMinutes, Weekdays: []int{0}, LocalStartTime: "00:00", Timezone: input.Timezone, EffectiveFrom: "2000-01-01"}
	if domain.ValidateScheduleSeries(series) != nil {
		return nil, ErrScheduleSeriesInvalid
	}
	if _, err := s.resolve(ctx, &series, true); err != nil {
		return nil, err
	}
	loc, _ := time.LoadLocation(input.Timezone)
	local := input.StartsAt.In(loc)
	now := s.now()
	return &domain.ScheduleSession{ID: id, OrganizationID: org, UnitID: unit, SeriesID: seriesID, Origin: origin, OccurrenceOn: occurrence, Status: domain.ScheduleSessionScheduled, TargetKind: input.TargetKind, TargetID: input.TargetID, TargetName: series.ReferenceHealth.TargetName, ProfessionalID: input.ProfessionalID, ProfessionalName: series.ReferenceHealth.ProfessionalName, SpaceID: input.SpaceID, SpaceName: series.ReferenceHealth.SpaceName, DeliveryMode: input.DeliveryMode, Capacity: input.Capacity, StartsAt: input.StartsAt.UTC(), EndsAt: input.StartsAt.Add(time.Duration(input.DurationMinutes) * time.Minute).UTC(), LocalDate: local.Format("2006-01-02"), LocalStartTime: local.Format("15:04"), Timezone: input.Timezone, ManualOverride: origin == "series", Revision: 1, PublishedAt: now, CreatedByUserID: actor, UpdatedByUserID: actor, CreatedAt: now, UpdatedAt: now}, nil
}
func (s *GridService) lockAndCheckSession(ctx context.Context, value domain.ScheduleSession, exclude domain.ID) error {
	spaces := []domain.ID{}
	if value.SpaceID != nil {
		spaces = append(spaces, *value.SpaceID)
	}
	if err := s.repo.LockResources(ctx, value.OrganizationID, []domain.ID{value.ProfessionalID}, spaces); err != nil {
		return err
	}
	existing, err := s.repo.ListSessionsAcrossOrganization(ctx, value.OrganizationID, value.StartsAt, value.EndsAt)
	if err != nil {
		return err
	}
	occ := domain.SchedulePreviewOccurrence{OccurrenceKey: string(value.ID), StartsAt: value.StartsAt, EndsAt: value.EndsAt, ProfessionalID: value.ProfessionalID, SpaceID: value.SpaceID}
	if hasSessionConflict([]domain.SchedulePreviewOccurrence{occ}, existing, exclude) {
		return ErrScheduleConflict
	}
	return nil
}
func hasSessionConflict(candidates []domain.SchedulePreviewOccurrence, existing []domain.ScheduleSession, exclude domain.ID) bool {
	for _, a := range candidates {
		for _, b := range existing {
			if b.ID == exclude || b.Status != domain.ScheduleSessionScheduled || !a.StartsAt.Before(b.EndsAt) || !a.EndsAt.After(b.StartsAt) {
				continue
			}
			if a.ProfessionalID == b.ProfessionalID {
				return true
			}
			if a.SpaceID != nil && b.SpaceID != nil && *a.SpaceID == *b.SpaceID && b.UnitID != "" {
				return true
			}
		}
	}
	return false
}
func sessionFromOccurrence(series domain.ScheduleSeries, o domain.SchedulePreviewOccurrence, actor domain.ID, now time.Time) domain.ScheduleSession {
	id := domain.ID(uuid.NewString())
	sid := series.ID
	date := o.LocalDate
	revision := series.Revision
	return domain.ScheduleSession{ID: id, OrganizationID: series.OrganizationID, UnitID: series.UnitID, SeriesID: &sid, Origin: "series", OccurrenceOn: &date, Status: domain.ScheduleSessionScheduled, TargetKind: series.TargetKind, TargetID: series.TargetID, TargetName: series.TargetNameSnapshot, ProfessionalID: series.ProfessionalID, ProfessionalName: series.ProfessionalNameSnapshot, SpaceID: series.SpaceID, SpaceName: series.SpaceNameSnapshot, DeliveryMode: series.DeliveryMode, Capacity: series.Capacity, StartsAt: o.StartsAt, EndsAt: o.EndsAt, LocalDate: o.LocalDate, LocalStartTime: o.LocalStartTime, Timezone: o.Timezone, SourceSeriesRevision: &revision, Revision: 1, PublishedAt: now, CreatedByUserID: actor, UpdatedByUserID: actor, CreatedAt: now, UpdatedAt: now}
}
func applyOccurrence(session *domain.ScheduleSession, series domain.ScheduleSeries, o domain.SchedulePreviewOccurrence) {
	session.TargetKind = series.TargetKind
	session.TargetID = series.TargetID
	session.TargetName = series.TargetNameSnapshot
	session.ProfessionalID = series.ProfessionalID
	session.ProfessionalName = series.ProfessionalNameSnapshot
	session.SpaceID = series.SpaceID
	session.SpaceName = series.SpaceNameSnapshot
	session.DeliveryMode = series.DeliveryMode
	session.Capacity = series.Capacity
	session.StartsAt = o.StartsAt
	session.EndsAt = o.EndsAt
	session.LocalDate = o.LocalDate
	session.LocalStartTime = o.LocalStartTime
	session.Timezone = o.Timezone
	session.SourceSeriesRevision = &series.Revision
}
func (s *GridService) audit(ctx context.Context, actor, org, unit domain.ID, action, entity string, id domain.ID, revision int, before, after map[string]any) error {
	return s.repo.CreateAuditEvent(ctx, &domain.SchedulingAuditEvent{ID: domain.ID(uuid.NewString()), OrganizationID: org, UnitID: unit, ActorUserID: actor, Action: action, EntityType: entity, EntityID: id, Revision: revision, ChangedFields: []string{"status", "revision"}, BeforeData: before, AfterData: after, CreatedAt: s.now()})
}
func (s *GridService) auditReason(ctx context.Context, actor, org, unit domain.ID, action, entity string, id domain.ID, revision int, reason string) error {
	return s.repo.CreateAuditEvent(ctx, &domain.SchedulingAuditEvent{ID: domain.ID(uuid.NewString()), OrganizationID: org, UnitID: unit, ActorUserID: actor, Action: action, EntityType: entity, EntityID: id, Revision: revision, ChangedFields: []string{"status", "revision"}, BeforeData: map[string]any{}, AfterData: map[string]any{"revision": revision}, Reason: reason, CreatedAt: s.now()})
}
func sessionState(v domain.ScheduleSession) map[string]any {
	return map[string]any{"session_id": v.ID, "revision": v.Revision, "organization_id": v.OrganizationID, "unit_id": v.UnitID, "target_kind": v.TargetKind, "target_id": v.TargetID, "professional_id": v.ProfessionalID, "space_id": v.SpaceID, "starts_at": v.StartsAt, "ends_at": v.EndsAt, "delivery_mode": v.DeliveryMode, "capacity": v.Capacity, "status": v.Status, "origin": v.Origin}
}
func (s *GridService) event(ctx context.Context, eventType string, before, after domain.ScheduleSession) error {
	value := after
	if value.ID == "" {
		value = before
	}
	return s.repo.CreateSessionEvent(ctx, &domain.SessionEventV1{ID: domain.ID(uuid.NewString()), OrganizationID: value.OrganizationID, UnitID: value.UnitID, SessionID: value.ID, SessionRevision: value.Revision, EventType: eventType, BeforeState: sessionState(before), AfterState: sessionState(after), OccurredAt: s.now(), SchemaVersion: 1})
}

var _ portrepo.PublishedSessionReader = (*GridService)(nil)
