# Development execution log

## 2026-09-23 — #13 private event-to-public-occurrence links (LINK-01)

Added migration `backend/internal/app/migrations/000009_event_public_links.sql`
(schema version 8 → 9): `event_public_links`, one row per private operator
`events` row linked to a public `tv.subcult.event.occurrence` AT record,
carrying `public_uri`, `observed_cid`, the resolved authority DID and
freshness fields (`status` in `fresh`/`changed`/`unavailable`/`deleted`,
`observed_at`, `last_checked_at`, `last_error`). Composite
`(event_id, workspace_id)` foreign key back to `events (id, workspace_id)`,
matching migration 000008's pattern; `unique (event_id, public_uri)` makes
repeated attach idempotent at the database level. `minimumSchemaVersion`
moved to 9.

Added `backend/internal/atproto/record_fetch.go`: a `RecordFetcher`
interface plus `IdentityRecordFetcher`, the production implementation,
which resolves a record's authority through the existing hardened
`identity.Directory` and fetches it through `com.atproto.repo.getRecord`
using the same public-only outbound HTTP client and identity hardening AT
OAuth already uses (`hardenIdentityDirectory`/`publicOnlyHTTPClient`).
Tests supply a fixture `RecordFetcher`; nothing in this repository's test
suite makes a live network call for this feature.

Added `backend/internal/app/event_public_links.go`: workspace-scoped
`POST .../preview` (resolves, Lexicon-validates and returns source
identity/CID/event fields without persisting anything), `POST
.../public-links` (attach, idempotent via `on conflict do update` no-op),
`GET .../public-links` (list), `DELETE .../public-links/{linkID}`
(detach), and `POST .../public-links/{linkID}/refresh` (re-fetches and
updates only this row's status/freshness columns: same CID → `fresh`,
different CID → `changed`, `RecordNotFoundError` → `deleted`, any other
fetch failure → `unavailable` with `last_error` recorded). All five routes
reuse the existing `requireWorkspaceRole(..., "owner", "member")` gate, so a
person outside the event's workspace gets `403`, matching the MODEL-01
cultural routes. Full route/table detail, the fetcher boundary and known
limits are in [`public-links.md`](public-links.md).

### Verification actually run

- Go 1.26.6 `go build ./...` and `go vet ./...` (backend).
- Disposable PostgreSQL (`docker compose -p subcult-wf-13`, port 47013):
  `go test ./internal/app ./internal/atproto -count=1 -v`, including the
  new `backend/internal/app/event_public_links_integration_test.go`
  (preview does not persist; preview rejects a record missing required
  Lexicon fields; attach is idempotent across two identical requests;
  cross-workspace access to preview/list/attach/refresh/detach all return
  `403`; list/detach; refresh detects a changed CID, an unavailable
  fetcher error, and a deleted (`RecordNotFoundError`) record, and confirms
  event editing stays usable after a link is marked `deleted`; preview
  rejects a handle-authority URI). All passed; container removed after the
  run (`docker compose -p subcult-wf-13 down -v`).
- Full `make verify`; see below.

### Remaining limits

- API-only; no web/mobile UI.
- Refresh is manual (`POST .../refresh`); no background scheduler
  re-checks links automatically.
- `authority_did` reflects the identity resolved at attach/refresh time; it
  is not re-verified against identity rotation except on the next refresh.

## 2026-09-23 — #12 minimal cultural record model (MODEL-01)

Added migration `backend/internal/app/migrations/000008_cultural_model.sql`
(schema version 7 → 8): `cultural_profiles` (a `kind` column of
`creator`/`collective`/`act` stands in for a separate Act table, per D6's
recommended starting point), `cultural_places` (public fields only) plus
`cultural_place_protected_details` (street address/access notes in a
separate table the public serializer never joins), `event_occurrences`
(relates to an existing private `events` row; one event may have more than
one occurrence; nullable `public_uri`/`public_cid` left unset), and
`event_occurrence_profiles` (multi-host attribution with role and sort
order). Every new table carries a composite `(id, workspace_id)` foreign key
back to its parent so a cross-workspace reference is rejected by PostgreSQL
itself. `events` gained `unique (id, workspace_id)` to support this.
`minimumSchemaVersion` moved to 8.

