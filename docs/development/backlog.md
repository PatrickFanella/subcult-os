# Development backlog

Status: local work items for the Subcult OS platform core. No hosted issue has been created, queued or assigned. Sizes are rough engineering ranges, not commitments.

## Dependency order

BASE-01 → API-01 + DB-01 + INV-01.
DB-01 + INV-01 → IDENT-01 + AT-01.
IDENT-01 + AT-01 → MODEL-01 → DISC-01 → PUB-01 → UX-01 → QUAL-01.
CONSENT-01 is a boundary task, not authorization to send. COMMONS-01 follows demonstrated conformance reuse. LIFE-01 and OFFLINE-01 remain later discovery.

Review [decisions](decisions.md), [architecture](architecture.md), and the [extraction inventory](extraction-inventory.md) before implementation.

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
- Status: implemented and focused-test-verified; aggregate `make verify` remains blocked by mobile dependency build policy
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
- Status: proposed
- Size: 4–7 engineering days

**Acceptance:** Pinned minimal Indigo surface; canonical syntax/DID/handle/URI/CID validation; reviewed Lexicon subset; identity-only OAuth link/unlink; no publication scope by default.

**Verification:** Shared valid/invalid fixture corpus passes Go and TypeScript `@atproto/lex`; bounded resolver and OAuth state/replay/revocation tests.

**Out of scope:** No full Indigo fork, PDS hosting, record publication, Jetstream/Tap service or all old Lexicons.

**Rollback:** Module remains disabled behind internal interfaces; remove it without changing existing OS event/ticket flows.

## MODEL-01 — Add the minimum cultural record model

- Repository: OS
- Priority: P1
- Depends on: IDENT-01, AT-01
- Status: proposed
- Size: 5–8 engineering days

**Acceptance:** Define only journey-required Profile/Act, Place/Venue, public Event occurrence and private operator-event relation; preserve public/private location and time semantics.

**Verification:** Fresh/upgrade DB cases, ownership and cross-workspace negatives, duplicate occurrence, DST/reschedule, protected location and serializer sentinels.

**Out of scope:** No wholesale touring/social schema, streaming, alliances, posts or reputation.

**Rollback:** Additive tables and nullable relationships; unlinked operator events retain existing behavior.

## DISC-01 — Build validated public projection and discovery

- Repository: OS
- Priority: P1
- Depends on: MODEL-01
- Status: proposed
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
- Status: proposed
- Size: 2–4 engineering days

**Acceptance:** Separate transactional notices from marketing; define sender/channel/purpose/scope/verification/suppression; no ticket/contact import grants consent.

**Verification:** No-consent/no-send, revoked-before-send, wrong-purpose, suppression and audit-redaction fixtures.

**Out of scope:** No autonomous marketing, SMS/social DM automation or legacy audience import.

**Rollback:** Delivery remains transactional-only and fail-closed until an accepted consent contract exists.

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
- Priority: Later
- Depends on: QUAL-01
- Status: proposed
- Size: discovery required

**Acceptance:** State matrix covers public record, operator plan, tickets, notice, refund policy, projection and archive continuity.

**Verification:** State-machine, stale-CID, provider test-mode and participant-notice cases before implementation completion.

**Out of scope:** No assumed automatic refund, deletion or live charge.

**Rollback:** Preserve prior published/ticket state and require explicit compensating action for external writes.

## OFFLINE-01 — Research disconnected door operations

- Repository: OS
- Priority: Later
- Depends on: QUAL-01
- Status: proposed
- Size: discovery required

**Acceptance:** Define snapshot expiry, revocation, duplicate check-in conflict, reconnect merge and device loss for a bounded pilot.

**Verification:** Two-device partition/reconnect experiments and real-device plan.

**Out of scope:** No production offline guarantee or permanent ticket cache.

**Rollback:** Offline capability remains disabled; server-authoritative door flow stays intact.
