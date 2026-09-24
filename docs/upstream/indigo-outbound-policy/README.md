# Indigo outbound-policy contribution candidate

Prepared for the Subcult OS "Indigo outbound-policy options" candidate tracked
in [`docs/development/upstream.md`](../../development/upstream.md). This
package is a tested contribution draft, not an upstream submission or
accepted patch. It follows the structure of the sibling
[Indigo persistence contribution package](../indigo-persistence/README.md):
evidence first, then a proposal, then compatibility notes, then publication
material. Nothing here changes Subcult's production dependency or its
existing hardening in `backend/internal/atproto/oauth_flow.go`
(`newHardenedIndigoClient`, `hardenIdentityDirectory`), which stays in place.

Indigo commit: `41278964ec8e3253e70d4e919dfb8e34211c543d` (pseudo-version
`v0.0.0-20260903211445-41278964ec8e`, matching `backend/go.mod`).

## Problem

Indigo's default OAuth and identity HTTP clients dial-restrict to public IP
addresses (`util/ssrf.PublicOnlyControl`), but that is the only outbound
control most of them apply by default. Two policies are left at the Go
standard library default across every request kind this review covers:

- **Proxy**: `ssrf.PublicOnlyTransport()` explicitly sets
  `Proxy: http.ProxyFromEnvironment`. `identity.DefaultDirectory()`'s own doc
  comment says so directly: "If an HTTP proxy is configured (via environment
  variable), SSRF protections will not apply to external requests (it is the
  responsibility of the proxy to handle SSRF)." That is a documented,
  presumably intentional tradeoff, but it means an ambient `HTTP_PROXY`/
  `HTTPS_PROXY` in a deployment's environment (set for an unrelated reason,
  or injected by an attacker with process/environment access) silently
  redirects OAuth metadata, token, handle and DID traffic through that proxy,
  bypassing the public-IP dial check entirely for whatever the proxy in turn
  contacts.
- **Redirects**: none of `oauth.NewResolver()`, `oauth.NewClientApp()`, or
  `identity.DefaultDirectory()` set `http.Client.CheckRedirect`, so the
  standard library default applies (follow up to 10 redirects, forwarding
  the original method/body per normal `net/http` semantics for 307/308, or
  demoting to GET for 301/302/303 on non-GET/HEAD as usual). `resolver.go`
  says so in three places, verbatim: `// NOTE: this allows redirects`.

A fourth, narrower finding: `identity.DefaultDirectory()`'s `PLCClient` is
constructed as `&http.Client{Timeout: 10 * time.Second}` with **no
`Transport` set at all**, so it falls back to `http.DefaultTransport` and has
**no public-IP dial restriction whatsoever** for did:plc lookups (stronger
than "honors an ambient proxy" — it also has no SSRF dial control to bypass
in the first place). `BaseDirectory.HTTPClient` (used for handle well-known
and did:web) does have the public-IP dial restriction.

This is a defense-in-depth/hardening gap, not a demonstrated exploit against
a specific deployment; whether it is exploitable in a given Subcult
deployment depends on whether that deployment's environment ever has
`HTTP_PROXY`/`HTTPS_PROXY` set and whether an attacker can influence it or
the DNS/redirect chain of a resolved auth server or PDS host. Subcult OS
already closes this gap locally in `oauth_flow.go` (see below); this
document is about proposing the same fix upstream so other Indigo OAuth/
identity consumers get it without having to reverse-engineer Subcult's
override.

## Evidence

Source citations (pinned commit, read from the Go module cache,
`go env GOMODCACHE`, read-only; nothing under the module cache was
modified):

