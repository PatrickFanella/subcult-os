# AT Protocol kernel

Status: syntax foundation implemented on 2026-09-20; OAuth, Lexicon admission, DID resolution and publication are not implemented.

## Dependency boundary

The Go backend pins Indigo at `v0.0.0-20260903211445-41278964ec8e` and imports only `github.com/bluesky-social/indigo/atproto/syntax`. Indigo explicitly describes its APIs as unstable, so `backend/internal/atproto` converts upstream values into OS-owned plain-string structures. Other application modules must not import Indigo directly without extending this boundary and its drift tests.

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

Before OAuth work continues, choose and document:

1. public client metadata and callback URLs for web and native variants;
2. confidential-client signing-key storage and rotation, or an explicitly public-client design;
3. encrypted storage for DPoP/private session material;
4. resolver network policy, including DNS rebinding and private-network denial;
5. state, PKCE verifier, issuer, nonce and callback replay lifetimes;
6. explicit link/unlink UX and the rule that a DID link grants no workspace or publication authority.

Use the pinned Indigo OAuth package only after reviewing its exact API and transitive surface. Do not copy the old Subcults OAuth service while its license/provenance gate remains unresolved.

## Lexicon boundary

No `tv.subcult.*` Lexicon has been admitted or published. The old schemas remain blocked by field review and the Subcults rights/license gate. `T-SYNTAX` is therefore implemented, while `T-LEX` remains open. When a minimal schema is independently authored and approved, validate the same JSON cases with Indigo's Lexicon package and the official TypeScript `@atproto/lex` package; reject unknown private-location and operational fields at the public projection boundary.

## Verification

```bash
cd backend && go test ./internal/atproto -count=1
cd web && pnpm run test -- atprotoSyntaxConformance
```

These checks prove local cross-language syntax agreement for the checked-in corpus. They do not prove handle resolution, live OAuth interoperability, a PDS write, Lexicon compatibility or publication authority.
