# Subcult.tv platform architecture

Status: target architecture accepted by [ADR 0005](../adr/0005-subcult-os-platform-core.md). Source extraction and runtime consolidation are not yet implemented.

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

## Data authority

| Concept | Authority |
| --- | --- |
| Canonical platform account and session | Subcult OS PostgreSQL identity module |
| Workspace membership and operator role | Subcult OS PostgreSQL authorization/operations modules |
| Public Profile/Act/Scene/Place/Venue/Event/Tour/Appearance after publication | Creator PDS record |
| Unpublished public-record draft | Private Subcult OS culture/publication storage |
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

The current OS `people` and `sessions` tables are prototype evidence, not a compatibility contract. Design the canonical users, verified email identities, rotating session families, recovery, and additive AT links directly. Replace the prototype schema and update clients in the same bounded slice unless a read-only inventory identifies real account data that needs an explicit migration.

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
