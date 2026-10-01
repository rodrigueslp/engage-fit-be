package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type ScheduleSeriesStatus string
type ScheduleSessionStatus string
type ScheduleTargetKind string
type ScheduleDeliveryMode string

const (
	ScheduleSeriesDraft        ScheduleSeriesStatus  = "draft"
	ScheduleSeriesArchived     ScheduleSeriesStatus  = "archived"
	ScheduleSeriesPublished    ScheduleSeriesStatus  = "published"
	ScheduleSeriesCancelled    ScheduleSeriesStatus  = "cancelled"
	ScheduleSessionScheduled   ScheduleSessionStatus = "scheduled"
	ScheduleSessionCancelled   ScheduleSessionStatus = "cancelled"
	ScheduleTargetModality     ScheduleTargetKind    = "modality"
	ScheduleTargetService      ScheduleTargetKind    = "service_offering"
	ScheduleDeliveryGroup      ScheduleDeliveryMode  = "group"
	ScheduleDeliveryIndividual ScheduleDeliveryMode  = "individual"
)

type ScheduleReference struct {
	ID              ID                 `json:"id"`
	Kind            ScheduleTargetKind `json:"kind,omitempty"`
	DisplayName     string             `json:"display_name"`
	Status          string             `json:"status"`
	EnabledInUnit   bool               `json:"enabled_in_unit"`
	DefaultCapacity *int               `json:"default_capacity,omitempty"`
}

type ScheduleResolvedReferences struct {
	Target ScheduleReference  `json:"target"`
	Space  *ScheduleReference `json:"space,omitempty"`
}

type ScheduleReferenceHealth struct {
	TargetAvailable       bool   `json:"target_available"`
	ProfessionalAvailable bool   `json:"professional_available"`
	SpaceAvailable        bool   `json:"space_available"`
	TargetName            string `json:"target_name"`
	ProfessionalName      string `json:"professional_name"`
	SpaceName             string `json:"space_name,omitempty"`
}

type ScheduleSeries struct {
	ID                       ID                      `json:"id"`
	OrganizationID           ID                      `json:"organization_id"`
	UnitID                   ID                      `json:"unit_id"`
	Status                   ScheduleSeriesStatus    `json:"status"`
	TargetKind               ScheduleTargetKind      `json:"target_kind"`
	TargetID                 ID                      `json:"target_id"`
	TargetNameSnapshot       string                  `json:"target_name_snapshot,omitempty"`
	ProfessionalID           ID                      `json:"professional_id"`
	ProfessionalNameSnapshot string                  `json:"professional_name_snapshot,omitempty"`
	SpaceID                  *ID                     `json:"space_id,omitempty"`
	SpaceNameSnapshot        string                  `json:"space_name_snapshot,omitempty"`
	DeliveryMode             ScheduleDeliveryMode    `json:"delivery_mode"`
	Capacity                 int                     `json:"capacity"`
	DurationMinutes          int                     `json:"duration_minutes"`
	Weekdays                 []int                   `json:"weekdays"`
	LocalStartTime           string                  `json:"local_start_time"`
	Timezone                 string                  `json:"timezone"`
	EffectiveFrom            string                  `json:"effective_from"`
	EffectiveUntil           *string                 `json:"effective_until,omitempty"`
	Revision                 int                     `json:"revision"`
	PublishedAt              *time.Time              `json:"published_at,omitempty"`
	PublishedByUserID        *ID                     `json:"published_by_user_id,omitempty"`
	CancelledAt              *time.Time              `json:"cancelled_at,omitempty"`
	CancelledByUserID        *ID                     `json:"cancelled_by_user_id,omitempty"`
	CancelledReason          string                  `json:"cancelled_reason,omitempty"`
	ArchivedAt               *time.Time              `json:"archived_at,omitempty"`
	ArchivedByUserID         *ID                     `json:"archived_by_user_id,omitempty"`
	CreatedByUserID          ID                      `json:"created_by_user_id"`
	UpdatedByUserID          ID                      `json:"updated_by_user_id"`
	CreatedAt                time.Time               `json:"created_at"`
	UpdatedAt                time.Time               `json:"updated_at"`
	ReferenceHealth          ScheduleReferenceHealth `json:"reference_health"`
}

