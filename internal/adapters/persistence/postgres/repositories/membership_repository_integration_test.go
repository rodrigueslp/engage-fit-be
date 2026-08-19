package repositories

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres"
	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/app/catalog"
	"boxengage/backend/internal/app/memberships"
	"boxengage/backend/internal/domain"
	"github.com/google/uuid"
)

func TestMembershipPlanLifecycleIdempotencyTenancyAndImmutability(t *testing.T) {
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
	boxes := []*domain.Box{{Name: "Membership Tenant One", Status: domain.BoxStatusActive, CreatedAt: now, UpdatedAt: now}, {Name: "Membership Tenant Two", Status: domain.BoxStatusActive, CreatedAt: now, UpdatedAt: now}}
	owners := []*domain.User{{Name: "Owner One", Email: uuid.NewString() + "@example.test", PasswordHash: "hash", AuthVersion: 1, Role: domain.UserRoleOwner, Active: true, CreatedAt: now, UpdatedAt: now}, {Name: "Owner Two", Email: uuid.NewString() + "@example.test", PasswordHash: "hash", AuthVersion: 1, Role: domain.UserRoleOwner, Active: true, CreatedAt: now, UpdatedAt: now}}
	for index := range boxes {
		if err := boxRepository.SaveWithOwner(context.Background(), boxes[index], owners[index]); err != nil {
			t.Fatal(err)
		}
	}
	sameOrganizationUnit := &domain.Box{OrganizationID: boxes[0].OrganizationID, Name: "Membership Tenant One Branch", Status: domain.BoxStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := boxRepository.Save(context.Background(), sameOrganizationUnit); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ids := []string{string(boxes[0].ID), string(boxes[1].ID)}
		unitIDs := []string{string(boxes[0].ID), string(boxes[1].ID), string(sameOrganizationUnit.ID)}
		_ = db.Exec("ALTER TABLE membership_plan_entitlements DISABLE TRIGGER USER").Error
		_ = db.Exec("ALTER TABLE membership_plan_versions DISABLE TRIGGER USER").Error
		_ = db.Exec("DELETE FROM membership_plan_command_requests WHERE organization_id IN ?", ids).Error
		_ = db.Exec("DELETE FROM membership_plan_audit_events WHERE organization_id IN ?", ids).Error
		_ = db.Exec("DELETE FROM membership_plan_entitlements WHERE organization_id IN ?", ids).Error
		_ = db.Exec("DELETE FROM membership_plan_versions WHERE organization_id IN ?", ids).Error
		_ = db.Exec("DELETE FROM membership_plans WHERE organization_id IN ?", ids).Error
		_ = db.Exec("ALTER TABLE membership_plan_versions ENABLE TRIGGER USER").Error
		_ = db.Exec("ALTER TABLE membership_plan_entitlements ENABLE TRIGGER USER").Error
		_ = db.Exec("DELETE FROM catalog_audit_events WHERE organization_id IN ?", ids).Error
		_ = db.Exec("DELETE FROM unit_service_offerings WHERE organization_id IN ?", ids).Error
		_ = db.Exec("DELETE FROM service_offerings WHERE organization_id IN ?", ids).Error
		_ = db.Exec("DELETE FROM unit_modalities WHERE organization_id IN ?", ids).Error
		_ = db.Exec("DELETE FROM modalities WHERE organization_id IN ?", ids).Error
		_ = db.Exec("DELETE FROM users WHERE box_id IN ?", unitIDs).Error
		_ = db.Exec("DELETE FROM boxes WHERE id IN ?", unitIDs).Error
		_ = db.Exec("DELETE FROM organizations WHERE id IN ?", ids).Error
	})

	catalogRepository := NewCatalogGormRepository(db)
	catalogService := catalog.NewService(catalogRepository, NewGormTransactionManager(db))
	modalityOne, err := catalogService.CreateModality(context.Background(), boxes[0].ID, owners[0].ID, "Pilates", "")
	if err != nil {
		t.Fatal(err)
	}
	modalityTwo, err := catalogService.CreateModality(context.Background(), boxes[1].ID, owners[1].ID, "Pilates", "")
	if err != nil {
		t.Fatal(err)
	}
	membershipRepository := NewMembershipGormRepository(db)
	service := memberships.NewService(membershipRepository, catalogRepository, boxRepository, NewGormTransactionManager(db))
	quantity, daily := 2, 1
	week := domain.MembershipPeriodWeek
	group := domain.MembershipDeliveryGroup
	input := memberships.PlanInput{Name: "  Pilates   2x ", PriceCents: 26000, BillingCycle: domain.MembershipBillingMonthly, Entitlements: []memberships.EntitlementInput{{ModalityID: string(modalityOne.ID), AccessType: domain.MembershipAccessPerPeriod, Quantity: &quantity, PeriodUnit: &week, DailyLimit: &daily, DeliveryMode: &group, BookingRequired: true}}}

	created, replayed, err := service.Create(context.Background(), boxes[0].ID, owners[0].ID, "create-plan-001", input)
	if err != nil || replayed || created.Draft == nil || created.Draft.Summary != "Pilates · 2 sessões por semana · R$ 260/mês" {
		t.Fatalf("create result=%+v replayed=%v err=%v", created, replayed, err)
	}
	replayedPlan, replayed, err := service.Create(context.Background(), boxes[0].ID, owners[0].ID, "create-plan-001", input)
	if err != nil || !replayed || replayedPlan.ID != created.ID {
		t.Fatalf("idempotent replay=%+v replayed=%v err=%v", replayedPlan, replayed, err)
	}
	if _, _, err := service.Create(context.Background(), sameOrganizationUnit.ID, owners[0].ID, "create-plan-001", input); !errors.Is(err, memberships.ErrPlanNotFound) {
		t.Fatalf("same organization cross-unit replay=%v", err)
	}
	different := input
	different.PriceCents = 27000
	if _, _, err := service.Create(context.Background(), boxes[0].ID, owners[0].ID, "create-plan-001", different); !errors.Is(err, memberships.ErrIdempotencyReused) {
		t.Fatalf("key reuse=%v", err)
	}
	if _, _, err := service.Create(context.Background(), boxes[0].ID, domain.ID(uuid.NewString()), "rollback-plan-001", memberships.PlanInput{Name: "Rollback Plan", PriceCents: 10000, BillingCycle: domain.MembershipBillingMonthly, Entitlements: input.Entitlements}); err == nil {
		t.Fatal("expected audit rollback")
	}
	var rollbackCount int64
	if err := db.Model(&models.MembershipPlanModel{}).Where("organization_id = ? AND name_key = 'rollback plan'", string(boxes[0].OrganizationID)).Count(&rollbackCount).Error; err != nil || rollbackCount != 0 {
		t.Fatalf("audit rollback count=%d err=%v", rollbackCount, err)
	}

	badTarget := input
	badTarget.Name = "Foreign target"
	badTarget.Entitlements = []memberships.EntitlementInput{{ModalityID: string(modalityTwo.ID), AccessType: domain.MembershipAccessOpen}}
	if _, _, err := service.Create(context.Background(), boxes[0].ID, owners[0].ID, "foreign-target-1", badTarget); !errors.Is(err, memberships.ErrTargetInvalid) {
		t.Fatalf("foreign target=%v", err)
	}
	if _, err := service.Get(context.Background(), boxes[1].ID, created.ID); !errors.Is(err, memberships.ErrPlanNotFound) {
		t.Fatalf("cross tenant get=%v", err)
	}

	published, replayed, err := service.Publish(context.Background(), boxes[0].ID, owners[0].ID, created.ID, "publish-plan-01", created.Draft.Revision)
	if err != nil || replayed || published.Published == nil || published.Draft != nil {
		t.Fatalf("publish=%+v replayed=%v err=%v", published, replayed, err)
	}
	originalVersionID := published.Published.ID
	if err := db.Model(&models.MembershipPlanVersionModel{}).Where("id = ?", string(originalVersionID)).Update("price_cents", 1).Error; err == nil {
		t.Fatal("postgres allowed published version update")
	}

	withDraft, replayed, err := service.NewVersion(context.Background(), boxes[0].ID, owners[0].ID, created.ID, "new-version-01")
	if err != nil || replayed || withDraft.Draft == nil || withDraft.Draft.VersionNumber != 2 || withDraft.Draft.PriceCents != 26000 {
		t.Fatalf("new version=%+v replayed=%v err=%v", withDraft, replayed, err)
	}
	if _, _, err := service.NewVersion(context.Background(), boxes[0].ID, owners[0].ID, created.ID, "new-version-02"); !errors.Is(err, memberships.ErrDraftExists) {
		t.Fatalf("second draft=%v", err)
	}
	updatedInput := input
	updatedInput.PriceCents = 30000
	updatedInput.ExpectedRevision = withDraft.Draft.Revision
	updated, err := service.UpdateDraft(context.Background(), boxes[0].ID, owners[0].ID, created.ID, updatedInput)
	if err != nil || updated.Draft == nil || updated.Draft.Revision != 2 {
		t.Fatalf("update=%+v err=%v", updated, err)
	}
	if _, err := service.UpdateDraft(context.Background(), boxes[0].ID, owners[0].ID, created.ID, updatedInput); !errors.Is(err, memberships.ErrRevisionConflict) {
		t.Fatalf("stale update=%v", err)
	}
	secondPublished, _, err := service.Publish(context.Background(), boxes[0].ID, owners[0].ID, created.ID, "publish-plan-02", updated.Draft.Revision)
	if err != nil || secondPublished.Published == nil || secondPublished.Published.VersionNumber != 2 || secondPublished.Published.PriceCents != 30000 {
		t.Fatalf("second publish=%+v err=%v", secondPublished, err)
	}
	var original models.MembershipPlanVersionModel
	if err := db.Where("id = ?", string(originalVersionID)).Take(&original).Error; err != nil || original.Status != "retired" || original.PriceCents != 26000 {
		t.Fatalf("original changed=%+v err=%v", original, err)
	}
	firstPublishReplay, replayed, err := service.Publish(context.Background(), boxes[0].ID, owners[0].ID, created.ID, "publish-plan-01", 1)
	if err != nil || !replayed || firstPublishReplay.Published == nil || firstPublishReplay.Published.ID != originalVersionID || firstPublishReplay.Published.PriceCents != 26000 || firstPublishReplay.Draft != nil {
		t.Fatalf("publish replay was not stable: %+v replayed=%v err=%v", firstPublishReplay, replayed, err)
	}
	if _, err := catalogService.SetModalityActive(context.Background(), boxes[0].ID, boxes[0].ID, owners[0].ID, modalityOne.ID, false); err != nil {
		t.Fatal(err)
	}
	withWarning, err := service.Get(context.Background(), boxes[0].ID, created.ID)
	if err != nil || withWarning.Published == nil || len(withWarning.Published.Warnings) == 0 {
		t.Fatalf("missing unavailable-reference warning: %+v err=%v", withWarning, err)
	}
	var wait sync.WaitGroup
	versionResults := make(chan error, 2)
	for _, key := range []string{"concurrent-version-01", "concurrent-version-02"} {
		wait.Add(1)
		go func(idempotencyKey string) {
			defer wait.Done()
			_, _, createErr := service.NewVersion(context.Background(), boxes[0].ID, owners[0].ID, created.ID, idempotencyKey)
			versionResults <- createErr
		}(key)
	}
	wait.Wait()
	close(versionResults)
	succeeded, conflicted := 0, 0
	for resultErr := range versionResults {
		switch {
		case resultErr == nil:
			succeeded++
		case errors.Is(resultErr, memberships.ErrDraftExists):
			conflicted++
		default:
			t.Fatalf("unexpected concurrent version error: %v", resultErr)
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("concurrent versions succeeded=%d conflicted=%d", succeeded, conflicted)
	}

	archived, err := service.SetArchived(context.Background(), boxes[0].ID, owners[0].ID, created.ID, true)
	if err != nil || archived.Status != domain.MembershipPlanArchived || archived.Draft == nil || archived.Draft.VersionNumber != 3 {
		t.Fatalf("archive=%+v err=%v", archived, err)
	}
	if _, _, err := service.Publish(context.Background(), boxes[0].ID, owners[0].ID, created.ID, "archive-pub-01", archived.Draft.Revision); !errors.Is(err, memberships.ErrPlanArchived) {
		t.Fatalf("archived publish=%v", err)
	}
	if _, err := service.SetArchived(context.Background(), boxes[0].ID, owners[0].ID, created.ID, false); err != nil {
		t.Fatal(err)
	}

	var auditCount int64
	if err := db.Model(&models.MembershipPlanAuditEventModel{}).Where("plan_id = ?", string(created.ID)).Count(&auditCount).Error; err != nil || auditCount < 7 {
		t.Fatalf("audit count=%d err=%v", auditCount, err)
	}
}
