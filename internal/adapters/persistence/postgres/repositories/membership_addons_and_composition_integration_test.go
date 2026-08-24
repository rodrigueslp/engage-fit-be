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
	"boxengage/backend/internal/app/memberships"
	"boxengage/backend/internal/domain"
	"github.com/google/uuid"
)

func TestMembershipAddonsAndCompositionLifecycle(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	boxRepository := NewBoxGormRepository(db)

	box := &domain.Box{Name: "Addon Unit " + uuid.NewString()[:8], Status: domain.BoxStatusActive, CreatedAt: now, UpdatedAt: now}
	owner := &domain.User{Name: "Owner Addon", Email: uuid.NewString() + "@example.test", PasswordHash: "hash", AuthVersion: 1, Role: domain.UserRoleOwner, Active: true, CreatedAt: now, UpdatedAt: now}
	if err := boxRepository.SaveWithOwner(ctx, box, owner); err != nil {
		t.Fatal(err)
	}

	organizationIDs := []string{string(box.OrganizationID)}
	unitIDs := []string{string(box.ID)}
	t.Cleanup(func() {
		_ = db.Exec("ALTER TABLE enrollment_addons DISABLE TRIGGER USER").Error
		_ = db.Exec("DELETE FROM enrollment_addons WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("ALTER TABLE enrollment_addons ENABLE TRIGGER USER").Error
		_ = db.Exec("DELETE FROM enrollment_events WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM enrollment_command_requests WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM enrollments WHERE organization_id IN ?", organizationIDs).Error

		_ = db.Exec("ALTER TABLE membership_plan_version_addon_entitlements DISABLE TRIGGER USER").Error
		_ = db.Exec("ALTER TABLE membership_plan_version_addons DISABLE TRIGGER USER").Error
		_ = db.Exec("ALTER TABLE membership_plan_entitlements DISABLE TRIGGER USER").Error
		_ = db.Exec("ALTER TABLE membership_plan_versions DISABLE TRIGGER USER").Error

		_ = db.Exec("DELETE FROM membership_plan_version_addon_entitlements WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM membership_plan_version_addons WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM membership_plan_command_requests WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM membership_plan_audit_events WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM membership_plan_entitlements WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM membership_plan_versions WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM membership_plans WHERE organization_id IN ?", organizationIDs).Error

		_ = db.Exec("ALTER TABLE membership_plan_versions ENABLE TRIGGER USER").Error
		_ = db.Exec("ALTER TABLE membership_plan_entitlements ENABLE TRIGGER USER").Error
		_ = db.Exec("ALTER TABLE membership_plan_version_addons ENABLE TRIGGER USER").Error
		_ = db.Exec("ALTER TABLE membership_plan_version_addon_entitlements ENABLE TRIGGER USER").Error

		_ = db.Exec("DELETE FROM catalog_audit_events WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM unit_service_offerings WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM service_offerings WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM unit_modalities WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM modalities WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM people_audit_events WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM person_unit_relationships WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM students WHERE box_id IN ?", unitIDs).Error
		_ = db.Exec("DELETE FROM persons WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM users WHERE box_id IN ?", unitIDs).Error
		_ = db.Exec("DELETE FROM boxes WHERE id IN ?", unitIDs).Error
		_ = db.Exec("DELETE FROM organizations WHERE id IN ?", organizationIDs).Error
	})

	transactionManager := NewGormTransactionManager(db)
	catalogRepository := NewCatalogGormRepository(db)
	catalogService := catalog.NewService(catalogRepository, transactionManager)

	modalityCrossfit, err := catalogService.CreateModality(ctx, box.ID, owner.ID, "CrossFit", "")
	if err != nil {
		t.Fatal(err)
	}
	modalityPilates, err := catalogService.CreateModality(ctx, box.ID, owner.ID, "Pilates", "")
	if err != nil {
		t.Fatal(err)
	}
	serviceRecovery, err := catalogService.CreateServiceOffering(ctx, box.ID, owner.ID, "Recovery", "", "#0F766E")
	if err != nil {
		t.Fatal(err)
	}

	membershipRepository := NewMembershipGormRepository(db)
	membershipService := memberships.NewService(membershipRepository, catalogRepository, boxRepository, transactionManager)

	quantity := 2
	period := domain.MembershipPeriodWeek
	planInput := memberships.PlanInput{
		Name:         "CrossFit Plus " + uuid.NewString()[:8],
		PriceCents:   30000,
		BillingCycle: domain.MembershipBillingMonthly,
		Entitlements: []memberships.EntitlementInput{
			{ModalityID: string(modalityCrossfit.ID), AccessType: domain.MembershipAccessOpen},
		},
		Addons: []memberships.AddonInput{
			{
				Name:            "Pilates 2x",
				Description:     "Aulas de Pilates adicionais",
				PriceDeltaCents: 12000,
				Entitlements: []memberships.AddonEntitlementInput{
					{ModalityID: string(modalityPilates.ID), AccessType: domain.MembershipAccessPerPeriod, Quantity: &quantity, PeriodUnit: &period},
				},
			},
			{
				Name:            "Recovery Livre",
				Description:     "Acesso à banheira de gelo e bota de compressão",
				PriceDeltaCents: 0,
				Entitlements: []memberships.AddonEntitlementInput{
					{ServiceOfferingID: string(serviceRecovery.ID), AccessType: domain.MembershipAccessOpen},
				},
			},
		},
	}

	plan, _, err := membershipService.Create(ctx, box.ID, owner.ID, "plan-addon-create-"+uuid.NewString(), planInput)
	if err != nil {
		t.Fatalf("create plan with addons: %v", err)
	}
	if plan.Draft == nil || len(plan.Draft.Addons) != 2 {
		t.Fatalf("expected 2 draft addons, got: %+v", plan.Draft)
	}
	if plan.Draft.Addons[0].Summary != "Pilates · 2 sessões por semana" {
		t.Fatalf("unexpected addon summary: %q", plan.Draft.Addons[0].Summary)
	}
	if plan.Draft.Addons[1].Summary != "Recovery · acesso livre" {
		t.Fatalf("unexpected addon summary: %q", plan.Draft.Addons[1].Summary)
	}

	// Publish plan
	publishedPlan, _, err := membershipService.Publish(ctx, box.ID, owner.ID, plan.ID, "plan-addon-pub-"+uuid.NewString(), plan.Draft.Revision)
	if err != nil {
		t.Fatalf("publish plan with addons: %v", err)
	}
	if publishedPlan.Published == nil || len(publishedPlan.Published.Addons) != 2 {
		t.Fatalf("expected 2 published addons, got: %+v", publishedPlan.Published)
	}

	// Test trigger: published addon immutability
	publishedAddonID := string(publishedPlan.Published.Addons[0].ID)
	if err := db.Model(&models.MembershipPlanVersionAddonModel{}).Where("id = ?", publishedAddonID).Update("price_delta_cents", 9999).Error; err == nil {
		t.Fatal("expected error on published addon update due to immutability trigger")
	}
	if err := db.Where("id = ?", publishedAddonID).Delete(&models.MembershipPlanVersionAddonModel{}).Error; err == nil {
		t.Fatal("expected error on published addon delete due to immutability trigger")
	}

	// Test version cloning clones addons with new IDs
	v2Plan, _, err := membershipService.NewVersion(ctx, box.ID, owner.ID, plan.ID, "plan-addon-v2-"+uuid.NewString())
	if err != nil {
		t.Fatalf("new version cloning: %v", err)
	}
	if v2Plan.Draft == nil || len(v2Plan.Draft.Addons) != 2 {
		t.Fatalf("expected 2 cloned addons in draft v2, got: %+v", v2Plan.Draft)
	}
	if string(v2Plan.Draft.Addons[0].ID) == publishedAddonID {
		t.Fatal("cloned addon should have received a new UUID")
	}

	// Test enrollment composition
	personID, studentID, relationshipID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	if err := db.Exec(`INSERT INTO persons (id, organization_id, name) VALUES (?, ?, 'Aluno Composição')`, personID, string(box.OrganizationID)).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO students (id, box_id, person_id, name, source, contact_status) VALUES (?, ?, ?, 'Aluno Composição', 'box_member', 'unknown')`, studentID, string(box.ID), personID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO person_unit_relationships (id, organization_id, person_id, unit_id, started_at) VALUES (?, ?, ?, ?, NOW())`, relationshipID, string(box.OrganizationID), personID, string(box.ID)).Error; err != nil {
		t.Fatal(err)
	}

	enrollmentRepository := NewEnrollmentGormRepository(db)
	peopleRepository := NewPeopleGormRepository(db)
	enrollmentService := memberships.NewEnrollmentService(enrollmentRepository, peopleRepository, catalogRepository, boxRepository, transactionManager, memberships.WithEnrollmentClock(func() time.Time { return now }))

	// Reject duplicate addon IDs
	duplicateAddonInput := memberships.CreateEnrollmentInput{
		PlanID:        string(publishedPlan.ID),
		PlanVersionID: string(publishedPlan.Published.ID),
		StartsOn:      "2026-08-20",
		AddonIDs:      []string{publishedAddonID, publishedAddonID},
	}
	if _, _, err := enrollmentService.Create(ctx, box.ID, domain.ID(studentID), owner.ID, "enroll-addon-dup-"+uuid.NewString(), duplicateAddonInput); !errors.Is(err, memberships.ErrEnrollmentAddonDuplicate) {
		t.Fatalf("expected ErrEnrollmentAddonDuplicate, got: %v", err)
	}

	// Reject foreign addon ID
	foreignAddonInput := memberships.CreateEnrollmentInput{
		PlanID:        string(publishedPlan.ID),
		PlanVersionID: string(publishedPlan.Published.ID),
		StartsOn:      "2026-08-20",
		AddonIDs:      []string{uuid.NewString()},
	}
	if _, _, err := enrollmentService.Create(ctx, box.ID, domain.ID(studentID), owner.ID, "enroll-addon-foreign-"+uuid.NewString(), foreignAddonInput); !errors.Is(err, memberships.ErrEnrollmentAddonNotFound) {
		t.Fatalf("expected ErrEnrollmentAddonNotFound, got: %v", err)
	}

	// Successful enrollment with both addons: base 30000 + addon1 12000 + addon2 0 = 42000
	validEnrollInput := memberships.CreateEnrollmentInput{
		PlanID:        string(publishedPlan.ID),
		PlanVersionID: string(publishedPlan.Published.ID),
		StartsOn:      "2026-08-20",
		AddonIDs:      []string{string(publishedPlan.Published.Addons[0].ID), string(publishedPlan.Published.Addons[1].ID)},
	}
	enrollment, _, err := enrollmentService.Create(ctx, box.ID, domain.ID(studentID), owner.ID, "enroll-addon-success-"+uuid.NewString(), validEnrollInput)
	if err != nil {
		t.Fatalf("create enrollment with addons: %v", err)
	}
	if enrollment.BasePriceCents != 30000 || enrollment.AddonsPriceCents != 12000 || enrollment.TotalPriceCents != 42000 {
		t.Fatalf("pricing calculation mismatch: base=%d addons=%d total=%d", enrollment.BasePriceCents, enrollment.AddonsPriceCents, enrollment.TotalPriceCents)
	}
	if enrollment.Terms.TotalPriceCents != 42000 || enrollment.Terms.BasePriceCents != 30000 || enrollment.Terms.AddonsPriceCents != 12000 || enrollment.Terms.PriceCents != 42000 {
		t.Fatalf("terms pricing mismatch: %+v", enrollment.Terms)
	}
	if len(enrollment.Addons) != 2 {
		t.Fatalf("expected 2 enrollment addons, got: %+v", enrollment.Addons)
	}

	// Test trigger: enrollment_addons snapshot immutability
	enrollmentAddonID := string(enrollment.Addons[0].ID)
	if err := db.Model(&models.EnrollmentAddonModel{}).Where("id = ?", enrollmentAddonID).Update("price_delta_cents", 99999).Error; err == nil {
		t.Fatal("expected error on enrollment_addon update due to immutability trigger")
	}
	if err := db.Where("id = ?", enrollmentAddonID).Delete(&models.EnrollmentAddonModel{}).Error; err == nil {
		t.Fatal("expected error on enrollment_addon delete due to immutability trigger")
	}

	// Test Get and ListForStudent include composition
	fetched, err := enrollmentService.Get(ctx, box.ID, enrollment.ID)
	if err != nil || fetched.TotalPriceCents != 42000 || len(fetched.Addons) != 2 {
		t.Fatalf("fetched enrollment: %+v err=%v", fetched, err)
	}

	history, err := enrollmentService.ListForStudent(ctx, box.ID, domain.ID(studentID))
	if err != nil || history.Current == nil || history.Current.TotalPriceCents != 42000 || len(history.Current.Addons) != 2 {
		t.Fatalf("history current: %+v err=%v", history, err)
	}

	// Test Privacy Export includes addons snapshot
	privacy, err := enrollmentService.ListPrivacy(ctx, box.ID, domain.ID(personID))
	if err != nil || len(privacy) == 0 || len(privacy[0].Enrollment.Addons) != 2 {
		t.Fatalf("privacy export with addons: %+v err=%v", privacy, err)
	}
}
