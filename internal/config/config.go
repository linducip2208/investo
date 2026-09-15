package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	DBHost           string
	DBPort           string
	DBUser           string
	DBPass           string
	DBName           string
	AppName          string
	AppURL           string
	AppEnv           string
	JWTSecret        string
	SessionSecret    string
	TelegramBotToken string
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv("INVESTO_" + key); ok && val != "" {
		return val
	}
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:             getEnv("PORT", "8000"),
		DBHost:           getEnv("DB_HOST", "127.0.0.1"),
		DBPort:           getEnv("DB_PORT", "3306"),
		DBUser:           getEnv("DB_USER", "root"),
		DBPass:           getEnv("DB_PASS", ""),
		DBName:           getEnv("DB_NAME", "investo"),
		AppName:          getEnv("APP_NAME", "Investo"),
		AppURL:           getEnv("APP_URL", "http://localhost:8000"),
		AppEnv:           getEnv("APP_ENV", "development"),
		JWTSecret:        getEnv("JWT_SECRET", "investo-jwt-secret-key"),
		SessionSecret:    getEnv("SESSION_SECRET", "investo-session-secret-key"),
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
	}
}

// Validate checks configuration invariants that must hold before production
// traffic is served. Development keeps its convenient local defaults.
func (c *Config) Validate() error {
	if !strings.EqualFold(strings.TrimSpace(c.AppEnv), "production") {
		return nil
	}

	var problems []string
	if strings.TrimSpace(c.DBPass) == "" {
		problems = append(problems, "DB_PASS must be set")
	}
	if !isStrongSecret(c.JWTSecret) {
		problems = append(problems, "JWT_SECRET must be at least 32 characters and must not be a placeholder")
	}
	if !isStrongSecret(c.SessionSecret) {
		problems = append(problems, "SESSION_SECRET must be at least 32 characters and must not be a placeholder")
	}
	if c.JWTSecret == c.SessionSecret {
		problems = append(problems, "JWT_SECRET and SESSION_SECRET must be different")
	}

	appURL, err := url.ParseRequestURI(strings.TrimSpace(c.AppURL))
	if err != nil || appURL.Scheme != "https" || appURL.Host == "" {
		problems = append(problems, "APP_URL must be an absolute HTTPS URL")
	}

	if len(problems) > 0 {
		return fmt.Errorf("invalid production configuration: %s", strings.Join(problems, "; "))
	}
	return nil
}

func isStrongSecret(secret string) bool {
	secret = strings.TrimSpace(secret)
	if len(secret) < 32 {
		return false
	}
	lower := strings.ToLower(secret)
	for _, marker := range []string{"change-me", "changeme", "placeholder", "example", "investo-jwt-secret", "investo-session-secret"} {
		if strings.Contains(lower, marker) {
			return false
		}
	}
	return true
}
