package oauth_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
)

type rejectingStateStore struct {
	oauth.ClientAuthStore
	err   error
	calls int
}

func (s *rejectingStateStore) SaveAuthRequestInfo(context.Context, oauth.AuthRequestData) error {
	s.calls++
	return s.err
}

type fixtureTransport func(*http.Request) (*http.Response, error)

func (f fixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Synthetic HTTPS documents and an in-memory transport make this independent
// of DNS, databases, accounts, provider availability and any Subcult service.
func TestStartAuthFlowRejectsUnstoredState(t *testing.T) {
	wantErr := errors.New("fixture: state database unavailable")
	store := &rejectingStateStore{ClientAuthStore: oauth.NewMemStore(), err: wantErr}
	config := oauth.NewPublicConfig("https://client.example/metadata", "https://client.example/callback", []string{"atproto"})
	client := oauth.NewClientApp(&config, store)
	requests := 0
	transport := fixtureTransport(func(r *http.Request) (*http.Response, error) {
		requests++
		var body string
		switch r.Method + " " + r.URL.String() {
		case "GET https://auth.example/.well-known/oauth-authorization-server":
			body = `{"issuer":"https://auth.example","authorization_endpoint":"https://auth.example/authorize","token_endpoint":"https://auth.example/token","response_types_supported":["code"],"grant_types_supported":["authorization_code","refresh_token"],"code_challenge_methods_supported":["S256"],"token_endpoint_auth_methods_supported":["none","private_key_jwt"],"token_endpoint_auth_signing_alg_values_supported":["ES256"],"scopes_supported":["atproto"],"authorization_response_iss_parameter_supported":true,"require_pushed_authorization_requests":true,"pushed_authorization_request_endpoint":"https://auth.example/par","dpop_signing_alg_values_supported":["ES256"],"client_id_metadata_document_supported":true}`
		case "POST https://auth.example/par":
			body = `{"request_uri":"urn:ietf:params:oauth:request_uri:fixture","expires_in":90}`
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	client.Client = &http.Client{Transport: transport}
	client.Resolver.Client = client.Client
	redirect, err := client.StartAuthFlow(t.Context(), "https://auth.example")
	if requests != 2 || store.calls != 1 {
		t.Fatalf("fixture did not reach persistence: requests=%d saves=%d", requests, store.calls)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("StartAuthFlow error=%v; want wrapped persistence failure", err)
	}
	if redirect != "" {
		t.Fatal("must not redirect to authorization when callback state was not stored")
	}
}
