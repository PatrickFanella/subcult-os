package atproto

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/util/ssrf"
)

type OAuthLinkResult struct {
	DID string
}

var ErrOAuthDenied = errors.New("AT OAuth authorization was denied")

type indigoOAuthRunner interface {
	StartAuthFlow(context.Context, string) (string, error)
	ProcessCallback(context.Context, url.Values) (*oauth.ClientSessionData, error)
}

type OAuthFlow struct {
	clientID  string
	store     oauth.ClientAuthStore
	newRunner func(oauth.ClientAuthStore) indigoOAuthRunner
}

// NewOAuthFlow binds the public client identity to encrypted durable storage.
// Each start gets an isolated store wrapper so an Indigo persistence error
// cannot be lost by the upstream StartAuthFlow helper.
func (c *OAuthClient) NewOAuthFlow(store oauth.ClientAuthStore) (*OAuthFlow, error) {
	if c == nil || store == nil {
		return nil, errors.New("AT OAuth flow requires client configuration and storage")
	}
	config := c.config
	return &OAuthFlow{
		clientID: config.ClientID,
		store:    store,
		newRunner: func(flowStore oauth.ClientAuthStore) indigoOAuthRunner {
			flowConfig := config
			return newHardenedIndigoClient(&flowConfig, flowStore)
		},
	}, nil
}

func (f *OAuthFlow) StartLink(ctx context.Context, personID, rawIdentifier string) (string, error) {
	identifier, err := ParseAccountIdentifier(rawIdentifier)
	if err != nil {
		return "", err
	}
	if personID == "" {
		return "", ErrOAuthLinkContext
	}
	capture := &captureSaveStore{ClientAuthStore: f.store}
	runner := f.newRunner(capture)
	redirectURL, err := runner.StartAuthFlow(WithOAuthLinkPerson(ctx, personID), identifier.Value)
	if err != nil {
		return "", fmt.Errorf("start AT OAuth flow: %w", err)
	}
	if capture.saveErr != nil {
		return "", fmt.Errorf("persist AT OAuth request: %w", capture.saveErr)
	}
	if !capture.saved {
		return "", errors.New("AT OAuth request was not persisted")
	}
	if err := validateAuthorizationRedirect(redirectURL, f.clientID); err != nil {
		return "", err
	}
	return redirectURL, nil
}

func (f *OAuthFlow) CompleteLink(ctx context.Context, params url.Values) (OAuthLinkResult, error) {
	session, err := f.newRunner(f.store).ProcessCallback(ctx, params)
	if err != nil {
		var callbackErr *oauth.AuthRequestCallbackError
		if errors.As(err, &callbackErr) {
			return OAuthLinkResult{}, fmt.Errorf("%w: %s", ErrOAuthDenied, callbackErr.ErrorCode)
		}
		return OAuthLinkResult{}, fmt.Errorf("complete AT OAuth flow: %w", err)
	}
	if session == nil || session.AccountDID.String() == "" {
		return OAuthLinkResult{}, errors.New("AT OAuth callback returned no account DID")
	}
	return OAuthLinkResult{DID: session.AccountDID.String()}, nil
}

type captureSaveStore struct {
	oauth.ClientAuthStore
	saved   bool
	saveErr error
}

func (s *captureSaveStore) SaveAuthRequestInfo(ctx context.Context, info oauth.AuthRequestData) error {
	s.saveErr = s.ClientAuthStore.SaveAuthRequestInfo(ctx, info)
	s.saved = s.saveErr == nil
	return s.saveErr
}

func validateAuthorizationRedirect(raw, clientID string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" {
		return errors.New("AT OAuth authorization redirect is not a safe HTTPS URL")
	}
	query := parsed.Query()
	if query.Get("client_id") != clientID || query.Get("request_uri") == "" {
		return errors.New("AT OAuth authorization redirect is missing bound request parameters")
	}
	return nil
}

func newHardenedIndigoClient(config *oauth.ClientConfig, store oauth.ClientAuthStore) indigoOAuthRunner {
	client := oauth.NewClientApp(config, store)
	client.Client = publicOnlyHTTPClient(30 * time.Second)
	client.Resolver.Client = publicOnlyHTTPClient(10 * time.Second)
	if !hardenIdentityDirectory(client.Dir) {
		return failedIndigoRunner{err: errors.New("AT OAuth identity resolver shape is unsupported")}
	}
	return client
}

type failedIndigoRunner struct{ err error }

func (r failedIndigoRunner) StartAuthFlow(context.Context, string) (string, error) {
	return "", r.err
}

func (r failedIndigoRunner) ProcessCallback(context.Context, url.Values) (*oauth.ClientSessionData, error) {
	return nil, r.err
}

func publicOnlyHTTPClient(timeout time.Duration) *http.Client {
	transport := ssrf.PublicOnlyTransport()
	transport.Proxy = nil
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func hardenIdentityDirectory(directory identity.Directory) bool {
	cache, ok := directory.(*identity.CacheDirectory)
	if !ok {
		return false
	}
	base, ok := cache.Inner.(*identity.BaseDirectory)
	if !ok {
		return false
	}
	base.HTTPClient = *publicOnlyHTTPClient(10 * time.Second)
	base.PLCClient = publicOnlyHTTPClient(10 * time.Second)
	return true
}