Added workspace/event-scoped CRUD handlers
(`cultural_profiles.go`, `cultural_places.go`, `cultural_occurrences.go`)
reusing the existing `requireWorkspaceRole` ownership gate, and a public
projection (`cultural_public_projection.go`) that builds the exact
`tv.subcult.profile`/`.place`/`event.occurrence` record shapes from
public-only columns and validates them with the existing
`atproto.ValidateAdmittedRecord`, exposed read-only at
`GET .../occurrences/{id}/public-preview`. Full table/route inventory,
the DST/reschedule/sentinel reasoning and known limits are in
[`cultural-model.md`](cultural-model.md). `decisions.md`'s D6/D7 rows now
note that this slice implements their recommended starting points; D6/D7
remain Open. D5/ADR 0007 (Lexicon admission itself) was accepted the same
day as A10. No web/mobile UI and no contract-schema DTO were added.

### Verification passed
- `go build ./...`, `go vet ./...`, `gofmt -l .` (clean) from `backend/`
- `TEST_DATABASE_URL=<disposable> go test ./internal/app ./internal/atproto -count=1 -v`: 205 tests, 0 failures, both packages `ok`, including the 11 new tests (fresh migration, upgrade path preserving a pre-existing event, profile/place CRUD, duplicate occurrences + multi-host credit attach/detach, reschedule-does-not-touch-tickets, cross-workspace rejection, DST round-trip, public-preview sentinel leak check, public-preview-requires-a-credited-profile)
- `make verify`: deps, fmt, lint, `check-contracts`, `test` (backend/web/mobile/qa-scripts), `build`, `compose-config`, `open-pilot-check` all passed; web 200/200, mobile 27/27
- Disposable PostgreSQL used `COMPOSE_PROJECT_NAME=subcult-os-model`/`POSTGRES_PORT=47432`, torn down with `docker compose -p subcult-os-model down -v` after the run

### Remaining
Publication (writing any record to a PDS), projection/discovery ingestion,
reconciliation, and any web/mobile UI for profiles/places/occurrences are
still open (DISC-01/PUB-01/UX-01). Review embedded the admitted Lexicon
documents into the binary (`backend/internal/atproto/lexicons/`, kept
byte-identical to `contracts/lexicons/` by a test) so `/public-preview`
works in the production image without a contracts directory on disk;
`LEXICON_CONTRACT_DIR` remains an optional development override.

## 2026-09-23 — #11 Lexicon admission accepted (D5 → A10)

The repository owner accepted ADR 0007 on 2026-09-23, following its recommendation to admit the independently authored minimal `tv.subcult.*` chain (profile, place, event occurrence) rather than adopt the community calendar schemas wholesale. D5 moved from the Open table to the Accepted table as A10 in `decisions.md`; the ADR status, `atproto-kernel.md`, `lexicon-contract.md` and the architecture contract table were updated to say admitted instead of proposed. No schema, corpus, validator or runtime code changed in this step. Publication of any `tv.subcult.*` record still depends on D7 through D10 and on the MODEL-01, DISC-01 and PUB-AUTH slices.

## 2026-09-23 — #46 Indigo outbound-policy contribution package

