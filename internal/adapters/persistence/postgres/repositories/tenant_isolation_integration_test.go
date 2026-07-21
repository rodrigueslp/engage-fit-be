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
	portrepo "boxengage/backend/internal/ports/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestStudentRepositoryIsolatesTenants(t *testing.T) {
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
	boxOne := uuid.NewString()
	boxTwo := uuid.NewString()
	studentOne := uuid.NewString()
	studentTwo := uuid.NewString()
	for _, box := range []models.BoxModel{
		{ID: boxOne, Name: "tenant-one", RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now},
		{ID: boxTwo, Name: "tenant-two", RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now},
	} {
		if err := db.Create(&box).Error; err != nil {
			t.Fatal(err)
		}
	}
	defer db.Where("id IN ?", []string{boxOne, boxTwo}).Delete(&models.BoxModel{})
	for _, student := range []models.StudentModel{
		{ID: studentOne, BoxID: boxOne, Name: "Student One", Source: "totalpass", ExternalID: "same-external", RiskStatus: "active", ContactStatus: "unknown", CreatedAt: now, UpdatedAt: now},
		{ID: studentTwo, BoxID: boxTwo, Name: "Student Two", Source: "totalpass", ExternalID: "same-external", RiskStatus: "active", ContactStatus: "unknown", CreatedAt: now, UpdatedAt: now},
	} {
		if err := db.Create(&student).Error; err != nil {
			t.Fatal(err)
		}
	}

	repository := NewStudentGormRepository(db)
	students, err := repository.List(context.Background(), domain.ID(boxOne), portrepo.StudentFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(students) != 1 || students[0].ID != domain.ID(studentOne) {
		t.Fatalf("tenant list leaked data: %+v", students)
	}
	if _, err := repository.FindByID(context.Background(), domain.ID(boxOne), domain.ID(studentTwo)); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("cross-tenant lookup should be hidden as not found, got %v", err)
	}
	if err := repository.UpdateRiskStatus(context.Background(), domain.ID(boxOne), domain.ID(studentTwo), domain.StudentRiskStatusPaused); err != nil {
		t.Fatal(err)
	}
	var persisted models.StudentModel
	if err := db.Where("id = ?", studentTwo).First(&persisted).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.RiskStatus != "active" {
		t.Fatalf("cross-tenant update changed another tenant: %s", persisted.RiskStatus)
	}
}

func TestRewardRepositoryIsolatesTenants(t *testing.T) {
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
	campaignOne, campaignTwo := uuid.NewString(), uuid.NewString()
	rewardOne, rewardTwo := uuid.NewString(), uuid.NewString()
	for _, box := range []models.BoxModel{
		{ID: boxOne, Name: "reward-tenant-one", RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now},
		{ID: boxTwo, Name: "reward-tenant-two", RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now},
	} {
		if err := db.Create(&box).Error; err != nil {
			t.Fatal(err)
		}
	}
	defer db.Where("id IN ?", []string{boxOne, boxTwo}).Delete(&models.BoxModel{})
	for _, campaign := range []models.CampaignModel{
		{ID: campaignOne, BoxID: boxOne, Name: "Campaign One", StartDate: now, EndDate: now.AddDate(0, 1, 0), Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: campaignTwo, BoxID: boxTwo, Name: "Campaign Two", StartDate: now, EndDate: now.AddDate(0, 1, 0), Active: true, CreatedAt: now, UpdatedAt: now},
	} {
		if err := db.Create(&campaign).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, reward := range []models.RewardModel{
		{ID: rewardOne, CampaignID: campaignOne, Name: "Reward One", Quantity: 1},
		{ID: rewardTwo, CampaignID: campaignTwo, Name: "Reward Two", Quantity: 2},
	} {
		if err := db.Create(&reward).Error; err != nil {
			t.Fatal(err)
		}
	}

	repository := NewRewardGormRepository(db)
	items, err := repository.ListByCampaign(context.Background(), domain.ID(boxOne), domain.ID(campaignTwo))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("cross-tenant reward list leaked data: %+v", items)
	}
	if _, err := repository.FindByID(context.Background(), domain.ID(boxOne), domain.ID(rewardTwo)); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("cross-tenant reward lookup should be hidden as not found, got %v", err)
	}
	if err := repository.Update(context.Background(), domain.ID(boxOne), domain.Reward{ID: domain.ID(rewardTwo), CampaignID: domain.ID(campaignTwo), Name: "Changed", Quantity: 99}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("cross-tenant update should be hidden as not found, got %v", err)
	}
	if err := repository.Delete(context.Background(), domain.ID(boxOne), domain.ID(rewardTwo)); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("cross-tenant delete should be hidden as not found, got %v", err)
	}
	var persisted models.RewardModel
	if err := db.Where("id = ?", rewardTwo).First(&persisted).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.Name != "Reward Two" || persisted.Quantity != 2 {
		t.Fatalf("cross-tenant update changed another tenant: %+v", persisted)
	}
}
