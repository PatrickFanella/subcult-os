package atproto

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestOAuthFlowNormalizesIdentifierAndRequiresPersistedRequest(t *testing.T) {
	store := &personCheckingStore{ClientAuthStore: oauth.NewMemStore(), wantPersonID: "person-1"}
	var seenIdentifier string
	flow := &OAuthFlow{
		clientID: "https://subcults.subcult.tv/api/v1/auth/atproto/client-metadata",
		store:    store,
		newRunner: func(flowStore oauth.ClientAuthStore) indigoOAuthRunner {
			return &fakeIndigoRunner{store: flowStore, seenIdentifier: &seenIdentifier}
		},
	}
	redirect, err := flow.StartLink(t.Context(), "person-1", "User.Example.COM")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(redirect)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Host != "auth.example.com" || parsed.Query().Get("client_id") != flow.clientID || parsed.Query().Get("request_uri") != "urn:example:request" {
		t.Fatalf("unexpected authorization redirect: %s", redirect)
	}
	if seenIdentifier != "user.example.com" {
		t.Fatalf("start identifier = %q", seenIdentifier)
	}
}

func TestOAuthFlowSurfacesIgnoredUpstreamPersistenceError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	store := &failingSaveStore{ClientAuthStore: oauth.NewMemStore(), err: wantErr}
	flow := &OAuthFlow{
		clientID: "https://subcults.subcult.tv/api/v1/auth/atproto/client-metadata",
		store:    store,
		newRunner: func(flowStore oauth.ClientAuthStore) indigoOAuthRunner {
			return &fakeIndigoRunner{store: flowStore}
		},
	}
	if _, err := flow.StartLink(t.Context(), "person-1", "user.example.com"); !errors.Is(err, wantErr) {
		t.Fatalf("StartLink() error = %v, want persistence error", err)
	}
}

func TestOAuthFlowRejectsUnsafeOrUnboundAuthorizationRedirect(t *testing.T) {
	for _, redirect := range []string{
		"http://auth.example.com/authorize?client_id=client&request_uri=urn:test",
		"https://auth.example.com/authorize?client_id=wrong&request_uri=urn:test",
		"https://auth.example.com/authorize?client_id=client",
	} {
		t.Run(redirect, func(t *testing.T) {
			store := oauth.NewMemStore()
			flow := &OAuthFlow{
				clientID: "client",
				store:    store,
				newRunner: func(flowStore oauth.ClientAuthStore) indigoOAuthRunner {
					return &fakeIndigoRunner{store: flowStore, redirect: redirect}
				},
			}
			if _, err := flow.StartLink(t.Context(), "person-1", "user.example.com"); err == nil {
				t.Fatal("unsafe authorization redirect unexpectedly accepted")
			}
		})
	}
}

func TestOAuthFlowCompletesToPlainDIDResult(t *testing.T) {
	did, err := syntax.ParseDID("did:plc:vwzwgnygau7ed7b7wt5ux7y2")
	if err != nil {
		t.Fatal(err)
	}
	flow := &OAuthFlow{
		store: oauth.NewMemStore(),
		newRunner: func(oauth.ClientAuthStore) indigoOAuthRunner {
			return &fakeIndigoRunner{callbackSession: &oauth.ClientSessionData{AccountDID: did}}
		},
	}
	result, err := flow.CompleteLink(t.Context(), url.Values{"state": {"state-1"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.DID != did.String() {
		t.Fatalf("DID = %q", result.DID)
	}
}

func TestOAuthFlowClassifiesProviderDenialWithoutDescription(t *testing.T) {
	flow := &OAuthFlow{
		store: oauth.NewMemStore(),
		newRunner: func(oauth.ClientAuthStore) indigoOAuthRunner {
			return &fakeIndigoRunner{callbackErr: &oauth.AuthRequestCallbackError{
				ErrorCode: "access_denied", ErrorDescription: "untrusted provider description",
			}}
		},
	}
	_, err := flow.CompleteLink(t.Context(), url.Values{"state": {"state-1"}})
	if !errors.Is(err, ErrOAuthDenied) {
		t.Fatalf("CompleteLink() error = %v, want denial", err)
	}
	if err != nil && strings.Contains(err.Error(), "untrusted provider description") {
		t.Fatal("denial error exposed provider description")
	}
}

func TestPublicOnlyHTTPClientDisablesProxyAndRedirects(t *testing.T) {
	client := publicOnlyHTTPClient(0)
	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport.Proxy != nil {
		t.Fatalf("unexpected hardened transport: %#v", client.Transport)
	}
	request, err := http.NewRequest(http.MethodGet, "https://other.example", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.CheckRedirect(request, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("redirect policy error = %v", err)
	}
}

func TestIdentityDirectoryHardeningFailsClosedOnSDKShapeDrift(t *testing.T) {
	if !hardenIdentityDirectory(identity.DefaultDirectory()) {
		t.Fatal("current Indigo default directory was not hardened")
	}
	if hardenIdentityDirectory(&identity.MockDirectory{}) {
		t.Fatal("unsupported directory shape unexpectedly hardened")
	}
}

type fakeIndigoRunner struct {
	store           oauth.ClientAuthStore
	redirect        string
	seenIdentifier  *string
	callbackSession *oauth.ClientSessionData
	callbackErr     error
}

func (r *fakeIndigoRunner) StartAuthFlow(ctx context.Context, identifier string) (string, error) {
	if r.seenIdentifier != nil {
		*r.seenIdentifier = identifier
	}
	if r.store != nil {
		_ = r.store.SaveAuthRequestInfo(ctx, oauth.AuthRequestData{
			State:      "state-1",
			Scopes:     []string{"atproto"},
			RequestURI: "urn:example:request",
		})
	}
	if r.redirect != "" {
		return r.redirect, nil
	}
	return "https://auth.example.com/authorize?client_id=https%3A%2F%2Fsubcults.subcult.tv%2Fapi%2Fv1%2Fauth%2Fatproto%2Fclient-metadata&request_uri=urn%3Aexample%3Arequest", nil
}

func (r *fakeIndigoRunner) ProcessCallback(context.Context, url.Values) (*oauth.ClientSessionData, error) {
	return r.callbackSession, r.callbackErr
}

type failingSaveStore struct {
	oauth.ClientAuthStore
	err error
}

func (s *failingSaveStore) SaveAuthRequestInfo(context.Context, oauth.AuthRequestData) error {
	return s.err
}

type personCheckingStore struct {
	oauth.ClientAuthStore
	wantPersonID string
}

func (s *personCheckingStore) SaveAuthRequestInfo(ctx context.Context, info oauth.AuthRequestData) error {
	personID, _ := ctx.Value(oauthLinkPersonContextKey{}).(string)
	if personID != s.wantPersonID {
		return errors.New("missing bound person")
	}
	return s.ClientAuthStore.SaveAuthRequestInfo(ctx, info)
}