type ScheduleSession struct {
	ID                    ID                    `json:"session_id"`
	OrganizationID        ID                    `json:"organization_id"`
	UnitID                ID                    `json:"unit_id"`
	SeriesID              *ID                   `json:"series_id,omitempty"`
	Origin                string                `json:"origin"`
	OccurrenceOn          *string               `json:"occurrence_on,omitempty"`
	Status                ScheduleSessionStatus `json:"status"`
	TargetKind            ScheduleTargetKind    `json:"target_kind"`
	TargetID              ID                    `json:"target_id"`
	TargetName            string                `json:"target_name"`
	ProfessionalID        ID                    `json:"professional_id"`
	ProfessionalName      string                `json:"professional_name"`
	AssignedToCurrentUser bool                  `json:"assigned_to_current_user"`
	SpaceID               *ID                   `json:"space_id,omitempty"`
	SpaceName             string                `json:"space_name,omitempty"`
	DeliveryMode          ScheduleDeliveryMode  `json:"delivery_mode"`
	Capacity              int                   `json:"capacity"`
	StartsAt              time.Time             `json:"starts_at"`
	EndsAt                time.Time             `json:"ends_at"`
	LocalDate             string                `json:"local_date"`
	LocalStartTime        string                `json:"local_start_time"`
	Timezone              string                `json:"timezone"`
	SourceSeriesRevision  *int                  `json:"source_series_revision,omitempty"`
	ManualOverride        bool                  `json:"manual_override"`
	Revision              int                   `json:"revision"`
	PublishedAt           time.Time             `json:"published_at"`
	CancelledAt           *time.Time            `json:"cancelled_at,omitempty"`
	CancelledByUserID     *ID                   `json:"cancelled_by_user_id,omitempty"`
	CancelledReason       string                `json:"cancelled_reason,omitempty"`
	CreatedByUserID       ID                    `json:"created_by_user_id"`
	UpdatedByUserID       ID                    `json:"updated_by_user_id"`
	CreatedAt             time.Time             `json:"created_at"`
	UpdatedAt             time.Time             `json:"updated_at"`
}

type SchedulePreviewOccurrence struct {
	OccurrenceKey    string               `json:"occurrence_key"`
	SeriesID         *ID                  `json:"series_id,omitempty"`
	LocalDate        string               `json:"local_date"`
	LocalStartTime   string               `json:"local_start_time"`
	Timezone         string               `json:"timezone"`
	StartsAt         time.Time            `json:"starts_at"`
	EndsAt           time.Time            `json:"ends_at"`
	TargetKind       ScheduleTargetKind   `json:"target_kind"`
	TargetID         ID                   `json:"target_id"`
	TargetName       string               `json:"target_name"`
	ProfessionalID   ID                   `json:"professional_id"`
	ProfessionalName string               `json:"professional_name"`
	SpaceID          *ID                  `json:"space_id,omitempty"`
	SpaceName        string               `json:"space_name,omitempty"`
	DeliveryMode     ScheduleDeliveryMode `json:"delivery_mode"`
	Capacity         int                  `json:"capacity"`
}

type SchedulePreviewIssue struct {
	Code          string `json:"code"`
	ResourceType  string `json:"resource_type,omitempty"`
	ResourceID    ID     `json:"resource_id,omitempty"`
	OccurrenceKey string `json:"occurrence_key,omitempty"`
	ConflictsWith string `json:"conflicts_with,omitempty"`
}

type SchedulePreview struct {
	PreviewFrom               string                      `json:"preview_from"`
	PreviewUntil              string                      `json:"preview_until"`
	GeneratedAt               time.Time                   `json:"generated_at"`
	Fingerprint               string                      `json:"fingerprint"`
	ReadyForFuturePublication bool                        `json:"ready_for_future_publication"`
	Occurrences               []SchedulePreviewOccurrence `json:"occurrences"`
	Issues                    []SchedulePreviewIssue      `json:"issues"`
}

type SessionEventV1 struct {
	Sequence        int64          `json:"sequence"`
	ID              ID             `json:"event_id"`
	OrganizationID  ID             `json:"organization_id"`
	UnitID          ID             `json:"unit_id"`
	SessionID       ID             `json:"session_id"`
	SessionRevision int            `json:"session_revision"`
	EventType       string         `json:"event_type"`
	BeforeState     map[string]any `json:"before_state"`
	AfterState      map[string]any `json:"after_state"`
	OccurredAt      time.Time      `json:"occurred_at"`
	SchemaVersion   int            `json:"schema_version"`
}

