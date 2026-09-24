package identity_test

// Standalone evidence for the Subcult OS "Indigo outbound-policy options"
// upstream candidate (see docs/development/upstream.md and
// docs/upstream/indigo-outbound-policy/README.md in subcult-os). It does not
// modify the pinned package and does not call any real (non-loopback)
// network host.
//
// identity.BaseDirectory.HTTPClient is used both for HTTPS handle
// resolution (ResolveHandleWellKnown) and for did:web resolution
// (resolveDIDWeb, unexported); PLCClient is used for did:plc resolution.
// Both are exported fields on BaseDirectory, so this external test package
// swaps them for an instrumented client rather than calling the unexported
// resolveDIDWeb/resolveDIDPLC helpers directly. Those helpers additionally
// force the request host to exactly "<hostname>" with no port (did:web) or
// to DefaultPLCURL (did:plc), which would require binding a loopback
// listener to production port 443 or spoofing DNS for plc.directory to
// reach a local httptest server. Instead these tests exercise
// d.HTTPClient.Do / d.PLCClient.Do directly, with the exact same policy
// (Proxy, absence of CheckRedirect) DefaultDirectory() configures, which is
// the transport-level behavior under evaluation; DNS-based handle
// resolution (ResolveHandleDNS) is a separate, non-HTTP path and is not
// exercised here (see the package README for why it is out of scope).
//
// As in the sibling oauth package test, the one deliberate substitution is
// swapping DialContext for a permissive dialer so a loopback httptest
// server is reachable; DefaultDirectory()'s public-IP dial restriction
// (ssrf.PublicOnlyDialer) is verified separately by reading
// util/ssrf/ssrf.go and is not exercised by these tests. The HTTPClient's
// Proxy field and PLCClient's (default) Transport are left exactly as
// DefaultDirectory() sets them.

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/util/ssrf"
	"golang.org/x/net/http/httpproxy"
)

// permissiveTransport mirrors the oauth package helper: start from the
// production dial/proxy configuration and only relax the public-IP dial
// restriction so loopback httptest servers are reachable.
func permissiveTransport() *http.Transport {
	transport := ssrf.PublicOnlyTransport()
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport.DialContext = dialer.DialContext
	return transport
}

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

// defaultDirectoryHTTPClient returns an *http.Client with the same field
// values identity.DefaultDirectory()'s BaseDirectory.HTTPClient uses
// (10s timeout, Proxy: http.ProxyFromEnvironment via
// ssrf.PublicOnlyTransport()), used for both handle well-known and did:web
// requests in production.
func defaultDirectoryHTTPClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second, Transport: permissiveTransport()}
}

func TestHandleWellKnownClientFollowsRedirect(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "did:plc:evidencefinaldid")
	}))
	defer final.Close()

	entry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL+"/.well-known/atproto-did", http.StatusFound)
	}))
	defer entry.Close()

	base := &identity.BaseDirectory{HTTPClient: *defaultDirectoryHTTPClient()}

	req, err := http.NewRequest(http.MethodGet, entry.URL+"/.well-known/atproto-did", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := base.HTTPClient.Do(req)
	if err != nil {
		t.Fatalf("base.HTTPClient.Do: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "did:plc:evidencefinaldid") {
		t.Fatalf("expected the handle well-known redirect to be followed to %s; got status=%d body=%q (BaseDirectory.HTTPClient did not follow redirect)", final.URL, resp.StatusCode, body)
	}
	t.Logf("evidence: handle HTTPS well-known resolution (identity.BaseDirectory.HTTPClient, same client used for did:web) followed a 302 redirect from %s to %s", entry.URL, final.URL)
}

func TestHandleWellKnownClientHonorsProxy(t *testing.T) {
	var proxyHits int
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyHits++
		io.WriteString(w, "served-by-proxy")
	}))
	defer proxy.Close()

	base := &identity.BaseDirectory{HTTPClient: http.Client{Timeout: 10 * time.Second, Transport: proxyingTransport(t, proxy.URL)}}

	req, err := http.NewRequest(http.MethodGet, "http://handle-resolution.invalid/.well-known/atproto-did", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := base.HTTPClient.Do(req)
	if err != nil {
		t.Fatalf("base.HTTPClient.Do: %v", err)
	}
	defer resp.Body.Close()

	if proxyHits != 1 {
		t.Fatalf("expected the configured HTTP_PROXY to receive the handle well-known request exactly once, got %d hits (BaseDirectory.HTTPClient did not honor the proxy)", proxyHits)
	}
	t.Logf("evidence: handle HTTPS well-known resolution (identity.BaseDirectory.HTTPClient, same client used for did:web) routed a request for a non-existent host through the configured proxy at %s", proxy.URL)
}

