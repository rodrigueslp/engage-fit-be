package repositories

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres"
	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/app/catalog"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"github.com/google/uuid"
)

func TestCatalogRepositoryTransactionsConstraintsAndTenantIsolation(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	boxRepository := NewBoxGormRepository(db)
	boxes := []*domain.Box{
		{Name: "Catalog Tenant One", Status: domain.BoxStatusActive, CreatedAt: now, UpdatedAt: now},
		{Name: "Catalog Tenant Two", Status: domain.BoxStatusActive, CreatedAt: now, UpdatedAt: now},
	}
	owners := []*domain.User{
		{Name: "Owner One", Email: uuid.NewString() + "@example.test", PasswordHash: "hash", AuthVersion: 1, Role: domain.UserRoleOwner, Active: true, CreatedAt: now, UpdatedAt: now},
		{Name: "Owner Two", Email: uuid.NewString() + "@example.test", PasswordHash: "hash", AuthVersion: 1, Role: domain.UserRoleOwner, Active: true, CreatedAt: now, UpdatedAt: now},
	}
	for index := range boxes {
		if err := boxRepository.SaveWithOwner(context.Background(), boxes[index], owners[index]); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		ids := []string{string(boxes[0].ID), string(boxes[1].ID)}
		_ = db.Exec("DELETE FROM catalog_audit_events WHERE organization_id IN ?", ids).Error
		_ = db.Exec("DELETE FROM unit_modalities WHERE unit_id IN ?", ids).Error
		_ = db.Exec("DELETE FROM spaces WHERE unit_id IN ?", ids).Error
		_ = db.Exec("DELETE FROM modalities WHERE organization_id IN ?", ids).Error
		_ = db.Exec("DELETE FROM users WHERE box_id IN ?", ids).Error
		_ = db.Exec("DELETE FROM boxes WHERE id IN ?", ids).Error
		_ = db.Exec("DELETE FROM organizations WHERE id IN ?", ids).Error
	})

	repository := NewCatalogGormRepository(db)
	service := catalog.NewService(repository, NewGormTransactionManager(db))
	ctx := context.Background()
	modalityOne, err := service.CreateModality(ctx, boxes[0].ID, owners[0].ID, "  Cross   Training ", "")
	if err != nil {
		t.Fatal(err)
	}
	if modalityOne.Name != "Cross Training" || modalityOne.NameKey != "cross training" || modalityOne.Color != domain.DefaultModalityColor || !modalityOne.ActiveInUnit {
		t.Fatalf("unexpected modality: %+v", modalityOne)
	}

	if _, err := service.CreateModality(ctx, boxes[0].ID, owners[0].ID, "cross training", "#FFFFFF"); !errors.Is(err, catalog.ErrModalityNameConflict) {
		t.Fatalf("same-tenant normalized duplicate = %v", err)
	}
	modalityTwo, err := service.CreateModality(ctx, boxes[1].ID, owners[1].ID, "Cross Training", "#ABCDEF")
	if err != nil {
		t.Fatalf("same name in another organization should be allowed: %v", err)
	}
	if _, err := service.UpdateModality(ctx, boxes[0].ID, owners[0].ID, modalityTwo.ID, catalog.ModalityChanges{Name: catalogStringPointer("Leaked")}); !errors.Is(err, catalog.ErrModalityNotFound) {
		t.Fatalf("cross-tenant modality update = %v", err)
	}

	spaceOne, err := service.CreateSpace(ctx, boxes[0].ID, boxes[0].ID, owners[0].ID, " Área   principal ", nil)
	if err != nil {
		t.Fatal(err)
	}
	if spaceOne.Name != "Área principal" || spaceOne.DefaultCapacity != nil {
		t.Fatalf("unexpected space: %+v", spaceOne)
	}
	if _, err := service.CreateSpace(ctx, boxes[0].ID, boxes[0].ID, owners[0].ID, "área principal", catalogIntPointer(20)); !errors.Is(err, catalog.ErrSpaceNameConflict) {
		t.Fatalf("same-unit normalized duplicate = %v", err)
	}
	spaceTwo, err := service.CreateSpace(ctx, boxes[1].ID, boxes[1].ID, owners[1].ID, "Área principal", nil)
	if err != nil {
		t.Fatalf("same name in another unit should be allowed: %v", err)
	}
	if _, err := service.UpdateSpace(ctx, boxes[0].ID, boxes[0].ID, owners[0].ID, spaceTwo.ID, catalog.SpaceChanges{Name: catalogStringPointer("Leaked")}); !errors.Is(err, catalog.ErrSpaceNotFound) {
		t.Fatalf("cross-tenant space update = %v", err)
	}

	archived := domain.CatalogStatusArchived
	updated, err := service.UpdateModality(ctx, boxes[0].ID, owners[0].ID, modalityOne.ID, catalog.ModalityChanges{Status: &archived})
	if err != nil || updated.ID != modalityOne.ID {
		t.Fatalf("archive failed: %+v %v", updated, err)
	}
	active := domain.CatalogStatusActive
	updated, err = service.UpdateModality(ctx, boxes[0].ID, owners[0].ID, modalityOne.ID, catalog.ModalityChanges{Status: &active})
	if err != nil || updated.ID != modalityOne.ID || !updated.ActiveInUnit {
		t.Fatalf("reactivation did not preserve association: %+v %v", updated, err)
	}

	invalidAssociation := domain.UnitModality{OrganizationID: boxes[1].OrganizationID, UnitID: boxes[0].ID, ModalityID: modalityTwo.ID, Active: true, CreatedAt: now, UpdatedAt: now}
	if err := repository.CreateUnitModality(ctx, &invalidAssociation); err == nil {
		t.Fatal("database accepted cross-organization unit modality")
	}

	failedName := "Must Roll Back"
	if _, err := service.CreateModality(ctx, boxes[0].ID, domain.ID(uuid.NewString()), failedName, ""); err == nil {
		t.Fatal("expected audit foreign-key failure")
	}
	var rolledBackCount int64
	if err := db.Model(&models.ModalityModel{}).Where("organization_id = ? AND name_key = ?", string(boxes[0].OrganizationID), "must roll back").Count(&rolledBackCount).Error; err != nil {
		t.Fatal(err)
	}
	if rolledBackCount != 0 {
		t.Fatalf("failed audit left %d modalities behind", rolledBackCount)
	}

	var auditCount int64
	if err := db.Model(&models.CatalogAuditEventModel{}).Where("organization_id = ?", string(boxes[0].OrganizationID)).Count(&auditCount).Error; err != nil {
		t.Fatal(err)
	}
	if auditCount < 5 {
		t.Fatalf("expected catalog audit history, got %d events", auditCount)
	}

	if _, err := repository.FindModality(ctx, boxes[0].OrganizationID, boxes[0].ID, modalityTwo.ID); !errors.Is(err, portrepo.ErrCatalogModalityNotFound) {
		t.Fatalf("cross-organization lookup should be hidden: %v", err)
	}
}

func catalogStringPointer(value string) *string { return &value }
func catalogIntPointer(value int) *int          { return &value }
