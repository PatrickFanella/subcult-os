package app

import "os"

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

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
