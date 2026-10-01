package repositories

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres"
	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/app/catalog"
	"boxengage/backend/internal/app/organizations"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"boxengage/backend/migrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func TestOrganizationRepositoryMultiUnitAccessAndSharedCatalog(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	adminConfig, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	adminDB := stdlib.OpenDB(*adminConfig)
	defer adminDB.Close()
	testDatabase := "spec001_c_repo_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := adminDB.ExecContext(ctx, "CREATE DATABASE "+testDatabase); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if _, err := adminDB.ExecContext(cleanupCtx, "DROP DATABASE IF EXISTS "+testDatabase+" WITH (FORCE)"); err != nil {
			t.Errorf("drop test database: %v", err)
		}
	}()
	testConfig := adminConfig.Copy()
	testConfig.Database = testDatabase
	sqlDB := stdlib.OpenDB(*testConfig)
	if _, err := migrations.New(sqlDB).Up(ctx); err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	testURL, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	testURL.Path = "/" + testDatabase
	db, err := postgres.Open(testURL.String())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	boxRepository := NewBoxGormRepository(db)
	owner := &domain.User{Name: "Owner Multiunit", Email: uuid.NewString() + "@example.test", PasswordHash: "hash", AuthVersion: 1, Role: domain.UserRoleOwner, Active: true, CreatedAt: now, UpdatedAt: now}
	firstUnit := &domain.Box{Name: "Unidade Centro " + uuid.NewString(), Status: domain.BoxStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := boxRepository.SaveWithOwner(ctx, firstUnit, owner); err != nil {
		t.Fatal(err)
	}
	organizationRepository := NewOrganizationGormRepository(db)
	organizationService := organizations.NewService(organizationRepository)
	secondUnit, err := organizationService.CreateUnit(ctx, owner.ID, firstUnit.ID, "Unidade Norte", "America/Manaus")
	if err != nil {
		t.Fatal(err)
	}
	organizationID := firstUnit.OrganizationID

	units, err := organizationService.ListUnits(ctx, owner.ID, firstUnit.ID)
	if err != nil || len(units) != 2 {
		t.Fatalf("accessible units = %+v, %v", units, err)
	}
	access, err := organizationRepository.ResolveUserUnitAccess(ctx, owner.ID, secondUnit.ID)
	if err != nil || access.Organization.ID != organizationID || access.Role != domain.UserRoleOwner {
		t.Fatalf("second unit access = %+v, %v", access, err)
	}

	catalogRepository := NewCatalogGormRepository(db)
	catalogService := catalog.NewService(catalogRepository, NewGormTransactionManager(db))
	modality, err := catalogService.CreateModality(ctx, firstUnit.ID, owner.ID, "Funcional", "")
	if err != nil {
		t.Fatal(err)
	}
	secondUnitModalities, err := catalogService.ListModalities(ctx, secondUnit.ID, secondUnit.ID, false)
	if err != nil || len(secondUnitModalities) != 1 || secondUnitModalities[0].ID != modality.ID || secondUnitModalities[0].ActiveInUnit {
		t.Fatalf("shared modality before activation = %+v, %v", secondUnitModalities, err)
	}
	activated, err := catalogService.SetModalityActive(ctx, secondUnit.ID, secondUnit.ID, owner.ID, modality.ID, true)
	if err != nil || !activated.ActiveInUnit {
		t.Fatalf("shared modality activation = %+v, %v", activated, err)
	}

	coach := &domain.User{BoxID: firstUnit.ID, Name: "Coach Multiunit", Email: uuid.NewString() + "@example.test", PasswordHash: "hash", AuthVersion: 1, Role: domain.UserRoleCoach, Active: true, CreatedAt: now, UpdatedAt: now}
	if err := NewUserGormRepository(db).Save(ctx, coach); err != nil {
		t.Fatal(err)
	}
	if _, err := organizationRepository.ResolveUserUnitAccess(ctx, coach.ID, firstUnit.ID); err != nil {
		t.Fatalf("coach home access: %v", err)
	}
	updatedIDs, err := organizationService.ReplaceMemberUnitAccess(ctx, owner.ID, firstUnit.ID, coach.ID, []domain.ID{secondUnit.ID})
	if err != nil || len(updatedIDs) != 1 || updatedIDs[0] != secondUnit.ID {
		t.Fatalf("replace coach access = %+v, %v", updatedIDs, err)
	}
	if _, err := organizationRepository.ResolveUserUnitAccess(ctx, coach.ID, firstUnit.ID); !errors.Is(err, portrepo.ErrOrganizationAccessNotFound) {
		t.Fatalf("revoked unit remained accessible: %v", err)
	}
	if _, err := organizationRepository.ResolveUserUnitAccess(ctx, coach.ID, secondUnit.ID); err != nil {
		t.Fatalf("granted unit inaccessible: %v", err)
	}
	if err := db.WithContext(ctx).Model(&models.UserModel{}).Where("id = ?", stringID(coach.ID)).Update("active", false).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := organizationRepository.FindOrganizationMember(ctx, organizationID, coach.ID); err != nil {
		t.Fatalf("inactive coach must remain administrable: %v", err)
	}
	if _, err := organizationRepository.ResolveUserUnitAccess(ctx, coach.ID, secondUnit.ID); !errors.Is(err, portrepo.ErrOrganizationAccessNotFound) {
		t.Fatalf("inactive membership remained authorized: %v", err)
	}
	members, err := NewTeamGormRepository(db).ListMembers(ctx, secondUnit.ID)
	if err != nil || len(members) != 2 {
		t.Fatalf("organization-wide team from second unit = %+v, %v", members, err)
	}
	organizationOwner, err := NewUserGormRepository(db).FindOwnerByBoxID(ctx, secondUnit.ID)
	if err != nil || organizationOwner.ID != owner.ID {
		t.Fatalf("organization owner from second unit = %+v, %v", organizationOwner, err)
	}
}
