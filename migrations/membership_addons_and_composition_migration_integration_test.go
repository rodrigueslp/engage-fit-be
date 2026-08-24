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

func TestMembershipAddonsAndCompositionUpgradeAndNoBackfill(t *testing.T) {
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
	testDatabase := "spec003_b_migration_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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

	// Apply 001..058 first
	through58, err := catalogMigrationFilesThrough(58)
	if err != nil {
		t.Fatal(err)
	}
	if applied, err := NewWithFS(testDB, through58).Up(ctx); err != nil || len(applied) != 58 {
		t.Fatalf("apply through 058: count=%d err=%v", len(applied), err)
	}

	const (
		unitOne      = "10000000-0000-4000-8000-000000000001"
		ownerOne     = "20000000-0000-4000-8000-000000000001"
		personOne    = "40000000-0000-4000-8000-000000000001"
		studentOne   = "30000000-0000-4000-8000-000000000001"
		relationship = "50000000-0000-4000-8000-000000000001"
		modalityOne  = "60000000-0000-4000-8000-000000000001"
		serviceOne   = "60000000-0000-4000-8000-000000000002"
		planOne      = "70000000-0000-4000-8000-000000000001"
		versionOne   = "80000000-0000-4000-8000-000000000001"
		versionTwo   = "80000000-0000-4000-8000-000000000002"
		entitlement  = "90000000-0000-4000-8000-000000000001"
		enrollmentID = "a0000000-0000-4000-8000-000000000001"
		addonID      = "b0000000-0000-4000-8000-000000000001"
		addonID2     = "b0000000-0000-4000-8000-000000000002"
	)
	if _, err := testDB.ExecContext(ctx, `INSERT INTO boxes (id, name, status, created_at, updated_at) VALUES ($1, 'Unit One', 'active', NOW(), NOW())`, unitOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO users (id, box_id, name, email, password_hash, role, auth_version, active, created_at, updated_at) VALUES ($1, $2, 'Owner One', 'owner-b-migration@example.test', 'hash', 'OWNER', 1, TRUE, NOW(), NOW())`, ownerOne, unitOne); err != nil {
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
	if _, err := testDB.ExecContext(ctx, `INSERT INTO service_offerings (id, organization_id, name, name_key) VALUES ($1, $2, 'Recovery', 'recovery')`, serviceOne, unitOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO unit_service_offerings (organization_id, unit_id, service_offering_id) VALUES ($1, $1, $2)`, unitOne, serviceOne); err != nil {
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
	if _, err := testDB.ExecContext(ctx, `INSERT INTO membership_plan_versions (id, organization_id, plan_id, version_number, revision, status, name, price_cents, currency, billing_cycle) VALUES ($1, $2, $3, 2, 1, 'draft', 'CrossFit Mensal v2', 26000, 'BRL', 'monthly')`, versionTwo, unitOne, planOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO enrollments (id, organization_id, person_id, home_unit_id, plan_version_id, starts_on, created_by_user_id) VALUES ($1, $2, $3, $2, $4, '2026-08-19', $5)`, enrollmentID, unitOne, personOne, versionOne, ownerOne); err != nil {
		t.Fatal(err)
	}

	// Upgrade to 059
	through59, err := catalogMigrationFilesThrough(59)
	if err != nil {
		t.Fatal(err)
	}
	upgraded, err := NewWithFS(testDB, through59).Up(ctx)
	if err != nil || len(upgraded) != 1 || upgraded[0].Version != 59 {
		t.Fatalf("upgrade to 059: applied=%+v err=%v", upgraded, err)
	}
	for _, table := range []string{"membership_plan_version_addons", "membership_plan_version_addon_entitlements", "enrollment_addons"} {
		var count int
		if err := testDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("unexpected backfill in %s: count=%d err=%v", table, count, err)
		}
	}

	// Test triggers on draft vs published addons
	if _, err := testDB.ExecContext(ctx, `INSERT INTO membership_plan_version_addons (id, organization_id, plan_version_id, name, price_delta_cents) VALUES ($1, $2, $3, 'Recovery', 5000)`, addonID, unitOne, versionTwo); err != nil {
		t.Fatalf("database rejected adding addon to draft version: %v", err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO membership_plan_version_addons (id, organization_id, plan_version_id, name, price_delta_cents) VALUES ($1, $2, $3, 'Recovery', 5000)`, addonID2, unitOne, versionOne); err == nil {
		t.Fatal("database allowed adding an addon to a published plan version")
	}

	// Test triggers on enrollment addons
	const enrollmentAddonID = "c0000000-0000-4000-8000-000000000001"
	if _, err := testDB.ExecContext(ctx, `INSERT INTO enrollment_addons (id, organization_id, enrollment_id, plan_version_addon_id, addon_name, price_delta_cents, entitlement_summary) VALUES ($1, $2, $3, $4, 'Recovery', 5000, 'Recovery · acesso livre')`, enrollmentAddonID, unitOne, enrollmentID, addonID); err != nil {
		t.Fatalf("insert enrollment addon failed: %v", err)
	}
	if _, err := testDB.ExecContext(ctx, `UPDATE enrollment_addons SET price_delta_cents=6000 WHERE id=$1`, enrollmentAddonID); err == nil {
		t.Fatal("database allowed mutation of enrollment addon snapshot")
	}
	if _, err := testDB.ExecContext(ctx, `DELETE FROM enrollment_addons WHERE id=$1`, enrollmentAddonID); err == nil {
		t.Fatal("database allowed deletion of enrollment addon snapshot")
	}

	// Verify idempotency of migrator on already upgraded database
	repeated, err := NewWithFS(testDB, through59).Up(ctx)
	if err != nil || len(repeated) != 0 {
		t.Fatalf("second migrator run: applied=%+v err=%v", repeated, err)
	}
}