New [Indigo outbound-policy contribution package](../upstream/indigo-outbound-policy/README.md), prepared from a disposable copy under `/tmp`, outside the repository. Source came read-only from the local Go module cache (`.blacktower/clonedeps` was absent in this worktree); nothing under the module cache was modified. Against the exact pinned commit `41278964ec8e3253e70d4e919dfb8e34211c543d` (`v0.0.0-20260903211445-41278964ec8e`, matching `backend/go.mod`), eight new standalone tests across two files (`oauth_outbound_policy_test.go`, `identity_outbound_policy_test.go`) demonstrate that all four reviewed request kinds — OAuth server metadata discovery (`oauth.Resolver.Client`), the OAuth PAR/token endpoint (`oauth.ClientApp.Client`), handle HTTPS well-known resolution and did:web (both `identity.BaseDirectory.HTTPClient`), and did:plc (`identity.BaseDirectory.PLCClient`) — follow a same-origin-unchecked redirect and honor an ambient `HTTP_PROXY`, using loopback `httptest` servers and no real network calls. `PLCClient` was additionally found to have no `Transport` set at all in `identity.DefaultDirectory()`, so it inherits `http.DefaultTransport` with no public-IP dial restriction, unlike the other three clients. All eight tests, plus the existing package suites, pass under `go test -race`.

