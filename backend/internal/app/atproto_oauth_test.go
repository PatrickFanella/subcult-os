package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestATProtoStartRequiresLocalAuthentication(t *testing.T) {
	settings := testAppOAuthSettings(t)
	application := New(oauthEnabledTestConfig(settings), nil)
	application.atprotoFlow = &fakeATProtoLinkFlow{}
	application.atprotoFlowErr = nil
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/atproto/start", strings.NewReader(`{"identifier":"user.example.com"}`))
	request.Header.Set("Content-Type", "application/json")
	application.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("start status = %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestATProtoCallbackUsesFixedSameOriginLanding(t *testing.T) {
	settings := testAppOAuthSettings(t)
	for _, test := range []struct {
		name       string
		flowErr    error
		wantStatus string
	}{
		{name: "linked", wantStatus: "linked"},
		{name: "cancelled", flowErr: atprotocol.ErrOAuthDenied, wantStatus: "cancelled"},
		{name: "error", flowErr: errors.New("contains secret provider detail"), wantStatus: "error"},
	} {
		t.Run(test.name, func(t *testing.T) {
			application := New(oauthEnabledTestConfig(settings), nil)
			application.atprotoFlow = &fakeATProtoLinkFlow{completeErr: test.flowErr}
			application.atprotoFlowErr = nil
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/atproto/callback?state=state-1&error_description=%3Cscript%3E", nil)
			application.Handler().ServeHTTP(recorder, request)
			if recorder.Code != http.StatusSeeOther {
				t.Fatalf("callback status = %d", recorder.Code)
			}
			want := "https://subcults.subcult.tv/workspace?atproto=" + test.wantStatus
			if got := recorder.Header().Get("Location"); got != want {
				t.Fatalf("callback location = %q, want %q", got, want)
			}
			if strings.Contains(recorder.Header().Get("Location"), "script") || strings.Contains(recorder.Body.String(), "provider detail") {
				t.Fatal("callback exposed untrusted or provider error detail")
			}
		})
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

func oauthEnabledTestConfig(settings atprotocol.OAuthClientSettings) Config {
	return Config{
		AppEnv:               "test",
		PublicWebURL:         "https://subcults.subcult.tv",
		ATProtoOAuthEnabled:  true,
		ATProtoOAuthClientID: settings.ClientID,
		ATProtoOAuthCallback: settings.CallbackURL,
		ATProtoOAuthJWKSURL:  settings.JWKSURL,
		ATProtoOAuthKey:      settings.PrivateKey,
		ATProtoOAuthKeyID:    settings.KeyID,
	}
}

type fakeATProtoLinkFlow struct {
	startErr        error
	startCalls      int
	completeErr     error
	startPersonID   string
	startIdentifier string
}

func (f *fakeATProtoLinkFlow) StartLink(_ context.Context, personID, identifier string) (string, error) {
	f.startCalls++
	f.startPersonID = personID
	f.startIdentifier = identifier
	return "https://auth.example/authorize", f.startErr
}

func (f *fakeATProtoLinkFlow) CompleteLink(context.Context, url.Values) (atprotocol.OAuthLinkResult, error) {
	return atprotocol.OAuthLinkResult{DID: "did:plc:vwzwgnygau7ed7b7wt5ux7y2"}, f.completeErr
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
