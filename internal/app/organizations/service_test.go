package organizations

import (
	"testing"

	"boxengage/backend/internal/domain"
)

func TestValidTimezone(t *testing.T) {
	if !validTimezone("America/Sao_Paulo") {
		t.Fatal("expected IANA timezone to be accepted")
	}
	if validTimezone("Sao Paulo") {
		t.Fatal("expected invalid timezone to be rejected")
	}
}

func TestUniqueIDs(t *testing.T) {
	got := uniqueIDs([]domain.ID{"unit-1", "unit-1", "", "unit-2"})
	if len(got) != 2 || got[0] != "unit-1" || got[1] != "unit-2" {
		t.Fatalf("uniqueIDs = %#v", got)
	}
}
