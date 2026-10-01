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

func TestMultiUnitAccessMigrationBackfillAndIsolation(t *testing.T) {
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
	testDatabase := "spec001_c_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	through61, err := migrationFilesThrough(61)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewWithFS(testDB, through61).Up(ctx); err != nil {
		t.Fatal(err)
	}
	const (
		unitOne = "10000000-0000-4000-8000-000000000001"
		owner   = "20000000-0000-4000-8000-000000000001"
		coach   = "20000000-0000-4000-8000-000000000002"
	)
	if _, err := testDB.ExecContext(ctx, `INSERT INTO boxes (id, name, status, created_at, updated_at) VALUES ($1, 'Unidade Centro', 'active', NOW(), NOW())`, unitOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `
		INSERT INTO users (id, box_id, name, email, password_hash, role, auth_version, active, created_at, updated_at)
		VALUES
			($2, $1, 'Owner', 'owner-multi@example.test', 'hash', 'OWNER', 1, TRUE, NOW(), NOW()),
			($3, $1, 'Coach', 'coach-multi@example.test', 'hash', 'COACH', 1, TRUE, NOW(), NOW())
	`, unitOne, owner, coach); err != nil {
		t.Fatal(err)
	}
	through62, err := migrationFilesThrough(62)
	if err != nil {
		t.Fatal(err)
	}
	applied, err := NewWithFS(testDB, through62).Up(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 1 || applied[0].Version != 62 {
		t.Fatalf("applied = %+v, want only 062", applied)
	}
	for _, table := range []string{"organization_user_memberships", "user_unit_access"} {
		var count int
		if err := testDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 2 {
			t.Fatalf("%s rows = %d, want 2", table, count)
		}
	}
	var timezone, nameKey string
	if err := testDB.QueryRowContext(ctx, `SELECT timezone, name_key FROM boxes WHERE id = $1`, unitOne).Scan(&timezone, &nameKey); err != nil {
		t.Fatal(err)
	}
	if timezone != "America/Sao_Paulo" || nameKey != "unidade centro" {
		t.Fatalf("unit defaults = %q, %q", timezone, nameKey)
	}
	const unitTwo = "10000000-0000-4000-8000-000000000002"
	if _, err := testDB.ExecContext(ctx, `INSERT INTO boxes (id, organization_id, name, name_key, timezone, status, created_at, updated_at) VALUES ($1, $2, 'Unidade Norte', 'unidade norte', 'America/Manaus', 'active', NOW(), NOW())`, unitTwo, unitOne); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO boxes (id, organization_id, name, status, created_at, updated_at) VALUES ('10000000-0000-4000-8000-000000000004', $1, '  unidade   NORTE ', 'active', NOW(), NOW())`, unitOne); err == nil {
		t.Fatal("database accepted a normalized duplicate unit name")
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO user_unit_access (organization_id, user_id, unit_id, active, granted_by_user_id) VALUES ($1, $2, $3, TRUE, $2)`, unitOne, owner, unitTwo); err != nil {
		t.Fatal(err)
	}
	const otherUnit = "10000000-0000-4000-8000-000000000003"
	if _, err := testDB.ExecContext(ctx, `INSERT INTO boxes (id, name, status, created_at, updated_at) VALUES ($1, 'Outra organização', 'active', NOW(), NOW())`, otherUnit); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO user_unit_access (organization_id, user_id, unit_id, active) VALUES ($1, $2, $3, TRUE)`, unitOne, owner, otherUnit); err == nil {
		t.Fatal("database accepted cross-organization unit access")
	}
	const newCoach = "20000000-0000-4000-8000-000000000003"
	if _, err := testDB.ExecContext(ctx, `INSERT INTO users (id, box_id, name, email, password_hash, role, auth_version, active, created_at, updated_at) VALUES ($1, $2, 'Novo Coach', 'new-coach@example.test', 'hash', 'COACH', 1, TRUE, NOW(), NOW())`, newCoach, unitOne); err != nil {
		t.Fatal(err)
	}
	var triggerRows int
	if err := testDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_unit_access WHERE user_id = $1 AND unit_id = $2 AND active`, newCoach, unitOne).Scan(&triggerRows); err != nil {
		t.Fatal(err)
	}
	if triggerRows != 1 {
		t.Fatalf("new coach trigger access rows = %d", triggerRows)
	}
	repeated, err := NewWithFS(testDB, through62).Up(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(repeated) != 0 {
		t.Fatalf("second migrator run applied %+v", repeated)
	}
}
