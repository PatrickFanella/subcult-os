# Decision register

Accepted architecture is recorded in repository ADRs. Open entries below must not be labeled accepted until reviewed. Subcults decisions are evidence to evaluate during extraction; they do not silently become OS decisions.

## Accepted

| ID | Decision | Result | Evidence |
| --- | --- | --- | --- |
| A1 | Receiving repository | Subcult OS absorbs selected Subcults capabilities; no wholesale merge | [ADR 0005](../adr/0005-subcult-os-platform-core.md) |
| A2 | Backend language | Go production backend; TypeScript frontend and independent AT conformance | [ADR 0005](../adr/0005-subcult-os-platform-core.md) |
| A3 | Runtime topology | One supported API, one identity authority and one PostgreSQL database; bounded modules and optional worker commands | [architecture](architecture.md) |
| A4 | Extraction policy | Product-need-driven admission; no migration/test/application bulk import | [extraction inventory](extraction-inventory.md) |
| A5 | Prototype compatibility | No API/schema/identifier/session compatibility by default; preserve only inventoried real data or external state | [ADR 0006](../adr/0006-no-prototype-compatibility-contract.md) |
| A6 | Production web and AT OAuth identity | Replace the legacy service at `subcults.subcult.tv` only after qualification; preserve its metadata/callback/JWKS URLs as a confidential web client and narrow requested scope to `atproto`. Amended 2026-10-02: the canonical origin and AT OAuth client URLs moved to `os.subcult.tv`, and the legacy host redirects pages there | [AT kernel](atproto-kernel.md), [cutover runbook](../runbooks/subcults-cutover.md) |
| A7 | First AT dependency surface | Pinned Indigo syntax and OAuth packages stay behind the OS-owned adapter; extend only through drift-tested wrappers | [AT kernel](atproto-kernel.md) |
| A8 | OS ordered migrations (formerly D1) | Embedded ordered runner with immutable checksum ledger, transaction/advisory locking and startup compatibility gate | DB-01 and [execution log](execution-log.md) |
| A9 | Canonical account shape (formerly D2) | Verified email identity and rotating session families; browser/native transports and recovery are implemented | IDENT-01 and [execution log](execution-log.md); device/provider qualification remains separate |
| A10 | Lexicon admission (formerly D5) | Minimal independently authored `tv.subcult.*` chain: profile, place and event occurrence, with strict public allowlists and additive-only evolution; community calendar schemas reviewed as prior art, not adopted | [ADR 0007](../adr/0007-minimal-lexicon-admission.md), [lexicon contract](lexicon-contract.md); accepted 2026-09-23, publication still gated by D7 to D10 |
| A11 | Projection ingestion (formerly D10) | Small restart-safe Jetstream-based worker admitting only the three accepted collections, with an atomically-committed cursor and bounded quarantine | DISC-01 and [projection.md](projection.md); implemented 2026-09-24. Backfill, rebuild/compare and reconcile against an approved-authority allowlist added 2026-09-24 (migration 000012; [projection.md](projection.md)) so a missed stream window or a rejected/quarantined event can be recovered from the authoritative PDS rather than only from the live stream. Publication (D8, D9) and discovery UI (UX-01) remain open; this worker does not itself read or write any `cultural_*` table |

## Open

| ID | Decision | Recommended starting point | Blocks / evidence needed |
| --- | --- | --- | --- |
| D3 | Retained account migration | None unless a read-only inventory finds real accounts; then require fresh proof and never email equality alone | IDENT-01; data inventory, ambiguity and replay fixtures |
| D6 | Public cultural model | Minimal Profile/Act, Place/Venue, Event occurrence and optional Tour/Appearance | MODEL-01; user journeys and existing OS event comparison. MODEL-01 implements this recommended starting point (`cultural_profiles` with a `kind` column including `act`, `cultural_places`, `event_occurrences`; see [cultural-model.md](cultural-model.md)); the owner accepts D6 by ADR, not by this implementation alone |
| D7 | Operator/public event relationship | Private operator event owns operations; creator PDS owns published occurrence; private URI/CID state relates them | MODEL-01; time, location, cancellation and duplication cases. MODEL-01 implements this recommended starting point (`event_occurrences` rows relate to an existing `events` row via a workspace-checked foreign key and carry nullable `public_uri`/`public_cid` columns left unset until a later PUB-01 publish); the owner accepts D7 by ADR, not by this implementation alone |
| D8 | Publishing actor authority | Require workspace permission, creator/profile delegation and current scoped OAuth | PUB-01; revocation and cross-tenant negative fixtures. AUTH-01 implements the workspace-permission and creator-delegation halves of this recommended starting point ([authority-model.md](authority-model.md)); current scoped OAuth is not yet checked, and the owner accepts D8 by ADR, not by this implementation alone |
| D9 | Public editing conflict policy | CID preconditions plus explicit human conflict resolution | PUB-01; concurrent client and timeout-after-write fixtures |
| D11 | License and provenance | Record exact source revision/path and adapted/generated status for every extraction | INV-01 and each extraction; contributor/dependency audit |
| D12 | Marketing integration | No attendance/contact inference or sending until purpose, verification and suppression contracts pass | CONSENT-01; no-consent/no-send and revocation cases |

## Record a decision

Capture the problem, alternatives, accepted choice, consequences, affected files, migration/rollback, evidence and approving reviewer. Add an OS ADR when the decision changes long-lived architecture. Do not amend Subcults merely to make its historical design match the new platform.
