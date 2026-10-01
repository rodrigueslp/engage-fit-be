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

func TestMemberBillingRepositoryPostgresLedgerAndTenancy(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	ctx := context.Background()
	if _, err = migrations.New(sqlDB).Up(ctx); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	boxRepo := NewBoxGormRepository(db)
	box := &domain.Box{Name: "Member Billing " + uuid.NewString(), Status: domain.BoxStatusActive, CreatedAt: now, UpdatedAt: now}
	owner := &domain.User{Name: "Owner", Email: uuid.NewString() + "@example.test", PasswordHash: "hash", Role: domain.UserRoleOwner, Active: true, CreatedAt: now, UpdatedAt: now}
	if err = boxRepo.SaveWithOwner(ctx, box, owner); err != nil {
		t.Fatal(err)
	}
	personID, planID, versionID, enrollmentID := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	org, unit := string(box.OrganizationID), string(box.ID)
	t.Cleanup(func() {
		db.Exec("DELETE FROM member_billing_command_requests WHERE organization_id=?", org)
		db.Exec("DELETE FROM member_payments WHERE organization_id=?", org)
		db.Exec("DELETE FROM member_charges WHERE organization_id=?", org)
		db.Exec("DELETE FROM member_billing_contracts WHERE organization_id=?", org)
		db.Exec("DELETE FROM enrollments WHERE organization_id=?", org)
		db.Exec("ALTER TABLE membership_plan_versions DISABLE TRIGGER USER")
		db.Exec("DELETE FROM membership_plan_versions WHERE organization_id=?", org)
		db.Exec("ALTER TABLE membership_plan_versions ENABLE TRIGGER USER")
		db.Exec("DELETE FROM membership_plans WHERE organization_id=?", org)
		db.Exec("DELETE FROM persons WHERE organization_id=?", org)
		db.Exec("DELETE FROM users WHERE box_id=?", unit)
		db.Exec("DELETE FROM boxes WHERE id=?", unit)
		db.Exec("DELETE FROM organizations WHERE id=?", org)
	})
	if err = db.Exec("INSERT INTO persons(id,organization_id,name) VALUES (?,?,?)", personID, org, "Pessoa Teste").Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Exec("INSERT INTO membership_plans(id,organization_id,home_unit_id,name,name_key,status) VALUES (?,?,?,?,?,'active')", planID, org, unit, "Plano Teste", uuid.NewString()).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Exec("INSERT INTO membership_plan_versions(id,organization_id,plan_id,version_number,revision,status,name,description,price_cents,currency,billing_cycle,published_at) VALUES (?,?,?,1,1,'published','Plano Teste','',15000,'BRL','monthly',NOW())", versionID, org, planID).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Exec("INSERT INTO enrollments(id,organization_id,person_id,home_unit_id,plan_version_id,lifecycle_status,starts_on,revision,created_by_user_id) VALUES (?,?,?,?,?,'active',CURRENT_DATE,1,?)", enrollmentID, org, personID, unit, versionID, owner.ID).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewMemberBillingGormRepository(db)
	next := now.AddDate(0, 1, 0)
	contract := &domain.MemberBillingContract{ID: domain.ID(uuid.NewString()), OrganizationID: box.OrganizationID, UnitID: box.ID, PersonID: domain.ID(personID), EnrollmentID: domain.ID(enrollmentID), BillingCycle: domain.MembershipBillingMonthly, AmountCents: 15000, Currency: "BRL", Status: "active", PaymentMethod: "pix", StartedOn: now, NextDueOn: &next, CreatedByUserID: owner.ID, CreatedAt: now, UpdatedAt: now}
	if err = repo.CreateContract(ctx, contract); err != nil {
		t.Fatal(err)
	}
	charge := &domain.MemberCharge{ID: domain.ID(uuid.NewString()), OrganizationID: box.OrganizationID, UnitID: box.ID, PersonID: domain.ID(personID), EnrollmentID: domain.ID(enrollmentID), ContractID: contract.ID, Sequence: 1, DueOn: now, AmountCents: 15000, Currency: "BRL", Status: "pending", PaymentMethod: "pix", Provider: "local_simulator", ProviderReference: "sim_" + uuid.NewString(), PaymentInstructions: map[string]any{"simulation": true}, CreatedAt: now, UpdatedAt: now}
	if err = repo.CreateCharge(ctx, charge); err != nil {
		t.Fatal(err)
	}
	payment := &domain.MemberPayment{ID: domain.ID(uuid.NewString()), OrganizationID: box.OrganizationID, ChargeID: charge.ID, Kind: "manual", Method: "cash", AmountCents: 15000, OccurredAt: now, ActorUserID: &owner.ID, Reason: "Baixa de teste", CreatedAt: now}
	if err = repo.CreatePayment(ctx, payment); err != nil {
		t.Fatal(err)
	}
	charge.PaidCents = 15000
	charge.Status = "paid"
	charge.PaidAt = &now
	if err = repo.UpdateCharge(ctx, charge); err != nil {
		t.Fatal(err)
	}
	loaded, err := repo.GetCharge(ctx, box.OrganizationID, box.ID, charge.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "paid" || len(loaded.Payments) != 1 || loaded.PaymentInstructions["simulation"] != true {
		t.Fatalf("unexpected charge: %+v", loaded)
	}
	if _, err = repo.GetCharge(ctx, domain.ID(uuid.NewString()), box.ID, charge.ID); err != portrepo.ErrMemberBillingNotFound {
		t.Fatalf("tenant lookup err=%v", err)
	}
	duplicate := *payment
	duplicate.ID = domain.ID(uuid.NewString())
	duplicate.ProviderEventID = "event-1"
	if err = repo.CreatePayment(ctx, &duplicate); err != nil {
		t.Fatal(err)
	}
	duplicate.ID = domain.ID(uuid.NewString())
	if err = repo.CreatePayment(ctx, &duplicate); err != portrepo.ErrMemberBillingConflict {
		t.Fatalf("duplicate provider event err=%v", err)
	}
}
