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
