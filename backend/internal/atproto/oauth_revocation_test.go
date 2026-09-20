package atproto

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
)

type revocationTransport func(*http.Request) (*http.Response, error)

func (f revocationTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRevocationClientBothTokensAndNonce(t *testing.T) {
	key, err := GenerateOAuthClientPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewOAuthClient(OAuthClientSettings{ClientID: "https://app.example/client", CallbackURL: "https://app.example/callback", JWKSURL: "https://app.example/jwks", PrivateKey: key, KeyID: "test"})
	if err != nil {
		t.Fatal(err)
	}
	data := oauth.ClientSessionData{AuthServerURL: "https://provider.example", AuthServerRevocationEndpoint: "https://provider.example/revoke", Scopes: []string{"atproto"}, AccessToken: "test-access", RefreshToken: "test-refresh", DPoPPrivateKeyMultibase: key}
	calls := 0
	transport := revocationTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != data.AuthServerRevocationEndpoint || r.Method != "POST" {
			t.Fatal("wrong revocation destination")
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("client_assertion") == "" || r.Header.Get("DPoP") == "" {
			t.Fatal("missing confidential/DPoP proof")
		}
		if calls <= 2 && (r.Form.Get("token_type_hint") != "access_token" || r.Form.Get("token") != "test-access") {
			t.Fatal("wrong access revocation")
		}
		if calls == 3 && (r.Form.Get("token_type_hint") != "refresh_token" || r.Form.Get("token") != "test-refresh") {
			t.Fatal("wrong refresh revocation")
		}
		status, body := 200, ""
		headers := http.Header{}
		if calls == 1 {
			status = 400
			body = `{"error":"use_dpop_nonce"}`
			headers.Set("DPoP-Nonce", "fresh-test-nonce")
		}
		return &http.Response{StatusCode: status, Header: headers, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	if err := client.revokeSession(t.Context(), data, &http.Client{Transport: transport}); err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatalf("calls=%d want 3", calls)
	}
}

func TestRevocationClientRedactsFailuresAndRejectsInvalidInput(t *testing.T) {
	key, err := GenerateOAuthClientPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewOAuthClient(OAuthClientSettings{ClientID: "https://app.example/client", CallbackURL: "https://app.example/callback", JWKSURL: "https://app.example/jwks", PrivateKey: key, KeyID: "test"})
	if err != nil {
		t.Fatal(err)
	}
	data := oauth.ClientSessionData{AuthServerURL: "https://provider.example", AuthServerRevocationEndpoint: "https://provider.example/revoke", Scopes: []string{"atproto"}, AccessToken: "access-secret", RefreshToken: "refresh-secret", DPoPPrivateKeyMultibase: key}
	broken := &http.Client{Transport: revocationTransport(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("access-secret: malicious provider detail")
	})}
	if err := client.revokeSession(context.Background(), data, broken); !errors.Is(err, ErrRevocationProvider) || strings.Contains(err.Error(), "secret") {
		t.Fatalf("unsafe error: %v", err)
	}
	data.AuthServerRevocationEndpoint = ""
	if err := client.revokeSession(t.Context(), data, broken); !errors.Is(err, ErrRevocationUnsupported) {
		t.Fatalf("missing endpoint: %v", err)
	}
	data.AuthServerRevocationEndpoint = "http://provider.example/revoke"
	if err := client.revokeSession(t.Context(), data, broken); !errors.Is(err, ErrRevocationPayload) {
		t.Fatalf("unsafe endpoint: %v", err)
	}
}
