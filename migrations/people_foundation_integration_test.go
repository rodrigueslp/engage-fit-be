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

func TestPeopleFoundationBackfillsOneToOneAndReconcilesLegacyStudents(t *testing.T) {
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
	testDatabase := "spec002_a_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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

	through54, err := migrationFilesThrough(54)
	if err != nil {
		t.Fatal(err)
	}
	if applied, err := NewWithFS(testDB, through54).Up(ctx); err != nil || len(applied) != 54 {
		t.Fatalf("apply through 054: %d migrations, %v", len(applied), err)
	}
	const (
		unitOne     = "10000000-0000-0000-0000-000000000001"
		unitTwo     = "10000000-0000-0000-0000-000000000002"
		studentOne  = "30000000-0000-0000-0000-000000000001"
		studentTwo  = "30000000-0000-0000-0000-000000000002"
		studentAnon = "30000000-0000-0000-0000-000000000003"
	)
	if _, err := testDB.ExecContext(ctx, `
		INSERT INTO boxes (id, name, status, created_at, updated_at) VALUES
		($1, 'Unit One', 'active', NOW(), NOW()), ($2, 'Unit Two', 'active', NOW(), NOW())
	`, unitOne, unitTwo); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `
		INSERT INTO students (id, box_id, name, email, phone, source, external_id, risk_status, contact_status, contact_status_source, membership_started_at, membership_started_source, anonymized_at, created_at, updated_at) VALUES
		($1, $4, ' Maria   Silva ', 'MARIA@EXAMPLE.TEST', '(11) 99999-9999', 'wellhub', 'wellhub-1', 'active', 'opted_in', 'legacy-consent', '2025-01-02', 'manual', NULL, NOW() - INTERVAL '1 year', NOW()),
		($2, $4, 'Maria Silva', 'second@example.test', '5511999999999', 'totalpass', 'totalpass-1', 'paused', 'unknown', '', NULL, NULL, NULL, NOW() - INTERVAL '6 months', NOW()),
		($3, $5, 'Aluno anonimizado', 'secret@example.test', '11988887777', 'wellhub', 'anonymized:legacy', 'paused', 'opted_out', 'privacy_anonymization', NULL, NULL, NOW(), NOW() - INTERVAL '2 years', NOW())
	`, studentOne, studentTwo, studentAnon, unitOne, unitTwo); err != nil {
		t.Fatal(err)
	}

	through56, err := migrationFilesThrough(56)
	if err != nil {
		t.Fatal(err)
	}
	applied, err := NewWithFS(testDB, through56).Up(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 2 || applied[0].Version != 55 || applied[1].Version != 56 {
		t.Fatalf("applied migrations = %+v, want 055 and 056", applied)
	}
	var students, linked, persons, relationships, distinctPeople int
	if err := testDB.QueryRowContext(ctx, `SELECT COUNT(*), COUNT(person_id), COUNT(DISTINCT person_id) FROM students`).Scan(&students, &linked, &distinctPeople); err != nil {
		t.Fatal(err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM persons`).Scan(&persons); err != nil {
		t.Fatal(err)
	}
	if err := testDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM person_unit_relationships`).Scan(&relationships); err != nil {
		t.Fatal(err)
	}
	if students != 3 || linked != students || distinctPeople != students || persons != students || relationships != students {
		t.Fatalf("reconciliation students=%d linked=%d distinct=%d persons=%d relationships=%d", students, linked, distinctPeople, persons, relationships)
	}
	var personID, name, phone, email, source, externalID, contactStatus, contactSource string
	if err := testDB.QueryRowContext(ctx, `
		SELECT s.person_id, p.name, COALESCE(p.phone_e164, ''), COALESCE(p.email, ''), s.source, s.external_id, s.contact_status, s.contact_status_source
		FROM students s JOIN persons p ON p.id = s.person_id WHERE s.id = $1
	`, studentOne).Scan(&personID, &name, &phone, &email, &source, &externalID, &contactStatus, &contactSource); err != nil {
		t.Fatal(err)
	}
	if personID != studentOne || name != "Maria Silva" || phone != "+5511999999999" || email != "maria@example.test" || source != "wellhub" || externalID != "wellhub-1" || contactStatus != "opted_in" || contactSource != "legacy-consent" {
		t.Fatalf("legacy projection changed unexpectedly: %q %q %q %q %q %q %q %q", personID, name, phone, email, source, externalID, contactStatus, contactSource)
	}
	var anonymizedStatus, anonymizedPhone, anonymizedEmail string
	if err := testDB.QueryRowContext(ctx, `SELECT profile_status, COALESCE(phone_e164, ''), COALESCE(email, '') FROM persons WHERE id = $1`, studentAnon).Scan(&anonymizedStatus, &anonymizedPhone, &anonymizedEmail); err != nil {
		t.Fatal(err)
	}
	if anonymizedStatus != "anonymized" || anonymizedPhone != "" || anonymizedEmail != "" {
		t.Fatalf("anonymized person leaked PII: %q %q %q", anonymizedStatus, anonymizedPhone, anonymizedEmail)
	}

	legacyAfterMigration := uuid.NewString()
	if _, err := testDB.ExecContext(ctx, `INSERT INTO students (id, box_id, name, source, external_id, risk_status, contact_status, created_at, updated_at) VALUES ($1, $2, 'Future import', 'wellhub', 'future-1', 'active', 'unknown', NOW(), NOW())`, legacyAfterMigration, unitOne); err != nil {
		t.Fatalf("nullable compatibility person_id rejected a future import: %v", err)
	}
	var futurePersonID *string
	if err := testDB.QueryRowContext(ctx, `SELECT person_id FROM students WHERE id = $1`, legacyAfterMigration).Scan(&futurePersonID); err != nil || futurePersonID != nil {
		t.Fatalf("future legacy student person_id = %v, %v", futurePersonID, err)
	}
	if _, err := testDB.ExecContext(ctx, `INSERT INTO person_unit_relationships (id, organization_id, person_id, unit_id, relationship_status, started_at) VALUES ($1, $2, $3, $4, 'active', NOW())`, uuid.NewString(), unitTwo, studentOne, unitTwo); err == nil {
		t.Fatal("database accepted a cross-organization person relationship")
	}
	repeated, err := NewWithFS(testDB, through56).Up(ctx)
	if err != nil || len(repeated) != 0 {
		t.Fatalf("second migrator run = %+v, %v", repeated, err)
	}
}
