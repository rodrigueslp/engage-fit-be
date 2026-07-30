package imports

import (
	"testing"

	"boxengage/backend/internal/ports/services"
)

func TestStudentIdentityNormalizesNameWhenNoStableIdentifierExists(t *testing.T) {
	first := studentIdentity(services.ParsedCheckin{StudentName: "  Adriana   Segatelli "})
	second := studentIdentity(services.ParsedCheckin{StudentName: "ADRIANA SEGATELLI"})

	if first != "adriana segatelli" {
		t.Fatalf("expected normalized identity, got %q", first)
	}
	if first != second {
		t.Fatalf("expected name variants to share an identity: %q != %q", first, second)
	}
}

func TestStudentIdentityStillPrefersStableExternalID(t *testing.T) {
	identity := studentIdentity(services.ParsedCheckin{
		StudentName:       "Adriana Segatelli",
		StudentExternalID: " MEMBER-123 ",
	})

	if identity != "member-123" {
		t.Fatalf("expected stable external ID, got %q", identity)
	}
}
