package app

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	AppEnv        string
	DatabaseURL   string
	SessionSecret string
	PublicWebURL  string
	Addr          string
}

func LoadConfig() Config {
	return Config{
		AppEnv:        env("APP_ENV", "development"),
		DatabaseURL:   env("DATABASE_URL", ""),
		SessionSecret: env("SESSION_SECRET", "dev-session-secret-change-me"),
		PublicWebURL:  env("PUBLIC_WEB_URL", "http://localhost:5173"),
		Addr:          env("API_ADDR", ":8080"),
	}
}

func (c Config) Validate() error {
	var problems []string
	if strings.TrimSpace(c.Addr) == "" {
		problems = append(problems, "API_ADDR is required")
	}
	if strings.EqualFold(strings.TrimSpace(c.AppEnv), "production") {
		if strings.TrimSpace(c.DatabaseURL) == "" {
			problems = append(problems, "DATABASE_URL is required in production")
		}
		if strings.TrimSpace(c.PublicWebURL) == "" {
			problems = append(problems, "PUBLIC_WEB_URL is required in production")
		}
		secret := strings.TrimSpace(c.SessionSecret)
		if secret == "" || secret == "dev-session-secret-change-me" || len(secret) < 24 {
			problems = append(problems, "SESSION_SECRET must be a non-default value with at least 24 characters in production")
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("invalid config: %s", strings.Join(problems, "; "))
	}
	return nil
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
