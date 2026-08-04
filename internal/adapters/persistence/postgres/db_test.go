package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestSafeDatabaseErrorDetailsClassifiesWithoutReturningRawMessage(t *testing.T) {
	kind, sqlState := safeDatabaseErrorDetails(&pgconn.PgError{Code: "23505", Message: "sensitive database detail"})
	if kind != "unique_violation" || sqlState != "23505" {
		t.Fatalf("unexpected postgres classification: %q %q", kind, sqlState)
	}

	kind, sqlState = safeDatabaseErrorDetails(errors.New("extended protocol limited to 65535 parameters: sensitive detail"))
	if kind != "parameter_limit" || sqlState != "" {
		t.Fatalf("unexpected parameter classification: %q %q", kind, sqlState)
	}
}
