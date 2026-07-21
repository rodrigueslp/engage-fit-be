package repositories

import (
	"context"
	"errors"
	"fmt"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/services"
	"gorm.io/gorm"
)

type SecretRotationResult struct {
	WhatsappScanned int
	WhatsappRotated int
	EmailScanned    int
	EmailRotated    int
}

type secretRotationRow struct {
	ID    string
	BoxID string
	Value string
}

func RotatePersistedSecrets(ctx context.Context, db *gorm.DB, cipher services.SecretCipher) (SecretRotationResult, error) {
	var result SecretRotationResult
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		whatsappRows, err := loadSecretRows(tx, "whatsapp_settings", "api_key_encrypted")
		if err != nil {
			return err
		}
		result.WhatsappScanned = len(whatsappRows)
		for _, row := range whatsappRows {
			if row.Value == "" || !cipher.NeedsRotation(row.Value) {
				continue
			}
			plaintext, err := cipher.Decrypt(row.Value, whatsappSecretAAD(domain.ID(row.BoxID)))
			if err != nil {
				return fmt.Errorf("decrypt whatsapp credential for row %s: %w", row.ID, err)
			}
			encrypted, err := cipher.Encrypt(plaintext, whatsappSecretAAD(domain.ID(row.BoxID)))
			if err != nil {
				return fmt.Errorf("encrypt whatsapp credential for row %s: %w", row.ID, err)
			}
			if err := tx.Table("whatsapp_settings").Where("id = ?", row.ID).Update("api_key_encrypted", encrypted).Error; err != nil {
				return err
			}
			result.WhatsappRotated++
		}

		emailRows, err := loadSecretRows(tx, "email_settings", "password_encrypted")
		if err != nil {
			return err
		}
		result.EmailScanned = len(emailRows)
		for _, row := range emailRows {
			if row.Value == "" || !cipher.NeedsRotation(row.Value) {
				continue
			}
			plaintext, err := cipher.Decrypt(row.Value, emailSecretAAD(domain.ID(row.BoxID)))
			if err != nil {
				return fmt.Errorf("decrypt email credential for row %s: %w", row.ID, err)
			}
			encrypted, err := cipher.Encrypt(plaintext, emailSecretAAD(domain.ID(row.BoxID)))
			if err != nil {
				return fmt.Errorf("encrypt email credential for row %s: %w", row.ID, err)
			}
			if err := tx.Table("email_settings").Where("id = ?", row.ID).Update("password_encrypted", encrypted).Error; err != nil {
				return err
			}
			result.EmailRotated++
		}
		return nil
	})
	return result, err
}

func loadSecretRows(tx *gorm.DB, table, column string) ([]secretRotationRow, error) {
	allowed := map[string]string{
		"whatsapp_settings.api_key_encrypted": "SELECT id, box_id, api_key_encrypted AS value FROM whatsapp_settings FOR UPDATE",
		"email_settings.password_encrypted":   "SELECT id, box_id, password_encrypted AS value FROM email_settings FOR UPDATE",
	}
	query, ok := allowed[table+"."+column]
	if !ok {
		return nil, errors.New("unsupported secret rotation target")
	}
	var rows []secretRotationRow
	if err := tx.Raw(query).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
