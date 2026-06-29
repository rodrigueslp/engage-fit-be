package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv                            string
	HTTPHost                          string
	HTTPPort                          string
	DatabaseURL                       string
	JWTSecret                         string
	WhatsappAllowRealSend             bool
	WhatsappDevRecipientPhone         string
	WhatsappDevAllowedRecipientPhones string
	EmailAllowRealSend                bool
	EmailDevRecipientEmail            string
	AutomationWorkerEnabled           bool
	AutomationWorkerIntervalSeconds   int
}

func Load() Config {
	loadEnvFiles(".env", filepath.Join("..", ".env"))

	return Config{
		AppEnv:                            getEnv("APP_ENV", "development"),
		HTTPHost:                          getEnv("HTTP_HOST", "0.0.0.0"),
		HTTPPort:                          getEnv("HTTP_PORT", "8080"),
		DatabaseURL:                       getEnv("DATABASE_URL", ""),
		JWTSecret:                         getEnv("JWT_SECRET", "change-me"),
		WhatsappAllowRealSend:             getEnv("WHATSAPP_ALLOW_REAL_SEND", "false") == "true",
		WhatsappDevRecipientPhone:         getEnv("WHATSAPP_DEV_RECIPIENT_PHONE", ""),
		WhatsappDevAllowedRecipientPhones: getEnv("WHATSAPP_DEV_ALLOWED_RECIPIENT_PHONES", ""),
		EmailAllowRealSend:                getEnv("EMAIL_ALLOW_REAL_SEND", "false") == "true",
		EmailDevRecipientEmail:            getEnv("EMAIL_DEV_RECIPIENT_EMAIL", ""),
		AutomationWorkerEnabled:           getEnv("AUTOMATION_WORKER_ENABLED", "false") == "true",
		AutomationWorkerIntervalSeconds:   getEnvInt("AUTOMATION_WORKER_INTERVAL_SECONDS", 60),
	}
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
