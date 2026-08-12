package migrations

import (
	"context"
	"io/fs"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func TestOrganizationFoundationUpgradesPopulatedSchema(t *testing.T) {
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

	testDatabase := "spec001_a_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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

	through52, err := migrationFilesThrough(52)
	if err != nil {
		t.Fatal(err)
	}
	legacyApplied, err := NewWithFS(testDB, through52).Up(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(legacyApplied) != 52 {
		t.Fatalf("applied %d legacy migrations, want 52", len(legacyApplied))
	}

	const (
		activeBoxID       = "10000000-0000-0000-0000-000000000001"
		suspendedBoxID    = "10000000-0000-0000-0000-000000000002"
		archivedBoxID     = "10000000-0000-0000-0000-000000000003"
		suspendedOwnerID  = "20000000-0000-0000-0000-000000000002"
		legacyInsertBoxID = "10000000-0000-0000-0000-000000000004"
	)
	if _, err := testDB.ExecContext(ctx, `
		INSERT INTO boxes (id, name, status, status_reason, created_at, updated_at)
		VALUES
			($1, 'Legacy Active', 'active', '', NOW() - INTERVAL '3 years', NOW() - INTERVAL '2 years'),
			($2, 'Legacy Suspended', 'suspended', 'billing review', NOW() - INTERVAL '3 years', NOW() - INTERVAL '1 year'),
			($3, 'Legacy Archived', 'archived', 'closed', NOW() - INTERVAL '4 years', NOW() - INTERVAL '6 months')
	`, activeBoxID, suspendedBoxID, archivedBoxID); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `
		INSERT INTO users (id, box_id, name, email, password_hash, role, auth_version, active, created_at, updated_at)
		VALUES
			('20000000-0000-0000-0000-000000000001', $1, 'Owner Active', 'active@example.test', 'hash', 'OWNER', 1, TRUE, NOW(), NOW()),
			($4, $2, 'Owner Suspended', 'suspended@example.test', 'hash', 'OWNER', 1, TRUE, NOW(), NOW()),
			('20000000-0000-0000-0000-000000000003', $3, 'Owner Archived', 'archived@example.test', 'hash', 'OWNER', 1, TRUE, NOW(), NOW())
	`, activeBoxID, suspendedBoxID, archivedBoxID, suspendedOwnerID); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `
		UPDATE boxes
		SET status_changed_by = $2, status_changed_at = NOW() - INTERVAL '1 year'
		WHERE id = $1
	`, suspendedBoxID, suspendedOwnerID); err != nil {
		t.Fatal(err)
	}

	applied, err := New(testDB).Up(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 1 || applied[0].Version != 53 {
		t.Fatalf("applied migrations = %+v, want only 053", applied)
	}

	var withoutOrganization int
	if err := testDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM boxes WHERE organization_id IS NULL`).Scan(&withoutOrganization); err != nil {
		t.Fatal(err)
	}
	if withoutOrganization != 0 {
		t.Fatalf("boxes without organization = %d", withoutOrganization)
	}

	rows, err := testDB.QueryContext(ctx, `
		SELECT b.id, b.name, b.status, o.id, o.name, o.status,
		       COALESCE(o.status_reason, ''), COALESCE(o.status_changed_by::text, '')
		FROM boxes b
		JOIN organizations o ON o.id = b.organization_id
		ORDER BY b.id
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	type organizationSnapshot struct {
		boxID, boxName, boxStatus              string
		organizationID, organizationName       string
		organizationStatus, organizationReason string
		statusChangedBy                        string
	}
	want := map[string]organizationSnapshot{
		activeBoxID: {
			boxID: activeBoxID, boxName: "Legacy Active", boxStatus: "active",
			organizationID: activeBoxID, organizationName: "Legacy Active", organizationStatus: "active",
		},
		suspendedBoxID: {
			boxID: suspendedBoxID, boxName: "Legacy Suspended", boxStatus: "suspended",
			organizationID: suspendedBoxID, organizationName: "Legacy Suspended", organizationStatus: "suspended",
			organizationReason: "billing review", statusChangedBy: suspendedOwnerID,
		},
		archivedBoxID: {
			boxID: archivedBoxID, boxName: "Legacy Archived", boxStatus: "archived",
			organizationID: archivedBoxID, organizationName: "Legacy Archived", organizationStatus: "archived",
			organizationReason: "closed",
		},
	}
	seen := 0
	for rows.Next() {
		var got organizationSnapshot
		if err := rows.Scan(
			&got.boxID, &got.boxName, &got.boxStatus,
			&got.organizationID, &got.organizationName, &got.organizationStatus,
			&got.organizationReason, &got.statusChangedBy,
		); err != nil {
			t.Fatal(err)
		}
		expected, exists := want[got.boxID]
		if !exists {
			t.Fatalf("unexpected backfilled box %q", got.boxID)
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("backfill for box %s = %+v, want %+v", got.boxID, got, expected)
		}
		seen++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if seen != len(want) {
		t.Fatalf("backfilled rows = %d, want %d", seen, len(want))
	}

	if _, err := testDB.ExecContext(ctx, `
		INSERT INTO boxes (id, name, status, created_at, updated_at)
		VALUES ($1, 'Legacy Binary Insert', 'active', NOW(), NOW())
	`, legacyInsertBoxID); err != nil {
		t.Fatal(err)
	}
	var legacyOrganizationID, legacyOrganizationName string
	if err := testDB.QueryRowContext(ctx, `
		SELECT b.organization_id, o.name
		FROM boxes b
		JOIN organizations o ON o.id = b.organization_id
		WHERE b.id = $1
	`, legacyInsertBoxID).Scan(&legacyOrganizationID, &legacyOrganizationName); err != nil {
		t.Fatal(err)
	}
	if legacyOrganizationID != legacyInsertBoxID || legacyOrganizationName != "Legacy Binary Insert" {
		t.Fatalf("legacy insert organization = (%q, %q)", legacyOrganizationID, legacyOrganizationName)
	}

	repeated, err := New(testDB).Up(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(repeated) != 0 {
		t.Fatalf("second migrator run applied %+v", repeated)
	}
}

func migrationFilesThrough(version int64) (fstest.MapFS, error) {
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
