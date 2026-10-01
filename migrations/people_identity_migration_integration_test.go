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

func TestPeopleIdentityMigrationBackfillCreatesSafeReviews(t *testing.T) {
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
	testDatabase := "spec002_identity_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	through62, err := migrationFilesThrough(62)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewWithFS(testDB, through62).Up(ctx); err != nil {
		t.Fatal(err)
	}
	ids := []string{
		"10000000-0000-4000-8000-000000000001", "20000000-0000-4000-8000-000000000001",
		"30000000-0000-4000-8000-000000000001", "30000000-0000-4000-8000-000000000002",
		"40000000-0000-4000-8000-000000000001", "40000000-0000-4000-8000-000000000002",
		"50000000-0000-4000-8000-000000000001", "60000000-0000-4000-8000-000000000001",
	}
	unitID, ownerID, personOne, personTwo, studentOne, studentTwo, athleteID, membershipID := ids[0], ids[1], ids[2], ids[3], ids[4], ids[5], ids[6], ids[7]
	statements := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO organizations (id, name, status) VALUES ($1, 'Rede Teste', 'active')`, []any{unitID}},
		{`INSERT INTO boxes (id, organization_id, name, name_key, timezone, status) VALUES ($1, $1, 'Centro', 'centro', 'America/Sao_Paulo', 'active')`, []any{unitID}},
		{`INSERT INTO users (id, box_id, name, email, password_hash, role, auth_version, active) VALUES ($2, $1, 'Owner', 'owner-spec002@example.test', 'hash', 'OWNER', 1, TRUE)`, []any{unitID, ownerID}},
		{`INSERT INTO persons (id, organization_id, name) VALUES ($2, $1, 'Maria Um'), ($3, $1, 'Maria Dois')`, []any{unitID, personOne, personTwo}},
		{`INSERT INTO person_unit_relationships (id, organization_id, person_id, unit_id, started_at) VALUES (gen_random_uuid(), $1, $2, $1, NOW()), (gen_random_uuid(), $1, $3, $1, NOW())`, []any{unitID, personOne, personTwo}},
		{`INSERT INTO students (id, box_id, person_id, name, source, external_id, contact_status) VALUES ($2, $1, $3, 'Maria Um', 'wellhub', 'shared-external', 'unknown'), ($4, $1, $5, 'Maria Dois', 'wellhub', 'shared-external', 'unknown')`, []any{unitID, studentOne, personOne, studentTwo, personTwo}},
		{`INSERT INTO athlete_accounts (id, name, email, password_hash) VALUES ($1, 'Maria', 'athlete-spec002@example.test', 'hash')`, []any{athleteID}},
		{`INSERT INTO athlete_box_memberships (id, athlete_account_id, box_id) VALUES ($1, $2, $3)`, []any{membershipID, athleteID, unitID}},
		{`INSERT INTO athlete_student_links (id, membership_id, student_id, link_method) VALUES (gen_random_uuid(), $1, $2, 'manual_review'), (gen_random_uuid(), $1, $3, 'manual_review')`, []any{membershipID, studentOne, studentTwo}},
	}
	for _, statement := range statements {
		if _, err := testDB.ExecContext(ctx, statement.sql, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	through63, err := migrationFilesThrough(63)
	if err != nil {
		t.Fatal(err)
	}
	applied, err := NewWithFS(testDB, through63).Up(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 1 || applied[0].Version != 63 {
		t.Fatalf("applied = %+v", applied)
	}
	var identities, externalReviews, athleteReviews, athleteLinks int
	if err := testDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM person_external_identities`).Scan(&identities); err != nil {
		t.Fatal(err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM person_identity_reviews WHERE reason = 'external_identity_conflict'`).Scan(&externalReviews); err != nil {
		t.Fatal(err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM person_identity_reviews WHERE reason = 'athlete_account_conflict'`).Scan(&athleteReviews); err != nil {
		t.Fatal(err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM person_athlete_accounts`).Scan(&athleteLinks); err != nil {
		t.Fatal(err)
	}
	if identities != 1 || externalReviews != 1 || athleteReviews != 1 || athleteLinks != 0 {
		t.Fatalf("identity=%d external_reviews=%d athlete_reviews=%d athlete_links=%d", identities, externalReviews, athleteReviews, athleteLinks)
	}
	if second, err := NewWithFS(testDB, through63).Up(ctx); err != nil || len(second) != 0 {
		t.Fatalf("second migration run = %+v, %v", second, err)
	}
}
