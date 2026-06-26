package repositories

import (
	"crypto/rand"
	"fmt"

	"boxengage/backend/internal/domain"
)

func ensureID(id *domain.ID) error {
	if *id != "" {
		return nil
	}

	generated, err := newID()
	if err != nil {
		return err
	}
	*id = generated
	return nil
}

func newID() (domain.ID, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}

	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return domain.ID(fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])), nil
}

func stringID(id domain.ID) string {
	return string(id)
}

func domainID(id string) domain.ID {
	return domain.ID(id)
}
