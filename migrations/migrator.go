package migrations

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"
)

//go:embed *.sql
var Files embed.FS

const advisoryLockID int64 = 731947521

var migrationFilename = regexp.MustCompile(`^(\d+)_([a-z0-9_]+)\.sql$`)

type Migration struct {
	Version  int64
	Name     string
	Filename string
	Checksum string
	SQL      string
}

type AppliedMigration struct {
	Version         int64
	Name            string
	Checksum        string
	AppliedAt       time.Time
	ExecutionTimeMS int64
}

type Status struct {
	Migration Migration
	Applied   *AppliedMigration
}

type Migrator struct {
	db    *sql.DB
	files fs.FS
}

func New(db *sql.DB) *Migrator {
	return &Migrator{db: db, files: Files}
}

func NewWithFS(db *sql.DB, files fs.FS) *Migrator {
	return &Migrator{db: db, files: files}
}

func Load(files fs.FS) ([]Migration, error) {
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}
	migrations := make([]Migration, 0, len(entries))
	seen := make(map[int64]string)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		parts := migrationFilename.FindStringSubmatch(entry.Name())
		if parts == nil {
			return nil, fmt.Errorf("invalid migration filename %q", entry.Name())
		}
		version, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || version <= 0 {
			return nil, fmt.Errorf("invalid migration version in %q", entry.Name())
		}
		if previous, exists := seen[version]; exists {
			return nil, fmt.Errorf("migration version %d duplicated by %q and %q", version, previous, entry.Name())
		}
		body, err := fs.ReadFile(files, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", entry.Name(), err)
		}
		digest := sha256.Sum256(body)
		seen[version] = entry.Name()
		migrations = append(migrations, Migration{
			Version:  version,
			Name:     parts[2],
			Filename: entry.Name(),
			Checksum: hex.EncodeToString(digest[:]),
			SQL:      string(body),
		})
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })
	for index, migration := range migrations {
		expected := int64(index + 1)
		if migration.Version != expected {
			return nil, fmt.Errorf("migration sequence has a gap: expected %d, found %d", expected, migration.Version)
		}
	}
	if len(migrations) == 0 {
		return nil, errors.New("no migrations found")
	}
	return migrations, nil
}

func (m *Migrator) Up(ctx context.Context) ([]Migration, error) {
	all, err := Load(m.files)
	if err != nil {
		return nil, err
	}
	conn, release, err := m.lock(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	applied, err := loadApplied(ctx, conn)
	if err != nil {
		return nil, err
	}
	if len(applied) == 0 {
		nonEmpty, err := schemaHasApplicationTables(ctx, conn)
		if err != nil {
			return nil, err
		}
		if nonEmpty {
			return nil, errors.New("schema is not empty and has no migration history; verify it and run migrate baseline --through=<version>")
		}
	}
	if err := verifyChecksums(all, applied); err != nil {
		return nil, err
	}

	executed := make([]Migration, 0)
	for _, migration := range all {
		if _, exists := applied[migration.Version]; exists {
			continue
		}
		startedAt := time.Now()
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return executed, fmt.Errorf("begin migration %s: %w", migration.Filename, err)
		}
		if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
			_ = tx.Rollback()
			return executed, fmt.Errorf("apply migration %s: %w", migration.Filename, err)
		}
		executionTime := time.Since(startedAt).Milliseconds()
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, name, checksum, execution_time_ms) VALUES ($1, $2, $3, $4)`, migration.Version, migration.Name, migration.Checksum, executionTime); err != nil {
			_ = tx.Rollback()
			return executed, fmt.Errorf("record migration %s: %w", migration.Filename, err)
		}
		if err := tx.Commit(); err != nil {
			return executed, fmt.Errorf("commit migration %s: %w", migration.Filename, err)
		}
		executed = append(executed, migration)
	}
	return executed, nil
}

func (m *Migrator) Baseline(ctx context.Context, through int64) ([]Migration, error) {
	all, err := Load(m.files)
	if err != nil {
		return nil, err
	}
	if through <= 0 || through > all[len(all)-1].Version {
		return nil, fmt.Errorf("baseline version must be between 1 and %d", all[len(all)-1].Version)
	}
	conn, release, err := m.lock(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	applied, err := loadApplied(ctx, conn)
	if err != nil {
		return nil, err
	}
	if len(applied) > 0 {
		return nil, errors.New("baseline refused because migration history already exists")
	}
	nonEmpty, err := schemaHasApplicationTables(ctx, conn)
	if err != nil {
		return nil, err
	}
	if !nonEmpty {
		return nil, errors.New("baseline refused on an empty schema; run migrate up instead")
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	marked := make([]Migration, 0, through)
	for _, migration := range all {
		if migration.Version > through {
			break
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, name, checksum, execution_time_ms) VALUES ($1, $2, $3, 0)`, migration.Version, migration.Name, migration.Checksum); err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("baseline migration %s: %w", migration.Filename, err)
		}
		marked = append(marked, migration)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return marked, nil
}

