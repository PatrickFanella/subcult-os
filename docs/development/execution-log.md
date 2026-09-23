# Development execution log

## 2026-09-22 — #113 event editor read recovery

The report loader treats only a 404 as an absent report. Workspace authority failures are explicit rather than silently hiding owner tools. Both reads have independent error state and scoped retries; neither retry reloads the event form. Authority-dependent controls remain unavailable until the workspace read succeeds. An already known report snapshot is retained during a refresh, with any refresh error still visible.

Local `make verify` passed, including status/transport regression cases. Browser checks against a disposable synthetic API confirmed an authority outage, recovery of owner tools while an unsaved title stayed intact, and report outage/retry recovery. The fixture recorded no event refetch during authority recovery and only the report endpoint during report retry. These are controlled UI checks, not production or real-backend qualification. No live email or deployment was performed.

## 2026-09-22 — #111 workspace loading boundaries

Workspace selection now falls back only after an explicit 403 or 404. Server, authentication and transport failures remain errors, and fallback membership is loaded from the authorized workspace endpoint rather than fabricated from the account summary. Event and archive outages no longer become empty lists. The dashboard is committed only after its required overview loads, with a retry action for initial failures.

Local `make verify` passed. Fifteen loader cases cover selection errors, authorized fallback, overview outages and private-panel denial. A real browser against a disposable synthetic API confirmed event/archive error states, successful retry into the chosen workspace, no fallback on a selection outage, and an explicit notice when access denial permits fallback. This verifies controlled UI behavior, not production or a real backend lifecycle. Hosted CI remains a separate PR gate. No schema, live mail or deployment changes were made.

## 2026-09-22 — #105 signed email feedback

Added a disabled-by-default Resend webhook, raw-body signature verification and duplicate-safe minimal receipts. Tests use the independent published Svix vector and locally signed synthetic requests. Early receipts correlate after acknowledgement; adverse outcomes cannot be cleared by late delivery events. Suppression uses only our stored recipient, and workers exclude both durable suppression and unprocessed adverse receipts. Generic provider failures do not suppress a recipient.

The owner deferred live Resend setup. No account, DNS, live delivery or production changes are included. Full local and hosted verification precede the batch merge checkpoint. Minimal receipt retention and audited unsuppression remain follow-ups; no cleanup silently enables sending.

Hosted worker runs 9254/9255 exposed a test-fixture race: the version-five fixture executed the database-wide extension creation outside the migration advisory lock while AT integration tests ran in another package. The fixture now uses the same lock and one transaction. Production migration SQL and assertions are unchanged. The failed hosted results remain part of the record; verification is repeated against a fresh disposable database as well as the existing synthetic test database.

## 2026-09-22 — #104 durable transactional delivery

Added an opt-in delivery command and additive version-6 ledger. Historical and disabled-mode inserts remain held; enabled inserts freeze the sender and reply-to. Leased claims use row locking and fenced acknowledgements. Retries retain one provider idempotency key, stop after eight attempts or 23 hours, and respect identity-challenge expiry/consumption. Terminal outcomes clear message bodies. Aggregate status does not contact Resend.

Focused disposable-PostgreSQL tests passed under the race detector, including concurrent claims, stale acknowledgements, crash recovery, terminal failures and identity deadlines. The full verification and database gates are rerun before publication. Live Resend setup is deferred at the owner's request; no credentials, DNS, live mail or production migration is part of this change. Approved reply-to and controlled test recipient: `info@subcult.tv`.

## 2026-09-20 — Five-issue delivery batch after merge checkpoint

### #103 — Resend provider adapter

Added one standard-library Go HTTPS adapter with stable message idempotency, bounded responses/timeouts, no redirects/proxy and typed redacted errors. Synthetic transport cases cover successful acceptance, malformed/oversized responses, credential rejection, both idempotency conflict classes, throttling, provider failure and invalid input. See `docs/runbooks/transactional-email.md` for sourced contracts and setup boundaries. This does not activate a worker or live sending.

