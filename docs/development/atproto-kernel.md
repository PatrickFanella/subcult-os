# AT Protocol kernel

Status: syntax, encrypted identity-only OAuth persistence, confidential-client documents, authenticated start/callback, link/list/unlink UI and durable revocation processing implemented locally. Real provider interoperability and deployed worker qualification remain open; Lexicon admission and publication are not implemented.

## Dependency boundary

The Go backend pins Indigo at `v0.0.0-20260903211445-41278964ec8e`. Production imports are confined to the OS-owned `backend/internal/atproto` boundary: `atproto/syntax` supplies validated syntax values and `atproto/auth/oauth` supplies the exact persistence interface and payload types. Indigo explicitly describes its APIs as unstable, so syntax values leave this package only as OS-owned plain-string structures. Other application modules must not import Indigo directly without extending this boundary and its drift tests.

The independent TypeScript path pins `@atproto/syntax` at `0.7.6`. Both implementations consume `contracts/atproto-syntax.fixtures.json`. The fixture cases were reduced from the current public [handle](https://atproto.com/specs/handle), [NSID](https://atproto.com/specs/nsid), [AT URI](https://atproto.com/specs/at-uri-scheme), and [record-key](https://atproto.com/specs/record-key) specifications; no Subcults source or fixtures were copied.

Indigo and `@atproto/syntax` are both offered under the MIT or Apache-2.0 licenses. Pinning Indigo also advances the repository's compatible `golang.org/x/crypto`, `x/net`, `x/sync`, `x/sys`, and `x/text` modules plus `klauspost/cpuid`; those transitive changes are part of this reviewed dependency unit, not an unrelated bulk upgrade.

The first kernel supports:

- DID or handle syntax validation, with lowercase handle normalization;
- collection NSID validation and normalization;
- restricted AT URI parsing that requires an exact collection record;
- strong references that require a DID-authority record URI and syntactically valid CID.

Syntax validation is not resolution or authority proof. A handle must be resolved and confirmed against the DID document in both directions before it can identify an account. A strong reference does not prove that a record exists, is current or is controlled by a local user.

## OAuth boundary

Do not implement OAuth as a conventional authorization-code shortcut. The current AT Protocol [OAuth specification](https://atproto.com/specs/oauth) requires PKCE, pushed authorization requests, DPoP with server nonces, automated client metadata, issuer/resource-server discovery, returned-scope checks and mandatory `sub` validation. Identity-only linking still requires the `atproto` scope but must request no repository permissions by default.

The product owner first selected `https://subcults.subcult.tv` as the production web origin and replaced the legacy service there on 2026-09-30. On 2026-10-02 the owner moved the canonical origin to `https://os.subcult.tv`. The legacy host now redirects pages to it and still proxies `/api`. The AT OAuth client identity moves with the origin, so the legacy client ID is no longer preserved. AT linking was still disabled, and the legacy database had no OAuth links, sessions or requests. The production client uses these URLs:

- client ID and metadata: `https://os.subcult.tv/api/v1/auth/atproto/client-metadata`
- callback: `https://os.subcult.tv/api/v1/auth/atproto/callback`
- public JWKS: `https://os.subcult.tv/api/v1/auth/atproto/jwks`

The replacement is a confidential web client using `private_key_jwt`, ES256 and a P-256 signing key supplied only through the deployment secret store. Its metadata requests only `atproto`; the broader repository scopes advertised by the legacy service are deliberately not inherited. `GET` metadata/JWKS, authenticated `POST` start, and public state-bound `GET` callback endpoints exist behind `ATPROTO_OAUTH_ENABLED`. That flag remains false by default until link UI and bounded live interoperability are qualified.

Before the OAuth flow is enabled, complete and document:

1. signing-key import/rotation and rollback behavior for the existing `subcults-1` key ID;
2. a bounded live interoperability test covering PAR, callback, refresh and revocation without repository scopes;
3. remote provider token revocation that cannot prevent immediate local unlink;
4. whether existing legacy grants can be revoked or should simply require fresh authorization after cutover.

Use the pinned Indigo OAuth package only after reviewing its exact API and transitive surface. Do not copy the old Subcults OAuth service while its license/provenance gate remains unresolved.

### Pinned implementation review

The source for the exact Indigo commit is registered as a read-only inspection dependency under `.blacktower/clonedeps/repos/bluesky-social__indigo/`. The 2026-09-20 review found that its client already implements PAR, PKCE, DPoP proof generation and nonce retries, callback `iss` checks, token `sub` checks, public-only HTTP transports and session refresh/revocation. Subcult OS should adapt those behaviors instead of reimplementing them.

The application still owns security properties that Indigo intentionally delegates to its caller:

- durable, concurrent `ClientAuthStore` persistence and expired-request collection;
- encryption at rest for PKCE verifiers, DPoP private keys and access/refresh tokens;
- one-time callback consumption and a local-person link intent that cannot merge accounts;
- returned-scope policy and explicit local link status;
- stricter redirect and outbound-resolution policy around metadata discovery;
- atomic persistence when token or DPoP nonce rotation updates a session.

Migration 4 and `backend/internal/atproto.OAuthStore` now implement that local persistence layer. It binds a start request to an already-authenticated local person, HMAC-indexes state, encrypts the full request/session payload with a protocol-specific derived key, atomically claims each callback once, expires requests after ten minutes, accepts only the single `atproto` identity scope, atomically creates the DID link and session, rejects cross-account DID claims, records the link audit event, and refuses to refresh a revoked link. Expired request cleanup is explicit and safe to schedule. The same deployment root key is used through a separate derivation domain; no protocol secret shares ciphertext or lookup keys with email identity data.

Importing the pinned Indigo OAuth package to satisfy its store interface adds its current JWT, identity, CID/multibase and Prometheus-related transitive modules to the backend build. This is the reviewed cost of compiling against the actual unstable interface rather than maintaining a lookalike local contract.

The application can publish the selected client metadata and public JWKS without exposing the private key. Configuration validation requires all three endpoints to share the HTTPS web origin and rejects invalid booleans, missing keys and incompatible signing curves. The start route accepts only a syntactically valid handle or DID from an authenticated local person, binds that person into encrypted request state, and returns a validated HTTPS authorization URL. The callback atomically claims state, delegates PAR/PKCE/DPoP/token and subject checks to pinned Indigo, persists only exact `atproto` sessions, and redirects to a fixed same-origin result without reflecting provider error text.

Outbound OAuth, handle and DID discovery uses public-IP-only dial controls with environment proxy use and HTTP redirects disabled. This closes the proxy bypass and redirect rebinding gaps left open by the SDK defaults. The adapter fails closed if a future pinned Indigo release changes the default identity-directory shape so the hardening can no longer be applied. Live interoperability remains open; these local tests do not prove that a real PDS accepts the client.

Authenticated users can list every active DID link and begin a new identity-only link from the operator home. The UI states that a DID grants neither workspace membership nor publication authority and requires a second confirmation to unlink. Local unlink atomically marks that person's active DID revoked, transfers encrypted credentials to the revocation outbox, deletes active sessions and records `atproto_did_unlinked`. It succeeds without network availability. The bounded worker separately revokes both tokens through Indigo's confidential-client/DPoP flow with public-only, no-proxy, no-redirect transport.

Migration 5 provides two-minute leases, fencing tokens, exponential one-minute-to-64-minute retry delays and a maximum of eight attempts. A timed-out operation may already have succeeded; retry is expected. A late refresh replaces encrypted revocation material and fences the old acknowledgement without restoring local access. An expired lease is reclaimable, including terminal handling of a crash on attempt eight. Unsupported/invalid/exhausted work is quarantined and secrets removed; seven-day retention expiry also purges credentials without claiming remote success. Relinking after unlink requires fresh authorization.

`atproto-revoke -status` reports aggregate counts only. The one-shot command processes at most ten jobs by default; `-limit` accepts 1–100 and `-watch` repeats bounded batches every 30 seconds. The optional `atproto-workers` Compose profile runs it without an additional API/database. New-link enablement is independent of draining old jobs. The deployment must supervise the worker, monitor quarantined/backlogged counts and retain the correct protection/signing keys. A rollback must stop the worker before using a pre-migration-5 binary and must preserve the outbox; losing queued credentials makes remote revocation unprovable. Physical deployment/provider qualification remains #10 and the cutover gate.

`make generate-atproto-key` emits an Indigo-compatible multibase P-256 client key. Its stdout is secret material and should be piped directly to the deployment secret manager, never stored in committed configuration or captured in logs.

## Lexicon boundary

A minimal `tv.subcult.*` chain (`tv.subcult.profile`, `tv.subcult.place`,
`tv.subcult.event.occurrence`) was independently authored and admitted on
2026-09-23 by [ADR 0007](../adr/0007-minimal-lexicon-admission.md)
(decision A10 in [`decisions.md`](decisions.md)). Admission covers the schema
contract and validators only; no `tv.subcult.*` record is published until the
publication decisions D7 through D10 are accepted and implemented. The old
Subcults schemas remain blocked by the rights/license gate and were not
consulted; the admitted chain reviewed the MIT-licensed
`community.lexicon.calendar.*`/`community.lexicon.location.*` schemas as
prior art. `T-SYNTAX` is implemented; `T-LEX` is admitted as a contract, with
no runtime consumer yet.

Field allowlists, bounds, and public time/location semantics are specified in
[`lexicon-contract.md`](lexicon-contract.md). The same JSON corpus
(`contracts/atproto-lexicon.fixtures.json`) is validated by the pinned
Indigo `atproto/lexicon` package (`backend/internal/atproto/lexicon.go` and
`lexicon_conformance_test.go`) and the official TypeScript `@atproto/lexicon`
package (`web/src/atprotoLexiconConformance.test.ts`). Both validators
additionally reject unknown private-location and operational fields at the
public projection boundary, because official Lexicon validation intentionally
allows additive unknown fields by protocol design and does not do this by
itself; see `lexicon-contract.md` for why that check lives outside the wire
schema.

## Verification

```bash
cd backend && go test ./internal/atproto -count=1
cd backend && TEST_DATABASE_URL=postgres://... go test ./internal/atproto -run TestOAuthStore -count=1
cd backend && TEST_DATABASE_URL=postgres://... go test ./internal/app -run TestIdentityATProtoStart -count=1
cd backend && TEST_DATABASE_URL=postgres://... go test ./internal/app -run TestIdentityATProtoLinkListAndUnlink -count=1
cd web && pnpm run test -- atprotoSyntaxConformance
cd backend && go test ./internal/atproto -run TestSharedLexiconConformanceFixture -count=1
cd web && pnpm run test -- atprotoLexiconConformance
```

These checks prove local cross-language syntax agreement, authenticated person binding, hardened redirect policy, local unlink behavior and, with PostgreSQL configured, the encrypted store's expiry, replay, scope, rotation, audit and non-merging invariants. The Lexicon fixture checks additionally prove that the admitted `tv.subcult.*` chain's field allowlist, bounds and public-time strictness agree between the pinned Indigo validator and the official TypeScript validator. They do not prove live handle/DID resolution, OAuth interoperability, provider token revocation, a PDS write, or publication authority, and they do not prove publication authority.
