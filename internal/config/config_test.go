package config

import "testing"

func TestValidateAllowsDevelopmentDefaults(t *testing.T) {
	t.Parallel()
	cfg := &Config{AppEnv: "development"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("development config should remain permissive: %v", err)
	}
}

func TestValidateRejectsUnsafeProductionConfig(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		AppEnv:        "production",
		AppURL:        "http://localhost:8000",
		JWTSecret:     "change-me-to-a-random-string-please",
		SessionSecret: "change-me-to-a-random-string-please",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("unsafe production config must be rejected")
	}
}

func TestValidateAcceptsStrongProductionConfig(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		AppEnv:        "production",
		AppURL:        "https://investo.example.com",
		DBPass:        "database-password",
		JWTSecret:     "0de8973cc640f73b5dcfd37085caa564",
		SessionSecret: "5738ae09e4992c75cdf5d411421d8498",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid production config rejected: %v", err)
	}
}