The owner approved batches of five to ten issues, followed by combined verification and merging before further feature work. Baseline: `966a111` on main, verified by hosted runs 8929/8930 and local database/race tests. Current batch: #101 reservation inventory, #102 protected-session expiry, #103 Resend adapter, #104 durable delivery, #105 signed delivery feedback/suppression. Each receives a stacked PR. Parent roadmap issues remain open where live-provider, device or deployment acceptance is still missing.

### #101 — Reservation inventory

Free RSVP now reloads authoritative event inventory instead of leaving pre-reservation counts or decrementing locally. A failed refresh preserves the issued ticket and labels availability unknown. Unit checks cover confirmed, failed-refresh and rejected-reservation cases. `make verify` passed (123 web and 27 mobile tests). A real Chromium browser against the disposable PostgreSQL/API/Vite runtime reserved the last ticket: both availability indicators changed to sold out while the confirmation and ticket link remained visible. This was synthetic local data, not delivery or production qualification.

### #102 — Protected operator session expiry

Workspace/event namespaces now authenticate before resource authorization: missing or expired sessions return 401, while authenticated foreign-workspace denial stays 403. The request-scoped identity avoids redundant authentication and preserves the server-owned logging route template. Anonymous media requests now require authentication before exposing adapter availability; the media test covers both that boundary and authenticated 503 behavior.

The complete database suite passed, including expired access, refresh, denied mutation and cross-workspace regression. Web tests prove one refresh/retry with the unchanged mutation payload and no retry for real 403. Chromium loaded an event after controlled access expiry and saved an allowed location edit after a second expiry; session generations rotated. A disallowed published-title change remained 409 and was not falsely called successful. No natural expiry soak, native device or production behavior is claimed.

## 2026-09-20 — ARCH-01 protocol failure isolation

Reused the complete local-event lifecycle assertion for a second scenario with an enabled AT link flow that returns a synthetic provider error. The AT request returns 502, then local create/publish/free-reserve/duplicate-check-in/closeout succeeds with one settlement/archive. The provider is called exactly once and no DID link is created. Focused and complete database gates plus `make verify` pass. This tests the existing application seam, not a live network partition or offline mobile mode.

The architecture document now distinguishes implemented account/session and AT seams from design-only cultural validator/projection/publication contracts. It removes obsolete prototype identity/status claims and explicitly defers unused interface packages until a real caller exists. No public schema, record mapping or PDS publication was added. Hosted baseline runs 8911/8912 passed at `226fdf4`; later PostgreSQL service qualification is still queued.

## 2026-09-20 — Hosted database verification gate

With the full local database suite restored, Gitea now declares a job-scoped PostgreSQL 17 service and runs the complete `make test-db` after `make verify`. The service has disposable test-only credentials, readiness checks, no host port and no retained volume. Test cases use private schemas. No application database, runner settings or deployment service is targeted.

Actionlint and local `make verify` pass; the immediately preceding expanded local database gate passes on isolated PostgreSQL 18. Hosted service-network and PostgreSQL 17 execution remain pending until this revision runs. Parent run 8911 has successfully provisioned Node/pnpm/Go and reached dependency/build verification, clearing the earlier missing-pnpm step; it is not yet a completed green run.

## 2026-09-20 — QUAL-BASE full database gate restored

Resolved all seven isolated failures from #95. Two product defects were confirmed: malformed event IDs reached PostgreSQL UUID conversion and returned 500, and publication preserved the database slug but incorrectly returned/audited the ID-derived fallback. Both normal and locking event loaders now reject malformed UUIDs as not found before querying; publication uses `RETURNING public_slug` so storage, response and audit agree. Regression tests retain the seeded-slug scenario and explicitly compare all three values.

The remaining setup corrections preserve behavior assertions: closed/private public applications are explicitly denied, then synthetic private rows perturb source state to prove archive immutability and roster filtering; paid-event privacy checks assert free-RSVP rejection and use a paid ticket fixture; member staffing authorization is checked through the supported PATCH method. Public-event privacy is observed before close and closed lookup must return 404. No public admission/payment policy was relaxed and no test was removed or skipped.

