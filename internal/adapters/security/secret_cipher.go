package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

const encryptedSecretPrefix = "enc:v1:"

var keyIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

type SecretCipher struct {
	activeKeyID string
	keys        map[string][]byte
	allowLegacy bool
}

func NewSecretCipher(activeKeyID, encodedKeyring string) (*SecretCipher, error) {
	return newSecretCipher(activeKeyID, encodedKeyring, false)
}

func NewRotationSecretCipher(activeKeyID, encodedKeyring string) (*SecretCipher, error) {
	return newSecretCipher(activeKeyID, encodedKeyring, true)
}

func newSecretCipher(activeKeyID, encodedKeyring string, allowLegacy bool) (*SecretCipher, error) {
	activeKeyID = strings.TrimSpace(activeKeyID)
	if !keyIDPattern.MatchString(activeKeyID) {
		return nil, errors.New("active encryption key id is invalid")
	}
	keys := make(map[string][]byte)
	for _, item := range strings.Split(encodedKeyring, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		keyID, encoded, ok := strings.Cut(item, ":")
		keyID = strings.TrimSpace(keyID)
		if !ok || !keyIDPattern.MatchString(keyID) {
			return nil, errors.New("encryption keyring contains an invalid key id")
		}
		key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
		if err != nil || len(key) != 32 {
			return nil, fmt.Errorf("encryption key %q must be exactly 32 bytes encoded as base64", keyID)
		}
		keys[keyID] = key
	}
	if _, exists := keys[activeKeyID]; !exists {
		return nil, fmt.Errorf("active encryption key %q is absent from keyring", activeKeyID)
	}
	return &SecretCipher{activeKeyID: activeKeyID, keys: keys, allowLegacy: allowLegacy}, nil
}

func (c *SecretCipher) Encrypt(plaintext, associatedData string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	block, err := aes.NewCipher(c.keys[c.activeKeyID])
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nonce, nonce, []byte(plaintext), []byte(associatedData))
	return encryptedSecretPrefix + c.activeKeyID + ":" + base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (c *SecretCipher) Decrypt(value, associatedData string) (string, error) {
	if value == "" {
		return value, nil
	}
	if !strings.HasPrefix(value, encryptedSecretPrefix) {
		if c.allowLegacy {
			return value, nil
		}
		return "", errors.New("unencrypted secret rejected; run engagefit-rotate-secrets")
	}
	remainder := strings.TrimPrefix(value, encryptedSecretPrefix)
	keyID, encoded, ok := strings.Cut(remainder, ":")
	if !ok {
		return "", errors.New("encrypted secret envelope is malformed")
	}
	key, exists := c.keys[keyID]
	if !exists {
		return "", fmt.Errorf("encryption key %q is not available", keyID)
	}
	sealed, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", errors.New("encrypted secret payload is malformed")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(sealed) < aead.NonceSize() {
		return "", errors.New("encrypted secret payload is too short")
	}
	nonce, ciphertext := sealed[:aead.NonceSize()], sealed[aead.NonceSize():]
	plaintext, err := aead.Open(nil, nonce, ciphertext, []byte(associatedData))
	if err != nil {
		return "", errors.New("encrypted secret authentication failed")
	}
	return string(plaintext), nil
}

func (c *SecretCipher) NeedsRotation(value string) bool {
	return value != "" && !strings.HasPrefix(value, encryptedSecretPrefix+c.activeKeyID+":")
}

type PlaintextSecretCipher struct{}

func NewPlaintextSecretCipher() PlaintextSecretCipher { return PlaintextSecretCipher{} }

func (PlaintextSecretCipher) Encrypt(plaintext, _ string) (string, error) { return plaintext, nil }
func (PlaintextSecretCipher) Decrypt(value, _ string) (string, error)     { return value, nil }
func (PlaintextSecretCipher) NeedsRotation(string) bool                   { return false }
