package security

import (
	"context"
	"testing"
)

func TestPasswordServiceHashAndCompare(t *testing.T) {
	service := NewPasswordService()

	hash, err := service.Hash(context.Background(), "secret")
	if err != nil {
		t.Fatalf("expected hash to succeed: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == "secret" {
		t.Fatal("expected password hash to differ from plain password")
	}

	if err := service.Compare(context.Background(), hash, "secret"); err != nil {
		t.Fatalf("expected password comparison to succeed: %v", err)
	}
	if err := service.Compare(context.Background(), hash, "wrong"); err == nil {
		t.Fatal("expected password comparison to fail")
	}
}