The full app database suite passed twice in one process: 336 passing test/subtest events, zero failures/skips. The expanded `make test-db` then passed for the complete app and AT packages; it no longer hides lifecycle cases behind a name filter. `make verify` passed (120 web, 27 mobile, configured Go/vet/build/contracts/Compose). Original failed ledgers remain unchanged; successful local ledgers are `/tmp/subcult-lifecycle-contracts-full.jsonl` and `/tmp/subcult-test-db-expanded.log`.

A separate actual Chromium rehearsal created a workspace and event, published, reserved a free ticket, opened ticket lookup, checked in through Door, and generated End of Night with reserved/check-in/no-show counts 1/1/0 and zero-dollar settlement. This rehearsal used the earlier running `07b83ea` API binary and current web sources (unchanged since `c7817d2`), not a rebuilt/deployed claim for these backend fixes. Broad API coverage qualifies additional workspace/privacy/staffing/template/role/reminder boundaries; physical-device and full browser coverage of those panels remain open under #5. The date field required DOM input-event entry because the preview typing helper misfocused the native date input; this is not native date-picker qualification.

## 2026-09-20 — Hosted CI toolchain setup

Run 8907/job 16364 failed immediately at `make deps-web`: `pnpm: command not found`. The workflow previously assumed Go/pnpm existed in the shared runner image. It now provisions Node 24 (matching the frontend image major), pnpm 10.33.0 (the web package pin), and locally qualified Go 1.26.6, reports versions, and retains the unchanged `make verify` gate. Added actions use verified upstream commit pins; no runner or global host configuration changed. Dependency caching is disabled for Go until the Gitea cache path is independently qualified. A 20-minute job bound and explicit CI environment prevent unbounded setup and interactive dependency prompts.

Local `make verify` and actionlint pass. Actionlint initially hit an unset mise shim; explicitly selecting its installed Go 1.25.13 tool environment passed without changing global defaults. Hosted execution must be read back after publication; this entry does not claim hosted success or full database qualification.

## 2026-09-20 — QUAL-BASE test isolation and retained failures

The broad app database run at `c7817d2` produced 151 passing / 11 failing test events. Unlike the identity tests, lifecycle fixtures shared the public schema and persisted data across runs. Each lifecycle fixture now owns a disposable schema through the existing migration helper. Secondary cross-workspace actors explicitly share the first application's database/session authority; they remain valid authenticated users, so access-denial checks do not degrade into invalid-session tests. The events-table migration check is also schema-local.

Isolation removes four failures (notification/reminder static keys and discovery counts): the same broad suite reaches 155 passing / seven failing events. Issue #95 retains the seven archive, paid-privacy, staffing, slug and private-role cases for behavioral reconciliation. Original local JSONL ledgers remain at `/tmp/subcult-lifecycle-baseline.jsonl` and `/tmp/subcult-lifecycle-isolated.jsonl`; no failing test was deleted or skipped. A new fixture test proves private schemas and valid authenticated cross-workspace denial. That test and the four repaired cases pass twice in one process. `make verify` passes; the full database suite remains red and the maintained narrow target is not presented as full coverage.

Hosted runs now finish but fail before tests: run 8907/job 16364 at `c7817d2` reports `pnpm: command not found`. CI toolchain setup is the next independent fix; local verification is not hosted success. No runner service was modified.

## 2026-09-20 — AUTH-RETURN same-origin navigation

Issue #93 records a browser-confirmed URL-normalization gap: the previous `next` check allowed slash/backslash and slash/control/slash inputs that Chromium resolves off-site. The new pure return-path validator rejects those forms, checks the parsed origin, and rejects network-path results after dot-segment normalization. It preserves valid invitation/workspace paths and query/fragment data. No cookie disclosure is claimed; this addresses post-authentication off-site navigation.

