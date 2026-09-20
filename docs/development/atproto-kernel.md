# AT Protocol kernel

Status: syntax foundation, encrypted identity-only OAuth persistence, and configurable confidential-client metadata/JWKS endpoints implemented on 2026-09-20; authorization/callback flows, Lexicon admission, DID resolution and publication are not implemented.

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

The product owner selected `https://subcults.subcult.tv` as the production web origin and authorized eventual replacement of the legacy service there. Subcult OS preserves the legacy public OAuth identity at these exact URLs:

- client ID and metadata: `https://subcults.subcult.tv/api/v1/auth/atproto/client-metadata`
- callback: `https://subcults.subcult.tv/api/v1/auth/atproto/callback`
- public JWKS: `https://subcults.subcult.tv/api/v1/auth/atproto/jwks`

The replacement is a confidential web client using `private_key_jwt`, ES256 and a P-256 signing key supplied only through the deployment secret store. Its metadata requests only `atproto`; the broader repository scopes advertised by the legacy service are deliberately not inherited. `GET` metadata and JWKS endpoints exist behind `ATPROTO_OAUTH_ENABLED`, but that flag must remain false until start/callback behavior is implemented and qualified. The callback URL is reserved, not yet served.

Before the OAuth flow is enabled, complete and document:

1. signing-key import/rotation and rollback behavior for the existing `subcults-1` key ID;
2. resolver network policy, including DNS rebinding and private-network denial;
3. explicit link/unlink UX and the rule that a DID link grants no workspace or publication authority;
4. a bounded live interoperability test covering PAR, callback, refresh and revocation without repository scopes;
5. whether existing legacy grants can be revoked or should simply require fresh authorization after cutover.

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

The application can now publish the selected client metadata and public JWKS without exposing the private key. Configuration validation requires all three endpoints to share the HTTPS web origin and rejects invalid booleans, missing keys and incompatible signing curves. Production flow work is no longer blocked on URL or client-type selection, but start/callback handling, outbound resolver policy and live interoperability remain open.

## Lexicon boundary

No `tv.subcult.*` Lexicon has been admitted or published. The old schemas remain blocked by field review and the Subcults rights/license gate. `T-SYNTAX` is therefore implemented, while `T-LEX` remains open. When a minimal schema is independently authored and approved, validate the same JSON cases with Indigo's Lexicon package and the official TypeScript `@atproto/lex` package; reject unknown private-location and operational fields at the public projection boundary.

## Verification

```bash
cd backend && go test ./internal/atproto -count=1
cd backend && TEST_DATABASE_URL=postgres://... go test ./internal/atproto -run TestOAuthStore -count=1
cd web && pnpm run test -- atprotoSyntaxConformance
```

These checks prove local cross-language syntax agreement for the checked-in corpus and, with PostgreSQL configured, the encrypted store's expiry, replay, scope, rotation, audit and non-merging invariants. They do not prove handle resolution, live OAuth interoperability, a PDS write, Lexicon compatibility or publication authority.