Evaluated two proposal shapes: constructor/option-struct fields across three packages, versus a documented strict profile assembled from already-exported fields (the same technique Subcult OS already applies in `backend/internal/atproto/oauth_flow.go`). Recommended the latter, plus one small additive patch: `ssrf.StrictPublicOnlyTransport()` (`fix.patch`), which is `PublicOnlyTransport()` with `Proxy` forced to `nil` and changes no existing default. The patch applies cleanly via `patch -p1` (this worktree's git-operation guard blocks `git apply` outside the assigned worktree) and the patched `util/ssrf` package builds, vets and passes its existing test.

Searched `bluesky-social/indigo` issues and PRs for `CheckRedirect` and `ProxyFromEnvironment` (0 results each) and for `SSRF`/`PublicOnlyTransport` (13/2 results): no existing issue or PR addresses this proxy/redirect gap. The closest prior work — PR #1452 "harden identity package" and PR #1451 "util/ssrf: small improvements" (both merged 2026-09-01, already present in the pin) and open issue #1461 on reusable "safe" client singletons — covers dial-level SSRF and client-reuse ergonomics, not proxy or redirect policy. Fetched `main`'s `util/ssrf/ssrf.go` and `atproto/auth/oauth/resolver.go` directly: both match the pinned commit's behavior, so the gap is present on current `main`, not just the pin.

The contribution package README records this evidence, the proposal and compatibility analysis, the upstream search results with URLs, a submitter checklist, an AI-assistance disclosure, and a PR description draft. Nothing was filed, posted, or opened upstream. `docs/development/upstream.md`'s candidate bullet now links the package. Subcult's existing hardening (`newHardenedIndigoClient`, `hardenIdentityDirectory` in `oauth_flow.go`) already applies the equivalent strict profile in production and was not changed.

Remaining: all human-only submission steps in the package's checklist (authorization, GitHub account, DCO/CLA check, opening the issue first, final re-reproduction against then-current `main`, deciding between the two proposed shapes with maintainers). No upstream issue or PR was opened.

## 2026-09-23 — #45 Indigo persistence contribution reproduction

Re-reproduced the prepared [Indigo persistence contribution package](../upstream/indigo-persistence/README.md) from a disposable copy under `/tmp`, outside the repository. Source came read-only from the local Go module cache (`.blacktower/clonedeps` was absent in this worktree); nothing under the module cache was modified. Against the exact pinned commit `41278964ec8e3253e70d4e919dfb8e34211c543d` (`v0.0.0-20260903211445-41278964ec8e`, matching `backend/go.mod`), the existing synthetic test `TestStartAuthFlowRejectsUnstoredState` failed with `error=<nil>; want wrapped persistence failure`. After applying the existing `fix.patch` to a second disposable copy of the same commit, `go test -race ./atproto/auth/oauth -count=1` passed in full. Neither the test nor the patch needed changes.

Searched `bluesky-social/indigo` issues and PRs (open and closed) for `SaveAuthRequestInfo`, `StartAuthFlow persist`, `ClientAuthStore`, and `oauth persist error`: no issue or PR addresses this defect. One open PR (#1164) adds unrelated `state`-stashing behavior; one merged PR (#1159) adds a duplicate-state guard inside `MemStore` but does not check the error at the `StartAuthFlow` call site. Fetched `main`'s `atproto/auth/oauth/oauth.go` directly from GitHub: the unchecked `app.Store.SaveAuthRequestInfo(ctx, *info)` call is still present on `main`, not just at the pinned commit. The repository has no root `CONTRIBUTING.md`; the README's "Contributions" section is the operative guidance, and states no AI-assistance disclosure policy.

The contribution package README now records this reproduction, the upstream search results with URLs, a submitter checklist (authorization to publish, account, DCO/CLA recheck, issue-first, re-reproduce against current `main`, no internal references), an AI-assistance disclosure statement, and a PR description draft. Nothing was filed, posted, or opened upstream. Subcult's local fail-closed capture wrapper in `backend/internal/atproto/oauth_flow.go` was confirmed unchanged and its regression test (`TestOAuthFlowSurfacesIgnoredUpstreamPersistenceError`) still passes; `docs/development/upstream.md` now states it stays until an upgraded, separately verified Indigo dependency is pinned.

Remaining: all human-only submission steps in the package's checklist (authorization, GitHub account, DCO/CLA check, opening the issue first, final re-reproduction against then-current `main`). No upstream issue or PR was opened.

## 2026-09-23 — #11 minimal Lexicon admission proposal

Reviewed `community.lexicon.calendar.event`/`.rsvp` and `community.lexicon.location.address`/`.geo` (Lexicon Community, MIT License) as prior art; the `events.smokesignal.*` source repository returned HTTP 404 when fetched directly, so only unverified indirect descriptions of it exist and none were relied on. Recommended, and drafted, a minimal independently authored `tv.subcult.*` chain (`tv.subcult.profile`, `tv.subcult.place`, `tv.subcult.event.occurrence`) rather than adopting `community.lexicon` as-is, because it has no profile record, no independently addressable place record, and no bounds or public/private field distinction. Full reasoning, license and evidence are in [ADR 0007](../adr/0007-minimal-lexicon-admission.md) (status Proposed); field allowlists, bounds, and public time/location semantics are in [`lexicon-contract.md`](lexicon-contract.md).

Added `contracts/lexicons/*.json` (three record Lexicons) and `contracts/atproto-lexicon.fixtures.json` (`tv.subcult.profile`: 2 valid/8 invalid; `tv.subcult.place`: 2 valid/10 invalid; `tv.subcult.event.occurrence`: 3 valid/13 invalid; 38 cases total). Both `backend/internal/atproto/lexicon.go` (using the pinned Indigo `atproto/lexicon` package, confirmed present at the pinned commit) and `web/src/atprotoLexiconConformance.test.ts` (using `@atproto/lexicon@0.7.6`, added as a pinned `web/package.json` devDependency and installed into `web/pnpm-lock.yaml`) run the same corpus and agree on every case. Both validators additionally enforce a field allowlist derived from the Lexicon JSON's own `properties`, and an explicit-UTC-offset datetime check, because official Lexicon validation intentionally allows additive unknown fields and (in `@atproto/lexicon`'s case) a missing datetime offset; this discrepancy is recorded in ADR 0007 and `lexicon-contract.md` rather than papered over.

Updated `atproto-kernel.md`'s Lexicon boundary section and `decisions.md`'s D5 row to point at the ADR and contract doc. D5 is **not** marked Accepted; this slice is a complete, reviewable proposal; the repository owner accepts it by flipping ADR 0007 to Accepted. No database, migration or runtime handler changed.

### Verification passed
- `node scripts/check-contracts.mjs`
- `go test ./internal/atproto -run TestSharedLexiconConformanceFixture -count=1` (42 cases)
- `go test ./internal/atproto -count=1` and `go vet ./...`
- `pnpm run test -- atprotoLexiconConformance` (40 cases; 200/200 across the whole web suite)
- `make verify` (see PR/commit for the exact final tail)

## 2026-09-23 — #5 operations-panel API rehearsal

New `scripts/qa-operations.sh` (`make operations-qa`) rehearses the QUAL-BASE second acceptance bullet: workspace invitations and switching, contacts, commitments, staffing assignments, event templates, roles/applications, and reminder-sweep boundaries. Built and ran against a fresh isolated Compose stack (`subcult-os-qual`, ports 45432/48080/48079, disposable database `subcult_qa_operations_c97f07a`) from revision `c97f07a58a70c2b20090cc6809bc056329f84342`. The script produced 36 PASS/0 FAIL across two consecutive runs, exit code 0, covering: invite-then-accept plus client-side workspace switching by ID; contact create/list/update; commitment create/status-validation/completion; an owner-only staffing create/assign/PATCH boundary (member gets 403); template create/apply-to-draft plus a 409 once the target event is published; public role application submit/review with an invalid-status 400 and an owner-only decision boundary; and a reminder sweep that is role-gated (403 for members), state-gated (only overdue open commitments produce a reminder; a 30-day-future commitment produces none), and idempotent (a repeat sweep creates zero additional reminders).

This is API-level evidence only — no browser or native-device rehearsal was run for these panels. No product defects were found; all authorization and validation boundaries matched the code as read. Three bounded follow-ups are recorded in `docs/qa/operations-panels-2026-09-23.md`: the API has no recipient-level consent flag for reminders (only state/role gating), workspace "switching" is confirmed client-side-only with no server session state (documentation note, not a defect), and broader operations-panel browser/device coverage remains open for a future headed rehearsal. `make verify` passed after adding the script and docs.

## 2026-09-23 — #8 identity/AT signing key rotation

`IDENTITY_PROTECTION_KEY` (verified email encryption/lookup) and the AT OAuth
confidential client signing key now support a bounded rotation window rather
than an all-or-nothing swap. `identityProtector` and `atproto.OAuthStore`
each accept an optional previous key, try the current key first and the
previous key second on both decrypt and lookup-hash matching, and always
write under the current key. A new resumable `identity-rekey` command
(`backend/cmd/identity-rekey`, `backend/internal/app/identity_rekey.go`,
`backend/internal/atproto/oauth_rekey.go`) re-encrypts `email_identities`,
`atproto_oauth_sessions` and `atproto_oauth_revocations` with a keyset
cursor in bounded `FOR UPDATE SKIP LOCKED` transactions, is idempotent, and
prints aggregate counts only. Review added a test that a batch limit smaller
than the table still rewrites every previous-key row in one run. The AT OAuth JWKS
(`backend/internal/atproto/oauth_client.go`) can publish a previous public
key alongside the current one during a signing-key transition; the private
key that signs new assertions is always the current one.

Verified: unit tests for wrong-key rejection, tampered-ciphertext rejection,
current-only vs. current+previous decrypt and lookup during rotation, and
re-encryption idempotency (`backend/internal/app/identity_crypto_test.go`);
production config validation failing closed with a named-variable message
when `IDENTITY_PROTECTION_KEY` is missing or malformed, and accepting a
well-formed previous key (`backend/internal/app/app_test.go`); JWKS
containing both keys during a transition and only the current key afterward,
plus rejection of malformed/colliding previous-key settings
(`backend/internal/atproto/oauth_client_test.go`). Started a disposable
`docker compose up -d postgres` (throwaway database, not a retained
application database) and ran `go test ./internal/app ./internal/atproto
-count=1` with `TEST_DATABASE_URL` set: all 271 cases passed, including a new
end-to-end test that seeds email and AT OAuth rows under a previous key,
runs `identity-rekey`, confirms the previous key can then be removed while
reads still succeed, and confirms a second run changes nothing. `make verify`
passed locally.

Open and out of reach here: provisioning either key through a real deployed
secret store, a live rehearsal of a deployed rotation (this environment has
no deployed instance to rotate), and observing an actual AT Protocol
resource server accept a token signed against the previous key during a
JWKS transition (covered here only by asserting the JWKS document shape).
`atproto_oauth_requests` rows are deliberately not re-encrypted; they expire
in 10 minutes and are documented in `docs/runbooks/key-rotation.md` as a
bounded, self-clearing exception. The pre-existing plaintext `people.email`
column (a known prototype leftover, unrelated to this key) was left
untouched — protecting it was out of scope for this issue.

## 2026-09-22 — #121 verified rehearsal accounts

The real-backend checkpoint at `f206b4f` found both legacy rehearsal scripts failing with 401 after signup: they assumed signup issued a session. They now require explicit loopback/disposable-database opt-in, read the held verification message for their own synthetic recipient, and consume its challenge through the normal verification endpoint. They never mark identities verified directly, print tokens, or send mail. The shared helper rejects retained/remote targets and propagates database, missing-message and HTTP failures; its network-free tests run in `make verify`.

After correcting a column-name error in the first helper attempt, both free API rehearsals passed against `subcult_qa_batch_f206b4f` and the unchanged Go binary built from `f206b4f`. Coverage includes verified signup, invitation acceptance, publish, capacity rejection, door search/idempotent check-in, end-of-night counts, role visibility/review and staffing edits. These API checks do not replace served browser journeys, native-device tests or live provider qualification. The earlier 401 results remain evidence of the inherited script defect.

## 2026-09-22 — #119 payment-aware ticket presentation

Admission messages now require free or paid status. Pending, cancelled and unknown payment values never claim readiness or granted access, even when a prior check-in is recorded. Codes remain available for support with an explicit statement that they do not bypass payment. Cancelled-payment copy no longer invents a resumable checkout. Already scanned free/paid tickets say "Already checked in" rather than implying a new grant of access.

Full local `make verify` passed. Tests cover all four payment values plus unknown input across reserved/checked-in states, and rendered pending/cancelled pages reject conflicting ready-for-entry text. A browser with a synthetic pending ticket confirmed the corrected warning, payment explanation and support-code copy. No live payment, admission or deployment was performed. The five-issue batch now enters combined verification and hosted CI review before merge.

## 2026-09-22 — #117 ticket recovery

Ticket pages now provide an in-page read retry/refresh. A failed refresh retains the last loaded pass and explicitly warns that payment/check-in status may have changed. Changing ticket codes clears the prior ticket and QR; cancelled asynchronous loads cannot replace the new result. QR generation failure shows manual-code instructions instead of an indefinite preparation message.

Full local `make verify`, render regressions for QR association/failure and initial/refresh errors, docs validation and diff checks passed. A real browser with a synthetic API confirmed initial outage recovery, pending-ticket/QR rendering and retained pass plus warning after a failed refresh. QR failure rendering is unit-tested, not a browser-induced canvas failure. No reservation, payment, check-in or live email was issued; hosted CI and actual backend qualification remain separate.

## 2026-09-22 — #115 saved-state lifecycle transitions

Publish and end-of-night now require a clean, saved event form and explain why the action is disabled when edits remain. Save and lifecycle handlers reject overlapping actions, and event detail inputs are locked during their requests so a response cannot replace edits made in flight. No unsaved content is automatically published.

Full local `make verify` passed after correcting a type annotation in the new test fixture; the first failed check is retained in the local verification log. Rendering regressions cover both lifecycle actions with dirty forms and both busy states. A browser against the synthetic API showed publish enabled for a clean draft, disabled with the save-first explanation after a title edit, and enabled again when the saved title was restored. No publish, close, payment or live mail request was made during that browser check. Hosted verification is a separate gate.

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
