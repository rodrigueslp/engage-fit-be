package checkiningestion

import "testing"

func TestGeneratedIngestionTokenIsOnlyValidatedByHash(t *testing.T) {
	token, hash, err := newToken()
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || hash == "" || token == hash {
		t.Fatal("expected distinct token and stored hash")
	}
	if !tokenMatches(token, hash) {
		t.Fatal("generated token should match its hash")
	}
	if tokenMatches(token+"changed", hash) {
		t.Fatal("changed token must not match")
	}
}

func TestIdempotencyKeyRequiresStableConnectorValue(t *testing.T) {
	if !idempotencyPattern.MatchString("wellhub-2026-07-28") {
		t.Fatal("expected stable connector key to be accepted")
	}
	if idempotencyPattern.MatchString("short") || idempotencyPattern.MatchString("contains spaces") {
		t.Fatal("expected unsafe idempotency key to be rejected")
	}
}
