# Subcult.tv platform architecture

Status: target architecture accepted by [ADR 0005](../adr/0005-subcult-os-platform-core.md). The receiving Go API, canonical identity schema and narrow AT adapter are implemented in the review stack. Public cultural schemas, projection and publication remain planned; production consolidation is not deployed.

## Direction

Subcult OS is the receiving repository and becomes the Subcult.tv platform core. Subcults is a read-only source of selected protocol, identity, privacy, migration, domain-model, and test-fixture knowledge. There is no wholesale repository merge and no permanent service bridge.

The target is a Go modular monolith with one supported API, one identity authority, one PostgreSQL database, and separately runnable background workers only where operationally necessary. TypeScript remains the web/mobile language and supplies an independent AT Protocol conformance path through `@atproto/lex`.

## Bounded modules

| Module | Owns | Must not own |
| --- | --- | --- |
| Identity | Canonical users, verified email identities, sessions, recovery, additive DID links | Workspace authority, creator delegation inferred from email, public profile content |
| Authorization | Platform roles, workspace membership, creator/profile delegation policy | Authentication secrets or protocol payloads |
| Culture | Profiles/Acts, Scenes, Places/Venues, Events, Tours/Appearances and public drafts | Tickets, contacts, staffing, settlement, consent receipts |
| Operations | Workspaces, contacts, commitments, staffing, private event plan and archive | Canonical public AT records or marketing permission |
| Ticketing | Inventory, reservation/purchase, ticket codes, admission and check-in | Public cultural authorship or audience consent |
| Publication | Approved public projection, OAuth authority, durable intents and reconciliation | Private operational DTO serialization or implicit publish-on-save |
| Discovery | Validated public projections, search and public availability links | Public-record authorship or private operational authority |
| Audience | Purpose-scoped consent, verification, revocation and suppression | Permission inferred from attendance, membership, follow, or purchase |
| Delivery | Transactional outbox, reminders and approved signal delivery | Deciding consent or public authority |

Dependencies point inward through small interfaces. HTTP handlers validate external input and call application services; they do not coordinate unrelated repositories directly. AT Protocol and provider clients are adapters around explicit ports.

## Current seams and future contracts

Keep the existing operations handlers in `backend/internal/app` until a concrete
caller needs a separate service. Do not manufacture empty packages/interfaces to
make the target module table look implemented. Identity has one session authority
in that package; browser cookies and native headers adapt the same persisted
family lifecycle. `backend/internal/atproto` owns unstable Indigo types, syntax,
OAuth network policy and encrypted store behavior.

| Boundary | Contract and failure behavior | State |
| --- | --- | --- |
| Identity to HTTP | Verified canonical person and session family, never raw credentials in JSON; failed proof grants no identity or workspace membership | Implemented |
| App to AT link flow | `StartLink(context, personID, identifier)` and `CompleteLink(context, callbackValues)`; bounded external failure is reported only by the AT action | Implemented `atprotoLinkFlow` |
| Revocation worker to provider | One decrypted persisted session passed to a bounded revoke operation; sanitized error class determines retry/quarantine; lease token fences acknowledgement | Implemented adapter/worker seam |
| Culture to schema validator | One versioned public record and its declared type; return validated public fields or bounded field errors; never accept a private event DTO as a record | Contract admitted (ADR 0007); MODEL-01 exercises this: `GET .../occurrences/{id}/public-preview` builds `tv.subcult.event.occurrence`/`.place`/`.profile` records from allowlisted columns and runs `atproto.ValidateAdmittedRecord` before returning them. Still blocked on D5/ADR 0007 acceptance for any real publish path, and the current production Docker image does not ship `contracts/lexicons`, so this endpoint is dev/test-exercised, not production-deployed |
| Projection to ingestion | Validated URI/CID/source observation plus an upsert or tombstone; atomically commit projection with checkpoint, or retain checkpoint and quarantine/retry | Design only, no consumer or projection tables |
| Operations to publication | Actor, workspace, local event revision, explicit public-field preview/digest, destination authority and expected CID; return a durable intent ID and pending/confirmed/conflict/unknown/failed state | Design only, no PDS write |
| Publication to reconciliation | Read authoritative record identity/revision after ambiguous writes; reconcile the original intent without creating another occurrence or silently changing local ticketed facts | Design only |

Future contracts must be implemented alongside their first real caller and shared
client DTOs, not as unused compatibility scaffolding. The public-record allowlist,
time/place bounds and evolution rules are admitted in #11 before these planned
contracts become executable. A schema's shape is not consent or authorship proof.

