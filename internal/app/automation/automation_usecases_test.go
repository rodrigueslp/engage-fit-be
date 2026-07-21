package automation

import (
	"errors"
	"testing"
	"time"

	"boxengage/backend/internal/domain"
)

func TestNormalizeAndValidateSchedule(t *testing.T) {
	schedule := domain.AutomationSchedule{
		Name:       "  Rotina diaria  ",
		Mode:       ScheduleModeFullDaily,
		RunTime:    "08:30",
		Timezone:   "America/Sao_Paulo",
		DaysOfWeek: "5,1,1,3",
	}
	if err := normalizeAndValidateSchedule(&schedule); err != nil {
		t.Fatal(err)
	}
	if schedule.Name != "Rotina diaria" || schedule.DaysOfWeek != "1,3,5" {
		t.Fatalf("schedule was not normalized: %+v", schedule)
	}
}

func TestNormalizeAndValidateScheduleRejectsInvalidValues(t *testing.T) {
	tests := []domain.AutomationSchedule{
		{Name: "", Mode: ScheduleModeFullDaily, RunTime: "08:00", Timezone: defaultTimezone},
		{Name: "x", Mode: "unknown", RunTime: "08:00", Timezone: defaultTimezone},
		{Name: "x", Mode: ScheduleModeFullDaily, RunTime: "25:00", Timezone: defaultTimezone},
		{Name: "x", Mode: ScheduleModeFullDaily, RunTime: "08:00", Timezone: "Mars/Olympus"},
		{Name: "x", Mode: ScheduleModeFullDaily, RunTime: "08:00", Timezone: defaultTimezone, DaysOfWeek: "1,7"},
	}
	for _, schedule := range tests {
		if err := normalizeAndValidateSchedule(&schedule); !errors.Is(err, ErrInvalidSchedule) {
			t.Fatalf("expected invalid schedule for %+v, got %v", schedule, err)
		}
	}
}

func TestIsScheduleDueUsesConfiguredTimezoneAndMinute(t *testing.T) {
	now := time.Date(2026, 7, 20, 11, 30, 0, 0, time.UTC) // 08:30 in Sao Paulo.
	schedule := domain.AutomationSchedule{Enabled: true, RunTime: "08:30", Timezone: defaultTimezone, DaysOfWeek: "1"}
	if !IsScheduleDue(schedule, now) {
		t.Fatal("expected schedule to be due")
	}
	lastRun := now.Add(20 * time.Second)
	schedule.LastRunAt = &lastRun
	if IsScheduleDue(schedule, now) {
		t.Fatal("expected same-minute execution to be suppressed")
	}
}

func TestIsScheduleDueWithinAllowsBoundedCatchup(t *testing.T) {
	schedule := domain.AutomationSchedule{Enabled: true, RunTime: "08:30", Timezone: defaultTimezone, DaysOfWeek: "1"}
	fiveMinutesLate := time.Date(2026, 7, 20, 11, 35, 0, 0, time.UTC)
	if !IsScheduleDueWithin(schedule, fiveMinutesLate, 15*time.Minute) {
		t.Fatal("expected execution inside catchup window")
	}
	twentyMinutesLate := time.Date(2026, 7, 20, 11, 50, 0, 0, time.UTC)
	if IsScheduleDueWithin(schedule, twentyMinutesLate, 15*time.Minute) {
		t.Fatal("expected execution outside catchup window to be skipped")
	}
}
