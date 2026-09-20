# Development execution log

## 2026-09-20 — Pinned Indigo OAuth implementation review

The exact Indigo source commit behind the Go module pin was cloned into the ignored read-only dependency workspace and registered in `.blacktower/clonedeps.json`. Review was limited to its AT OAuth package; no dependency scripts were run and no upstream source was edited or copied into Subcult OS.

The implementation supplies PAR, PKCE, DPoP, nonce retry, callback issuer and token subject validation, public-only HTTP transports, refresh and revocation. OS remains responsible for encrypted durable storage, expiry and replay handling, local-person link intent, scope policy and stricter redirect/resolver policy. No OAuth route, external request, credential, repository scope or DID link was created.

## 2026-09-20 — AT-01 syntax foundation

The first AT-01 slice pins Indigo at `v0.0.0-20260903211445-41278964ec8e` and limits production imports to `atproto/syntax` behind the OS-owned `backend/internal/atproto` adapter. It parses normalized account identifiers and collection NSIDs, exact-record AT URIs and DID-authority strong references. Application packages receive plain strings rather than unstable Indigo types.

The web dev toolchain independently pins `@atproto/syntax` `0.7.6`. Go and TypeScript consume the same provenance-tagged JSON corpus reduced from current public protocol specifications. Focused Go and TypeScript conformance tests pass. No Subcults source, fixture or Lexicon was copied.

This is syntax qualification only. AT OAuth still requires a reviewed implementation of PKCE, PAR, DPoP/nonces, client metadata, issuer/resource discovery and `sub`/scope validation. No handle/DID resolution, network call, OAuth link, PDS write, Lexicon publication or repository permission exists. The missing Subcults license continues to block copying the legacy schemas or OAuth implementation; `T-LEX` remains open until a minimal schema is independently authored and approved.

## 2026-09-20 — IDENT-01

Subcult OS now owns the canonical account/session foundation. Signup creates an unverified email identity and one-time verification challenge; verification is required before login. Authentication lookup uses an HMAC of normalized email, while the address is encrypted with a deployment key. The existing `people.email` value remains a plaintext operational projection for current workspace workflows and is not used as the authentication lookup authority.

Sessions use 15-minute access credentials and rotating 30-day refresh families. Browser transport uses scoped HttpOnly cookies. Native auth endpoints are isolated under `/api/mobile/auth/*` and use separate access and refresh headers backed by secure storage; ordinary browser auth responses do not expose JavaScript-readable token headers. Refresh replay revokes the whole descendant family. Logout, logout-all, expiry, password recovery, email normalization conflicts, DID uniqueness, challenge expiry and account non-merging have database-backed coverage. Web and mobile signup now stop at a verification-required state; both clients have verification entry points, and the web client includes recovery request/completion surfaces. Neither client receives session credentials in a JSON DTO.

Migration 2 refuses populated prototype accounts before creating the canonical identity tables. Migration 3 refuses populated prototype sessions before removing that table. This repository has no authorized retained-account migration, legacy password reader or dual-session compatibility path. Any discovered retained database must be inventoried and backed up before a separately reviewed migration is written.

Verification passed for the focused identity and migration suites, the repository's configured lifecycle database tests, Go non-database tests/vet/build, shared contracts, web TypeScript/ESLint/tests/build, mobile TypeScript/tests, Compose configuration and aggregate `make verify` under noninteractive CI mode. pnpm still reports that the `esbuild@0.27.7` install script is ignored; no build-script approval or supply-chain policy was changed. The broader historical database suite reaches several pre-existing lifecycle assertions outside IDENT-01 that conflict with current end-of-night/public behavior; those failures are recorded as non-identity follow-up rather than weakened. No deployment, external email, real-user migration or PDS operation was performed.

A built disposable stack also passed a headed Chromium signup and email-verification journey with synthetic data, landing in the authenticated operator home. Response-header inspection confirmed that normal browser login emits two HttpOnly cookies and no access/refresh token headers, while the native login route emits the two native token headers. The local outbox row was inspected only to bridge the deliberately unsent development verification email. The browser, containers, test databases and disposable volume were removed afterward. Native-device and full browser recovery qualification remain open.

The next eligible unit is AT-01. Its implementation remains an OS-native rewrite: no Subcults source or Lexicon may be copied until the recorded rights/license gate is resolved.

## 2026-09-20 — DB-01 and INV-01

### DB-01 result

Subcult OS now embeds a gap-free ordered migration set. `schema.sql` is immutable version 1; later files use `backend/internal/app/migrations/NNNNNN_name.sql`. The runner applies migrations in one PostgreSQL transaction under a transaction-scoped advisory lock and records version, name, SHA-256 checksum and time in `schema_migrations`. It rejects changed applied migrations and any database whose ledger is ahead of the binary. Startup and the explicit `cmd/migrate` path use the same runner; the Make target no longer pipes `schema.sql` directly into `psql`.

