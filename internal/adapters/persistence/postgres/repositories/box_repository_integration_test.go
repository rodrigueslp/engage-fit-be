package repositories

import (
	"context"
	"os"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres"
	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	"github.com/google/uuid"
)

func TestBoxRepositoryCreatesOrganizationAndOwnerAtomically(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	box := &domain.Box{Name: "Organization Foundation", Status: domain.BoxStatusActive, CreatedAt: now, UpdatedAt: now}
	owner := &domain.User{
		Name: "Owner", Email: uuid.NewString() + "@example.com", PasswordHash: "hash", AuthVersion: 1,
		Role: domain.UserRoleOwner, Active: true, CreatedAt: now, UpdatedAt: now,
	}

	repository := NewBoxGormRepository(db)
	if err := repository.SaveWithOwner(context.Background(), box, owner); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Exec("DELETE FROM boxes WHERE id = ?", string(box.ID)).Error
		_ = db.Exec("DELETE FROM organizations WHERE id = ?", string(box.OrganizationID)).Error
	})

	if box.ID == "" || box.OrganizationID != box.ID || owner.BoxID != box.ID {
		t.Fatalf("unexpected identifiers: box=%+v owner=%+v", box, owner)
	}

	var organization models.OrganizationModel
	if err := db.Where("id = ?", string(box.OrganizationID)).First(&organization).Error; err != nil {
		t.Fatal(err)
	}
	if organization.Name != box.Name || organization.Status != string(domain.OrganizationStatusActive) {
		t.Fatalf("unexpected organization: %+v", organization)
	}

	loaded, err := repository.FindByID(context.Background(), box.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.OrganizationID != box.OrganizationID {
		t.Fatalf("organization id = %q, want %q", loaded.OrganizationID, box.OrganizationID)
	}
}

func TestLegacyBoxInsertReceivesOrganization(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	boxID := uuid.NewString()
	legacy := &models.BoxModel{
		ID: boxID, Name: "Legacy Box Insert", Status: "active", RiskInactiveDays: 7,
		RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Omit("OrganizationID").Create(legacy).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Exec("DELETE FROM boxes WHERE id = ?", boxID).Error
		_ = db.Exec("DELETE FROM organizations WHERE id = ?", boxID).Error
	})

	var persisted models.BoxModel
	if err := db.Where("id = ?", boxID).First(&persisted).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.OrganizationID == nil || *persisted.OrganizationID != boxID {
		t.Fatalf("organization id = %v, want %q", persisted.OrganizationID, boxID)
	}
}

func TestBoxRepositoryRollsBackOrganizationWhenOwnerCreationFails(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	duplicateEmail := uuid.NewString() + "@example.com"
	existingBox := &domain.Box{Name: "Existing Organization", Status: domain.BoxStatusActive, CreatedAt: now, UpdatedAt: now}
	existingOwner := &domain.User{
		Name: "Existing Owner", Email: duplicateEmail, PasswordHash: "hash", AuthVersion: 1,
		Role: domain.UserRoleOwner, Active: true, CreatedAt: now, UpdatedAt: now,
	}
	repository := NewBoxGormRepository(db)
	if err := repository.SaveWithOwner(context.Background(), existingBox, existingOwner); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Exec("DELETE FROM boxes WHERE id = ?", string(existingBox.ID)).Error
		_ = db.Exec("DELETE FROM organizations WHERE id = ?", string(existingBox.OrganizationID)).Error
	})

	failedBox := &domain.Box{Name: "Rolled Back Organization", Status: domain.BoxStatusActive, CreatedAt: now, UpdatedAt: now}
	duplicateOwner := &domain.User{
		Name: "Duplicate Owner", Email: duplicateEmail, PasswordHash: "hash", AuthVersion: 1,
		Role: domain.UserRoleOwner, Active: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := repository.SaveWithOwner(context.Background(), failedBox, duplicateOwner); err == nil {
		t.Fatal("expected duplicate owner email to fail")
	}

	var boxCount int64
	if err := db.Model(&models.BoxModel{}).Where("id = ?", string(failedBox.ID)).Count(&boxCount).Error; err != nil {
		t.Fatal(err)
	}
	var organizationCount int64
	if err := db.Model(&models.OrganizationModel{}).Where("id = ?", string(failedBox.OrganizationID)).Count(&organizationCount).Error; err != nil {
		t.Fatal(err)
	}
	if boxCount != 0 || organizationCount != 0 {
		t.Fatalf("transaction left rows behind: boxes=%d organizations=%d", boxCount, organizationCount)
	}
}