Seventeen hostile/valid-path cases pass. Actual browser sign-in with the malicious slash/backslash query remains on the local application origin and lands at the operator home. `make verify` passes (120 web and 27 mobile tests plus the full configured Go/build/contracts/Compose checks). This is a local fix; no production deployment occurred.

## 2026-09-20 — IDENT-02 web recovery and challenge qualification

The real Chromium/Vite Strict Mode journey reproduced two verification POSTs on one page load: one 200 and one rejected 401 replay. Verification now requires an explicit form submission, with an immediate in-flight guard; mounting the page does not consume the challenge. Successful verification replaces the token URL. Recovery success clears its query token and password field and removes the completed form. Recovery inputs have accessible names and result/error messages expose status/alert semantics.

On the disposable Unix-socket PostgreSQL 18 database, browser signup created no session before confirmation; one confirmation created exactly one session and reached the operator home. Recovery through the browser invalidated the existing session, rejected the old password, and accepted the new password. With only the synthetic account's access expiration advanced in the database, a reload refreshed the cookie session, advanced its generation from 0 to 1 and retained the authenticated workspace. This is controlled-expiration evidence, not a 15-minute natural soak. Development outbox reads bridged unsent emails; no external email was delivered. Tokens/passwords and account rows are not included in this record.

`make verify` passed (103 web tests, 27 mobile tests, Go/vet/build/contracts/Compose); `make test-db` passed including replay descendant revocation, revoke-one/all, recovery and account non-merging. Handler tests cover explicit submission, concurrent-submit suppression and missing tokens; actual browser behavior was checked separately. Issue #6 stays open: physical-device secure storage, restarts/deep links and native logout require real-device evidence. Hosted CI, production email and production deployment are not claimed.

## 2026-09-20 — UP-STATE reduced upstream reproduction

Prepared an original no-network test against Indigo's actual `StartAuthFlow`, rather than only a fake application runner. The pinned/current upstream commit ignores a failing store and returns nil error; the test reproduces that failure. A two-line error-propagation patch applied to a disposable source export makes the complete OAuth race suite pass. The inspected read-only dependency clone and application module pin remain unchanged. Current README contribution guidance requests issue discussion before an upstream PR; the package includes a submission draft and AI-assistance disclosure, but no external maintainer was contacted.

## 2026-09-20 — Request log privacy follow-up

Issue #89 records a concrete leak observed during lifecycle testing: the shared request logger wrote invitation/ticket values and linked DIDs from raw URL paths. It now writes only server-owned route templates, method, status and duration; unmatched, method-mismatch and pre-routing-denied requests use a constant marker. Query strings and path values are never a fallback. Focused race tests capture actual log output across those cases, and `make verify` passes. This changes future application logs only; production rollout and historic log retention are separate work.

## 2026-09-20 — AT-REVOKE worker and provider adapter

Added bounded leased processing, acknowledgement fencing, exponential retries, terminal quarantine and credential erasure. The SDK-backed adapter revokes access and refresh tokens with confidential-client assertions and DPoP nonce handling through the hardened public-only transport. Raw provider errors never enter the queue or command output. An opt-in worker command supports status, one-shot and watch modes using the same API image/database; disabling new OAuth links does not prevent draining existing work.

`make verify`, `make test-db`, the complete AT package race suite, the optional Compose profile render and a real CLI watch/SIGTERM smoke passed. Tests cover transient backoff, terminal exhaustion, unsupported providers, malformed encrypted payloads, retention expiry, crash recovery including the eighth attempt, concurrent workers and late-rotation fencing. The CLI produced aggregate counts with new links disabled. Docker became inactive after the earlier PostgreSQL 17 storage checks; worker tests used a signature-verified standalone PostgreSQL 18.6 package on a private Unix socket with no TCP listener. No shared daemon was started. The image was not built or deployed in that environment; real provider acceptance and production worker qualification remain open.

## 2026-09-20 — AT-REVOKE durable storage slice

