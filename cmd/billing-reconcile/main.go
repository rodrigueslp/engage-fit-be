package main

import (
	"context"
	"log"
	"time"

	billingadapter "boxengage/backend/internal/adapters/billing"
	"boxengage/backend/internal/adapters/persistence/postgres"
	pgrepo "boxengage/backend/internal/adapters/persistence/postgres/repositories"
	billingapp "boxengage/backend/internal/app/billing"
	"boxengage/backend/internal/config"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}
	if !cfg.FeatureBillingEnabled {
		log.Fatal("FEATURE_BILLING_ENABLED must be true")
	}
	db, err := postgres.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to configure database: %v", err)
	}
	defer sqlDB.Close()

	repository := pgrepo.NewBillingGormRepository(db)
	gateway := billingadapter.NewAsaasClient(cfg.AsaasBaseURL, cfg.AsaasAPIKey, time.Duration(cfg.AsaasTimeoutSeconds)*time.Second)
	service := billingapp.NewService(repository, gateway, nil, true, cfg.AsaasWebhookToken)
	if err := service.Reconcile(context.Background()); err != nil {
		log.Fatalf("billing reconciliation failed: %v", err)
	}
	log.Println("billing reconciliation completed")
}