### Local event operation during protocol failure

The database-backed `TestATProviderFailureDoesNotInterruptLocalEventLifecycle`
injects failure at the existing AT flow seam, observes the failed authorization
response, and then executes the same create/publish/free-reserve/duplicate-check-in/
closeout assertions used by the unlinked lifecycle test. It proves one provider
attempt, no DID link, and one settlement/archive. No local event operation calls
the AT flow. This is application-boundary failure injection, not a real-provider
outage, host network-partition test or offline mobile qualification.

## Data authority

| Concept | Authority |
| --- | --- |
| Canonical platform account and session | Subcult OS PostgreSQL identity module |
| Workspace membership and operator role | Subcult OS PostgreSQL authorization/operations modules |
| Public Profile/Act/Scene/Place/Venue/Event/Tour/Appearance after publication | Creator PDS record |
| Unpublished public-record draft | Private Subcult OS culture/publication storage: `cultural_profiles` (a `kind` column of `creator`/`collective`/`act` stands in for a separate Act table, per D6), `cultural_places` (public fields only) plus `cultural_place_protected_details` (street address/access notes, never read by the public serializer), and `event_occurrences`/`event_occurrence_profiles` (multi-host credits). See [cultural-model.md](cultural-model.md). `public_uri`/`public_cid` columns on these tables stay null until a later PUB-01 publish; nothing in MODEL-01 writes to a PDS |
| Validated public discovery representation | Rebuildable PostgreSQL projection with URI, CID and observation state |
| Event plan, contacts, commitments, staffing, tickets, door, settlement and archive | Private Subcult OS PostgreSQL records |
| Consent, verification, revocation and suppression | Private Subcult OS audience ledger |

One database does not mean every module may write every table. Each table has one owning module. Cross-module reads use application interfaces or deliberately reviewed query projections. Cross-module writes go through the owning service.

## Event identity

A public cultural occurrence and a private operator event are related but not identical.

- The operator event remains usable before publication and during PDS outages.
- Publishing creates or updates an approved public occurrence through a durable intent.
- The operator event stores the public AT URI, expected/observed CID and reconciliation state in private mapping state.
- Public time or venue divergence is surfaced for explicit review; it does not silently rewrite ticketed operational facts.
- Deleting a public record does not erase tickets, financial records, audit history or the private archive.

## Identity consolidation

The accepted identity slice uses `people` as the canonical person/operational
projection, with protected verified `email_identities`, one-time challenges,
rotating `identity_sessions` and additive `did_links`. Migration 3 removes the
prototype `sessions` table. Those migrations reject populated prototype account/
session tables; there is no authorized retained-account importer or dual authority.
Web and native transports share the same canonical identity model.

If any existing account data is later designated for migration, accounts must not be merged by email equality. Linking requires current proof such as a fresh verification challenge or another reviewed protocol proof. A DID link proves control of that AT identity; it does not grant a workspace membership or creator delegation automatically.

## Prototype compatibility

[ADR 0006](../adr/0006-no-prototype-compatibility-contract.md) permits deliberate breaking changes. Existing route names, DTO shapes, database identifiers, password sessions and client flows may be replaced rather than carried through compatibility layers. Preserve privacy, authorization, protocol, consent and recovery invariants; preserve data only when a read-only inventory identifies a real retention requirement.

## AT Protocol implementation

Production protocol code remains Go behind the narrow [AT Protocol kernel](atproto-kernel.md). Pin Indigo dependencies, own error/retry behavior locally, and keep admitted Lexicons as the schema source of truth. Syntax fixtures run against Indigo and official TypeScript `@atproto/syntax`; once a Lexicon is independently authored and approved, its record fixtures must also run against Indigo Lexicon validation and official TypeScript `@atproto/lex` so protocol drift is visible.

Public writes require both local authorization and current scoped OAuth authority. Use payload digests, idempotency keys, CID preconditions, durable reconciliation, bounded retry and quarantine. A service credential alone never proves creator permission.

## Extraction discipline

Use the [selective extraction inventory](extraction-inventory.md). Do not import Subcults' API composition, complete schema, migrations, test suite, frontend, deployment topology, or unused features. Each capability is admitted independently as adapted source, contract rewrite, fixtures only, or reference only.

## Deployment shape

Start with one API and one database. Tap/indexing, outbox delivery, and backfill may be separate Go commands while sharing contracts and migrations. Do not add a broker, distributed transaction, or microservice boundary before an evidenced reliability or scaling need.
