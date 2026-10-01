package repositories

import (
	"context"
	"os"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres"
	"boxengage/backend/internal/domain"
	"boxengage/backend/migrations"
	"github.com/google/uuid"
)

func TestTodayOperationsProjectionRunsAgainstMigratedPostgresAndScopesEmptyTenant(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	if _, err = migrations.New(sqlDB).Up(context.Background()); err != nil {
		t.Fatal(err)
	}
	from := time.Date(2026, 9, 16, 3, 0, 0, 0, time.UTC)
	result, err := NewTodayOperationsGormRepository(db).ReadToday(context.Background(), domain.ID(uuid.NewString()), domain.ID(uuid.NewString()), "2026-09-16", from, from.Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Sessions) != 0 || len(result.Charges) != 0 || len(result.PeopleActions) != 0 || result.AttendanceTotal != 0 || result.DeniedAccessTotal != 0 {
		t.Fatalf("unexpected cross-tenant data: %+v", result)
	}
}
