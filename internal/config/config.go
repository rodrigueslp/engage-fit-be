package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv                            string
	HTTPHost                          string
	HTTPPort                          string
	HTTPMaxBodyBytes                  int
	ImportMaxUploadBytes              int
	LoginRateLimitRequests            int
	LoginRateLimitWindowSeconds       int
	SetupRateLimitRequests            int
	SetupRateLimitWindowSeconds       int
	TrustedProxies                    string
	HTTPReadHeaderTimeoutSeconds      int
	HTTPReadTimeoutSeconds            int
	HTTPWriteTimeoutSeconds           int
	HTTPIdleTimeoutSeconds            int
	HTTPShutdownTimeoutSeconds        int
	DBMaxOpenConnections              int
	DBMaxIdleConnections              int
	DBConnectionMaxLifetimeSeconds    int
	DBConnectionMaxIdleTimeSeconds    int
	DatabaseURL                       string
	JWTSecret                         string
	AuthCookieName                    string
	AuthCookieSecure                  bool
	AuthCookieSameSite                string
	AuthSessionMaxAgeSeconds          int
	CORSAllowedOrigins                string
	PlatformAdminName                 string
	PlatformAdminEmail                string
	PlatformAdminPassword             string
	OwnerSetupEnabled                 bool
	OwnerSetupToken                   string
	WhatsappAllowRealSend             bool
	WhatsappDevRecipientPhone         string
	WhatsappDevAllowedRecipientPhones string
	WhatsappPlatformEnabled           bool
	WhatsappPlatformBaseURL           string
	WhatsappPlatformSender            string
	WhatsappPlatformAccountSID        string
	WhatsappPlatformAuthToken         string
	WhatsappPlatformAlmostThereSID    string
	WhatsappPlatformGoalReachedSID    string
	WhatsappPlatformWeMissYouSID      string
	EmailAllowRealSend                bool
	EmailDevRecipientEmail            string
	AutomationWorkerEnabled           bool
	AutomationWorkerIntervalSeconds   int
	AutomationStaleRunMinutes         int
	AutomationCatchupWindowMinutes    int
	OpenAIAPIKey                      string
	OpenAIModel                       string
	OpenAITimeoutSeconds              int
	OTelEnabled                       bool
	OTelServiceName                   string
	OTelServiceVersion                string
	OTelTraceSampleRatio              float64
	PrometheusEnabled                 bool
	PrometheusBearerToken             string
	DataEncryptionActiveKeyID         string
	DataEncryptionKeys                string
	AsaasBaseURL                      string
	AsaasAPIKey                       string
	AsaasWebhookToken                 string
	AsaasTimeoutSeconds               int
	FeatureWhatsappEnabled            bool
	FeatureEmailEnabled               bool
	FeatureAutomationEnabled          bool
	FeatureWorkoutsEnabled            bool
	FeatureLLMEnabled                 bool
	FeatureBillingEnabled             bool
	BuildVersion                      string
	BuildCommit                       string
	BuildTime                         string
}

