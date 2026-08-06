package config

import (
	"os"

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