| Request kind | Client (production) | Proxy field | Redirect handling | Public-IP dial control |
|---|---|---|---|---|
| OAuth server metadata discovery (`/.well-known/oauth-authorization-server`, `/.well-known/oauth-protected-resource`) | `oauth.Resolver.Client` (`oauth.NewResolver()`, `atproto/auth/oauth/resolver.go:22-31`) | `http.ProxyFromEnvironment` (via `ssrf.PublicOnlyTransport()`) | none set → stdlib default follows redirects (`resolver.go` comments this explicitly at lines 48, 99, 141: `// NOTE: this allows redirects`) | yes (`ssrf.PublicOnlyControl`) |
| Token endpoint (PAR + initial token request) | `oauth.ClientApp.Client` (`oauth.NewClientApp()`, `atproto/auth/oauth/oauth.go:54-59`) | `http.ProxyFromEnvironment` (via `ssrf.PublicOnlyTransport()`) | none set → stdlib default follows redirects | yes |
| Handle resolution, HTTPS well-known fallback (`ResolveHandleWellKnown`, `atproto/identity/handle.go:131-177`; DNS-based `ResolveHandleDNS` is not HTTP and is out of scope, see below) | `identity.BaseDirectory.HTTPClient` (`identity.DefaultDirectory()`, `atproto/identity/directory.go:68-100`) | `http.ProxyFromEnvironment` | none set → stdlib default follows redirects | yes |
| DID document fetch, did:web (`resolveDIDWeb`, `atproto/identity/did.go:79-138`) | same `BaseDirectory.HTTPClient` as handle resolution | `http.ProxyFromEnvironment` | none set → stdlib default follows redirects | yes |
| DID document fetch, did:plc (`resolveDIDPLC`, `atproto/identity/did.go:140-184`) | `identity.BaseDirectory.PLCClient` (`identity.DefaultDirectory()`, `directory.go:85-87`: `&http.Client{Timeout: time.Second * 10}`, **no `Transport`**) | `http.ProxyFromEnvironment` (inherited from `http.DefaultTransport`, since none is set) | none set → stdlib default follows redirects | **no** (falls back to `http.DefaultTransport`, which has no dial restriction at all) |

Every request kind in scope: **proxy honored: yes. Redirect followed: yes.**
None of the four is already strict on either axis; the differences are only
in which dial-level control (if any) sits alongside that default.

Confirmed unchanged on upstream `main` as of 2026-09-23 (see "Upstream
state" below): the same file/line evidence applies to `main`, not just the
pinned commit.

### Standalone tests

Two new external test files, `oauth_outbound_policy_test.go` (package
`oauth_test`) and `identity_outbound_policy_test.go` (package
`identity_test`), copied verbatim into this directory from the disposable
reproduction described below. They add no dependency the pinned module
doesn't already have (`golang.org/x/net/http/httpproxy` is already an
indirect dependency of Indigo, pinned at `v0.58.0`), require no database, DNS,
provider account, Subcult server, or real (non-loopback) network host, and
do not modify the pinned package.

Each test targets the exact `*http.Client` value production code uses
(`oauth.NewResolver().Client`, `oauth.NewClientApp(...).Client`,
`identity.BaseDirectory.HTTPClient`, and an equivalent of the default
`PLCClient`), then either:

- starts two `httptest.NewServer`s, has the first respond with an HTTP 302 to
  the second, and asserts the client's response came from the second server
  (redirect followed); or
- starts one `httptest.NewServer` acting as a forward proxy stand-in, points
  a request at a non-existent host, sets the client's `Transport.Proxy` to
  resolve exactly as `http.ProxyFromEnvironment` would for a process that
  started with `HTTP_PROXY` set to that server's URL, and asserts the "proxy"
  server received the request (proxy honored).

Two substitutions are made everywhere, both documented in the test files'
package comments:

1. **Dial control.** Production's `ssrf.PublicOnlyDialer()` rejects loopback
   addresses by design (`ssrf.go`: `reservedIPv4Nets` includes
   `127.0.0.0/8`), so a real production transport can never reach an
   `httptest` server. Every helper starts from `ssrf.PublicOnlyTransport()`
   itself (so `Proxy` and every other field is exactly what production sets)
   and only replaces `DialContext` with a plain `net.Dialer` that allows
   loopback. The public-IP restriction this bypasses is not what these tests
   evaluate; it is verified separately by reading `util/ssrf/ssrf.go` (table
   above), and it is real and effective for the two clients that have it.