func (m *Migrator) Status(ctx context.Context) ([]Status, error) {
	all, err := Load(m.files)
	if err != nil {
		return nil, err
	}
	conn, release, err := m.lock(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	applied, err := loadApplied(ctx, conn)
	if err != nil {
		return nil, err
	}
	if err := verifyChecksums(all, applied); err != nil {
		return nil, err
	}
	statuses := make([]Status, 0, len(all))
	for _, migration := range all {
		item := Status{Migration: migration}
		if record, exists := applied[migration.Version]; exists {
			copy := record
			item.Applied = &copy
		}
		statuses = append(statuses, item)
	}
	return statuses, nil
}

func (m *Migrator) lock(ctx context.Context) (*sql.Conn, func(), error) {
	conn, err := m.db.Conn(ctx)
	if err != nil {
		return nil, nil, err
	}
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, advisoryLockID); err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("acquire migration lock: %w", err)
	}
	if _, err := conn.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT PRIMARY KEY,
			name TEXT NOT NULL,
			checksum CHAR(64) NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			execution_time_ms BIGINT NOT NULL DEFAULT 0
		)`); err != nil {
		_, _ = conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, advisoryLockID)
		conn.Close()
		return nil, nil, fmt.Errorf("ensure migration table: %w", err)
	}
	release := func() {
		unlockContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = conn.ExecContext(unlockContext, `SELECT pg_advisory_unlock($1)`, advisoryLockID)
		_ = conn.Close()
	}
	return conn, release, nil
}

func loadApplied(ctx context.Context, conn *sql.Conn) (map[int64]AppliedMigration, error) {
	rows, err := conn.QueryContext(ctx, `SELECT version, name, checksum, applied_at, execution_time_ms FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	applied := make(map[int64]AppliedMigration)
	for rows.Next() {
		var record AppliedMigration
		if err := rows.Scan(&record.Version, &record.Name, &record.Checksum, &record.AppliedAt, &record.ExecutionTimeMS); err != nil {
			return nil, err
		}
		applied[record.Version] = record
	}
	return applied, rows.Err()
}

func verifyChecksums(all []Migration, applied map[int64]AppliedMigration) error {
	byVersion := make(map[int64]Migration, len(all))
	for _, migration := range all {
		byVersion[migration.Version] = migration
	}
	for version, record := range applied {
		migration, exists := byVersion[version]
		if !exists {
			return fmt.Errorf("database contains unknown migration version %d", version)
		}
		if record.Name != migration.Name || record.Checksum != migration.Checksum {
			return fmt.Errorf("migration %03d was changed after application", version)
		}
	}
	return nil
}

func schemaHasApplicationTables(ctx context.Context, conn *sql.Conn) (bool, error) {
	var exists bool
	err := conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = current_schema()
			  AND table_type = 'BASE TABLE'
			  AND table_name <> 'schema_migrations'
		)`).Scan(&exists)
	return exists, err
}
