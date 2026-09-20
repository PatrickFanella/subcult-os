# Subcults file-level extraction manifest

Status: completed read-only audit for INV-01 on 2026-09-20. This manifest authorizes no copy, data migration, provider call, namespace publication or change to the Subcults checkout.

## Audited source and rights boundary

- Source checkout: `/home/onnwee/Work/subcult/subcults`
- Revision: `3cf88ec66dffa52160331ecf2e24aee29d66e741`
- Working tree at audit: clean
- Candidate scope: 71 Go/Markdown files in six packages, nine `tv.subcult.*` Lexicon files and six selected migration pairs/fixtures from a 100-file migration directory
- Generated-source check: no candidate Go or Lexicon file declares itself generated or `DO NOT EDIT`
- Commit-author evidence: Patrick Fanella/onnwee aliases plus GitHub Copilot bot identities appear in the scoped history
- License evidence: no root `LICENSE`, `COPYING` or `NOTICE` file was present within two levels of the checkout

The missing license file is a release gate, even when the same owner controls both repositories. Before copying source into a distributable package, confirm contributor rights and choose/document the destination license. Until then, use the old implementation as private reference and recreate behavior from independently stated contracts. Preserve the source revision and old path in each adapted fixture. Do not copy bot-produced or third-party-derived code without the same review.

## Disposition vocabulary

- **Adapt source** — a small cohesive file may be copied and changed after rights review.
- **Rewrite from contract** — preserve the named invariant/API intent, not implementation text or old storage model.
- **Fixture only** — reduce test cases to language-neutral inputs and expected outcomes with provenance.
- **Reference only** — use design lessons while implementing a new OS-native component.
- **Reject** — do not bring the capability into the accepted platform scope.

No file is currently approved for verbatim copying. “Adapt source” below is conditional on the rights gate.

## AT Protocol and Lexicons

| Old paths | Disposition | Product need and retained evidence | Coupling and deliberate exclusions | Destination verification |
| --- | --- | --- | --- | --- |
| `lexicons/tv/subcult/event.json`, `place.json`, `venue.json`, `profile.json` | Adapt schemas after field and rights review | AT-01 and MODEL-01 need a minimal public Event/Place/Venue/Profile vocabulary. Retain required-field, URI-reference, time and disclosure cases. | Existing Event requires a Place URI and carries host-scene and coarse-geohash assumptions. Do not publish the namespace or accept fields just because they already exist. | Canonical JSON fixtures validated by pinned Go Indigo and TypeScript `@atproto/lex`; public-location negative cases. |
| `lexicons/tv/subcult/act.json`, `tour.json`, `appearance.json` | Fixture only now; reconsider for MODEL-01 | Preserve identity/reference and appearance-time examples for the later touring journey. | The OS core does not yet have an accepted Tour/Appearance journey or model. | Deferred contract review plus cross-language fixtures before admission. |
| `lexicons/tv/subcult/scene.json`, `assertion.json` | Reject from initial kernel | They document earlier scene and provenance ambitions. | Scene is not required by the accepted first lifecycle; assertion imports reconciliation/governance complexity. | New backlog decision required to reopen. |
| `internal/atprotocol/lexicon.go`, `lexicon_test.go` | Rewrite from contract; fixture only for cases | Retain canonical-vs-legacy namespace separation, collection/scope allowlisting and protected-coordinate rejection. | Validation is hand-written around old record shapes and imports Indigo syntax directly; do not make it the canonical Lexicon validator. | Table-driven Go adapter tests plus the same valid/invalid JSON corpus in TypeScript. |
| `oauth_service.go`, `oauth_store.go`, `oauth_service_test.go`, `oauth_store_test.go` | Rewrite from contract; adapt only cryptographic fixtures after rights review | AT-01 needs state/PKCE flow integrity, safe local return paths, encrypted session material, expiry, replay rejection and explicit DID linking. | Concrete Indigo client/store types, `database/sql`, `lib/pq`, old `SQLStore` tables and old account assumptions. No publishing scope by default and no automatic authority inference. | Resolver/OAuth fake tests for state, nonce, PKCE, replay, expiry, unlink/revoke and cross-account DID claims. |
| `publication.go`, `publication_store.go`, `publication_test.go` | Reference only; fixture-only invariants for PUB-01 | Retain entity/collection matching, payload allowlist/digest, protected-coordinate negative case, lock/idempotency and observed-result concepts. | Coupled to Indigo atclient, Redis, PostgreSQL mapping tables and legacy entity types. Publication is not part of AT-01. | Delayed until PUB-01: timeout-after-write, stale CID, duplicate intent, revoked grant and restart/replay cases. |
| `reconcile.go`, `reconcile_test.go`, `sync.go` | Fixture only | Retain authoritative-PDS URL safety, public-IP classification, cursor/event envelope and reconciliation failure cases. | Old `SQLStore`, old record mappings and Tap envelope assumptions; network validation requires a fresh SSRF threat review. | Language-neutral unsafe-origin corpus; bounded authoritative fake; no live PDS write. |
| `provisioning.go`, `provision_store.go`, `provisioning_test.go` | Reject | Handle-normalization and trusted-client-IP cases may be noted in future PDS-hosting discovery. | PDS provisioning, abuse response, recovery and invitation operations are explicitly out of initial scope. | A new operational ADR and threat model are required before reconsideration. |

