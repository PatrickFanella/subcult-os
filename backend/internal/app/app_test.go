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

func TestHealthThroughMiddleware(t *testing.T) {
	app := NewTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 through middleware, got %d: %s", rec.Code, rec.Body.String())
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected json content type, got %q", contentType)
	}
}

func TestCORSAllowsConfiguredPublicWebOrigin(t *testing.T) {
	app := NewTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Origin", "http://example.test")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://example.test" {
		t.Fatalf("expected configured origin CORS header, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected credentialed CORS header, got %q", got)
	}
}

func TestCORSAllowsLocalhostInDevelopment(t *testing.T) {
	app := New(Config{AppEnv: "development", PublicWebURL: "http://localhost:5173", SessionSecret: "test-secret"}, nil)
	req := httptest.NewRequest(http.MethodOptions, "/api/public/events", nil)
	req.Header.Set("Origin", "http://localhost:8081")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 preflight, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:8081" {
		t.Fatalf("expected localhost CORS header, got %q", got)
	}
}

func TestCORSRejectsUnconfiguredOrigin(t *testing.T) {
	app := NewTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Origin", "http://evil.example")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no CORS header for unconfigured origin, got %q", got)
	}
}

func TestStripeWebhookRejectsInvalidSignature(t *testing.T) {
	app := New(Config{AppEnv: "test", PublicWebURL: "http://example.test", SessionSecret: "test-secret", StripeWebhookSecret: "whsec_test"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/stripe/webhook", strings.NewReader(`{"id":"evt_test","type":"checkout.session.completed","data":{"object":{"id":"cs_test","object":"checkout.session","metadata":{"ticket_id":"ticket_1"}}}}`))
	req.Header.Set("Stripe-Signature", "t=1,v1=bogus")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid signature to return 400, got %d: %s", rec.Code, rec.Body.String())
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

func TestConfigValidateAllowsPaidTicketingDisabled(t *testing.T) {
	config := Config{
		AppEnv:        "production",
		Addr:          ":8080",
		DatabaseURL:   "postgres://app:secret@db:5432/app?sslmode=require",
		SessionSecret: "replace-with-a-long-random-secret",
		PublicWebURL:  "https://subcult.example",
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("production config without Stripe should validate when paid ticketing is disabled: %v", err)
	}
}

func TestConfigValidateRejectsPartialStripeConfig(t *testing.T) {
	config := Config{
		AppEnv:          "production",
		Addr:            ":8080",
		DatabaseURL:     "postgres://app:secret@db:5432/app?sslmode=require",
		SessionSecret:   "replace-with-a-long-random-secret",
		PublicWebURL:    "https://subcult.example",
		StripeSecretKey: "sk_test_123",
	}
	err := config.Validate()
	if err == nil || !strings.Contains(err.Error(), "STRIPE_WEBHOOK_SECRET") {
		t.Fatalf("expected webhook secret validation error, got %v", err)
	}
}

func TestConfigValidateAllowsCompleteStripeConfig(t *testing.T) {
	config := Config{
		AppEnv:              "production",
		Addr:                ":8080",
		DatabaseURL:         "postgres://app:secret@db:5432/app?sslmode=require",
		SessionSecret:       "replace-with-a-long-random-secret",
		PublicWebURL:        "https://subcult.example",
		StripeSecretKey:     "sk_test_123",
		StripeWebhookSecret: "whsec_123",
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("complete Stripe config should validate: %v", err)
	}
}
