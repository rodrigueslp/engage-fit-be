package repositories

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres"
	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestRetentionRepositoryIsolatesInterventionsByTenant(t *testing.T) {
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
	boxOne, boxTwo := uuid.NewString(), uuid.NewString()
	studentOne, studentTwo := uuid.NewString(), uuid.NewString()
	for _, box := range []models.BoxModel{
		{ID: boxOne, Name: "retention-one", RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now},
		{ID: boxTwo, Name: "retention-two", RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now},
	} {
		if err := db.Create(&box).Error; err != nil {
			t.Fatal(err)
		}
	}
	defer db.Where("id IN ?", []string{boxOne, boxTwo}).Delete(&models.BoxModel{})
	for _, student := range []models.StudentModel{
		{ID: studentOne, BoxID: boxOne, Name: "One", Source: "wellhub", RiskStatus: "active", ContactStatus: "unknown", CreatedAt: now, UpdatedAt: now},
		{ID: studentTwo, BoxID: boxTwo, Name: "Two", Source: "wellhub", RiskStatus: "active", ContactStatus: "unknown", CreatedAt: now, UpdatedAt: now},
	} {
		if err := db.Create(&student).Error; err != nil {
			t.Fatal(err)
		}
	}

	repository := NewRetentionGormRepository(db)
	item := domain.RetentionIntervention{BoxID: domain.ID(boxTwo), StudentID: domain.ID(studentTwo), Channel: "in_person", Status: "completed", Outcome: "contacted", CompletedAt: &now, CreatedAt: now, UpdatedAt: now}
	if err := repository.SaveIntervention(context.Background(), &item); err != nil {
		t.Fatal(err)
	}

	if _, err := repository.FindIntervention(context.Background(), domain.ID(boxOne), item.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("cross-tenant lookup should be hidden as not found, got %v", err)
	}
	items, err := repository.ListInterventions(context.Background(), domain.ID(boxOne), domain.ID(studentTwo))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("cross-tenant list leaked interventions: %+v", items)
	}
}
