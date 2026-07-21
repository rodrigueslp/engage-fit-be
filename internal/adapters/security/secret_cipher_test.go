package security

import (
	"encoding/base64"
	"strings"
	"testing"
)

func encodedKey(fill byte) string {
	return base64.StdEncoding.EncodeToString([]byte(strings.Repeat(string(fill), 32)))
}

func TestSecretCipherRoundTripAndAssociatedData(t *testing.T) {
	cipher, err := NewSecretCipher("key1", "key1:"+encodedKey('a'))
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := cipher.Encrypt("sensitive-value", "tenant-1:field")
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == "sensitive-value" || !strings.HasPrefix(encrypted, "enc:v1:key1:") {
		t.Fatalf("secret was not enveloped: %q", encrypted)
	}
	decrypted, err := cipher.Decrypt(encrypted, "tenant-1:field")
	if err != nil || decrypted != "sensitive-value" {
		t.Fatalf("unexpected decryption result %q, %v", decrypted, err)
	}
	if _, err := cipher.Decrypt(encrypted, "tenant-2:field"); err == nil {
		t.Fatal("expected associated data mismatch to fail authentication")
	}
}

func TestSecretCipherSupportsKeyRotationAndLegacyPlaintext(t *testing.T) {
	oldCipher, _ := NewSecretCipher("old", "old:"+encodedKey('o'))
	value, _ := oldCipher.Encrypt("secret", "scope")
	rotatedCipher, err := NewRotationSecretCipher("new", "new:"+encodedKey('n')+",old:"+encodedKey('o'))
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := rotatedCipher.Decrypt(value, "scope")
	if err != nil || decrypted != "secret" || !rotatedCipher.NeedsRotation(value) {
		t.Fatalf("old-key value should be readable and require rotation: %q, %v", decrypted, err)
	}
	legacy, err := rotatedCipher.Decrypt("legacy-plaintext", "scope")
	if err != nil || legacy != "legacy-plaintext" || !rotatedCipher.NeedsRotation(legacy) {
		t.Fatal("legacy plaintext should be readable only for controlled migration")
	}
}

func TestSecretCipherRejectsLegacyPlaintextInRuntimeMode(t *testing.T) {
	cipher, _ := NewSecretCipher("key1", "key1:"+encodedKey('a'))
	if _, err := cipher.Decrypt("legacy-plaintext", "scope"); err == nil {
		t.Fatal("runtime cipher must reject unencrypted persisted secrets")
	}
}

func TestSecretCipherRejectsInvalidKeyring(t *testing.T) {
	if _, err := NewSecretCipher("missing", "key1:"+encodedKey('a')); err == nil {
		t.Fatal("expected missing active key to fail")
	}
	if _, err := NewSecretCipher("key1", "key1:not-base64"); err == nil {
		t.Fatal("expected invalid key material to fail")
	}
}
