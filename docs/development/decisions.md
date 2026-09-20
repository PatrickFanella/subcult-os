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
| A6 | Production web and AT OAuth identity | Replace the legacy service at `subcults.subcult.tv` only after qualification; preserve its metadata/callback/JWKS URLs as a confidential web client and narrow requested scope to `atproto` | [AT kernel](atproto-kernel.md), [cutover runbook](../runbooks/subcults-cutover.md) |
| A7 | First AT dependency surface | Pinned Indigo syntax and OAuth packages stay behind the OS-owned adapter; extend only through drift-tested wrappers | [AT kernel](atproto-kernel.md) |

## Open

| ID | Decision | Recommended starting point | Blocks / evidence needed |
| --- | --- | --- | --- |
| D1 | OS ordered migration runner | Minimal embedded ordered runner with ledger, transaction/advisory lock and minimum-version startup gate | DB-01; fresh, replay/concurrency and failed-migration fixtures |
| D2 | Canonical account shape | Design verified email identities and rotating session families directly; update API/web/mobile together | IDENT-01; recovery and session threat review |
| D3 | Retained account migration | None unless a read-only inventory finds real accounts; then require fresh proof and never email equality alone | IDENT-01; data inventory, ambiguity and replay fixtures |
| D5 | Lexicon admission | Review each `tv.subcult.*` schema against current product journeys; begin with the smallest event/profile dependency chain | AT-01; namespace ownership, field and compatibility review |
| D6 | Public cultural model | Minimal Profile/Act, Place/Venue, Event occurrence and optional Tour/Appearance | MODEL-01; user journeys and existing OS event comparison |
| D7 | Operator/public event relationship | Private operator event owns operations; creator PDS owns published occurrence; private URI/CID state relates them | MODEL-01; time, location, cancellation and duplication cases |
| D8 | Publishing actor authority | Require workspace permission, creator/profile delegation and current scoped OAuth | PUB-01; revocation and cross-tenant negative fixtures |
| D9 | Public editing conflict policy | CID preconditions plus explicit human conflict resolution | PUB-01; concurrent client and timeout-after-write fixtures |
| D10 | Projection ingestion | Small restart-safe worker using accepted collections and quarantine | DISC-01; Tap/upstream evaluation and backfill fixture |
| D11 | License and provenance | Record exact source revision/path and adapted/generated status for every extraction | INV-01 and each extraction; contributor/dependency audit |
| D12 | Marketing integration | No attendance/contact inference or sending until purpose, verification and suppression contracts pass | CONSENT-01; no-consent/no-send and revocation cases |

## Record a decision

Capture the problem, alternatives, accepted choice, consequences, affected files, migration/rollback, evidence and approving reviewer. Add an OS ADR when the decision changes long-lived architecture. Do not amend Subcults merely to make its historical design match the new platform.
