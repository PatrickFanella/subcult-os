# Development brief

## Outcome

Subcult.tv becomes one coherent platform in the Subcult OS repository. An organizer uses one account and API to manage private event operations and deliberately publish portable cultural records. A participant can discover and reserve without gaining access to contacts, staffing, finances, attendance or consent records.

## Product responsibility

Subcult OS is the platform core: identity, workspaces, cultural records, event operations, tickets, door, settlement, archives, discovery, publication and consent-aware delivery. Creator PDS repositories remain authoritative for approved public AT records. PostgreSQL remains authoritative for private operations and provides rebuildable public projections.

Subcults is not the receiving application. It is a read-only source from which narrowly accepted capabilities and fixtures may be extracted under [ADR 0005](../adr/0005-subcult-os-platform-core.md) and the [extraction inventory](extraction-inventory.md).

## Initial consolidation slice

Establish a clean ordered OS schema and modular identity boundary. Inventory Subcults capabilities, then extract the smallest AT Protocol kernel: canonical Lexicons, syntax/identity validation, independently tested conformance fixtures, and identity-only OAuth linking. Do not begin with PDS provisioning, bulk data migration, a repository merge, or every old public feature.

After identity and migrations are qualified, introduce one private operator-event-to-public-occurrence relationship and a read-only validated projection. Publication follows only after local authority, scoped OAuth, preview, idempotency, CID conflict and reconciliation behavior are tested.

## Required success

- One platform account and session works across operator and public experiences.
- The new event/ticket lifecycle satisfies the accepted journeys without inheriting prototype identifiers or payload shapes.
- Accounts are not merged by matching email alone.
- A workspace actor cannot publish for an arbitrary DID, profile or organization.
- PDS or indexing outages do not prevent authorized local door operations.
- Delayed projection is visible as pending rather than reported as complete.
- Private contact, attendance, staffing, financial, archive and consent data never enters a public AT payload.
- Fresh install, migration replay/concurrency, failed migration, backup and restore are independently tested; populated legacy upgrade is required only for inventoried real data.
- No imported capability depends on the old Subcults application remaining online.

## Not in the first consolidated release

Wholesale Subcults import; old route aliases; streaming/LiveKit; trust ranking; alliances; social posts; general moderation automation; all historical migrations/tests; autonomous marketing; global reputation; new public PDS provisioning; production offline check-in; refunds expansion; native Subcults app; or simultaneous Patchwork/civic-suite integration.

## Stage gates

G0: accepted target architecture, reproducible baselines and public/private contract.
G1: ordered migrations plus selective-extraction inventory.
G2: unified identity foundation and cross-language protocol fixtures.
G3: read-only public occurrence projection in the unified API.
G4: explicitly authorized publication and reconciliation in disposable environments.
G5: browser/mobile rehearsal with a protected pilot workspace.
G6: separately approved production migration and qualification.

No date, funding deadline, or quantity of inherited tests bypasses a gate.

## Measurement

Track migration failures, duplicate or ambiguous account claims, rejected unauthorized publication, stale CID conflicts, projection lag, quarantine, orphaned mappings and organizer workflow time. Do not collect precise participant location or contact content in telemetry. These are engineering conditions, not growth forecasts.
