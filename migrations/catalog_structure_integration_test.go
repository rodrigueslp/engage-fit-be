package migrations

import (
	"context"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func TestCatalogStructureUpgradesPopulatedOrganizationSchema(t *testing.T) {
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

	testDatabase := "spec001_b_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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

	through53, err := catalogMigrationFilesThrough(53)
	if err != nil {
		t.Fatal(err)
	}
	applied, err := NewWithFS(testDB, through53).Up(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 53 {
		t.Fatalf("applied %d foundation migrations, want 53", len(applied))
	}

	const (
		unitOne  = "10000000-0000-0000-0000-000000000001"
		unitTwo  = "10000000-0000-0000-0000-000000000002"
		ownerTwo = "20000000-0000-0000-0000-000000000002"
	)
	if _, err := testDB.ExecContext(ctx, `
		INSERT INTO boxes (id, name, status, created_at, updated_at)
		VALUES
			($1, 'Catalog Unit One', 'active', NOW(), NOW()),
			($2, 'Catalog Unit Two', 'active', NOW(), NOW())
	`, unitOne, unitTwo); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `
		INSERT INTO users (id, box_id, name, email, password_hash, role, auth_version, active, created_at, updated_at)
		VALUES ($1, $2, 'Owner Two', 'owner-two@example.test', 'hash', 'OWNER', 1, TRUE, NOW(), NOW())
	`, ownerTwo, unitTwo); err != nil {
		t.Fatal(err)
	}

	through54, err := catalogMigrationFilesThrough(54)
	if err != nil {
		t.Fatal(err)
	}
	upgraded, err := NewWithFS(testDB, through54).Up(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(upgraded) != 1 || upgraded[0].Version != 54 {
		t.Fatalf("applied migrations = %+v, want only 054", upgraded)
	}

	for _, table := range []string{"modalities", "unit_modalities", "spaces", "catalog_audit_events"} {
		var count int
		if err := testDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
			t.Fatalf("query %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("migration backfilled %d rows into %s", count, table)
		}
	}

	const modalityID = "30000000-0000-0000-0000-000000000001"
	if _, err := testDB.ExecContext(ctx, `
		INSERT INTO modalities (id, organization_id, name, name_key)
		VALUES ($1, $2, 'Cross Training', 'cross training')
	`, modalityID, unitOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `
		INSERT INTO unit_modalities (organization_id, unit_id, modality_id)
		VALUES ($1, $2, $3)
	`, unitTwo, unitOne, modalityID); err == nil {
		t.Fatal("database accepted a modality/unit association across organizations")
	}
	if _, err := testDB.ExecContext(ctx, `
		INSERT INTO catalog_audit_events (id, organization_id, unit_id, actor_user_id, action, entity_type, entity_id)
		VALUES ('40000000-0000-0000-0000-000000000001', $1, $2, $3,
		        'space.created', 'space', $4)
	`, unitTwo, unitOne, ownerTwo, modalityID); err == nil {
		t.Fatal("database accepted a catalog audit event with a unit from another organization")
	}

	repeated, err := NewWithFS(testDB, through54).Up(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(repeated) != 0 {
		t.Fatalf("second migrator run applied %+v", repeated)
	}
}

func catalogMigrationFilesThrough(version int64) (fstest.MapFS, error) {
	loaded, err := Load(Files)
	if err != nil {
		return nil, err
	}
	result := make(fstest.MapFS, version)
	for _, migration := range loaded {
		if migration.Version > version {
			break
		}
		body, err := fs.ReadFile(Files, migration.Filename)
		if err != nil {
			return nil, err
		}
		result[migration.Filename] = &fstest.MapFile{Data: body}
	}
	return result, nil
}