## Identity and session security

| Old paths | Disposition | Product need and retained evidence | Coupling and deliberate exclusions | Destination verification |
| --- | --- | --- | --- | --- |
| `internal/identity/crypto.go` | Adapt algorithm choices only after rights review; otherwise rewrite | IDENT-01 needs authenticated contact encryption, keyed lookup material and hashed bearer tokens. | Key lifecycle and ciphertext/version format must be OS-owned; ephemeral-key convenience is not acceptable for retained accounts. | Round trip, wrong key, tamper, normalized lookup and token non-retention tests. |
| `model.go`, `repository.go`, `memory_repository.go` | Rewrite from contract | Retain separate User/verified-email/session concepts and repository-test seams. | Creator-access workflow and old internal-DID assumptions are not canonical OS identity. UUID/storage choices must match pgx and OS migrations. | Repository contract tests against memory and disposable PostgreSQL implementations. |
| `service.go`, `service_test.go` | Fixture only plus rewrite | Retain one-time magic-link, external-return rejection, session rotation and enumeration-resistance cases. | Imports legacy `internal/auth`, mail flow and creator approval. No wholesale service transplant. | State-machine tests for verification, token consumption, expiry, rotation and ambiguous/cross-account claims. |
| `session_family_test.go`, `session_family_integration_test.go` | Adapt fixtures after rights review | Strong direct evidence for family rotation, reuse detection, revoke-one and revoke-all semantics. | Integration fixture depends on old migrations and `internal/testutil`. | Recreate against the OS migration ledger and pgx repository. |
| `sql_repository.go` | Reference only | Query ordering and atomic consume/rotate behavior inform the new repository. | `database/sql`, `lib/pq` arrays, old table names and migrations 38/47. | Transactional concurrency tests on the OS schema; no SQL copy by default. |
| `postmark.go` | Reject from identity core | None for the canonical data/session model. | Concrete email provider and development-link logging create separate provider/privacy policy. | Later notification/provider task with secret and log-redaction review. |

## Discovery and projection

| Old paths | Disposition | Product need and retained evidence | Coupling and deliberate exclusions | Destination verification |
| --- | --- | --- | --- | --- |
| `internal/indexer/v2_client.go`, `v2_client_test.go` | Fixture only; rewrite minimal consumer | DISC-01 needs fatal/recoverable stream classification, no cursor advance after projection failure and bounded reconnect behavior. | Concrete Bluesky Jetstream client and legacy metrics. Prefer current upstream Tap/Jetstream interfaces when DISC-01 starts. | Deterministic fake stream with restart, duplicate, gap and cancellation cases. |
| `v2_projection.go`, `v2_projection_test.go`, `v2_projection_integration_test.go` | Fixture only | Retain atomic record-plus-cursor, account suppression, quarantine and shadow-isolation cases. | Large PostgreSQL projection tied to old tables, legacy fallback and Jetstream event types. | OS-native accepted-collection projection tests in isolated schemas. |
| `filter.go`, `filter_test.go`, `filter_cbor_test.go` | Rewrite from reviewed Lexicons; fixture only for malformed inputs | Retain collection allowlisting, malformed/oversized record rejection and mixed-batch cases. | Old filter accepts legacy scene/post shapes and relies on hand validation. | AT-01 corpus is the only accepted record-shape authority. |
| `car.go`, `car_test.go`, `cbor.go`, `cbor_test.go` | Reference only | Demonstrates CAR/CBOR edge cases if a future ingestion source requires them. | General protocol parsing should come from pinned upstream libraries, not a local fork. | Upstream conformance fixtures plus local size/resource bounds if admitted. |
| `mapper.go`, `mapper_test.go` | Reject | Old mappings reveal privacy and timestamp mistakes worth remembering. | Hard imports of legacy alliance, geo, post and scene domains; none matches the accepted OS model. | New mappers only after MODEL-01, generated from accepted contracts where practical. |
| `repository.go`, `repository_test.go`, `testhelpers_test.go` | Reject implementation; fixture-only atomicity cases | Retain duplicate, cursor and failure/quarantine expectations. | Roughly the center of the 9k-line package, coupled to old schema, tracing, trust recomputation and `lib/pq`. | Small projection repository written for OS tables only. |
| `cleanup.go`, `cleanup_test.go`, `consistency.go`, `consistency_test.go`, `duplicate_test.go` | Reference only | Operational cleanup/consistency cases may become useful after a projection exists. | Premature before accepted ingestion and retention contracts. | Later measured operations task. |
| `metrics.go`, `metrics_test.go`, `handler.go`, `handler_test.go`, `README.md` | Reference only | Metric names and internal-auth failure cases are operational examples. | Prometheus surface and old service topology are not the product contract. | Define telemetry from OS SLOs when the consumer exists. |

