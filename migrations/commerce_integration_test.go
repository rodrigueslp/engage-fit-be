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

func TestCommerceMigrationIsolationAndReconciliation(t *testing.T) {
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
	databaseName := "spec013_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = adminDB.ExecContext(ctx, "CREATE DATABASE "+databaseName); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		if _, dropErr := adminDB.ExecContext(cleanup, "DROP DATABASE IF EXISTS "+databaseName+" WITH (FORCE)"); dropErr != nil {
			t.Errorf("drop database: %v", dropErr)
		}
	}()
	testConfig := adminConfig.Copy()
	testConfig.Database = databaseName
	db := stdlib.OpenDB(*testConfig)
	defer db.Close()
	applied, err := New(db).Up(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) == 0 || applied[len(applied)-1].Version != 71 {
		t.Fatalf("last migration = %+v", applied)
	}
	const unitOne = "10000000-0000-4000-8000-000000000001"
	const unitTwo = "10000000-0000-4000-8000-000000000002"
	const owner = "20000000-0000-4000-8000-000000000001"
	if _, err = db.ExecContext(ctx, `INSERT INTO boxes (id, name) VALUES ($1, 'Centro')`, unitOne); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, `INSERT INTO boxes (id, organization_id, name) VALUES ($1, $2, 'Norte')`, unitTwo, unitOne); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, `INSERT INTO users (id, box_id, name, email, password_hash, role) VALUES ($1, $2, 'Owner', 'commerce@example.test', 'hash', 'OWNER')`, owner, unitOne); err != nil {
		t.Fatal(err)
	}
	product, sale := uuid.NewString(), uuid.NewString()
	if _, err = db.ExecContext(ctx, `INSERT INTO commerce_products (id, organization_id, unit_id, name, price_cents, created_by_user_id) VALUES ($1, $2, $2, 'Água', 500, $3)`, product, unitOne, owner); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO commerce_sales (id, organization_id, unit_id, total_cents, created_by_user_id, sold_at) VALUES ($1, $2, $2, 1000, $3, NOW())`, sale, unitOne, owner); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO commerce_sale_items (id, sale_id, organization_id, unit_id, product_id, product_name, quantity, unit_price_cents, total_cents) VALUES (gen_random_uuid(), $1, $2, $2, $3, 'Água', 2, 500, 1000)`, sale, unitOne, product); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO commerce_sale_payments (id, sale_id, organization_id, unit_id, method, amount_cents, occurred_at, actor_user_id, reference) VALUES (gen_random_uuid(), $1, $2, $2, 'cash', 1000, NOW(), $3, 'manual:test')`, sale, unitOne, owner); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	badTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	badSale := uuid.NewString()
	_, _ = badTx.ExecContext(ctx, `INSERT INTO commerce_sales (id, organization_id, unit_id, total_cents, created_by_user_id, sold_at) VALUES ($1, $2, $2, 500, $3, NOW())`, badSale, unitOne, owner)
	_, _ = badTx.ExecContext(ctx, `INSERT INTO commerce_sale_items (id, sale_id, organization_id, unit_id, product_id, product_name, quantity, unit_price_cents, total_cents) VALUES (gen_random_uuid(), $1, $2, $2, $3, 'Água', 1, 500, 500)`, badSale, unitOne, product)
	_, _ = badTx.ExecContext(ctx, `INSERT INTO commerce_sale_payments (id, sale_id, organization_id, unit_id, method, amount_cents, occurred_at, actor_user_id, reference) VALUES (gen_random_uuid(), $1, $2, $2, 'cash', 400, NOW(), $3, 'manual:bad')`, badSale, unitOne, owner)
	if err = badTx.Commit(); err == nil {
		t.Fatal("database accepted unreconciled sale")
	}
	if _, err = db.ExecContext(ctx, `INSERT INTO commerce_sale_items (id, sale_id, organization_id, unit_id, product_id, product_name, quantity, unit_price_cents, total_cents) VALUES (gen_random_uuid(), $1, $2, $3, $4, 'Água', 1, 500, 500)`, sale, unitOne, unitTwo, product); err == nil {
		t.Fatal("database accepted cross-unit sale item")
	}
	repeated, err := New(db).Up(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(repeated) != 0 {
		t.Fatalf("second migration run applied %+v", repeated)
	}
}
