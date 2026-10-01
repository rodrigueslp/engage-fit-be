package domain

import (
	"testing"
	"time"
)

func validSeries() ScheduleSeries {
	return ScheduleSeries{ID: "series-1", TargetKind: ScheduleTargetModality, TargetID: "modality-1", ProfessionalID: "professional-1", DeliveryMode: ScheduleDeliveryGroup, Capacity: 12, DurationMinutes: 60, Weekdays: []int{1, 3, 5}, LocalStartTime: "18:30", Timezone: "America/Sao_Paulo", EffectiveFrom: "2026-09-01"}
}

func TestScheduleSeriesValidationAndExpansion(t *testing.T) {
	value := validSeries()
	if err := ValidateScheduleSeries(value); err != nil {
		t.Fatalf("valid series: %v", err)
	}
	occ, issues, err := ExpandScheduleSeries(value, "2026-09-01", "2026-09-14")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("issues: %+v", issues)
	}
	if len(occ) != 6 {
		t.Fatalf("occurrences=%d want 6", len(occ))
	}
	if occ[0].LocalDate != "2026-09-02" || occ[0].StartsAt.Format(time.RFC3339) != "2026-09-02T21:30:00Z" {
		t.Fatalf("first=%+v", occ[0])
	}
	value.DeliveryMode = ScheduleDeliveryIndividual
	value.Capacity = 2
	if ValidateScheduleSeries(value) == nil {
		t.Fatal("individual capacity > 1 accepted")
	}
	value = validSeries()
	value.Weekdays = []int{1, 1}
	if ValidateScheduleSeries(value) == nil {
		t.Fatal("duplicate weekday accepted")
	}
}

func TestScheduleExpansionRejectsDSTGapAndFold(t *testing.T) {
	value := validSeries()
	value.Timezone = "America/New_York"
	value.Weekdays = []int{0}
	value.LocalStartTime = "02:30"
	value.EffectiveFrom = "2026-03-08"
	until := "2026-03-08"
	value.EffectiveUntil = &until
	occ, issues, err := ExpandScheduleSeries(value, "2026-03-08", "2026-03-08")
	if err != nil {
		t.Fatal(err)
	}
	if len(occ) != 0 || len(issues) != 1 || issues[0].Code != "invalid_local_time" {
		t.Fatalf("gap result occurrences=%d issues=%+v", len(occ), issues)
	}
	value.LocalStartTime = "01:30"
	value.EffectiveFrom = "2026-11-01"
	until = "2026-11-01"
	value.EffectiveUntil = &until
	occ, issues, err = ExpandScheduleSeries(value, "2026-11-01", "2026-11-01")
	if err != nil {
		t.Fatal(err)
	}
	if len(occ) != 0 || len(issues) != 1 {
		t.Fatalf("fold result occurrences=%d issues=%+v", len(occ), issues)
	}
}

func TestScheduleConflictUsesHalfOpenIntervals(t *testing.T) {
	base := SchedulePreviewOccurrence{OccurrenceKey: "a", ProfessionalID: "p", StartsAt: time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC), EndsAt: time.Date(2026, 9, 15, 11, 0, 0, 0, time.UTC)}
	adjacent := base
	adjacent.OccurrenceKey = "b"
	adjacent.StartsAt = base.EndsAt
	adjacent.EndsAt = adjacent.StartsAt.Add(time.Hour)
	if issues := DetectScheduleConflicts([]SchedulePreviewOccurrence{base, adjacent}); len(issues) != 0 {
		t.Fatalf("adjacent intervals conflict: %+v", issues)
	}
	overlap := adjacent
	overlap.StartsAt = base.EndsAt.Add(-time.Minute)
	if issues := DetectScheduleConflicts([]SchedulePreviewOccurrence{base, overlap}); len(issues) != 1 || issues[0].Code != "professional_overlap" {
		t.Fatalf("overlap issues: %+v", issues)
	}
}