func Load() Config {
	loadEnvFiles(".env", filepath.Join("..", ".env"))
	appEnv := getEnv("APP_ENV", "development")

	return Config{
		AppEnv:                            appEnv,
		HTTPHost:                          getEnv("HTTP_HOST", "0.0.0.0"),
		HTTPPort:                          getEnv("HTTP_PORT", getEnv("PORT", "8080")),
		HTTPMaxBodyBytes:                  getEnvInt("HTTP_MAX_BODY_BYTES", 1_048_576),
		ImportMaxUploadBytes:              getEnvInt("IMPORT_MAX_UPLOAD_BYTES", 10_485_760),
		LoginRateLimitRequests:            getEnvInt("LOGIN_RATE_LIMIT_REQUESTS", 10),
		LoginRateLimitWindowSeconds:       getEnvInt("LOGIN_RATE_LIMIT_WINDOW_SECONDS", 60),
		SetupRateLimitRequests:            getEnvInt("SETUP_RATE_LIMIT_REQUESTS", 5),
		SetupRateLimitWindowSeconds:       getEnvInt("SETUP_RATE_LIMIT_WINDOW_SECONDS", 3600),
		TrustedProxies:                    getEnv("TRUSTED_PROXIES", ""),
		HTTPReadHeaderTimeoutSeconds:      getEnvInt("HTTP_READ_HEADER_TIMEOUT_SECONDS", 5),
		HTTPReadTimeoutSeconds:            getEnvInt("HTTP_READ_TIMEOUT_SECONDS", 30),
		HTTPWriteTimeoutSeconds:           getEnvInt("HTTP_WRITE_TIMEOUT_SECONDS", 300),
		HTTPIdleTimeoutSeconds:            getEnvInt("HTTP_IDLE_TIMEOUT_SECONDS", 120),
		HTTPShutdownTimeoutSeconds:        getEnvInt("HTTP_SHUTDOWN_TIMEOUT_SECONDS", 20),
		DBMaxOpenConnections:              getEnvInt("DB_MAX_OPEN_CONNECTIONS", 10),
		DBMaxIdleConnections:              getEnvInt("DB_MAX_IDLE_CONNECTIONS", 5),
		DBConnectionMaxLifetimeSeconds:    getEnvInt("DB_CONNECTION_MAX_LIFETIME_SECONDS", 1800),
		DBConnectionMaxIdleTimeSeconds:    getEnvInt("DB_CONNECTION_MAX_IDLE_TIME_SECONDS", 300),
		DatabaseURL:                       getEnv("DATABASE_URL", ""),
		JWTSecret:                         getEnv("JWT_SECRET", "change-me"),
		AuthCookieName:                    getEnv("AUTH_COOKIE_NAME", "engagefit_session"),
		AuthCookieSecure:                  getEnvBool("AUTH_COOKIE_SECURE", appEnv == "production"),
		AuthCookieSameSite:                getEnv("AUTH_COOKIE_SAME_SITE", "lax"),
		AuthSessionMaxAgeSeconds:          getEnvInt("AUTH_SESSION_MAX_AGE_SECONDS", 86400),
		CORSAllowedOrigins:                getEnv("CORS_ALLOWED_ORIGINS", ""),
		PlatformAdminName:                 getEnv("PLATFORM_ADMIN_NAME", "Administrador EngageFit"),
		PlatformAdminEmail:                getEnv("PLATFORM_ADMIN_EMAIL", ""),
		PlatformAdminPassword:             getEnv("PLATFORM_ADMIN_PASSWORD", ""),
		OwnerSetupEnabled:                 getEnvBool("OWNER_SETUP_ENABLED", appEnv != "production"),
		OwnerSetupToken:                   getEnv("OWNER_SETUP_TOKEN", ""),
		WhatsappAllowRealSend:             getEnv("WHATSAPP_ALLOW_REAL_SEND", "false") == "true",
		WhatsappDevRecipientPhone:         getEnv("WHATSAPP_DEV_RECIPIENT_PHONE", ""),
		WhatsappDevAllowedRecipientPhones: getEnv("WHATSAPP_DEV_ALLOWED_RECIPIENT_PHONES", ""),
		WhatsappPlatformEnabled:           getEnv("WHATSAPP_PLATFORM_ENABLED", "false") == "true",
		WhatsappPlatformBaseURL:           getEnv("WHATSAPP_PLATFORM_BASE_URL", ""),
		WhatsappPlatformSender:            getEnv("WHATSAPP_PLATFORM_TWILIO_SENDER", ""),
		WhatsappPlatformAccountSID:        getEnv("WHATSAPP_PLATFORM_TWILIO_ACCOUNT_SID", ""),
		WhatsappPlatformAuthToken:         getEnv("WHATSAPP_PLATFORM_TWILIO_AUTH_TOKEN", ""),
		WhatsappPlatformAlmostThereSID:    getEnv("WHATSAPP_PLATFORM_TWILIO_CONTENT_SID_ALMOST_THERE", ""),
		WhatsappPlatformGoalReachedSID:    getEnv("WHATSAPP_PLATFORM_TWILIO_CONTENT_SID_GOAL_REACHED", ""),
		WhatsappPlatformWeMissYouSID:      getEnv("WHATSAPP_PLATFORM_TWILIO_CONTENT_SID_WE_MISS_YOU", ""),
		EmailAllowRealSend:                getEnv("EMAIL_ALLOW_REAL_SEND", "false") == "true",
		EmailDevRecipientEmail:            getEnv("EMAIL_DEV_RECIPIENT_EMAIL", ""),
		AutomationWorkerEnabled:           getEnv("AUTOMATION_WORKER_ENABLED", "false") == "true",
		AutomationWorkerIntervalSeconds:   getEnvInt("AUTOMATION_WORKER_INTERVAL_SECONDS", 60),
		AutomationStaleRunMinutes:         getEnvInt("AUTOMATION_STALE_RUN_MINUTES", 120),
		AutomationCatchupWindowMinutes:    getEnvInt("AUTOMATION_CATCHUP_WINDOW_MINUTES", 15),
		OpenAIAPIKey:                      getEnv("OPENAI_API_KEY", ""),
		OpenAIModel:                       getEnv("OPENAI_MODEL", "gpt-4.1-mini"),
		OpenAITimeoutSeconds:              getEnvInt("OPENAI_TIMEOUT_SECONDS", 30),
		OTelEnabled:                       getEnvBool("OTEL_ENABLED", false),
		OTelServiceName:                   getEnv("OTEL_SERVICE_NAME", "engagefit-api"),
		OTelServiceVersion:                getEnv("OTEL_SERVICE_VERSION", "dev"),
		OTelTraceSampleRatio:              getEnvFloat("OTEL_TRACE_SAMPLE_RATIO", 0.1),
		PrometheusEnabled:                 getEnvBool("PROMETHEUS_ENABLED", appEnv != "production"),
		PrometheusBearerToken:             getEnv("PROMETHEUS_BEARER_TOKEN", ""),
		DataEncryptionActiveKeyID:         getEnv("DATA_ENCRYPTION_ACTIVE_KEY_ID", ""),
		DataEncryptionKeys:                getEnv("DATA_ENCRYPTION_KEYS", ""),
		AsaasBaseURL:                      getEnv("ASAAS_BASE_URL", "https://api-sandbox.asaas.com/v3"),
		AsaasAPIKey:                       getEnv("ASAAS_API_KEY", ""),
		AsaasWebhookToken:                 getEnv("ASAAS_WEBHOOK_TOKEN", ""),
		AsaasTimeoutSeconds:               getEnvInt("ASAAS_TIMEOUT_SECONDS", 15),
		FeatureWhatsappEnabled:            getEnvBool("FEATURE_WHATSAPP_ENABLED", appEnv != "production"),
		FeatureEmailEnabled:               getEnvBool("FEATURE_EMAIL_ENABLED", appEnv != "production"),
		FeatureAutomationEnabled:          getEnvBool("FEATURE_AUTOMATION_ENABLED", appEnv != "production"),
		FeatureWorkoutsEnabled:            getEnvBool("FEATURE_WORKOUTS_ENABLED", appEnv != "production"),
		FeatureLLMEnabled:                 getEnvBool("FEATURE_LLM_ENABLED", appEnv != "production"),
		FeatureBillingEnabled:             getEnvBool("FEATURE_BILLING_ENABLED", false),
		BuildVersion:                      getEnv("BUILD_VERSION", ""),
		BuildCommit:                       getEnv("BUILD_COMMIT", ""),
		BuildTime:                         getEnv("BUILD_TIME", ""),
	}
}