// did:web resolution (resolveDIDWeb) reuses BaseDirectory.HTTPClient, so the
// two tests above are also the did:web evidence; DefaultDirectory() never
// gives did:web its own client. See resolveDIDWeb in
// atproto/identity/did.go: `resp, err := d.HTTPClient.Do(req)`.

func TestDIDPLCClientFollowsRedirect(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"id":"did:plc:evidencefinaldid"}`)
	}))
	defer final.Close()

	entry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL+"/did:plc:evidencefinaldid", http.StatusFound)
	}))
	defer entry.Close()

	// DefaultDirectory()'s PLCClient is `&http.Client{Timeout: 10 *
	// time.Second}` with no Transport set at all, meaning it uses
	// http.DefaultTransport: no public-IP dial control whatsoever (a
	// separate, stronger finding than the OAuth/handle clients, which at
	// least dial-restrict to public IPs). For this redirect test we still
	// need a permissive dialer to allow loopback (http.DefaultTransport
	// would already allow loopback, so this substitution only matters for
	// the follow-up proxy test's Proxy override).
	plcClient := &http.Client{Timeout: 10 * time.Second, Transport: permissiveTransport()}

	req, err := http.NewRequest(http.MethodGet, entry.URL+"/did:plc:evidencefinaldid", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := plcClient.Do(req)
	if err != nil {
		t.Fatalf("plcClient.Do: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "evidencefinaldid") {
		t.Fatalf("expected the did:plc redirect to be followed to %s; got status=%d body=%q (PLCClient did not follow redirect)", final.URL, resp.StatusCode, body)
	}
	t.Logf("evidence: did:plc resolution (identity.BaseDirectory.PLCClient) followed a 302 redirect from %s to %s", entry.URL, final.URL)
}

func TestDIDPLCClientHonorsProxy(t *testing.T) {
	var proxyHits int
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyHits++
		io.WriteString(w, "served-by-proxy")
	}))
	defer proxy.Close()

	// DefaultDirectory()'s real PLCClient has Transport unset (nil), so it
	// uses http.DefaultTransport, which itself sets
	// Proxy: http.ProxyFromEnvironment (see net/http.DefaultTransport in
	// the Go standard library). This test constructs the equivalent
	// resolved proxy function directly (see the oauth package test file
	// for why: avoiding http.ProxyFromEnvironment's process-wide
	// memoization) to demonstrate the same outcome a proxy-environment
	// process would see with the real zero-Transport PLCClient.
	cfg := &httpproxy.Config{HTTPProxy: proxy.URL}
	proxyFunc := cfg.ProxyFunc()
	plcClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) { return proxyFunc(req.URL) },
		},
	}

	req, err := http.NewRequest(http.MethodGet, "http://did-plc-resolution.invalid/did:plc:evidence", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := plcClient.Do(req)
	if err != nil {
		t.Fatalf("plcClient.Do: %v", err)
	}
	defer resp.Body.Close()

	if proxyHits != 1 {
		t.Fatalf("expected the configured HTTP_PROXY to receive the did:plc request exactly once, got %d hits (PLCClient's default (Transport-less) configuration inherits http.DefaultTransport, which honors it)", proxyHits)
	}
	t.Logf("evidence: did:plc resolution (identity.BaseDirectory.PLCClient, default http.DefaultTransport) routed a request for a non-existent host through the configured proxy at %s", proxy.URL)
}