Focused tests on the named disposable `subcult-os-db01-postgres` PostgreSQL 17 container passed for fresh/replay, concurrent runners, checksum tamper, forced SQL failure rollback, database-ahead rejection and the current create/publish/free-door/end-of-night and capacity/door lifecycle journeys. The focused migration suite also passed under Go's race detector. A logical dump of the populated disposable database restored into `db01_restore`; both source and restore reported schema version 1, two events, four people and two workspaces. The backend Dockerfile built both binaries, and `/app/migrate` replayed the populated schema successfully from the built image.

No external database was inspected, reset or migrated. No compatibility path was added because no retained real database has been identified. The test container and its temporary databases were removed after verification.

### INV-01 result

The clean Subcults checkout remained at `3cf88ec66dffa52160331ecf2e24aee29d66e741` and was not modified. The [file-level extraction manifest](subcults-extraction-manifest.md) classifies all scoped AT, identity, indexer, audience, signal, touring, Lexicon and selected migration candidates.

No package qualified for wholesale import. The roughly 9,045-line indexer is particularly coupled to Jetstream, PostgreSQL, Prometheus, OpenTelemetry and legacy scene/post/alliance/geo domains. Useful reuse is primarily privacy, session-family, consent, stream-recovery and publication-failure contracts and fixtures. The audited checkout had no root `LICENSE`, `COPYING` or `NOTICE` file, and scoped history includes owner aliases and Copilot bot identities; distributable copying therefore remains blocked on explicit rights/license review.

### Next eligible work

IDENT-01 and AT-01 are now eligible as separate units. Both must be OS-native rewrites with provenance-tagged fixtures; AT-01 does not include publication or PDS provisioning, and IDENT-01 does not infer account equality from email or DID.

## 2026-09-20 — BASE-01 and API-01

### BASE-01 result
Subcult OS baseline: branch `t3code/research-event-app-competitors`, HEAD `abf3f500364d2cce60879c449cb6711346cd675f`. Existing research and proposal files were preserved.

Subcults baseline: clean `fix/main-regression-recovery` at `3cf88ec66dffa52160331ecf2e24aee29d66e741`. Fresh focused checks passed for Go module integrity; `internal/atprotocol`, `internal/touring`, `internal/audience`, and `internal/signal`; nine canonical Lexicons; frontend i18n/lint/build; and mocked deploy recovery. Full Vitest, whole Go/race, PostGIS, browser, PDS/provider, restore and parity qualification were not run. The dated release status is stale for schema evidence: the checkout requires migration 47.

Local tool detail: the default Go shim was unconfigured; Go 1.26.6 was invoked from the installed pinned toolchain. The first `pnpm` on PATH was a hanging wrapper; `/usr/bin/pnpm` selected the package-manager version pinned by each project.

### API-01 defect and change
Before API-01, `GET /api/public/events` used an explicit allowlist, but `GET /api/public/events/{slug}` embedded internal `eventDTO`. Anonymous detail responses therefore included the private workspace identifier, ticket allocation, raw reservation/check-in aggregates and staffing-count keys. The route did not expose ticket-holder identity, staffing items, notes, settlements or archives.

API-01 now:
- serializes detail through a standalone allowlisted Go DTO;
- omits workspace, raw attendance/capacity and staffing aggregates;
- restores detail/list image parity by loading `image_url`;
- defines a narrow web and mobile `PublicEventDTO` rather than inheriting/copying `EventDTO`;
- makes the contract checker reject forbidden public client properties;
- adds Go serialization sentinel/absence coverage and anonymous endpoint regression coverage;
- updates stale discovery test fixtures to use the free-reservation contract and the fixture's actual public base URL.

No database schema, endpoint URL, authentication behavior, reservation workflow, AT record or deployed service changed.

### Verification passed
- `/usr/bin/node scripts/check-contracts.mjs`
- Go 1.26.6 non-DB `go test ./... -count=1`
- Go 1.26.6 `go vet ./...`
- Go 1.26.6 backend build
- disposable PostgreSQL selected lifecycle, discovery, public serializer and migration tests
- web TypeScript check, ESLint, 76 tests and production build
- mobile TypeScript check and 27 tests, invoked directly from installed binaries because the dependency-policy check blocks the package script
- `docker compose -p subcult-os config --quiet`

The disposable `subcult-os-api01-postgres` container used only test credentials and an isolated database. It was removed after verification.

### Remaining gate
The aggregate `make verify` is not green: mobile dependency installation rejects an unapproved `esbuild` build script. No build approval or supply-chain policy was changed. Full browser/device behavior and a running application stack were outside API-01. API-01 is implemented and focused-test-verified, not runtime- or production-verified.

### Next eligible work
DB-01 and INV-01 can proceed independently. IDENT-01 and AT-01 remain blocked until ordered OS migrations and the selective extraction audit establish their boundaries.

## 2026-09-20 — Architecture direction changed

The product owner selected Subcult OS as the receiving Subcult.tv repository. The older Subcults application will remain read-only source material while useful capabilities are evaluated and extracted selectively. The prior permanent cross-repository bridge, separate-account and separate-database proposal is superseded by ADR 0005.

Planning artifacts now require one Go API, one identity authority and one PostgreSQL database in OS, with modular ownership and a Go/TypeScript AT conformance boundary. No source, migration, test-suite or Git-history merge was performed. Subcults was not modified.