Migration 5 adds an encrypted revocation outbox. Local unlink atomically transfers every active session payload to the outbox, revokes the DID and removes active sessions. DID-scoped transaction locks serialize unlink and session persistence. A late refresh cannot reactivate the link: it updates pending encrypted revocation material and invalidates the older worker lease. Active-session reads also require a current active DID link.

Disposable PostgreSQL race tests prove failed enqueue rolls back unlink, secrets are not recognizable plaintext, late rotation fences stale work, and concurrent unlink/refresh leaves no active session. This slice intentionally does not execute network revocation; the next stacked PR adds bounded processing, provider handling and operator-visible status. Production OAuth remains off pending the real-provider gate.

## 2026-09-20 — BASE-01 reconciliation and stacked review

The main checkout moved to `Work/Subcult/subcult-os`; `git worktree repair` corrected this worktree's stale lowercase pointer without changing commits or user files. Remote main remains `abf3f50`; twelve local commits through `06816d2` contain the research and platform foundations. Research is separated at `13f88b7` into the first review branch, with the platform branch stacked above it. The Gitea roadmap now tracks the remaining work and supersedes the old statement that no hosted issues exist.

Reconciled implemented migration/identity decisions and stale AT HTTP/UI and aggregate-test statuses. `make verify` passed using the installed Go 1.26.6 toolchain with system pnpm on PATH (99 web tests, 27 mobile tests, Go tests/vet/build and Compose validation). Two earlier environment-only attempts failed because Go was absent from PATH or unset in mise; no dependency or package-script policy was weakened. All three documentation validators pass. Live-provider, physical-device and cutover gates remain open; no deployment is implied by PR publication.

## 2026-09-20 — Live legacy deployment reconciliation

Read-only host inspection corrected the historical deployment assumption. The NUC has no running Subcults containers or listeners on 3024/3025. Active Almaz Caddy instead routes Subcults API/health traffic to Dozor `10.0.0.57:3025` and frontend traffic to `10.0.0.57:3024`. Dozor runs the six-service `subcults` Compose project from `/srv/containers/subcults`; all roles were running with zero restarts, and exact current image IDs are recorded in the cutover runbook.

The live PostgreSQL database is at migration 46 with `dirty=false`. Row-count-only queries found zero users, events, profiles, OAuth links, OAuth sessions and OAuth requests. No credential values or personal rows were read. This evidence removes the need to design a retained-user migration for the currently deployed database, but it does not authorize deletion and must be reconfirmed at cutover. The unprivileged account could not inspect the protected backup directory, so a fresh backup plus isolated restore remains mandatory. No service, image, proxy, database or configuration was changed.

## 2026-09-20 — AT identity link lifecycle UI

Authenticated users can now list active AT Protocol links, start an identity-only authorization from the operator home, and unlink through a two-step confirmation. The UI says explicitly that a DID proves account control but grants no workspace membership or publishing authority. Callback result copy is fixed locally and never reflects provider text. Disabled installations receive no panel because the capability routes remain behind `ATPROTO_OAUTH_ENABLED`.

Local unlink transactionally marks only the current person's DID revoked, deletes every matching encrypted OAuth session, and records an `atproto_did_unlinked` audit event. This fails closed across accounts and does not depend on provider availability. It does not yet revoke tokens at the provider, so the production flag remains off until a bounded live revocation design is qualified. `make generate-atproto-key` now emits the required multibase P-256 secret for direct secret-manager capture.

A named disposable PostgreSQL 17 container passed the link-list/unlink database tests and focused race runs, then was removed. An independently named disposable Compose stack passed a headed Chromium journey for signup, verification, login, panel rendering, callback-success copy, invalid-identifier feedback, a synthetic linked-state fixture, two-step unlink, and UI/DB audit readback. The synthetic DB fixture qualified presentation and local unlink only; no external resolver, PDS, OAuth grant or provider revocation was exercised. The browser, containers, network and disposable volume were removed afterward.