The package is approximately 9,045 lines across 26 files and imports Jetstream, PostgreSQL, Prometheus, OpenTelemetry and four legacy domain packages. It is therefore explicitly not an extraction unit.

## Audience, signals and touring

| Old paths | Disposition | Product need and retained evidence | Coupling and deliberate exclusions | Destination verification |
| --- | --- | --- | --- | --- |
| `internal/audience/model.go`, `service_test.go` | Fixture only for CONSENT-01 | Retain purpose/sender/channel/scope consent, verification, disclosure-version and suppression-before-delivery negatives. | Attendance, a DID link or a relationship must never imply marketing consent. | Language-neutral consent decision table with no-send default. |
| `internal/audience/repository.go`, `service.go`, `sql_repository.go` | Reference only | Repository boundaries illustrate an auditable consent service. | Old schema and contact model are not yet accepted; implementation now would imply a delivery product. | Revisit only after CONSENT-01 contract approval. |
| all six `internal/signal/*.go` files | Reject from initial platform | Revision/idempotency and recheck-before-send tests are useful design notes. | Directly depends on audience and adds campaign/content/provider delivery not required by the first journey. | Separate proposal and explicit send authorization required. |
| `internal/touring/model.go`, `model_test.go` | Fixture only; rewrite minimal model | MODEL-01 can retain privacy-safe home territory, venue precision consent, time-window and appearance-kind cases. | File includes the full Place/Venue/Profile/Act/Tour/Appearance/Assertion aggregate, beyond current needs. | Admit one entity at a time against accepted journeys and public serializer sentinels. |
| `reconciliation.go` | Adapt small pure algorithm only after rights review | Ambiguity-preserving candidate classification may serve later imports. | No current ingestion journey requires it. | Pure deterministic tests; never mutate canonical events on ambiguous input. |
| `importer.go`, `importer_test.go` | Fixture only | Retain reject-extra-columns, source-assertion and ambiguous-afterparty cases. | CSV import and assertion persistence are not initial platform scope. | New import task must define provenance, preview and rollback. |
| `repository.go`, `repository_test.go`, `service.go`, `sql_repository.go` | Reference only | Tests document festivals, one-offs, multiple hosts, corrections and cursor caps. | Imports old scene model, uuid, `lib/pq` and broad search/repository behavior. | Fresh OS repositories after MODEL-01; no old table/API compatibility. |

## Migration evidence, not migration input

The old directory contains 100 migration files. None enters the OS migration chain. Only these lessons were audited:

| Old migration | Evidence retained | OS disposition |
| --- | --- | --- |
| `000038_public_beta_foundation.up.sql` | Email identities, one-time magic links, rotating sessions, roles, approval and protected-location grants were introduced together. | Rewrite only the IDENT-01 tables actually accepted; do not import internal DIDs, roles or creator approval by association. |
| `000041_atproto_canonical_publication.up.sql` | OAuth links/requests, provisioning, record mappings, checkpoints and failures were tightly combined. | Split identity linking (AT-01) from publication/provisioning (PUB-01/later). |
| `000042_atproto_sync_observations.up.sql` | Publication observations need durable history. | Reconsider only for PUB-01 reconciliation. |
| `000045_schema_version_ledger_repair.up.sql` | Historical ledger drift required a repair migration. | DB-01 uses an immutable checksum ledger and fails closed on binary/database mismatch. |
| `000046_jetstream_v2.up.sql` | Cursor, account, identity, reconciliation, shadow and comparison tables made migration/projection coupling explicit. | DISC-01 starts with one accepted projection and rebuildable state, not the shadow topology. |
| `000047_auth_session_families.up.sql` and its Go test | Session-family backfill, rotation and revocation semantics are security-relevant. | Recreate as fresh IDENT-01 migration/tests only if retained real accounts are inventoried. |

Down migrations were inspected as rollback history but are not evidence that an external PDS write, sent message or identity merge can be reversed.

## Approved extraction sequence

1. Keep Subcults read-only at the audited revision; re-audit if it moves.
2. Resolve license/contributor provenance before any source or fixture copy intended for distribution.
3. AT-01 creates a fresh package interface, pins only required upstream libraries and admits reviewed Lexicons/fixtures one at a time.
4. IDENT-01 creates OS-native migrations and repositories; no old database or account is assumed to exist.
5. MODEL-01 adds only journey-required cultural records.
6. DISC-01 rewrites a minimal consumer/projection from its failure contract; it does not import `internal/indexer`.
7. PUB-01 may reuse reduced publication/reconciliation fixtures only after authority and irreversible-write behavior are approved.
8. Every accepted artifact gets a provenance note naming this manifest, old revision/path, author/right review, modifications and tests.

## INV-01 conclusion

The audit found no package suitable for wholesale import and no evidence that prototype compatibility is required. The reusable value is concentrated in privacy, session-family, consent, stream-recovery and publication-failure invariants. Extract those as reviewed contracts and small fixtures; implement the platform against the current Subcult OS schema and journeys.