func (c Config) TrustedProxyList() []string {
	parts := strings.Split(c.TrustedProxies, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func (c Config) CORSAllowedOriginList() []string {
	parts := strings.Split(c.CORSAllowedOrigins, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return errors.New("DATABASE_URL deve ser configurada")
	}
	if c.HTTPMaxBodyBytes <= 0 || c.ImportMaxUploadBytes <= 0 || c.LoginRateLimitRequests <= 0 || c.LoginRateLimitWindowSeconds <= 0 || c.SetupRateLimitRequests <= 0 || c.SetupRateLimitWindowSeconds <= 0 {
		return errors.New("limites HTTP e de autenticacao devem ser maiores que zero")
	}
	if c.HTTPReadHeaderTimeoutSeconds <= 0 || c.HTTPReadTimeoutSeconds <= 0 || c.HTTPWriteTimeoutSeconds <= 0 || c.HTTPIdleTimeoutSeconds <= 0 || c.HTTPShutdownTimeoutSeconds <= 0 {
		return errors.New("timeouts HTTP devem ser maiores que zero")
	}
	sameSite := strings.ToLower(strings.TrimSpace(c.AuthCookieSameSite))
	if c.AuthCookieName == "" || c.AuthSessionMaxAgeSeconds <= 0 || (sameSite != "lax" && sameSite != "strict" && sameSite != "none") {
		return errors.New("configuracao da sessao e invalida")
	}
	if sameSite == "none" && !c.AuthCookieSecure {
		return errors.New("AUTH_COOKIE_SECURE deve ser true quando SameSite=None")
	}
	if c.DBMaxOpenConnections <= 0 || c.DBMaxIdleConnections < 0 || c.DBMaxIdleConnections > c.DBMaxOpenConnections || c.DBConnectionMaxLifetimeSeconds <= 0 || c.DBConnectionMaxIdleTimeSeconds <= 0 {
		return errors.New("configuracao do pool PostgreSQL e invalida")
	}
	if c.AutomationStaleRunMinutes <= 0 || c.AutomationCatchupWindowMinutes <= 0 {
		return errors.New("timeouts e janela de recuperacao da automacao devem ser maiores que zero")
	}
	if c.FeatureBillingEnabled && c.AsaasTimeoutSeconds <= 0 {
		return errors.New("ASAAS_TIMEOUT_SECONDS deve ser maior que zero")
	}
	if c.FeatureBillingEnabled {
		if strings.TrimSpace(c.AsaasAPIKey) == "" || len(c.AsaasWebhookToken) < 32 {
			return errors.New("FEATURE_BILLING_ENABLED exige ASAAS_API_KEY e ASAAS_WEBHOOK_TOKEN com ao menos 32 caracteres")
		}
		if !strings.HasPrefix(c.AsaasBaseURL, "https://") {
			return errors.New("ASAAS_BASE_URL deve usar HTTPS")
		}
	}
	if (c.OTelEnabled && strings.TrimSpace(c.OTelServiceName) == "") || c.OTelTraceSampleRatio < 0 || c.OTelTraceSampleRatio > 1 {
		return errors.New("configuracao OpenTelemetry e invalida")
	}
	if c.AutomationWorkerEnabled && !c.FeatureAutomationEnabled {
		return errors.New("AUTOMATION_WORKER_ENABLED exige FEATURE_AUTOMATION_ENABLED")
	}
	if c.FeatureLLMEnabled && !c.FeatureWorkoutsEnabled {
		return errors.New("FEATURE_LLM_ENABLED exige FEATURE_WORKOUTS_ENABLED")
	}
	if c.WhatsappAllowRealSend && !c.FeatureWhatsappEnabled {
		return errors.New("WHATSAPP_ALLOW_REAL_SEND exige FEATURE_WHATSAPP_ENABLED")
	}
	if c.EmailAllowRealSend && !c.FeatureEmailEnabled {
		return errors.New("EMAIL_ALLOW_REAL_SEND exige FEATURE_EMAIL_ENABLED")
	}
	if c.AppEnv != "production" {
		return nil
	}
	if len(c.JWTSecret) < 32 || c.JWTSecret == "change-me" {
		return errors.New("JWT_SECRET deve ter ao menos 32 caracteres em production")
	}
	if !c.AuthCookieSecure {
		return errors.New("AUTH_COOKIE_SECURE deve ser true em production")
	}
	if c.PlatformAdminEmail == "" || c.PlatformAdminPassword == "" {
		return errors.New("PLATFORM_ADMIN_EMAIL e PLATFORM_ADMIN_PASSWORD sao obrigatorios em production")
	}
	if c.OwnerSetupEnabled && len(c.OwnerSetupToken) < 32 {
		return errors.New("OWNER_SETUP_TOKEN deve ter ao menos 32 caracteres quando o setup estiver habilitado em production")
	}
	if c.PrometheusEnabled && len(c.PrometheusBearerToken) < 32 {
		return errors.New("PROMETHEUS_BEARER_TOKEN deve ter ao menos 32 caracteres quando Prometheus estiver habilitado em production")
	}
	if c.DataEncryptionActiveKeyID == "" || c.DataEncryptionKeys == "" {
		return errors.New("DATA_ENCRYPTION_ACTIVE_KEY_ID e DATA_ENCRYPTION_KEYS sao obrigatorios em production")
	}
	return nil
}

func loadEnvFiles(paths ...string) {
	for _, path := range paths {
		loadEnvFile(path)
	}
}

func loadEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		_ = os.Setenv(key, value)
	}
}

func (c Config) HTTPAddress() string {
	return c.HTTPHost + ":" + c.HTTPPort
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvFloat(key string, fallback float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		panic(fmt.Sprintf("%s deve ser true ou false", key))
	}
	return parsed
}