## 2026-09-20 — AT OAuth start and callback boundary

Subcult OS now exposes an authenticated `POST /api/v1/auth/atproto/start` and state-bound `GET /api/v1/auth/atproto/callback` when AT OAuth is explicitly enabled. Start accepts only a normalized handle or DID and binds the already-authenticated local person into the encrypted request. Callback delegates PAR, PKCE, DPoP, issuer, subject and token processing to the pinned Indigo client, while the OS store enforces one-time state, exact identity-only scope and non-merging DID ownership. Browser completion uses a fixed `/workspace?atproto=` landing and does not reflect untrusted provider error descriptions.

The adapter discovered that pinned Indigo ignores the error returned by `SaveAuthRequestInfo` in `StartAuthFlow`. An isolated capture wrapper now fails closed if persistence fails or never occurs; the regression is recorded as an upstream candidate. The adapter also replaces SDK HTTP defaults with public-IP-only, no-proxy, no-redirect clients across OAuth and identity discovery to close environment-proxy and redirect rebinding gaps.

Non-database tests and focused race tests passed. A named disposable PostgreSQL 17 container passed `make test-db`, including authenticated-person binding, plus race-enabled OAuth store tests; that container was removed. An existing local Compose database rejected the documented default password, so it was neither reset nor inspected and was returned to its prior stopped state. No external resolver, PDS, OAuth provider, credential, grant, DNS, proxy or deployed service was touched. The flow remains disabled by default pending UI and bounded live interoperability.

## 2026-09-20 — Production origin and OAuth client identity

The product owner selected `subcults.subcult.tv` as the replacement origin and authorized eventual controlled replacement of the legacy Subcults application. A read-only public inventory confirmed that the hostname currently serves the legacy app and exposes client metadata, JWKS and callback routes under `/api/v1/auth/atproto/`. The public metadata describes a confidential ES256 client with DPoP and broader repository scopes; no private secrets were read or recorded.

Subcult OS now models the replacement as a confidential web client and can serve the exact existing metadata and JWKS URLs when explicitly enabled. Configuration requires same-origin HTTPS client ID/callback/JWKS URLs, a P-256 private key and key ID, and rejects malformed enable flags. The generated public documents expose only the public JWK and request exactly `atproto`; they do not inherit the legacy repository scopes. Focused adapter, handler and configuration tests pass using generated ephemeral keys.

The callback and authorization-start flows are not implemented, so AT OAuth remains disabled by default. No live proxy, DNS, service, database, credential, OAuth grant or user data was changed. The [cutover runbook](../runbooks/subcults-cutover.md) requires current live-host inventory, proven backups/restores, an isolated candidate database, full API/browser qualification and an atomic proxy rollback before the legacy app may be stopped or replaced.

## 2026-09-20 — AT-01 encrypted OAuth persistence

Migration 4 adds AT OAuth requests and sessions without exposing protocol secrets as searchable plaintext. The OS-owned Indigo `ClientAuthStore` adapter binds each start request to an authenticated local person, HMAC-indexes state, encrypts request/session payloads under a protocol-specific derived key, atomically claims callbacks once, applies a ten-minute expiry, and provides expired-request cleanup.

Session creation accepts only the identity-level `atproto` scope. It atomically links the verified DID, stores authenticated-encrypted session material, consumes the request and writes an auth audit event. Existing sessions can persist rotated tokens only while their DID link remains active. Database uniqueness and a conditional conflict path reject a DID already owned by another local account; no email, handle or DID match merges people.

Focused PostgreSQL 17 tests passed for migration 4, concurrent callback claim, expiry and cleanup, recognizable-plaintext rejection, encrypted round trips, token rotation, audit creation, revoked-link refusal, over-scoped response refusal and cross-account DID rejection; the package also passed the race detector. The pinned OAuth import expands the Go transitive graph to the SDK's JWT, identity, CID/multibase and metrics dependencies. No OAuth HTTP route, live resolver request, external credential, PDS operation or repository scope was enabled.

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
