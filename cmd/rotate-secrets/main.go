package main

import (
	"context"
	"log"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres"
	"boxengage/backend/internal/adapters/persistence/postgres/repositories"
	"boxengage/backend/internal/adapters/security"
	"boxengage/backend/internal/config"
)

func main() {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL deve ser configurada")
	}
	if cfg.DataEncryptionActiveKeyID == "" || cfg.DataEncryptionKeys == "" {
		log.Fatal("DATA_ENCRYPTION_ACTIVE_KEY_ID e DATA_ENCRYPTION_KEYS devem ser configurados")
	}
	cipher, err := security.NewRotationSecretCipher(cfg.DataEncryptionActiveKeyID, cfg.DataEncryptionKeys)
	if err != nil {
		log.Fatalf("configuracao de criptografia invalida: %v", err)
	}
	db, err := postgres.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("falha ao conectar no PostgreSQL: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	result, err := repositories.RotatePersistedSecrets(ctx, db, cipher)
	if err != nil {
		log.Fatalf("rotacao cancelada sem alteracoes parciais: %v", err)
	}
	log.Printf("rotacao concluida: whatsapp=%d/%d email=%d/%d", result.WhatsappRotated, result.WhatsappScanned, result.EmailRotated, result.EmailScanned)
}
