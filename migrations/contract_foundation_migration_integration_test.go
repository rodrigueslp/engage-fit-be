package migrations

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func TestContractFoundationEmptyDBAndSecondRun(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	adminConfig, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	adminDB := stdlib.OpenDB(*adminConfig)
	defer adminDB.Close()
	testDatabase := "spec004_a_empty_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	testDB := stdlib.OpenDB(*testConfig)
	defer testDB.Close()

	// 1. Run all migrations on empty db
	migrator := New(testDB)
	applied, err := migrator.Up(ctx)
	if err != nil {
		t.Fatalf("migrator up on empty db failed: %v", err)
	}
	if len(applied) < 60 {
		t.Fatalf("expected at least 60 migrations, applied: %d", len(applied))
	}

	// 2. Run migrator again - must apply 0
	appliedSecond, err := migrator.Up(ctx)
	if err != nil {
		t.Fatalf("second migrator up failed: %v", err)
	}
	if len(appliedSecond) != 0 {
		t.Fatalf("expected 0 migrations on second run, got %d", len(appliedSecond))
	}
}

func TestContractFoundationUpgradeAndNoBackfill(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	adminConfig, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	adminDB := stdlib.OpenDB(*adminConfig)
	defer adminDB.Close()
	testDatabase := "spec004_a_upgrade_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	testDB := stdlib.OpenDB(*testConfig)
	defer testDB.Close()

	// Apply 001..059 first
	through59, err := catalogMigrationFilesThrough(59)
	if err != nil {
		t.Fatal(err)
	}
	if applied, err := NewWithFS(testDB, through59).Up(ctx); err != nil || len(applied) != 59 {
		t.Fatalf("apply through 059: count=%d err=%v", len(applied), err)
	}

	// Seed existing data before migration 060
	const (
		unitOne      = "10000000-0000-4000-8000-000000000001"
		ownerOne     = "20000000-0000-4000-8000-000000000001"
		personOne    = "40000000-0000-4000-8000-000000000001"
		studentOne   = "30000000-0000-4000-8000-000000000001"
		relationship = "50000000-0000-4000-8000-000000000001"
		modalityOne  = "60000000-0000-4000-8000-000000000001"
		planOne      = "70000000-0000-4000-8000-000000000001"
		versionOne   = "80000000-0000-4000-8000-000000000001"
		enrollmentID = "a0000000-0000-4000-8000-000000000001"
	)
	if _, err := testDB.ExecContext(ctx, `INSERT INTO boxes (id, name, status, created_at, updated_at) VALUES ($1, 'Unit One', 'active', NOW(), NOW())`, unitOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO users (id, box_id, name, email, password_hash, role, auth_version, active, created_at, updated_at) VALUES ($1, $2, 'Owner One', 'owner-contract-migration@example.test', 'hash', 'OWNER', 1, TRUE, NOW(), NOW())`, ownerOne, unitOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO persons (id, organization_id, name) VALUES ($1, $2, 'Ana Souza')`, personOne, unitOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO students (id, box_id, person_id, name, source, contact_status) VALUES ($1, $2, $3, 'Ana Souza', 'box_member', 'unknown')`, studentOne, unitOne, personOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO person_unit_relationships (id, organization_id, person_id, unit_id, started_at) VALUES ($1, $2, $3, $2, NOW())`, relationship, unitOne, personOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO modalities (id, organization_id, name, name_key) VALUES ($1, $2, 'CrossFit', 'crossfit')`, modalityOne, unitOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO unit_modalities (organization_id, unit_id, modality_id) VALUES ($1, $1, $2)`, unitOne, modalityOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO membership_plans (id, organization_id, home_unit_id, name, name_key) VALUES ($1, $2, $2, 'CrossFit Mensal', 'crossfit mensal')`, planOne, unitOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO membership_plan_versions (id, organization_id, plan_id, version_number, revision, status, name, price_cents, currency, billing_cycle, published_at) VALUES ($1, $2, $3, 1, 1, 'published', 'CrossFit Mensal', 25000, 'BRL', 'monthly', NOW())`, versionOne, unitOne, planOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO enrollments (id, organization_id, person_id, home_unit_id, plan_version_id, lifecycle_status, starts_on, revision, created_by_user_id) VALUES ($1, $2, $3, $2, $4, 'active', CURRENT_DATE, 1, $5)`, enrollmentID, unitOne, personOne, versionOne, ownerOne); err != nil {
		t.Fatal(err)
	}

	// Now apply only migration 060. Later migrations are covered by their own
	// upgrade tests and must not change this test's assertion.
	through60, err := catalogMigrationFilesThrough(60)
	if err != nil {
		t.Fatal(err)
	}
	appliedFinal, err := NewWithFS(testDB, through60).Up(ctx)
	if err != nil {
		t.Fatalf("apply 060 upgrade: %v", err)
	}
	if len(appliedFinal) != 1 || appliedFinal[0].Version != 60 {
		t.Fatalf("expected migration 060 applied, got %+v", appliedFinal)
	}

	// Verify NO contract templates or instances were backfilled
	var templateCount, instanceCount int
	if err := testDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM contract_templates").Scan(&templateCount); err != nil {
		t.Fatal(err)
	}
	if err := testDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM contract_instances").Scan(&instanceCount); err != nil {
		t.Fatal(err)
	}
	if templateCount != 0 || instanceCount != 0 {
		t.Fatalf("expected 0 templates and 0 instances after upgrade, got templates=%d instances=%d", templateCount, instanceCount)
	}
}
