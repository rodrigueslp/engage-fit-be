package repositories

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres"
	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	"github.com/google/uuid"
)

func TestAutomationScheduleClaimIsAtomic(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	boxID := uuid.NewString()
	scheduleID := uuid.NewString()
	now := time.Now().UTC()
	if err := db.Create(&models.BoxModel{ID: boxID, Name: "claim-test", RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Where("id = ?", boxID).Delete(&models.BoxModel{})
	if err := db.Create(&models.AutomationScheduleModel{ID: scheduleID, BoxID: boxID, Name: "claim", Mode: "recalculate_only", RunTime: "08:00", Timezone: "UTC", DaysOfWeek: "0,1,2,3,4,5,6", Enabled: true, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}

	repository := NewAutomationGormRepository(db)
	slot := now.Truncate(time.Minute)
	var claimed atomic.Int64
	var wait sync.WaitGroup
	for range 20 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			ok, err := repository.ClaimSchedule(context.Background(), domain.ID(boxID), domain.ID(scheduleID), slot)
			if err != nil {
				t.Errorf("claim failed: %v", err)
				return
			}
			if ok {
				claimed.Add(1)
			}
		}()
	}
	wait.Wait()
	if claimed.Load() != 1 {
		t.Fatalf("expected exactly one claim, got %d", claimed.Load())
	}
	if ok, err := repository.ClaimSchedule(context.Background(), domain.ID(boxID), domain.ID(scheduleID), slot.Add(time.Minute)); err != nil || !ok {
		t.Fatalf("next minute should be claimable: ok=%v err=%v", ok, err)
	}
}

func TestAutomationRepositorySkipsSuspendedBoxes(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	now := time.Now().UTC()
	activeBoxID, suspendedBoxID := uuid.NewString(), uuid.NewString()
	activeScheduleID, suspendedScheduleID := uuid.NewString(), uuid.NewString()
	for _, box := range []models.BoxModel{
		{ID: activeBoxID, Name: "active-automation", Status: string(domain.BoxStatusActive), RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now},
		{ID: suspendedBoxID, Name: "suspended-automation", Status: string(domain.BoxStatusSuspended), RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now},
	} {
		if err := db.Create(&box).Error; err != nil {
			t.Fatal(err)
		}
	}
	defer db.Where("id IN ?", []string{activeBoxID, suspendedBoxID}).Delete(&models.BoxModel{})
	for _, schedule := range []models.AutomationScheduleModel{
		{ID: activeScheduleID, BoxID: activeBoxID, Name: "active", Mode: "recalculate_only", RunTime: "08:00", Timezone: "UTC", DaysOfWeek: "0,1,2,3,4,5,6", Enabled: true, CreatedAt: now, UpdatedAt: now},
		{ID: suspendedScheduleID, BoxID: suspendedBoxID, Name: "suspended", Mode: "recalculate_only", RunTime: "08:00", Timezone: "UTC", DaysOfWeek: "0,1,2,3,4,5,6", Enabled: true, CreatedAt: now, UpdatedAt: now},
	} {
		if err := db.Create(&schedule).Error; err != nil {
			t.Fatal(err)
		}
	}

	repository := NewAutomationGormRepository(db)
	schedules, err := repository.ListEnabledSchedules(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	activeFound, suspendedFound := false, false
	for _, schedule := range schedules {
		activeFound = activeFound || schedule.ID == domain.ID(activeScheduleID)
		suspendedFound = suspendedFound || schedule.ID == domain.ID(suspendedScheduleID)
	}
	if !activeFound || suspendedFound {
		t.Fatalf("expected only active box schedule, active=%v suspended=%v", activeFound, suspendedFound)
	}
	claimed, err := repository.ClaimSchedule(context.Background(), domain.ID(suspendedBoxID), domain.ID(suspendedScheduleID), now.Truncate(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if claimed {
		t.Fatal("suspended box schedule must not be claimable")
	}
}
