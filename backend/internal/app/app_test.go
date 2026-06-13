package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func NewTestApp(t *testing.T) *App {
	t.Helper()
	return New(Config{AppEnv: "test", PublicWebURL: "http://example.test", SessionSecret: "test-secret"}, nil)
}

func TestHealth(t *testing.T) {
	app := NewTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}

func TestReadyWithoutDatabase(t *testing.T) {
	app := NewTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/api/ready", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestConfigValidateAllowsDevelopmentDefaults(t *testing.T) {
	config := Config{AppEnv: "development", Addr: ":8080"}
	if err := config.Validate(); err != nil {
		t.Fatalf("development config should validate: %v", err)
	}
}

func TestConfigValidateRejectsUnsafeProduction(t *testing.T) {
	config := Config{AppEnv: "production", Addr: ":8080", SessionSecret: "dev-session-secret-change-me"}
	err := config.Validate()
	if err == nil {
		t.Fatal("expected production config validation error")
	}
	message := err.Error()
	for _, want := range []string{"DATABASE_URL", "SESSION_SECRET", "PUBLIC_WEB_URL"} {
		if !strings.Contains(message, want) {
			t.Fatalf("expected %s in validation error, got %q", want, message)
		}
	}
}

func TestConfigValidateAllowsSafeProduction(t *testing.T) {
	config := Config{
		AppEnv:        "production",
		DatabaseURL:   "postgres://app:secret@db:5432/app?sslmode=require",
		SessionSecret: "replace-with-a-long-random-secret",
		PublicWebURL:  "https://subcult.example",
		Addr:          ":8080",
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("production config should validate: %v", err)
	}
}
