package repositories

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"boxengage/backend/migrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestFeatureFlagOverrideAndAuditIntegration(t *testing.T) {
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
	name := "featureflags_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := adminDB.ExecContext(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if _, err := adminDB.ExecContext(cleanupCtx, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("drop test database: %v", err)
		}
	}()
	testConfig := adminConfig.Copy()
	testConfig.Database = name
	testDB := stdlib.OpenDB(*testConfig)
	defer testDB.Close()
	if _, err := migrations.New(testDB).Up(ctx); err != nil {
		t.Fatal(err)
	}
	adminID := uuid.NewString()
	if _, err := testDB.ExecContext(ctx, `INSERT INTO users (id, name, email, password_hash, role) VALUES ($1, 'Admin', 'flags@example.test', 'hash', 'PLATFORM_ADMIN')`, adminID); err != nil {
		t.Fatal(err)
	}
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: testDB}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewFeatureFlagsRepository(gormDB)
	if err := repo.Set(ctx, "catalog", false, "Pausa de homologação", adminID, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Set(ctx, "catalog", true, "Retomada", adminID, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	items, err := repo.List(ctx)
	if err != nil || !items["catalog"] {
		t.Fatalf("unexpected overrides: %#v, %v", items, err)
	}
	var count int
	if err := testDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM admin_audit_logs WHERE target_type = 'feature_flag' AND target_id = 'catalog' AND reason IN ('Pausa de homologação', 'Retomada')`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("want two audit entries, got %d", count)
	}
}
