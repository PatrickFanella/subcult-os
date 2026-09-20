# Source baseline
Inspection date: 2026-09-20. Evidence tier: repository source and checked-in documentation, not fresh runtime or production audit.

| Repository | Checkout / revision | Observed boundary |
| --- | --- | --- |
| Subcult OS | Current task worktree; abf3f500364d2cce60879c449cb6711346cd675f | Go API, React/Vite web, Expo mobile, PostgreSQL |
| Subcults | /home/onnwee/Work/subcult/subcults; 3cf88ec66dffa52160331ecf2e24aee29d66e741 | Read-only extraction source: Go services, React web, public AT authoring/projection and historic private operations |

Subcults branch: fix/main-regression-recovery, clean at inspection. OS contained the earlier user-requested uncommitted research/funding documents; preserve them. These hashes identify source snapshots, not the deployed artifact.

Architecture direction after inspection: OS is the receiving repository. Subcults' size, historical test surface and existing feature breadth are not treated as requirements. See [ADR 0005](../adr/0005-subcult-os-platform-core.md) and the [extraction inventory](extraction-inventory.md).

## OS anchors
- backend/internal/app/app.go: current route registration.
- backend/internal/app/events.go: Event DTO, creation/edit/publication, report/settlement/archive paths.
- backend/internal/app/event_lifecycle.go and event_lifecycle_test.go: lifecycle rules and tests.
- backend/internal/app/public_discovery.go and discovery_test.go: public discovery seam.
- backend/internal/app/schema.sql, migrations/, and db.go: version-1 baseline plus the ordered checksum-ledger runner established by DB-01.
- contracts/api.schema.json, scripts/check-contracts.mjs, web/src/domain.ts, mobile/src/api/types.ts: cross-client DTO checks.
- web/src/modules/eventEditor and mobile/src/modules/events: editor models.
- README.md, CONTEXT.md, docs/adr/0004-event-discovery-ahead-of-first-cut.md: vocabulary and discovery scope.
- docs/runbooks/database-migrations.md: ordered migrations required before consequential external-data evolution.

BASE-01 confirmed that the anonymous event-detail handler embedded the operator Event DTO and exposed `workspaceId`, capacity, reservation/check-in counts and staffing-count keys. API-01 now uses an explicit public projection. The contract checker rejects named forbidden properties in the web/mobile public DTOs, and Go tests check the serialized response. It remains a lightweight source-contract checker, not general JSON Schema validation or exact cross-language type generation.

## Subcults anchors
- docs/adr/0007-scene-signals-touring-relationship-model.md: accepted domain and consent separation.
- docs/adr/0008-atproto-canonical-public-data.md: creator PDS canonical public data; PostgreSQL private state and validated projection; additive DID linking.
- internal/atprotocol/publication.go, publication_store.go, reconcile.go, sync.go, oauth_service.go: existing protocol boundary.
- internal/touring/sql_repository.go, internal/audience/service.go, internal/signal/delivery.go: cultural operations and delivery.
- lexicons/README.md: existing owned canonical namespace; reuse before inventing schemas.
- docs/product/PUBLIC_BETA_RELEASE_STATUS.md: dated release contract and unclosed qualification requirements.

The release status is dated August 9, 2026. BASE-01 freshly passed selected AT/touring/audience/Signal tests, Lexicon lint, frontend i18n/lint/build and mocked deploy recovery at the recorded clean revision. The document's full-Vitest failure counts remain historical. It also references schema 40/44 evidence while the checkout contains migrations 0–47 and requires schema 47. Full Go/race, PostGIS, browser, PDS/provider, backup and parity gates remain open. Do not infer readiness from documents named COMPLETE.

## Commands are not equivalent
OS make verify installs dependencies, formats/checks, lints, checks contracts, tests, builds and validates Compose/templates. Its fmt prerequisite can modify Go files.
Subcults make verify runs only go mod verify. A passing result there does not establish application correctness.
See [verification](verification.md) before interpreting any green command.
