package app

import (
	"net/http"
	"net/http/httptest"
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
