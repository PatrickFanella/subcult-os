package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	atprotocol "git.subcult.tv/PatrickFanella/subcult-os/internal/atproto"
)

func TestATProtoOAuthDocumentsAreUnavailableWhenDisabled(t *testing.T) {
	application := New(Config{AppEnv: "test", PublicWebURL: "https://subcults.subcult.tv"}, nil)
	for _, path := range []string{
		"/api/v1/auth/atproto/client-metadata",
		"/api/v1/auth/atproto/jwks",
	} {
		recorder := httptest.NewRecorder()
		application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("GET %s = %d, want 404", path, recorder.Code)
		}
	}
}

func TestATProtoOAuthDocumentsPreservePublicClientIdentity(t *testing.T) {
	settings := testAppOAuthSettings(t)
	application := New(Config{
		AppEnv:               "test",
		PublicWebURL:         "https://subcults.subcult.tv",
		ATProtoOAuthEnabled:  true,
		ATProtoOAuthClientID: settings.ClientID,
		ATProtoOAuthCallback: settings.CallbackURL,
		ATProtoOAuthJWKSURL:  settings.JWKSURL,
		ATProtoOAuthKey:      settings.PrivateKey,
		ATProtoOAuthKeyID:    settings.KeyID,
	}, nil)

	metadataBody := getOAuthDocument(t, application, "/api/v1/auth/atproto/client-metadata")
	var metadata map[string]any
	if err := json.Unmarshal(metadataBody, &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata["client_id"] != settings.ClientID || metadata["jwks_uri"] != settings.JWKSURL || metadata["scope"] != "atproto" {
		t.Fatalf("unexpected client metadata: %#v", metadata)
	}
	redirects, ok := metadata["redirect_uris"].([]any)
	if !ok || len(redirects) != 1 || redirects[0] != settings.CallbackURL {
		t.Fatalf("unexpected redirect URIs: %#v", metadata["redirect_uris"])
	}

	jwksBody := getOAuthDocument(t, application, "/api/v1/auth/atproto/jwks")
	if bytes.Contains(jwksBody, []byte(settings.PrivateKey)) || strings.Contains(string(jwksBody), `"d"`) {
		t.Fatal("public JWKS exposed confidential key material")
	}
	var jwks struct {
		Keys []map[string]any `json:"keys"`
	}
	if err := json.Unmarshal(jwksBody, &jwks); err != nil {
		t.Fatal(err)
	}
	if len(jwks.Keys) != 1 || jwks.Keys[0]["kid"] != settings.KeyID {
		t.Fatalf("unexpected public JWKS: %#v", jwks.Keys)
	}
}

func TestConfigValidateATProtoOAuth(t *testing.T) {
	settings := testAppOAuthSettings(t)
	valid := Config{
		AppEnv:               "development",
		Addr:                 ":8080",
		PublicWebURL:         "https://subcults.subcult.tv",
		ATProtoOAuthEnabled:  true,
		ATProtoOAuthClientID: settings.ClientID,
		ATProtoOAuthCallback: settings.CallbackURL,
		ATProtoOAuthJWKSURL:  settings.JWKSURL,
		ATProtoOAuthKey:      settings.PrivateKey,
		ATProtoOAuthKeyID:    settings.KeyID,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid AT OAuth configuration failed: %v", err)
	}

	invalid := valid
	invalid.ATProtoOAuthCallback = "https://other.example/callback"
	if err := invalid.Validate(); err == nil || !strings.Contains(err.Error(), "one HTTPS origin") {
		t.Fatalf("expected cross-origin validation error, got %v", err)
	}

	invalid = valid
	invalid.PublicWebURL = "https://other.example"
	if err := invalid.Validate(); err == nil || !strings.Contains(err.Error(), "share one origin") {
		t.Fatalf("expected public-origin validation error, got %v", err)
	}

	invalid = valid
	invalid.PublicWebURL = "http://subcults.subcult.tv"
	if err := invalid.Validate(); err == nil || !strings.Contains(err.Error(), "HTTPS origin") {
		t.Fatalf("expected HTTPS public-origin validation error, got %v", err)
	}
}

func TestLoadConfigRejectsInvalidOAuthEnabledBoolean(t *testing.T) {
	t.Setenv("ATPROTO_OAUTH_ENABLED", "sometimes")
	config := LoadConfig()
	if err := config.Validate(); err == nil || !strings.Contains(err.Error(), "ATPROTO_OAUTH_ENABLED must be a boolean") {
		t.Fatalf("expected invalid boolean error, got %v", err)
	}
}

func testAppOAuthSettings(t *testing.T) atprotocol.OAuthClientSettings {
	t.Helper()
	privateKey, err := atprotocol.GenerateOAuthClientPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	return atprotocol.OAuthClientSettings{
		ClientID:    "https://subcults.subcult.tv/api/v1/auth/atproto/client-metadata",
		CallbackURL: "https://subcults.subcult.tv/api/v1/auth/atproto/callback",
		JWKSURL:     "https://subcults.subcult.tv/api/v1/auth/atproto/jwks",
		PrivateKey:  privateKey,
		KeyID:       "subcults-1",
	}
}

func getOAuthDocument(t *testing.T, application *App, path string) []byte {
	t.Helper()
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET %s = %d: %s", path, recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("GET %s content type = %q", path, got)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=300" {
		t.Fatalf("GET %s cache control = %q", path, got)
	}
	return recorder.Body.Bytes()
}
