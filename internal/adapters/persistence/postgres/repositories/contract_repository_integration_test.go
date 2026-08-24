package repositories

import (
	"context"
	"os"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"boxengage/backend/migrations"
	"github.com/google/uuid"
)

func TestContractRepositoryPostgresTriggersAndTenancy(t *testing.T) {
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
	ctx := context.Background()
	if _, err := migrations.New(sqlDB).Up(ctx); err != nil {
		t.Fatalf("migrator.Up() err = %v", err)
	}
	now := time.Now().UTC()

	boxRepo := NewBoxGormRepository(db)
	box1 := &domain.Box{Name: "Contract Box 1 " + uuid.NewString(), Status: domain.BoxStatusActive, CreatedAt: now, UpdatedAt: now}
	box2 := &domain.Box{Name: "Contract Box 2 " + uuid.NewString(), Status: domain.BoxStatusActive, CreatedAt: now, UpdatedAt: now}
	owner1 := &domain.User{Name: "Owner 1", Email: uuid.NewString() + "@example.test", PasswordHash: "hash", Role: domain.UserRoleOwner, Active: true, CreatedAt: now, UpdatedAt: now}
	owner2 := &domain.User{Name: "Owner 2", Email: uuid.NewString() + "@example.test", PasswordHash: "hash", Role: domain.UserRoleOwner, Active: true, CreatedAt: now, UpdatedAt: now}

	if err := boxRepo.SaveWithOwner(ctx, box1, owner1); err != nil {
		t.Fatal(err)
	}
	if err := boxRepo.SaveWithOwner(ctx, box2, owner2); err != nil {
		t.Fatal(err)
	}

	orgIDs := []string{string(box1.OrganizationID), string(box2.OrganizationID)}
	unitIDs := []string{string(box1.ID), string(box2.ID)}

	t.Cleanup(func() {
		_ = db.Exec("DELETE FROM contract_acceptances WHERE organization_id IN ?", orgIDs).Error
		_ = db.Exec("DELETE FROM contract_acceptance_sessions WHERE organization_id IN ?", orgIDs).Error
		_ = db.Exec("DELETE FROM contract_events WHERE organization_id IN ?", orgIDs).Error
		_ = db.Exec("DELETE FROM contract_command_requests WHERE organization_id IN ?", orgIDs).Error
		_ = db.Exec("DELETE FROM contract_instances WHERE organization_id IN ?", orgIDs).Error
		_ = db.Exec("ALTER TABLE contract_template_versions DISABLE TRIGGER USER").Error
		_ = db.Exec("DELETE FROM contract_template_audit_events WHERE organization_id IN ?", orgIDs).Error
		_ = db.Exec("DELETE FROM contract_template_versions WHERE organization_id IN ?", orgIDs).Error
		_ = db.Exec("ALTER TABLE contract_template_versions ENABLE TRIGGER USER").Error
		_ = db.Exec("DELETE FROM contract_templates WHERE organization_id IN ?", orgIDs).Error
		_ = db.Exec("DELETE FROM enrollments WHERE organization_id IN ?", orgIDs).Error
		_ = db.Exec("ALTER TABLE membership_plan_versions DISABLE TRIGGER USER").Error
		_ = db.Exec("DELETE FROM membership_plan_versions WHERE organization_id IN ?", orgIDs).Error
		_ = db.Exec("ALTER TABLE membership_plan_versions ENABLE TRIGGER USER").Error
		_ = db.Exec("DELETE FROM membership_plans WHERE organization_id IN ?", orgIDs).Error
		_ = db.Exec("DELETE FROM person_unit_relationships WHERE organization_id IN ?", orgIDs).Error
		_ = db.Exec("DELETE FROM students WHERE box_id IN ?", unitIDs).Error
		_ = db.Exec("DELETE FROM persons WHERE organization_id IN ?", orgIDs).Error
		_ = db.Exec("DELETE FROM users WHERE box_id IN ?", unitIDs).Error
		_ = db.Exec("DELETE FROM boxes WHERE id IN ?", unitIDs).Error
		_ = db.Exec("DELETE FROM organizations WHERE id IN ?", orgIDs).Error
	})

	contractRepo := NewContractGormRepository(db)

	// 1. Create Template & Draft
	tmplID := domain.ID(uuid.NewString())
	draftID := domain.ID(uuid.NewString())
	tmpl := &domain.ContractTemplate{
		ID:             tmplID,
		OrganizationID: box1.OrganizationID,
		HomeUnitID:     box1.ID,
		Code:           "tmpl_01",
		Name:           "Contrato Principal",
		NameKey:        "contrato principal",
		ContractKind:   domain.ContractKindMembershipTerms,
		Status:         domain.ContractTemplateStatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	draft := &domain.ContractTemplateVersion{
		ID:                       draftID,
		OrganizationID:           box1.OrganizationID,
		TemplateID:               tmplID,
		VersionNumber:            1,
		Revision:                 1,
		Status:                   domain.ContractTemplateVersionStatusDraft,
		Title:                    "Termos",
		BodyMarkdown:             "Texto inicial",
		AcceptanceStatement:      "Li e concordo",
		PlaceholderSchemaVersion: 1,
		RendererVersion:          "markdown-v1",
		CreatedAt:                now,
		UpdatedAt:                now,
	}
	if err := contractRepo.SaveTemplateDraft(ctx, tmpl, draft); err != nil {
		t.Fatalf("SaveTemplateDraft() err = %v", err)
	}

	// 2. Name conflict on same organization
	dupTmpl := &domain.ContractTemplate{
		ID:             domain.ID(uuid.NewString()),
		OrganizationID: box1.OrganizationID,
		HomeUnitID:     box1.ID,
		Code:           "tmpl_02",
		Name:           "Contrato Principal",
		NameKey:        "contrato principal",
		ContractKind:   domain.ContractKindMembershipTerms,
		Status:         domain.ContractTemplateStatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	dupDraft := &domain.ContractTemplateVersion{
		ID:                       domain.ID(uuid.NewString()),
		OrganizationID:           box1.OrganizationID,
		TemplateID:               dupTmpl.ID,
		VersionNumber:            1,
		Revision:                 1,
		Status:                   domain.ContractTemplateVersionStatusDraft,
		Title:                    "Termos",
		BodyMarkdown:             "Texto",
		AcceptanceStatement:      "Li",
		PlaceholderSchemaVersion: 1,
		RendererVersion:          "markdown-v1",
		CreatedAt:                now,
		UpdatedAt:                now,
	}
	if err := contractRepo.SaveTemplateDraft(ctx, dupTmpl, dupDraft); err == nil || err != portrepo.ErrContractTemplateNameConflict {
		t.Fatalf("expected ErrContractTemplateNameConflict, got %v", err)
	}

	// 3. Publish Version
	contentSHA, err := domain.CalculateTemplateVersionContentSHA256(draft.Title, draft.BodyMarkdown, draft.AcceptanceStatement)
	if err != nil {
		t.Fatal(err)
	}
	if err := contractRepo.PublishTemplateVersion(ctx, box1.OrganizationID, box1.ID, tmplID, draftID, 1, now, contentSHA); err != nil {
		t.Fatalf("PublishTemplateVersion() err = %v", err)
	}

	// 4. Immutability trigger prevents direct SQL mutation on published template version
	err = db.Exec("UPDATE contract_template_versions SET body_markdown = 'Texto adulterado' WHERE id = ?", string(draftID)).Error
	if err == nil {
		t.Fatal("expected published contract template version mutation to be blocked by trigger")
	}

	// 5. Tenancy isolation: cannot find template from org2
	_, err = contractRepo.FindTemplateByID(ctx, box2.OrganizationID, box2.ID, tmplID)
	if err == nil || err != portrepo.ErrContractTemplateNotFound {
		t.Fatalf("expected ErrContractTemplateNotFound from different tenant, got %v", err)
	}

	// 6. Test Contract Instance Creation & Immutability Trigger
	personID := domain.ID(uuid.NewString())
	if err := db.Exec("INSERT INTO persons (id, organization_id, name, profile_status) VALUES (?, ?, ?, 'active')", string(personID), string(box1.OrganizationID), "Aluno Teste").Error; err != nil {
		t.Fatal(err)
	}
	planID := domain.ID(uuid.NewString())
	if err := db.Exec("INSERT INTO membership_plans (id, organization_id, home_unit_id, name, name_key) VALUES (?, ?, ?, 'Plano Teste', 'plano teste')", string(planID), string(box1.OrganizationID), string(box1.ID)).Error; err != nil {
		t.Fatal(err)
	}
	planVerID := domain.ID(uuid.NewString())
	if err := db.Exec("INSERT INTO membership_plan_versions (id, organization_id, plan_id, version_number, revision, status, name, price_cents, currency, billing_cycle, published_at) VALUES (?, ?, ?, 1, 1, 'published', 'Plano Teste', 20000, 'BRL', 'monthly', NOW())", string(planVerID), string(box1.OrganizationID), string(planID)).Error; err != nil {
		t.Fatal(err)
	}
	enrollmentID := domain.ID(uuid.NewString())
	if err := db.Exec("INSERT INTO enrollments (id, organization_id, person_id, home_unit_id, plan_version_id, lifecycle_status, starts_on, revision, created_by_user_id) VALUES (?, ?, ?, ?, ?, 'active', CURRENT_DATE, 1, ?)", string(enrollmentID), string(box1.OrganizationID), string(personID), string(box1.ID), string(planVerID), string(owner1.ID)).Error; err != nil {
		t.Fatal(err)
	}

	commSnap := domain.ContractCommercialSnapshot{
		SchemaVersion:      1,
		EnrollmentID:       enrollmentID,
		EnrollmentRevision: 1,
		PlanID:             planID,
		PlanVersionID:      planVerID,
		PlanVersionNumber:  1,
		PlanName:           "Plano Teste",
		BillingCycle:       domain.MembershipBillingMonthly,
		Currency:           "BRL",
		BasePriceCents:     20000,
		AddonsPriceCents:   0,
		TotalPriceCents:    20000,
		StartsOn:           "2026-08-24",
		BaseSummary:        "Acesso livre",
	}
	commSHA, _ := domain.CalculateCommercialSnapshotSHA256(commSnap)
	docSHA, _ := domain.CalculateDocumentSHA256("Título", "Texto", "Li e aceito", commSHA)
	retentionDue := now.Add(-1 * time.Hour) // expired for retention test

	instanceID := domain.ID(uuid.NewString())
	instance := &domain.ContractInstance{
		ID:                         instanceID,
		OrganizationID:             box1.OrganizationID,
		UnitID:                     box1.ID,
		PersonID:                   personID,
		EnrollmentID:               enrollmentID,
		TemplateVersionID:          draftID,
		ContractKind:               domain.ContractKindMembershipTerms,
		Status:                     domain.ContractInstanceStatusDraft,
		Revision:                   1,
		SnapshotSchemaVersion:      1,
		CommercialSnapshot:         commSnap,
		CommercialSnapshotSHA256:   commSHA,
		PartySnapshot:              domain.ContractPartySnapshot{OrganizationID: box1.OrganizationID, OrganizationName: box1.Name, UnitID: box1.ID, UnitName: box1.Name, PersonID: personID, DisplayName: "Aluno Teste"},
		DocumentTitle:              "Título",
		DocumentBodyMarkdown:       "Texto",
		DocumentSHA256:             docSHA,
		AcceptanceStatement:        "Li e aceito",
		AcceptanceStatementVersion: 1,
		CreatedByUserID:            owner1.ID,
		RetentionClass:             domain.ContractRetentionClassLocalSimulator,
		RetentionDueAt:             &retentionDue,
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}

	if err := contractRepo.SaveInstanceDraft(ctx, instance); err != nil {
		t.Fatalf("SaveInstanceDraft() err = %v", err)
	}

	// Immutability trigger prevents direct SQL mutation of commercial_snapshot on instance
	err = db.Exec("UPDATE contract_instances SET commercial_snapshot = '{\"adulterado\": true}' WHERE id = ?", string(instanceID)).Error
	if err == nil {
		t.Fatal("expected instance commercial_snapshot mutation to be blocked by trigger")
	}

	// Partial unique index prevents second non-discarded contract on same enrollment
	instance2 := *instance
	instance2.ID = domain.ID(uuid.NewString())
	if err := contractRepo.SaveInstanceDraft(ctx, &instance2); err == nil || err != portrepo.ErrContractInstanceExists {
		t.Fatalf("expected ErrContractInstanceExists, got %v", err)
	}

	// Retention purge test
	purgedCount, err := contractRepo.PurgeExpiredSimulatorRecords(ctx, now, true)
	if err != nil {
		t.Fatalf("PurgeExpiredSimulatorRecords() err = %v", err)
	}
	if purgedCount != 1 {
		t.Fatalf("expected 1 record purged, got %d", purgedCount)
	}
}
