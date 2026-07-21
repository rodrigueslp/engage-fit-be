package services

type SecretCipher interface {
	Encrypt(plaintext, associatedData string) (string, error)
	Decrypt(value, associatedData string) (string, error)
	NeedsRotation(value string) bool
}
