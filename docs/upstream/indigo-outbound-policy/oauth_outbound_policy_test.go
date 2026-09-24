package oauth_test

// Standalone evidence for the Subcult OS "Indigo outbound-policy options"
// upstream candidate (see docs/development/upstream.md and
// docs/upstream/indigo-outbound-policy/README.md in subcult-os). It does not
// modify the pinned package and does not call any real (non-loopback)
// network host.
//
// These tests exercise the exact *http.Client values that
// oauth.NewResolver() and oauth.NewClientApp() construct in production
// (same Proxy field, same absence of CheckRedirect). The one deliberate
// substitution is the Transport's DialContext: production uses
// ssrf.PublicOnlyDialer(), which rejects loopback addresses by design, so a
// loopback httptest server can never be dialed through it. Each helper below
// starts from ssrf.PublicOnlyTransport() (identical Proxy and other fields to
// production) and only swaps DialContext for a permissive dialer. The
// public-IP dial restriction itself is not exercised here; it is verified by
// reading util/ssrf/ssrf.go (see the package README).
//
// The metadata-discovery test also bypasses oauth.Resolver's higher-level
// Resolve* methods, which reject any host URL carrying an explicit port
// (see resolver.go: "u.Port() != \"\""). That check is a URL-shape
// requirement unrelated to proxy/redirect transport policy, and it would
// make it impossible to target a local httptest server without binding
// production port 443. Instead these tests call resolver.Client.Do and
// app.Client.Do directly with the same request shape the high-level helpers
// build, to isolate exactly the transport-level policy under evaluation.

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/util/ssrf"
	"golang.org/x/net/http/httpproxy"
)

// permissiveTransport starts from the exact production transport
// (ssrf.PublicOnlyTransport(), which sets Proxy: http.ProxyFromEnvironment)
// and only relaxes the dial-time public-IP restriction so it can reach
// loopback httptest servers. Every other field, including Proxy, is
// untouched.
func permissiveTransport() *http.Transport {
	transport := ssrf.PublicOnlyTransport()
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport.DialContext = dialer.DialContext
	return transport
}

// proxyingTransport is permissiveTransport with Proxy forced to route
// through proxyURL. http.ProxyFromEnvironment memoizes the environment on
// its first call in a process (see https://pkg.go.dev/net/http#ProxyFromEnvironment),
// so a test that mutates HTTP_PROXY after another test/init already called
// it would flake depending on run order. golang.org/x/net/http/httpproxy is
// the same library net/http's ProxyFromEnvironment delegates to; calling it
// directly, per test, reproduces the identical proxy-selection behavior
// production would exhibit for a process that starts with that environment
// variable set, without process-wide memoization.
func proxyingTransport(t *testing.T, proxyURL string) *http.Transport {
	t.Helper()
	transport := permissiveTransport()
	cfg := &httpproxy.Config{HTTPProxy: proxyURL}
	proxyFunc := cfg.ProxyFunc()
	transport.Proxy = func(req *http.Request) (*url.URL, error) {
		return proxyFunc(req.URL)
	}
	return transport
}

func TestOAuthMetadataResolverFollowsRedirect(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "final-metadata-document")
	}))
	defer final.Close()

	entry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL+"/.well-known/oauth-authorization-server", http.StatusFound)
	}))
	defer entry.Close()

	resolver := oauth.NewResolver()
	resolver.Client.Transport = permissiveTransport()

	req, err := http.NewRequest(http.MethodGet, entry.URL+"/.well-known/oauth-authorization-server", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := resolver.Client.Do(req)
	if err != nil {
		t.Fatalf("resolver.Client.Do: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// oauth.Resolver never sets CheckRedirect, so the http.Client default
	// (follow up to 10 redirects) applies: the final response comes from
	// the redirect target, not the 302 itself.
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "final-metadata-document") {
		t.Fatalf("expected the redirect to be followed to %s; got status=%d body=%q (resolver.Client did not follow redirect)", final.URL, resp.StatusCode, body)
	}
	if resp.Request.URL.String() != final.URL+"/.well-known/oauth-authorization-server" {
		t.Fatalf("expected final request URL to be the redirect target, got %s", resp.Request.URL)
	}
	t.Logf("evidence: OAuth server metadata discovery (oauth.Resolver.Client) followed a same-origin-unchecked 302 redirect to %s", final.URL)
}

func TestOAuthMetadataResolverHonorsProxy(t *testing.T) {
	var proxyHits int
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyHits++
		io.WriteString(w, "served-by-proxy")
	}))
	defer proxy.Close()

	resolver := oauth.NewResolver()
	resolver.Client.Transport = proxyingTransport(t, proxy.URL)

	// Target host does not need to exist: a correctly configured forward
	// proxy intercepts the absolute-form request before any DNS lookup or
	// dial to the target host happens.
	req, err := http.NewRequest(http.MethodGet, "http://oauth-metadata.invalid/.well-known/oauth-authorization-server", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := resolver.Client.Do(req)
	if err != nil {
		t.Fatalf("resolver.Client.Do: %v", err)
	}
	defer resp.Body.Close()

	if proxyHits != 1 {
		t.Fatalf("expected the configured HTTP_PROXY to receive the metadata request exactly once, got %d hits (resolver.Client did not honor the proxy)", proxyHits)
	}
	t.Logf("evidence: OAuth server metadata discovery (oauth.Resolver.Client) routed a request for a non-existent host through the configured proxy at %s", proxy.URL)
}

func TestOAuthTokenClientFollowsRedirect(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"request_uri":"urn:ietf:params:oauth:request_uri:evidence","expires_in":90}`)
	}))
	defer final.Close()

	entry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL+"/par", http.StatusFound)
	}))
	defer entry.Close()

	store := oauth.NewMemStore()
	config := oauth.NewPublicConfig("https://client.example/metadata", "https://client.example/callback", []string{"atproto"})
	app := oauth.NewClientApp(&config, store)
	app.Client.Transport = permissiveTransport()

	req, err := http.NewRequest(http.MethodPost, entry.URL+"/par", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := app.Client.Do(req)
	if err != nil {
		t.Fatalf("app.Client.Do: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "request_uri") {
		t.Fatalf("expected the PAR/token redirect to be followed to %s; got status=%d body=%q (app.Client did not follow redirect)", final.URL, resp.StatusCode, body)
	}
	t.Logf("evidence: OAuth PAR/token endpoint client (oauth.ClientApp.Client) followed a 302 redirect from %s to %s", entry.URL, final.URL)
}

func TestOAuthTokenClientHonorsProxy(t *testing.T) {
	var proxyHits int
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyHits++
		io.WriteString(w, "served-by-proxy")
	}))
	defer proxy.Close()

	store := oauth.NewMemStore()
	config := oauth.NewPublicConfig("https://client.example/metadata", "https://client.example/callback", []string{"atproto"})
	app := oauth.NewClientApp(&config, store)
	app.Client.Transport = proxyingTransport(t, proxy.URL)

	req, err := http.NewRequest(http.MethodPost, "http://token-endpoint.invalid/token", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := app.Client.Do(req)
	if err != nil {
		t.Fatalf("app.Client.Do: %v", err)
	}
	defer resp.Body.Close()

	if proxyHits != 1 {
		t.Fatalf("expected the configured HTTP_PROXY to receive the token request exactly once, got %d hits (app.Client did not honor the proxy)", proxyHits)
	}
	t.Logf("evidence: OAuth PAR/token endpoint client (oauth.ClientApp.Client) routed a request for a non-existent host through the configured proxy at %s", proxy.URL)
}
