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

func TestMembershipPlanCatalogUpgradeAndDatabaseImmutability(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	adminConfig, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	adminDB := stdlib.OpenDB(*adminConfig)
	defer adminDB.Close()
	testDatabase := "spec003_a_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	through56, err := catalogMigrationFilesThrough(56)
	if err != nil {
		t.Fatal(err)
	}
	if applied, err := NewWithFS(testDB, through56).Up(ctx); err != nil || len(applied) != 56 {
		t.Fatalf("apply through 056: count=%d err=%v", len(applied), err)
	}

	const (
		unitOne     = "10000000-0000-4000-8000-000000000001"
		unitTwo     = "10000000-0000-4000-8000-000000000002"
		ownerOne    = "20000000-0000-4000-8000-000000000001"
		modalityOne = "30000000-0000-4000-8000-000000000001"
		modalityTwo = "30000000-0000-4000-8000-000000000002"
		serviceOne  = "40000000-0000-4000-8000-000000000001"
	)
	if _, err := testDB.ExecContext(ctx, `INSERT INTO boxes (id, name, status, created_at, updated_at) VALUES ($1, 'Unit One', 'active', NOW(), NOW()), ($2, 'Unit Two', 'active', NOW(), NOW())`, unitOne, unitTwo); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO users (id, box_id, name, email, password_hash, role, auth_version, active, created_at, updated_at) VALUES ($1, $2, 'Owner', 'membership-owner@example.test', 'hash', 'OWNER', 1, TRUE, NOW(), NOW())`, ownerOne, unitOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO modalities (id, organization_id, name, name_key) VALUES ($1, $2, 'Pilates', 'pilates'), ($3, $4, 'Cross tenant', 'cross tenant')`, modalityOne, unitOne, modalityTwo, unitTwo); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO unit_modalities (organization_id, unit_id, modality_id) VALUES ($1, $1, $2), ($3, $3, $4)`, unitOne, modalityOne, unitTwo, modalityTwo); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO service_offerings (id, organization_id, name, name_key) VALUES ($1, $2, 'Recovery', 'recovery')`, serviceOne, unitOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO unit_service_offerings (organization_id, unit_id, service_offering_id) VALUES ($1, $1, $2)`, unitOne, serviceOne); err != nil {
		t.Fatal(err)
	}

	through57, err := catalogMigrationFilesThrough(57)
	if err != nil {
		t.Fatal(err)
	}
	upgraded, err := NewWithFS(testDB, through57).Up(ctx)
	if err != nil || len(upgraded) != 1 || upgraded[0].Version != 57 {
		t.Fatalf("upgrade to 057: applied=%+v err=%v", upgraded, err)
	}
	for _, table := range []string{"membership_plans", "membership_plan_versions", "membership_plan_entitlements", "membership_plan_command_requests", "membership_plan_audit_events"} {
		var count int
		if err := testDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("unexpected backfill in %s: count=%d err=%v", table, count, err)
		}
	}

	const planID = "50000000-0000-4000-8000-000000000001"
	const versionOne = "60000000-0000-4000-8000-000000000001"
	const versionTwo = "60000000-0000-4000-8000-000000000002"
	const entitlementOne = "70000000-0000-4000-8000-000000000001"
	if _, err := testDB.ExecContext(ctx, `INSERT INTO membership_plans (id, organization_id, home_unit_id, name, name_key) VALUES ($1, $2, $2, 'Pilates 2x', 'pilates 2x')`, planID, unitOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO membership_plan_versions (id, organization_id, plan_id, version_number, revision, status, name, price_cents, currency, billing_cycle) VALUES ($1, $2, $3, 1, 1, 'draft', 'Pilates 2x', 26000, 'BRL', 'monthly')`, versionOne, unitOne, planID); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO membership_plan_entitlements (id, organization_id, plan_version_id, modality_id, access_type, quantity, period_unit) VALUES ($1, $2, $3, $4, 'sessions_per_period', 2, 'week')`, entitlementOne, unitOne, versionOne, modalityOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `UPDATE membership_plan_entitlements SET daily_limit=1 WHERE id=$1`, entitlementOne); err != nil {
		t.Fatalf("database rejected draft entitlement mutation: %v", err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO membership_plan_entitlements (id, organization_id, plan_version_id, modality_id, access_type) VALUES ('70000000-0000-4000-8000-000000000002', $1, $2, $3, 'open_access')`, unitOne, versionOne, modalityTwo); err == nil {
		t.Fatal("database accepted a catalog target from another organization")
	}
	if _, err := testDB.ExecContext(ctx, `UPDATE membership_plan_versions SET status='published', published_at=NOW() WHERE id=$1`, versionOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `UPDATE membership_plan_versions SET price_cents=27000 WHERE id=$1`, versionOne); err == nil {
		t.Fatal("database allowed published version mutation")
	}
	if _, err := testDB.ExecContext(ctx, `UPDATE membership_plan_entitlements SET quantity=3 WHERE id=$1`, entitlementOne); err == nil {
		t.Fatal("database allowed published entitlement mutation")
	}
	if _, err := testDB.ExecContext(ctx, `DELETE FROM membership_plan_entitlements WHERE id=$1`, entitlementOne); err == nil {
		t.Fatal("database allowed published entitlement deletion")
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO membership_plan_versions (id, organization_id, plan_id, version_number, revision, status, name, price_cents, currency, billing_cycle) VALUES ($1, $2, $3, 2, 1, 'draft', 'Pilates 2x', 26000, 'BRL', 'monthly')`, versionTwo, unitOne, planID); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `UPDATE membership_plan_entitlements SET plan_version_id=$2 WHERE id=$1`, entitlementOne, versionTwo); err == nil {
		t.Fatal("database allowed published entitlement relocation to a draft")
	}
	var publishedRights, draftRights int
	if err := testDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM membership_plan_entitlements WHERE plan_version_id=$1`, versionOne).Scan(&publishedRights); err != nil {
		t.Fatal(err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM membership_plan_entitlements WHERE plan_version_id=$1`, versionTwo).Scan(&draftRights); err != nil {
		t.Fatal(err)
	}
	if publishedRights != 1 || draftRights != 0 {
		t.Fatalf("published entitlement relocation changed rights: published=%d draft=%d", publishedRights, draftRights)
	}
	if _, err := testDB.ExecContext(ctx, `UPDATE membership_plan_versions SET status='retired' WHERE id=$1`, versionOne); err != nil {
		t.Fatalf("published to retired transition rejected: %v", err)
	}
	if _, err := testDB.ExecContext(ctx, `UPDATE membership_plan_entitlements SET plan_version_id=$2 WHERE id=$1`, entitlementOne, versionTwo); err == nil {
		t.Fatal("database allowed retired entitlement relocation to a draft")
	}
	if _, err := testDB.ExecContext(ctx, `UPDATE membership_plan_versions SET status='published' WHERE id=$1`, versionOne); err == nil {
		t.Fatal("database allowed retired version mutation")
	}
	repeated, err := NewWithFS(testDB, through57).Up(ctx)
	if err != nil || len(repeated) != 0 {
		t.Fatalf("second migrator run: applied=%+v err=%v", repeated, err)
	}
}
