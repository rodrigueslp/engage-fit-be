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

func TestServiceOfferingsUpgradeCatalogSchema(t *testing.T) {
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
	testDatabase := "spec001_d_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	through54, err := catalogMigrationFilesThrough(54)
	if err != nil {
		t.Fatal(err)
	}
	if applied, err := NewWithFS(testDB, through54).Up(ctx); err != nil || len(applied) != 54 {
		t.Fatalf("apply through 054: count=%d err=%v", len(applied), err)
	}

	const (
		unitOne  = "10000000-0000-0000-0000-000000000001"
		unitTwo  = "10000000-0000-0000-0000-000000000002"
		ownerOne = "20000000-0000-0000-0000-000000000001"
	)
	if _, err := testDB.ExecContext(ctx, `
		INSERT INTO boxes (id, name, status, created_at, updated_at)
		VALUES ($1, 'Service Unit One', 'active', NOW(), NOW()),
		       ($2, 'Service Unit Two', 'active', NOW(), NOW())
	`, unitOne, unitTwo); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `
		INSERT INTO users (id, box_id, name, email, password_hash, role, auth_version, active, created_at, updated_at)
		VALUES ($1, $2, 'Owner One', 'service-owner@example.test', 'hash', 'OWNER', 1, TRUE, NOW(), NOW())
	`, ownerOne, unitOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `
		INSERT INTO modalities (id, organization_id, name, name_key)
		VALUES ('30000000-0000-0000-0000-000000000001', $1, 'Pilates', 'pilates')
	`, unitOne); err != nil {
		t.Fatal(err)
	}

	through55, err := catalogMigrationFilesThrough(55)
	if err != nil {
		t.Fatal(err)
	}
	upgraded, err := NewWithFS(testDB, through55).Up(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(upgraded) != 1 || upgraded[0].Version != 55 {
		t.Fatalf("applied migrations = %+v, want only 055", upgraded)
	}
	for _, table := range []string{"service_offerings", "unit_service_offerings"} {
		var count int
		if err := testDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("migration backfill in %s: count=%d err=%v", table, count, err)
		}
	}
	var modalityCount int
	if err := testDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM modalities WHERE organization_id = $1", unitOne).Scan(&modalityCount); err != nil || modalityCount != 1 {
		t.Fatalf("existing catalog was not preserved: count=%d err=%v", modalityCount, err)
	}

	const serviceID = "40000000-0000-0000-0000-000000000001"
	if _, err := testDB.ExecContext(ctx, `
		INSERT INTO service_offerings (id, organization_id, name, name_key)
		VALUES ($1, $2, 'Recovery', 'recovery')
	`, serviceID, unitOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `
		INSERT INTO unit_service_offerings (organization_id, unit_id, service_offering_id)
		VALUES ($1, $2, $3)
	`, unitOne, unitTwo, serviceID); err == nil {
		t.Fatal("database accepted service association across organizations")
	}
	if _, err := testDB.ExecContext(ctx, `
		INSERT INTO catalog_audit_events (id, organization_id, unit_id, actor_user_id, action, entity_type, entity_id)
		VALUES ('50000000-0000-0000-0000-000000000001', $1, $1, $2,
		        'service_offering.created', 'service_offering', $3)
	`, unitOne, ownerOne, serviceID); err != nil {
		t.Fatalf("new catalog audit type rejected: %v", err)
	}
	if _, err := testDB.ExecContext(ctx, `
		INSERT INTO catalog_audit_events (id, organization_id, unit_id, actor_user_id, action, entity_type, entity_id)
		VALUES ('50000000-0000-0000-0000-000000000002', $1, $1, $2,
		        'space.created', 'space', $3)
	`, unitOne, ownerOne, serviceID); err != nil {
		t.Fatalf("existing catalog audit type regressed: %v", err)
	}
	repeated, err := NewWithFS(testDB, through55).Up(ctx)
	if err != nil || len(repeated) != 0 {
		t.Fatalf("second migrator run: applied=%+v err=%v", repeated, err)
	}
}
