# Development backlog

Status: original engineering slices for the Subcult OS platform core. The [Gitea master roadmap](https://git.subcult.tv/subculture-collective/subcult-os/issues/1) now tracks 83 work items, including deferred and exploratory work, with native dependencies. No items were queued or assigned by roadmap publication. Sizes below are rough historical engineering ranges, not commitments.

## Dependency order

BASE-01 → API-01 + DB-01 + INV-01.
DB-01 + INV-01 → IDENT-01 + AT-01.
IDENT-01 + AT-01 → MODEL-01 → DISC-01 → PUB-01 → UX-01 → QUAL-01.
CONSENT-01 is a boundary task, not authorization to send. COMMONS-01 follows demonstrated conformance reuse. The owner promoted issues #50–71 for development on 2026-09-24; see the [expansion work order](../superpowers/plans/2026-09-24-expansion-50-71.md). Device, provider and production gates remain separate.

Review [decisions](decisions.md), [architecture](architecture.md), and the [extraction inventory](extraction-inventory.md) before implementation.

For the September 29 checkout and Gitea reconciliation, use the
[current delivery work order](current-delivery-work-order.md). Historical
implementation statuses below do not establish device, provider or deployment
qualification.

## BASE-01 — Capture current behavior and source evidence

- Repository: OS target; Subcults read-only
- Priority: P0
- Depends on: none
- Status: complete — baseline captured 2026-09-20; see execution-log.md
- Size: complete baseline slice

**Acceptance:** Record exact revisions, dirty state, tools and real check outcomes; distinguish current source evidence from historic release claims.

**Verification:** Run documented baseline commands; preserve failing output and prerequisite blockers.

**Out of scope:** No deployment, source import, dependency upgrade or test deletion.

**Rollback:** Documentation-only baseline; preserve prior files and source revisions.

## API-01 — Separate public event projection from operator DTO

- Repository: OS
- Priority: P0
- Depends on: BASE-01
- Status: implemented; focused privacy tests and subsequent aggregate `make verify` passed. Earlier dependency-policy failures are historical evidence, not the current gate.
- Size: implemented slice

**Acceptance:** Explicit public allowlist; private workspace/attendance/staffing properties rejected; necessary public UI retained.

**Verification:** Contract checker, serializer sentinel test, DB-backed anonymous route test, web/mobile type checks and focused suites.

**Out of scope:** No authentication replacement, AT dependency or public-data expansion.

**Rollback:** The prototype endpoint may change or disappear during consolidation, but no replacement may expose operational DTOs as a shortcut.

## DB-01 — Establish a clean ordered OS schema

- Repository: OS
- Priority: P0
- Depends on: BASE-01
- Status: implemented and focused-test-verified; no external database was reset or migrated
- Size: implemented foundation slice

**Acceptance:** Approve runner and clean platform schema; ledger each migration once; serialize concurrent runners; block incompatible startup; inventory every existing database before any reset and add legacy migration only for specifically retained real data.

**Verification:** New fresh, replay, concurrent-runner, failed-migration, backup and restore tests on disposable PostgreSQL. Add populated legacy-upgrade fixtures only if INV-01 finds data that must survive.

**Out of scope:** No Subcults migration import, live migration or unapproved destructive reset.

**Rollback:** Before replacing any non-empty database, classify and back up it read-only. For clean development databases, recreate from ordered migrations; do not build compatibility merely to restore prototype APIs.

## INV-01 — Complete selective Subcults extraction audit

- Repository: OS documentation; Subcults read-only
- Priority: P0
- Depends on: BASE-01
- Status: complete — file-level manifest recorded at the audited clean revision
- Size: complete audit slice

**Acceptance:** Classify candidate files as adapt, rewrite, fixture-only, reference-only or reject; record product need, revision, license/provenance, generated status, dependencies and coupled features.

**Verification:** Independent inventory review; every accepted candidate maps to a current journey and a focused test that does not require the old application.

**Out of scope:** No repository merge, source copy, migration import or cleanup of Subcults.

**Rollback:** Inventory changes are reversible documentation; rejected candidates remain untouched in Subcults.

## IDENT-01 — Build the canonical OS identity and session foundation

- Repository: OS
- Priority: P0
- Depends on: DB-01, INV-01
- Status: implemented and focused-test-verified; native-device and full recovery-browser qualification remain open
- Size: implemented foundation slice

**Acceptance:** Verified email identities, protected lookup material, rotating session families, revoke-one/revoke-all, recovery and additive DID-link slots; API, web and mobile move to the new model together.

**Verification:** Token replay/family revocation, expiry, account ambiguity, cross-account claim and web/mobile session fixtures. Add legacy-password/account migration cases only for inventoried real accounts.

**Out of scope:** No automatic email-equality merge, creator authority inference, Subcults account import or PDS provisioning.

**Rollback:** Recreate clean development state from migrations. If retained real accounts are discovered, specify a bounded migration and rollback before touching them; otherwise do not add dual-read or legacy-session code.

## AT-01 — Establish the minimal Go AT Protocol kernel

- Repository: OS
- Priority: P0
- Depends on: DB-01, INV-01
- Status: syntax, encrypted identity-only persistence, confidential-client metadata/JWKS, hardened resolver, start/callback and link/list/local-unlink UI implemented. Remote revocation, real provider qualification and Lexicon admission remain open in Gitea #9, #10 and #11.
- Size: active multi-slice item

**Acceptance:** Pinned minimal Indigo surface; canonical syntax/DID/handle/URI/CID validation; reviewed Lexicon subset; identity-only OAuth link/unlink; no publication scope by default.

**Verification:** Shared valid/invalid fixture corpus passes Go and TypeScript `@atproto/lex`; bounded resolver and OAuth state/replay/revocation tests.

**Out of scope:** No full Indigo fork, PDS hosting, record publication, Jetstream/Tap service or all old Lexicons.

**Rollback:** Module remains disabled behind internal interfaces; remove it without changing existing OS event/ticket flows.

## MODEL-01 — Add the minimum cultural record model

- Repository: OS
- Priority: P1
- Depends on: IDENT-01, AT-01
- Status: data model, workspace-scoped CRUD API and public-preview projection implemented 2026-09-23; see [cultural-model.md](cultural-model.md) and execution-log.md. Publication (writing to a PDS), projection/discovery ingestion and UI remain open in DISC-01/PUB-01/UX-01.
- Size: 5–8 engineering days

**Acceptance:** Define only journey-required Profile/Act, Place/Venue, public Event occurrence and private operator-event relation; preserve public/private location and time semantics.

**Verification:** Fresh/upgrade DB cases, ownership and cross-workspace negatives, duplicate occurrence, DST/reschedule, protected location and serializer sentinels.

**Out of scope:** No wholesale touring/social schema, streaming, alliances, posts or reputation.

**Rollback:** Additive tables and nullable relationships; unlinked operator events retain existing behavior.

## DISC-01 — Build validated public projection and discovery

- Repository: OS
- Priority: P1
- Depends on: MODEL-01
- Status: allowlisted, restart-safe Jetstream projection into `at_projection_*`
  implemented 2026-09-24 (migration 000011); see [projection.md](projection.md)
  and execution-log.md. Discovery UI, backfill tooling and reconciliation
  with `cultural_*` remain open in PUB-01/UX-01.
- Size: 4–7 engineering days

**Acceptance:** Accepted collections only; restart-safe cursor; URI/CID provenance; idempotent projection; delete/unavailable state; bounded backfill and quarantine.

**Verification:** Fixture stream replay, restart/cursor, out-of-order revision, deletion, malformed/oversized record, private-network resolution and recovery tests.

**Out of scope:** No complete AppView, every historical collection, ranking system or early broker/microservice split.

**Rollback:** Projections are rebuildable; disabling ingestion does not delete private operations or canonical PDS records.

## PUB-01 — Add authorized publication and reconciliation

- Repository: OS
- Priority: P1
- Depends on: IDENT-01, MODEL-01, DISC-01
- Status: proposed
- Size: 6–10 engineering days

**Acceptance:** Explicit preview/approval; workspace plus creator authority; separately scoped OAuth; payload allowlist/digest; stable idempotency; CID preconditions; exact observed outcome; bounded retries and quarantine.

**Verification:** Unauthorized actor, revoked grant, duplicate intent, stale CID, timeout-after-write, lost response, restart/replay and delayed projection tests against disposable authoritative fixtures.

**Out of scope:** No publish-on-save, broad service impersonation, exactly-once claim, new Lexicon field without review or production write.

**Rollback:** Stopping code cannot undo a PDS write; use reviewed compensating writes or preserve and reconcile the record with fresh authority.

## UX-01 — Deliver the unified account, operator and attendee journeys

- Repository: OS
- Priority: P2
- Depends on: DISC-01, PUB-01
- Status: proposed
- Size: 5–8 engineering days

**Acceptance:** One account/session; operator public-record preview and state; accessible pending/conflict UI; public discovery/reservation handoff; guests do not require an AT account.

**Verification:** Browser journeys, mobile real-device smoke, keyboard/accessibility checks, deep links, session refresh and PDS/indexing outage behavior.

**Out of scope:** No complete visual redesign or mandatory native app.

**Rollback:** Existing direct public URL and local event lifecycle stay usable while new cultural/publication UI is disabled.

## CONSENT-01 — Define delivery consent without attendance inference

- Repository: OS
- Priority: P2
- Depends on: IDENT-01
- Status: grant schema, `checkSendPermission` and send-time recheck implemented 2026-09-24 (migration 000013, issue #23); see execution-log.md and docs/development/consent.md. No audit-redaction fixture exists yet. SIGNAL-01 (issue #24, below) is the implemented announcement send path.
- Size: 2–4 engineering days

**Acceptance:** Separate transactional notices from marketing; define sender/channel/purpose/scope/verification/suppression; no ticket/contact import grants consent.

**Verification:** No-consent/no-send, revoked-before-send, wrong-purpose, suppression and audit-redaction fixtures.

**Out of scope:** No autonomous marketing, SMS/social DM automation or legacy audience import.

**Rollback:** Delivery remains transactional-only and fail-closed until an accepted consent contract exists.

## SIGNAL-01 — Implement one scoped announcement channel and delivery worker

- Repository: OS
- Priority: P2
- Depends on: CONSENT-01
- Status: verified email through the existing Resend outbox chosen as the one channel; migration 000014 (`announcements`, `announcement_deliveries`, `consent_grants.withdraw_token_hash`); draft/preview/schedule/cancel/list/get endpoints behind a new `manage_announcements` permission; `-announce` dispatch mode re-deriving the audience and enqueueing per-recipient `email_outbox` rows; synthetic grant/schedule/withdraw/dispatch/send journey implemented 2026-09-24 (issue #24); see execution-log.md and docs/development/announcements.md. No live deliverability test or permissioned pilot has been run (blocked on #7); no SMS or second channel.
- Size: 3–5 engineering days

**Acceptance:** Choose one verified channel using pilot need and observed cost; explicit scheduling with preview, cancellation and final consent checks; record provider delivery outcomes, bounded retries, cost and suppression; verify synthetic grant/schedule/revoke/send journey before a limited permissioned pilot.

**Verification:** Synthetic PostgreSQL journey (grant, confirm, schedule, preview, withdraw, dispatch, send with a fake sender); permission-boundary and past-scheduling rejection fixtures.

**Out of scope:** No SMS/social DM channel, no live send test, no UI, no re-send or per-recipient personalization beyond the withdraw link.

**Rollback:** Stop invoking `-announce`; existing transactional sending and CONSENT-01's grant/withdraw endpoints are unaffected. Roll back the application only; do not drop the additive migration while any dispatched announcement's delivery ledger must be retained for audit.

## QUAL-01 — Qualify the consolidated protected pilot

- Repository: OS
- Priority: P2
- Depends on: UX-01, CONSENT-01
- Status: proposed
- Size: determined by observation windows

**Acceptance:** Exact artifact/revision evidence; clean install and restore; provider/PDS/browser proof; privacy fixtures; outage behavior; independent review; rollback rehearsal. Add legacy-data migration evidence only if retained data exists.

**Verification:** Complete release evidence matrix without substituting health, registration or synthetic bypasses for natural behavior.

**Out of scope:** No public launch, destructive cleanup, external-state mutation or real-user migration without separate authorization.

**Rollback:** Preserve immutable migration ledgers and external-write evidence. A previous service/traffic return procedure is required only if a deployed consumer is found during qualification.

## COMMONS-01 — Publish reusable AT application conformance fixtures

- Repository: OS or a rights-reviewed extracted package
- Priority: P3
- Depends on: AT-01, PUB-01
- Status: proposed
- Size: 3–6 engineering days plus external review

**Acceptance:** Independent implementation can reproduce supported syntax, Lexicon, OAuth, publication and reconciliation cases without private Subcult services.

**Verification:** Second-language or second-implementation reproduction report with exact versions and known gaps.

**Out of scope:** No automatic standard, namespace governance claim, copied secrets/data or unreviewed relicensing.

**Rollback:** Keep package private until rights, maintenance and disclosure review pass.

## LIFE-01 — Specify cancellation and rescheduling before refunds

- Repository: OS
- Priority: active development by owner request, 2026-09-24
- Depends on: QUAL-01 for rollout; independent source work promoted
- Status: [state matrix, occurrence edit safeguards and private owner worklist](event-lifecycle-changes.md) implemented, including durable decision-key replay and unsent-only supersession. Coordinated dispatch and notice-failure recovery remain open. Merged worklist baseline `8166559` has passing hosted CI run 10160, rechecked 2026-09-29.
- Size: bounded multi-slice work

**Acceptance:** State matrix covers public record, operator plan, tickets, notice, refund policy, projection and archive continuity.

**Verification:** State-machine, stale-CID, provider test-mode and participant-notice cases before implementation completion.

**Out of scope:** No assumed automatic refund, deletion or live charge.

**Rollback:** Preserve prior published/ticket state and require explicit compensating action for external writes.

## OFFLINE-01 — Research disconnected door operations

- Repository: OS
- Priority: active source research by owner request, 2026-09-24
- Depends on: QUAL-01 for rollout; independent research promoted
- Status: [synthetic research slice implemented](../research/offline-door-experiment-2026-09-25.md), with Go merge-model tests and two separate Node client processes. The harness passed again 2026-09-29. Physical-device partition/reconnect, scanner, persistence and manual-fallback qualification remain open; disconnected admission remains unavailable.
- Size: research slice complete; device qualification required before a product slice

**Acceptance:** Define snapshot expiry, revocation, duplicate check-in conflict, reconnect merge and device loss for a bounded pilot.

**Verification:** Two-device partition/reconnect experiments and real-device plan.

**Out of scope:** No production offline guarantee or permanent ticket cache.

**Rollback:** Offline capability remains disabled; server-authoritative door flow stays intact.