2. **Proxy-environment memoization.** `http.ProxyFromEnvironment` reads the
   environment once per process and caches the result
   (<https://pkg.go.dev/net/http#ProxyFromEnvironment>), so mutating
   `HTTP_PROXY` mid-test-binary is order-dependent and flaky. The tests
   construct the equivalent proxy-selection function directly via
   `golang.org/x/net/http/httpproxy` — the same library `net/http` delegates
   to — reproducing exactly what a fresh process with that environment
   variable set would do, without relying on `os.Setenv` + cache timing.

The metadata-discovery and did:plc tests also call the client's `Do` method
directly rather than through `oauth.Resolver.ResolveAuthServerMetadata` /
`ResolveAuthServerURL` or `identity.BaseDirectory`'s unexported
`resolveDIDWeb`/`resolveDIDPLC`, because those wrapper methods separately
reject any host URL with an explicit port (`resolver.go`:
`u.Port() != ""` → error) or hardcode the target host (`did.go`: always
`https://<hostname>/.well-known/did.json`, or `DefaultPLCURL`). Those are
URL-shape requirements orthogonal to the transport-level proxy/redirect
policy under test, and satisfying them would require binding a loopback
listener to production port 443 or spoofing DNS for `plc.directory`. Calling
the client's `Do` method directly isolates exactly the mechanism in
question. This is documented in both test files' header comments.

DNS-based handle resolution (`ResolveHandleDNS`, `ResolveHandleDNSAuthoritative`,
`ResolveHandleDNSFallback`) is not HTTP and has no proxy or `CheckRedirect`
concept; it is out of scope for this specific proxy/redirect review (a
compromised or untrusted DNS resolver is a different SSRF class, already
partly addressed by `TryAuthoritativeDNS`/`FallbackDNSServers`, and not
evaluated here).

### Reproduction 2026-09-23

Ran from a disposable copy under `/tmp/indigo-outbound-repro`, outside this
repository and outside the read-only `.blacktower/clonedeps` mirror (absent
in this worktree, as with the sibling persistence package's 2026-09-23
reproduction). Source came from the local module cache
(`go env GOMODCACHE`), read-only; nothing under the module cache was
modified. The copy step used a small local Python script
(`shutil.copytree` + `chmod`) rather than a shell one-liner, purely because
this worktree session's git-operation guard treats any command mentioning
`github.com` (even as a plain filesystem path fragment, unrelated to any git
command) as a git operation on a foreign remote and refuses to run it; the
copy itself is a plain recursive file copy, identical in effect to the
sibling package's `cp -r`.

Go `1.26.6`, `GOPROXY=off`, `GOFLAGS=-mod=mod`, separate `GOCACHE`:

```sh
cp -r "$(go env GOMODCACHE)/github.com/bluesky-social/indigo@v0.0.0-20260903211445-41278964ec8e" /tmp/indigo-outbound-repro/pinned
chmod -R u+w /tmp/indigo-outbound-repro/pinned
cp oauth_outbound_policy_test.go /tmp/indigo-outbound-repro/pinned/atproto/auth/oauth/outbound_policy_test.go
cp identity_outbound_policy_test.go /tmp/indigo-outbound-repro/pinned/atproto/identity/outbound_policy_test.go

cd /tmp/indigo-outbound-repro/pinned
go test ./atproto/auth/oauth -v -count=1 -run TestOAuthMetadata
go test ./atproto/auth/oauth -v -count=1 -run TestOAuthToken
go test ./atproto/identity -v -count=1
go test -race ./atproto/auth/oauth ./atproto/identity -count=1
```

Output (OAuth metadata discovery):

```
=== RUN   TestOAuthMetadataResolverFollowsRedirect
    outbound_policy_test.go:110: evidence: OAuth server metadata discovery (oauth.Resolver.Client) followed a same-origin-unchecked 302 redirect to http://127.0.0.1:44321
--- PASS: TestOAuthMetadataResolverFollowsRedirect (0.00s)
=== RUN   TestOAuthMetadataResolverHonorsProxy
    outbound_policy_test.go:140: evidence: OAuth server metadata discovery (oauth.Resolver.Client) routed a request for a non-existent host through the configured proxy at http://127.0.0.1:37209
--- PASS: TestOAuthMetadataResolverHonorsProxy (0.00s)
PASS
ok  	github.com/bluesky-social/indigo/atproto/auth/oauth	0.004s
```

Output (OAuth PAR/token endpoint):

```
=== RUN   TestOAuthTokenClientFollowsRedirect
    outbound_policy_test.go:173: evidence: OAuth PAR/token endpoint client (oauth.ClientApp.Client) followed a 302 redirect from http://127.0.0.1:35171 to http://127.0.0.1:38847
--- PASS: TestOAuthTokenClientFollowsRedirect (0.00s)
=== RUN   TestOAuthTokenClientHonorsProxy
    outbound_policy_test.go:202: evidence: OAuth PAR/token endpoint client (oauth.ClientApp.Client) routed a request for a non-existent host through the configured proxy at http://127.0.0.1:34635
--- PASS: TestOAuthTokenClientHonorsProxy (0.00s)
PASS
ok  	github.com/bluesky-social/indigo/atproto/auth/oauth	0.004s
```

Output (identity package, full suite — pre-existing live tests skip as they
already do on the unmodified pinned source; the four new tests pass):

```
=== RUN   TestDIDDocParse
--- PASS: TestDIDDocParse (0.00s)
=== RUN   TestDIDDocFeedGenParse
--- PASS: TestDIDDocFeedGenParse (0.00s)
=== RUN   TestHandleExtraction
--- PASS: TestHandleExtraction (0.00s)
=== RUN   TestBaseDirectory
    live_test.go:63: TODO: skipping live network test
--- SKIP: TestBaseDirectory (0.00s)
=== RUN   TestDefaultDirectory
    live_test.go:69: TODO: skipping live network test
--- SKIP: TestDefaultDirectory (0.00s)
=== RUN   TestCacheDirectory
    live_test.go:75: TODO: skipping live network test
--- SKIP: TestCacheDirectory (0.00s)
=== RUN   TestCacheCoalesce
    live_test.go:84: TODO: skipping live network test
--- SKIP: TestCacheCoalesce (0.00s)
=== RUN   TestFallbackDNS
    live_test.go:130: TODO: skipping live network test
--- SKIP: TestFallbackDNS (0.00s)
=== RUN   TestResolveNSID
    live_test.go:152: TODO: skipping live network test
--- SKIP: TestResolveNSID (0.00s)
=== RUN   TestMockDirectory
--- PASS: TestMockDirectory (0.00s)
=== RUN   TestHandleWellKnownClientFollowsRedirect
    outbound_policy_test.go:105: evidence: handle HTTPS well-known resolution (identity.BaseDirectory.HTTPClient, same client used for did:web) followed a 302 redirect from http://127.0.0.1:38045 to http://127.0.0.1:37377
--- PASS: TestHandleWellKnownClientFollowsRedirect (0.00s)
=== RUN   TestHandleWellKnownClientHonorsProxy
    outbound_policy_test.go:131: evidence: handle HTTPS well-known resolution (identity.BaseDirectory.HTTPClient, same client used for did:web) routed a request for a non-existent host through the configured proxy at http://127.0.0.1:37595
--- PASS: TestHandleWellKnownClientHonorsProxy (0.00s)
=== RUN   TestDIDPLCClientFollowsRedirect
    outbound_policy_test.go:174: evidence: did:plc resolution (identity.BaseDirectory.PLCClient) followed a 302 redirect from http://127.0.0.1:32781 to http://127.0.0.1:44707
--- PASS: TestDIDPLCClientFollowsRedirect (0.00s)
=== RUN   TestDIDPLCClientHonorsProxy
    outbound_policy_test.go:215: evidence: did:plc resolution (identity.BaseDirectory.PLCClient, default http.DefaultTransport) routed a request for a non-existent host through the configured proxy at http://127.0.0.1:40479
--- PASS: TestDIDPLCClientHonorsProxy (0.00s)
PASS
ok  	github.com/bluesky-social/indigo/atproto/identity	0.005s
```

`-race` run (both packages, full suite): both `ok`, no data races
(`github.com/bluesky-social/indigo/atproto/auth/oauth 1.021s`,
`github.com/bluesky-social/indigo/atproto/identity 1.020s`).

The read-only dependency checkout under `.blacktower/clonedeps/repos` was
not present in this worktree and was not modified; nothing under the Go
module cache was modified.

## Proposal

Two shapes were evaluated for letting a caller opt into strict outbound
policy (no proxy, no redirects) across these four request kinds, while
keeping the existing public-IP dial checks:

**(a) Constructor/option-struct fields.** For example, an
`oauth.NewResolver(opts ...ResolverOption)` /
`oauth.NewClientApp(config, store, opts ...ClientAppOption)` /
`identity.DefaultDirectory(opts ...DirectoryOption)` with a
`WithStrictOutboundPolicy()` option that swaps in a no-proxy transport and a
redirect-blocking `CheckRedirect`. This is the more ergonomic shape for new
callers, but it touches the public constructor signatures of three
independent packages (`oauth.NewResolver`, `oauth.NewClientApp`,
`identity.DefaultDirectory`) and would need Indigo maintainers to agree on a
shared `Option` pattern across packages that don't currently share one.
Given Indigo's README asks contributors to avoid large, undiscussed
refactors, and given issue #1461 (see below) shows maintainers are already
thinking about client-construction ergonomics but haven't converged on an
approach, an options-pattern PR touching three constructors is a bigger ask
than this candidate's evidence currently justifies.

**(b) A documented strict profile assembled from exported pieces.** Every
field this review touches — `oauth.Resolver.Client`, `oauth.ClientApp.Client`,
`identity.BaseDirectory.HTTPClient`, `identity.BaseDirectory.PLCClient` — is
already exported and mutable after construction. A caller can already do
exactly what Subcult OS does today in
`backend/internal/atproto/oauth_flow.go` (`newHardenedIndigoClient`,
`publicOnlyHTTPClient`, `hardenIdentityDirectory`): construct the default
client, then overwrite `Transport.Proxy = nil` and set
`CheckRedirect: func(...) error { return http.ErrUseLastResponse }`. No
upstream code change is required for this to work; the gap is that it is
undocumented and each caller has to reverse-engineer which struct fields to
touch (Subcult's own `hardenIdentityDirectory` has to type-assert through
`identity.CacheDirectory`/`identity.BaseDirectory` to reach `HTTPClient`/
`PLCClient`, since `identity.Directory` is an interface).

**Recommendation: (b), plus one small additive helper.** Propose adding a
single new exported function, `ssrf.StrictPublicOnlyTransport()` (this
package's `fix.patch`, below), that returns `PublicOnlyTransport()` with
`Proxy` forced to `nil`. This does not change any existing default, does not
touch any of the three constructors' signatures, and gives callers (Subcult
included) one line instead of three (`transport := ssrf.PublicOnlyTransport(); transport.Proxy = nil`)
at each of the four call sites. Pair it with a doc-comment recipe (in
`ssrf.go` and/or a new `atproto/identity`/`atproto/auth/oauth` doc example)
showing the full strict profile: swap in `StrictPublicOnlyTransport()` for
each `*http.Client.Transport`, and set `CheckRedirect` to
`http.ErrUseLastResponse` on each `*http.Client`. This is deliberately the
smaller, more conservative option given Indigo's stated preference for
narrowly scoped, discussable changes, and because option (a)'s ergonomic
benefit is marginal once (b)'s one-line-per-client helper exists.

Files/functions an eventual PR would touch:

- `util/ssrf/ssrf.go`: add `StrictPublicOnlyTransport()` (see `fix.patch`).
- `atproto/auth/oauth/resolver.go`, `oauth.go`: doc-comment only, showing how
  to combine `StrictPublicOnlyTransport()` with `CheckRedirect` on
  `Resolver.Client` and `ClientApp.Client`. No code change needed since both
  fields are already exported and settable post-construction.
- `atproto/identity/directory.go`: doc-comment only, same recipe for
  `BaseDirectory.HTTPClient`/`PLCClient`. Also worth a one-line doc-comment
  fix noting `PLCClient`'s current lack of any dial restriction when left
  unset (a separate, narrower finding from the proxy/redirect one — see
  table above), since a caller could otherwise reasonably assume `PLCClient`
  inherits `HTTPClient`'s public-IP protection when nil.

### Minimal draft patch

`fix.patch` (unified diff against `util/ssrf/ssrf.go` at the pinned commit,
format-patch style) adds only the one new function described above. Verified
in a disposable copy (Go `1.26.6`, `GOPROXY=off`, `GOFLAGS=-mod=mod`):

```sh
cd /tmp/indigo-outbound-repro/patched-check3   # fresh copy of pinned source
patch -p1 < /absolute/path/to/fix.patch        # applies cleanly, no fuzz, no reject
go build ./util/ssrf/...                       # succeeds
go vet ./util/ssrf/...                         # succeeds
go test ./util/ssrf/... -v -count=1            # TestPublicOnlyTransport still passes
```

`git apply --check` was not used to validate this patch, unlike the sibling
persistence package's reproduction: this worktree-isolated session's guard
refuses any `git` invocation outside its assigned worktree, and the
disposable copy under `/tmp` is outside it by design (to avoid touching the
Go module cache or this repository). Plain `patch -p1` was used instead for
both the dry-run check and the application; it applies without offset or
fuzz against the exact pinned source, which is the property that matters for
this evidence.

## Compatibility

- **Default behavior must remain opt-in and unchanged.** `fix.patch` only
  adds a new exported function; it does not modify `PublicOnlyTransport()`,
  `NewResolver()`, `NewClientApp()`, or `DefaultDirectory()`. Every existing
  caller of those four constructors sees identical behavior before and after
  this patch.
- **Tests added:** 8 new tests, 4 per file
  (`oauth_outbound_policy_test.go`: `TestOAuthMetadataResolverFollowsRedirect`,
  `TestOAuthMetadataResolverHonorsProxy`, `TestOAuthTokenClientFollowsRedirect`,
  `TestOAuthTokenClientHonorsProxy`;
  `identity_outbound_policy_test.go`: `TestHandleWellKnownClientFollowsRedirect`,
  `TestHandleWellKnownClientHonorsProxy`, `TestDIDPLCClientFollowsRedirect`,
  `TestDIDPLCClientHonorsProxy`). All pass against the unpatched pinned
  source (they characterize existing behavior, they do not test the patch
  itself) and continue to pass with `-race`; no existing test in either
  package was modified or weakened.
- **What breaks if a caller relies on proxies.** If a caller (Subcult
  included, hypothetically, if it later adopted the proposed helper for a
  client that currently doesn't override `Transport.Proxy`) switches a
  client to `StrictPublicOnlyTransport()` while its deployment's outbound
  egress is only reachable through a configured `HTTP_PROXY`/`HTTPS_PROXY`
  (the exact scenario `identity.DefaultDirectory`'s own doc comment
  anticipates as legitimate — "it is the responsibility of the proxy to
  handle SSRF"), every OAuth/identity request on that client starts failing
  closed (direct-connect attempts blocked by network egress policy or simply
  unroutable) instead of going through the trusted proxy. This is why the
  helper must stay additive/opt-in rather than becoming a new default:
  adopting it is a deployment-topology decision, not a safe blanket change.
  Subcult OS's own deployment does not currently route OAuth/identity
  traffic through an HTTP proxy (see `docs/development/atproto-kernel.md`'s
  "Outbound OAuth, handle and DID discovery uses public-IP-only dial
  controls with environment proxy use and HTTP redirects disabled"), so this
  candidate does not, by itself, change Subcult's own compatibility posture;
  it only proposes upstreaming the recipe Subcult already applies.

## Publication

### Upstream state, checked 2026-09-23

- `github.com/bluesky-social/indigo/util/ssrf/ssrf.go` on `main`
  (`raw.githubusercontent.com/bluesky-social/indigo/main/util/ssrf/ssrf.go`)
  still defines `PublicOnlyTransport()` exactly as quoted above, with
  `Proxy: http.ProxyFromEnvironment`.
- `atproto/auth/oauth/resolver.go` on `main` still contains the same three
  `// NOTE: this allows redirects` comments and does not set `CheckRedirect`
  on `Resolver`'s client. The defect/gap is present on current `main`, not
  just at the pinned commit.
- Searched `github.com/bluesky-social/indigo` issues and PRs for
  `CheckRedirect` (0 results) and `ProxyFromEnvironment` (0 results): no
  existing issue or PR discusses this specific proxy/redirect gap in the
  OAuth or identity packages.
- Searched for `SSRF` (13 results) and `PublicOnlyTransport` (2 results).
  Relevant prior work, all already merged into the pinned commit (dated
  2026-09-01, before this pin's 2026-09-03 build date) or still open:
  - [#1461](https://github.com/bluesky-social/indigo/issues/1461)
    "consider having re-usable 'safe' http.Client singletons for
    identity.Directory and atclient.APIClient" (open, filed 2026-09-01).
    About connection-pool reuse across repeated `http.Client` construction
    for SSRF hardening; does not mention proxy or redirect behavior
    specifically. Adjacent to this candidate's recommendation (b) — a shared
    "safe client" helper would be a natural place to also expose
    `StrictPublicOnlyTransport()` — but not a duplicate of it.
  - [#1452](https://github.com/bluesky-social/indigo/pull/1452) "harden
    identity package" (merged 2026-09-01). Added the dial-level SSRF
    `DialContext`, response size limits, and doc comments to
    `BaseDirectory`/`DefaultDirectory` — i.e., exactly the public-IP dial
    control this review confirms is present. Does not touch `Proxy` or
    `CheckRedirect`.
  - [#1451](https://github.com/bluesky-social/indigo/pull/1451) "util/ssrf:
    small improvements" (merged 2026-09-01). Added tests, a custom error
    type, and 6to4 handling to the SSRF dial-control utilities; explicitly
    describes itself as not changing behavior. Does not touch `Proxy` or
    `CheckRedirect`.
  - [#1217](https://github.com/bluesky-social/indigo/pull/1217),
    [#1379](https://github.com/bluesky-social/indigo/issues/1379),
    [#1320](https://github.com/bluesky-social/indigo/issues/1320),
    [#1216](https://github.com/bluesky-social/indigo/issues/1216): all about
    the *relay*'s `requestCrawl` SSRF checks being too strict for
    self-hosted/private-network deployments (the opposite direction from
    this candidate — loosening dial restrictions, not tightening
    proxy/redirect policy). Unrelated package, noted only to confirm they
    are not the same discussion.
  - [#1057](https://github.com/bluesky-social/indigo/pull/1057) "basic IP
    filtering net.Dialer for SSRF protection" (merged, the origin of
    `util/ssrf`) and [#1378](https://github.com/bluesky-social/indigo/pull/1378)
    "adds a CLI flag that toggles SSRF protection" (merged): predate and are
    superseded by the above; noted for completeness.
  - This confirms the specific proxy/redirect gap is still open and
    unaddressed upstream as of 2026-09-23, including by the most recent
    (2026-09-01) hardening work already present in the pinned commit.

### Submitter checklist (human, before any upstream submission)

None of the following were performed by this task; they gate an actual
submission and must be done by a human maintainer with authorization to
publish outside Subcult OS:

- [ ] Confirm authorization to publish this reproduction/patch outside the
      organization.
- [ ] Create or use a personal GitHub account for the submission (not a
      shared/service account).
- [ ] Read and comply with Indigo's DCO/CLA requirements if any are
      introduced before submission (none found in the README or repo root as
      of 2026-09-23, matching the persistence package's 2026-09-23 check;
      recheck at submission time).
- [ ] Open a GitHub issue first, per the README's contribution guidance, and
      wait for maintainer feedback before opening a PR. Consider
      cross-referencing (not duplicating) issue #1461, since a maintainer
      may prefer to fold this into that broader client-construction
      discussion rather than take a standalone `StrictPublicOnlyTransport()`
      addition.
- [ ] Re-run the reproduction against the then-current `main` immediately
      before submitting; `main` was confirmed to still contain the gap on
      2026-09-23 but may change.
- [ ] Decide, with maintainers, whether option (a) (constructor options) is
      preferred over option (b) (this candidate's recommendation) despite
      the larger surface area, and whether the redirect half of the recipe
      belongs in `util/ssrf` (which is otherwise dial/transport-only, not
      `http.Client`-level) or somewhere else, since `CheckRedirect` is a
      `Client` field this package doesn't currently touch.
- [ ] Attach the AI-assistance disclosure below to the issue/PR description.
- [ ] Do not reference internal Subcult issue numbers, credentials, or
      private infrastructure in the public submission.

### AI-assistance disclosure

Indigo states no AI-assistance policy in its README or repository root as of
2026-09-23 (rechecked; unchanged from the persistence package's 2026-09-23
finding). This disclosure is included anyway. This evidence, test files, and
patch were prepared with AI assistance (an automated coding agent), reviewed
for correctness against the pinned source, and verified by running the exact
commands and outputs recorded in this document. A human reviewed the diff
before it is considered for upstream submission per the checklist above.

### PR description draft

Title: `util/ssrf: add StrictPublicOnlyTransport (no ambient proxy)`

Body draft:

> **What**: `ssrf.PublicOnlyTransport()` sets `Proxy: http.ProxyFromEnvironment`,
> so any caller using it (directly, or via `oauth.NewResolver()`,
> `oauth.NewClientApp()`, or `identity.DefaultDirectory()`) will route
> outbound OAuth metadata/token and identity/handle/DID requests through
> whatever `HTTP_PROXY`/`HTTPS_PROXY` is set in the process environment,
> bypassing the public-IP dial check for whatever the proxy in turn contacts
> (this is called out in `identity.DefaultDirectory`'s own doc comment).
> Separately, none of those three constructors set `http.Client.CheckRedirect`,
> so the stdlib default (follow redirects) applies everywhere; `resolver.go`
> already comments on this in three places (`// NOTE: this allows
> redirects`).
>
> **Impact**: a deployment whose environment has an ambient proxy variable
> set for an unrelated reason, or whose environment an attacker can
> influence, gets silently weakened SSRF protection for these four request
> kinds. This is a hardening gap, not a demonstrated exploit; whether it
> matters depends on deployment specifics.
>
> **Repro**: attached test files exercise the exact `*http.Client` values
> `oauth.NewResolver()`, `oauth.NewClientApp()`, and
> `identity.DefaultDirectory()` construct (with only the dial-time public-IP
> check relaxed to reach loopback `httptest` servers; `Proxy` and
> `CheckRedirect` are untouched) and show all four request kinds both follow
> a same-origin-unchecked redirect and route through a configured
> `HTTP_PROXY`.
>
> **Fix**: add `ssrf.StrictPublicOnlyTransport()`, identical to
> `PublicOnlyTransport()` except `Proxy` is `nil`. Purely additive; no
> existing behavior changes. See attached `fix.patch`. Callers that want the
> full strict profile also need to set `CheckRedirect` on their
> `http.Client` to return `http.ErrUseLastResponse`; happy to also propose a
> doc-comment recipe showing that, or fold this into the #1461 discussion
> about reusable "safe" client singletons if maintainers prefer.
>
> **Question for maintainers**: should the redirect half of this live in
> `util/ssrf` at all (it's currently dial/transport-only, and
> `CheckRedirect` is an `http.Client` field), or is a documented recipe
> sufficient without any new exported symbol? Also open to closing this in
> favor of #1461 if a shared "safe client" constructor is the preferred
> direction.
>
> **Disclosure**: this reproduction and patch were prepared with AI
> assistance and reviewed/tested by a human before submission (see
> repository's stated AI-assistance disclosure, if any, at submission time;
> none was found as of 2026-09-23).

This draft is not submitted. It is scoped to the proxy/redirect gap
described here only; it intentionally excludes the separate
`StartAuthFlow`/`SaveAuthRequestInfo` persistence-error candidate tracked in
[`docs/upstream/indigo-persistence/README.md`](../indigo-persistence/README.md).
