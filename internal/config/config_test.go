package config

import "testing"

func TestProductionConfigValidation(t *testing.T) {
	valid := Config{
		AppEnv:                         "production",
		DatabaseURL:                    "postgres://database",
		JWTSecret:                      "01234567890123456789012345678901",
		PlatformAdminEmail:             "admin@example.com",
		PlatformAdminPassword:          "strong-password",
		HTTPMaxBodyBytes:               1,
		ImportMaxUploadBytes:           1,
		LoginRateLimitRequests:         1,
		LoginRateLimitWindowSeconds:    1,
		SetupRateLimitRequests:         1,
		SetupRateLimitWindowSeconds:    1,
		HTTPReadHeaderTimeoutSeconds:   1,
		HTTPReadTimeoutSeconds:         1,
		HTTPWriteTimeoutSeconds:        1,
		HTTPIdleTimeoutSeconds:         1,
		HTTPShutdownTimeoutSeconds:     1,
		DBMaxOpenConnections:           2,
		DBMaxIdleConnections:           1,
		DBConnectionMaxLifetimeSeconds: 1,
		DBConnectionMaxIdleTimeSeconds: 1,
		DataEncryptionActiveKeyID:      "primary",
		DataEncryptionKeys:             "primary:configured-outside-database",
		AutomationStaleRunMinutes:      120,
		AutomationCatchupWindowMinutes: 15,
		AuthCookieName:                 "engagefit_session",
		AuthCookieSecure:               true,
		AuthCookieSameSite:             "lax",
		AuthSessionMaxAgeSeconds:       86400,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected production config to be valid: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "weak jwt", mutate: func(cfg *Config) { cfg.JWTSecret = "change-me" }},
		{name: "missing admin", mutate: func(cfg *Config) { cfg.PlatformAdminEmail = "" }},
		{name: "enabled setup without token", mutate: func(cfg *Config) { cfg.OwnerSetupEnabled = true }},
		{name: "enabled prometheus without token", mutate: func(cfg *Config) { cfg.PrometheusEnabled = true }},
		{name: "enabled otel without service name", mutate: func(cfg *Config) { cfg.OTelEnabled = true }},
		{name: "missing encryption keyring", mutate: func(cfg *Config) { cfg.DataEncryptionKeys = "" }},
		{name: "insecure production cookie", mutate: func(cfg *Config) { cfg.AuthCookieSecure = false }},
		{name: "insecure Twilio webhook", mutate: func(cfg *Config) { cfg.TwilioInboundWebhookURL = "http://example.com/webhook" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := valid
			test.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("expected invalid production config")
			}
		})
	}
}

func TestBillingReconciliationValidationRequiresOnlyDatabaseAndAsaas(t *testing.T) {
	valid := Config{
		AppEnv:                         "production",
		DatabaseURL:                    "postgres://database",
		DBMaxOpenConnections:           2,
		DBMaxIdleConnections:           1,
		DBConnectionMaxLifetimeSeconds: 1,
		DBConnectionMaxIdleTimeSeconds: 1,
		FeatureBillingEnabled:          true,
		AsaasBaseURL:                   "https://api.asaas.com/v3",
		AsaasAPIKey:                    "production-api-key",
		AsaasWebhookToken:              "01234567890123456789012345678901",
		AsaasTimeoutSeconds:            15,
	}

	if err := valid.ValidateBillingReconciliation(); err != nil {
		t.Fatalf("expected reconciliation config to be valid: %v", err)
	}
	if err := valid.Validate(); err == nil {
		t.Fatal("full API validation must still reject missing HTTP, session and production secrets")
	}

	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "missing database", mutate: func(cfg *Config) { cfg.DatabaseURL = "" }},
		{name: "invalid database pool", mutate: func(cfg *Config) { cfg.DBMaxOpenConnections = 0 }},
		{name: "billing disabled", mutate: func(cfg *Config) { cfg.FeatureBillingEnabled = false }},
		{name: "missing API key", mutate: func(cfg *Config) { cfg.AsaasAPIKey = "" }},
		{name: "short webhook token", mutate: func(cfg *Config) { cfg.AsaasWebhookToken = "short" }},
		{name: "insecure Asaas URL", mutate: func(cfg *Config) { cfg.AsaasBaseURL = "http://api.asaas.com/v3" }},
		{name: "invalid timeout", mutate: func(cfg *Config) { cfg.AsaasTimeoutSeconds = 0 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := valid
			test.mutate(&cfg)
			if err := cfg.ValidateBillingReconciliation(); err == nil {
				t.Fatal("expected invalid reconciliation config")
			}
		})
	}
}
