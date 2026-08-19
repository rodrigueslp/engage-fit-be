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

func TestPersonPlanEnrollmentUpgradeConstraintsAndNoBackfill(t *testing.T) {
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
	testDatabase := "spec003_c_migration_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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

	through57, err := catalogMigrationFilesThrough(57)
	if err != nil {
		t.Fatal(err)
	}
	if applied, err := NewWithFS(testDB, through57).Up(ctx); err != nil || len(applied) != 57 {
		t.Fatalf("apply through 057: count=%d err=%v", len(applied), err)
	}

	const (
		unitOne      = "10000000-0000-4000-8000-000000000001"
		unitTwo      = "10000000-0000-4000-8000-000000000002"
		ownerOne     = "20000000-0000-4000-8000-000000000001"
		ownerTwo     = "20000000-0000-4000-8000-000000000002"
		studentOne   = "30000000-0000-4000-8000-000000000001"
		personOne    = "40000000-0000-4000-8000-000000000001"
		relationship = "50000000-0000-4000-8000-000000000001"
		modalityOne  = "60000000-0000-4000-8000-000000000001"
		planOne      = "70000000-0000-4000-8000-000000000001"
		versionOne   = "80000000-0000-4000-8000-000000000001"
		entitlement  = "90000000-0000-4000-8000-000000000001"
		enrollmentID = "a0000000-0000-4000-8000-000000000001"
	)
	if _, err := testDB.ExecContext(ctx, `INSERT INTO boxes (id, name, status, created_at, updated_at) VALUES ($1, 'Unit One', 'active', NOW(), NOW()), ($2, 'Unit Two', 'active', NOW(), NOW())`, unitOne, unitTwo); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO users (id, box_id, name, email, password_hash, role, auth_version, active, created_at, updated_at) VALUES ($1, $2, 'Owner One', 'owner-one-enrollment@example.test', 'hash', 'OWNER', 1, TRUE, NOW(), NOW()), ($3, $4, 'Owner Two', 'owner-two-enrollment@example.test', 'hash', 'OWNER', 1, TRUE, NOW(), NOW())`, ownerOne, unitOne, ownerTwo, unitTwo); err != nil {
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
	if _, err := testDB.ExecContext(ctx, `INSERT INTO membership_plan_versions (id, organization_id, plan_id, version_number, revision, status, name, price_cents, currency, billing_cycle) VALUES ($1, $2, $3, 1, 1, 'draft', 'CrossFit Mensal', 25000, 'BRL', 'monthly')`, versionOne, unitOne, planOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO membership_plan_entitlements (id, organization_id, plan_version_id, modality_id, access_type) VALUES ($1, $2, $3, $4, 'open_access')`, entitlement, unitOne, versionOne, modalityOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `UPDATE membership_plan_versions SET status='published', published_at=NOW() WHERE id=$1`, versionOne); err != nil {
		t.Fatal(err)
	}

	through58, err := catalogMigrationFilesThrough(58)
	if err != nil {
		t.Fatal(err)
	}
	upgraded, err := NewWithFS(testDB, through58).Up(ctx)
	if err != nil || len(upgraded) != 1 || upgraded[0].Version != 58 {
		t.Fatalf("upgrade to 058: applied=%+v err=%v", upgraded, err)
	}
	for _, table := range []string{"enrollments", "enrollment_command_requests", "enrollment_events"} {
		var count int
		if err := testDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("unexpected backfill in %s: count=%d err=%v", table, count, err)
		}
	}

	if _, err := testDB.ExecContext(ctx, `INSERT INTO enrollments (id, organization_id, person_id, home_unit_id, plan_version_id, starts_on, created_by_user_id) VALUES ($1, $2, $3, $2, $4, '2026-08-19', $5)`, enrollmentID, unitOne, personOne, versionOne, ownerOne); err != nil {
		t.Fatalf("valid enrollment rejected: %v", err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO enrollments (id, organization_id, person_id, home_unit_id, plan_version_id, starts_on, created_by_user_id) VALUES (gen_random_uuid(), $1, $2, $1, $3, '2026-08-20', $4)`, unitOne, personOne, versionOne, ownerOne); err == nil {
		t.Fatal("partial unique index accepted two open enrollments")
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO enrollments (id, organization_id, person_id, home_unit_id, plan_version_id, starts_on, created_by_user_id) VALUES (gen_random_uuid(), $1, $2, $3, $4, '2026-08-20', $5)`, unitOne, personOne, unitTwo, versionOne, ownerOne); err == nil {
		t.Fatal("composite tenant FK accepted a unit from another organization")
	}
	if _, err := testDB.ExecContext(ctx, `UPDATE enrollments SET lifecycle_status='ended', ended_on='2026-08-19' WHERE id=$1`, enrollmentID); err == nil {
		t.Fatal("end-field constraint accepted an incomplete transition")
	}
	if repeated, err := NewWithFS(testDB, through58).Up(ctx); err != nil || len(repeated) != 0 {
		t.Fatalf("second migrator run: applied=%+v err=%v", repeated, err)
	}
}