func ValidateScheduleSeries(value ScheduleSeries) error {
	if value.TargetKind != ScheduleTargetModality && value.TargetKind != ScheduleTargetService {
		return fmt.Errorf("invalid target")
	}
	if value.TargetID == "" || value.ProfessionalID == "" {
		return fmt.Errorf("missing reference")
	}
	if value.DeliveryMode != ScheduleDeliveryGroup && value.DeliveryMode != ScheduleDeliveryIndividual {
		return fmt.Errorf("invalid mode")
	}
	if value.Capacity < 1 || value.Capacity > 10000 || (value.DeliveryMode == ScheduleDeliveryIndividual && value.Capacity != 1) {
		return fmt.Errorf("invalid capacity")
	}
	if value.DurationMinutes < 5 || value.DurationMinutes > 1440 {
		return fmt.Errorf("invalid duration")
	}
	if _, err := time.LoadLocation(value.Timezone); err != nil || strings.Contains(value.Timezone, ":") || !strings.Contains(value.Timezone, "/") {
		return fmt.Errorf("invalid timezone")
	}
	if _, err := time.Parse("15:04", value.LocalStartTime); err != nil || len(value.LocalStartTime) != 5 {
		return fmt.Errorf("invalid time")
	}
	from, err := time.Parse("2006-01-02", value.EffectiveFrom)
	if err != nil {
		return fmt.Errorf("invalid from")
	}
	if value.EffectiveUntil != nil {
		until, err := time.Parse("2006-01-02", *value.EffectiveUntil)
		if err != nil || until.Before(from) {
			return fmt.Errorf("invalid until")
		}
	}
	if len(value.Weekdays) < 1 || len(value.Weekdays) > 7 {
		return fmt.Errorf("invalid weekdays")
	}
	seen := map[int]bool{}
	for _, day := range value.Weekdays {
		if day < 0 || day > 6 || seen[day] {
			return fmt.Errorf("invalid weekdays")
		}
		seen[day] = true
	}
	return nil
}

func NormalizeScheduleSeries(value *ScheduleSeries) {
	sort.Ints(value.Weekdays)
	value.LocalStartTime = strings.TrimSpace(value.LocalStartTime)
	value.Timezone = strings.TrimSpace(value.Timezone)
}

func ScheduleSeriesFingerprint(value ScheduleSeries) string {
	copyValue := value
	copyValue.ID = ""
	copyValue.OrganizationID = ""
	copyValue.UnitID = ""
	copyValue.ReferenceHealth = ScheduleReferenceHealth{}
	copyValue.CreatedAt = time.Time{}
	copyValue.UpdatedAt = time.Time{}
	copyValue.CreatedByUserID = ""
	copyValue.UpdatedByUserID = ""
	NormalizeScheduleSeries(&copyValue)
	b, _ := json.Marshal(copyValue)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func ExpandScheduleSeries(series ScheduleSeries, fromValue, untilValue string) ([]SchedulePreviewOccurrence, []SchedulePreviewIssue, error) {
	if err := ValidateScheduleSeries(series); err != nil {
		return nil, nil, err
	}
	from, err := time.Parse("2006-01-02", fromValue)
	if err != nil {
		return nil, nil, err
	}
	until, err := time.Parse("2006-01-02", untilValue)
	if err != nil || until.Before(from) || int(until.Sub(from).Hours()/24)+1 > 56 {
		return nil, nil, fmt.Errorf("invalid range")
	}
	effectiveFrom, _ := time.Parse("2006-01-02", series.EffectiveFrom)
	if from.Before(effectiveFrom) {
		from = effectiveFrom
	}
	if series.EffectiveUntil != nil {
		effectiveUntil, _ := time.Parse("2006-01-02", *series.EffectiveUntil)
		if until.After(effectiveUntil) {
			until = effectiveUntil
		}
	}
	if until.Before(from) {
		return []SchedulePreviewOccurrence{}, []SchedulePreviewIssue{}, nil
	}
	weekdaySet := map[int]bool{}
	for _, day := range series.Weekdays {
		weekdaySet[day] = true
	}
	result := []SchedulePreviewOccurrence{}
	issues := []SchedulePreviewIssue{}
	for current := from; !current.After(until); current = current.AddDate(0, 0, 1) {
		if !weekdaySet[int(current.Weekday())] {
			continue
		}
		date := current.Format("2006-01-02")
		start, ok := uniqueLocalInstant(date, series.LocalStartTime, series.Timezone)
		keyPrefix := "series:" + string(series.ID)
		if series.ID == "" {
			keyPrefix = "candidate:" + ScheduleSeriesFingerprint(series)
		}
		key := keyPrefix + ":" + date
		if !ok {
			issues = append(issues, SchedulePreviewIssue{Code: "invalid_local_time", OccurrenceKey: key})
			continue
		}
		end := start.Add(time.Duration(series.DurationMinutes) * time.Minute)
		loc, _ := time.LoadLocation(series.Timezone)
		startLocal, endLocal := start.In(loc), end.In(loc)
		if endLocal.Format("2006-01-02") != startLocal.Format("2006-01-02") && endLocal.Format("2006-01-02") != startLocal.AddDate(0, 0, 1).Format("2006-01-02") {
			issues = append(issues, SchedulePreviewIssue{Code: "invalid_local_time", OccurrenceKey: key})
			continue
		}
		var sid *ID
		if series.ID != "" {
			id := series.ID
			sid = &id
		}
		result = append(result, SchedulePreviewOccurrence{OccurrenceKey: key, SeriesID: sid, LocalDate: date, LocalStartTime: series.LocalStartTime, Timezone: series.Timezone, StartsAt: start.UTC(), EndsAt: end.UTC(), TargetKind: series.TargetKind, TargetID: series.TargetID, TargetName: series.ReferenceHealth.TargetName, ProfessionalID: series.ProfessionalID, ProfessionalName: series.ReferenceHealth.ProfessionalName, SpaceID: series.SpaceID, SpaceName: series.ReferenceHealth.SpaceName, DeliveryMode: series.DeliveryMode, Capacity: series.Capacity})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].StartsAt.Equal(result[j].StartsAt) {
			return result[i].OccurrenceKey < result[j].OccurrenceKey
		}
		return result[i].StartsAt.Before(result[j].StartsAt)
	})
	return result, issues, nil
}

func uniqueLocalInstant(dateValue, timeValue, zone string) (time.Time, bool) {
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return time.Time{}, false
	}
	d, err := time.Parse("2006-01-02", dateValue)
	if err != nil {
		return time.Time{}, false
	}
	tm, err := time.Parse("15:04", timeValue)
	if err != nil {
		return time.Time{}, false
	}
	nominal := time.Date(d.Year(), d.Month(), d.Day(), tm.Hour(), tm.Minute(), 0, 0, loc)
	matches := map[int64]time.Time{}
	for delta := -3 * time.Hour; delta <= 3*time.Hour; delta += 30 * time.Minute {
		candidate := nominal.Add(delta).In(loc)
		if candidate.Year() == d.Year() && candidate.Month() == d.Month() && candidate.Day() == d.Day() && candidate.Hour() == tm.Hour() && candidate.Minute() == tm.Minute() {
			matches[candidate.UTC().Unix()] = candidate.UTC()
		}
	}
	if len(matches) != 1 {
		return time.Time{}, false
	}
	for _, match := range matches {
		return match, true
	}
	return time.Time{}, false
}

func DetectScheduleConflicts(occurrences []SchedulePreviewOccurrence) []SchedulePreviewIssue {
	issues := []SchedulePreviewIssue{}
	seen := map[string]bool{}
	for i := 0; i < len(occurrences); i++ {
		for j := i + 1; j < len(occurrences); j++ {
			a, b := occurrences[i], occurrences[j]
			if !a.StartsAt.Before(b.EndsAt) || !a.EndsAt.After(b.StartsAt) {
				continue
			}
			code, resource, id := "", "", ID("")
			if a.ProfessionalID == b.ProfessionalID {
				code, resource, id = "professional_overlap", "professional", a.ProfessionalID
			} else if a.SpaceID != nil && b.SpaceID != nil && *a.SpaceID == *b.SpaceID {
				code, resource, id = "space_overlap", "space", *a.SpaceID
			}
			if code == "" {
				continue
			}
			key := code + ":" + a.OccurrenceKey + ":" + b.OccurrenceKey
			if seen[key] {
				continue
			}
			seen[key] = true
			issues = append(issues, SchedulePreviewIssue{Code: code, ResourceType: resource, ResourceID: id, OccurrenceKey: a.OccurrenceKey, ConflictsWith: b.OccurrenceKey})
		}
	}
	sort.Slice(issues, func(i, j int) bool { return fmt.Sprint(issues[i]) < fmt.Sprint(issues[j]) })
	return issues
}
