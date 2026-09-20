# Subcults selective extraction inventory

Status: source-backed capability inventory. The completed [file-level manifest](subcults-extraction-manifest.md) supplies the INV-01 dispositions and provenance gate. Subcults remains read-only. An entry here is not approval to copy code, alter Subcults, migrate data, publish records, or enable a provider.

## Admission rule

A Subcults capability enters Subcult OS only when all of the following are recorded:

1. A current Subcult.tv user journey requires it.
2. The smallest useful contract is defined without importing unrelated product behavior.
3. Source ownership, license, generated-code status, and dependency implications are reviewed.
4. Public/private authority and failure behavior are explicit.
5. Focused fixtures can fail independently of the old application.
6. The extraction leaves Subcult OS runnable and preserves rollback.

Prefer reimplementation against a small contract when the old code is coupled to obsolete tables, routes, services, or product assumptions. Adapt source only when it is cohesive, rights-clear, and cheaper to verify than to rewrite. Preserve a provenance note either way.

## Priority candidates

| Capability | Source evidence | Starting disposition | Minimum accepted slice | Explicit exclusions |
| --- | --- | --- | --- | --- |
| Owned `tv.subcult.*` Lexicons | `lexicons/tv/`; `internal/atprotocol/lexicon.go` | Adapt schemas after namespace and field review | Lintable canonical schemas plus valid/invalid shared fixtures | Legacy `app.subcult.*`, speculative private records, unused fields |
| AT OAuth and identity linking | `internal/atprotocol/oauth_service.go`; OAuth store and tests | Reimplement behind an OS interface; adapt focused fixtures | Identity-only link, separately scoped publishing grant, revocation and status | Broad service impersonation, automatic DID ownership inference, PDS provisioning |
| Publication intent and reconciliation | `internal/atprotocol/publication.go`, `publication_store.go`, `reconcile.go`, `sync.go` | Extract algorithms and failure fixtures, not the surrounding service graph | Durable intent, payload digest, CID precondition, retry/replay, exact observed result | Publish-on-save, exactly-once claims, silent conflict resolution |
| DID, handle, URI, CID, and record-key validation | Existing Indigo-backed AT packages and local validators | Prefer pinned upstream Indigo APIs with local adapters | Bounded resolution and canonical syntax validation | Copying general-purpose protocol internals already maintained upstream |
| Tap/indexer operating model | `cmd/tap`, `cmd/indexer`, `internal/indexer`, sync checkpoints | Reference first; implement only the projection path required by accepted records | Restart-safe cursor, validated projection, quarantine, backfill fixture | Full historical app-view behavior, every old collection, early distributed deployment |
| Passwordless identity and session security | `internal/identity`; migrations 38 and 47 | Reimplement in OS migrations and auth module; adapt security cases | Verified email identity, encrypted lookup material, rotating session families, revoke-all | Wholesale user-table import, automatic merge by matching email, creator-approval UI |
| Ordered migrations and readiness gate | `migrations/`; `internal/db/schema_version.go` | Recreate a minimal OS-native runner and tests | Ledger, transactional ordering, fresh/upgrade/failure tests, minimum-version startup guard | Importing 47 historical migrations or pretending they describe OS state |
| Public/private location rules | Subcults `docs/adr/0005-privacy-first-location.md`, location access handlers and tests | Reuse policy and fixtures when a journey requires protected locations | Coarse public projection and explicit protected grant | Storing precise coordinates by default, importing legacy geo/search surface |
| Audience consent and suppression | `internal/audience`, `internal/signal` | Defer implementation; extract vocabulary and negative fixtures first | Purpose-scoped consent, revocation, suppression-before-send | Ticket attendance as marketing consent, autonomous sends, contact export |
| Touring cultural model | ADR 0007; `internal/touring`; relevant `tv.subcult.*` schemas | Redesign minimally around accepted journeys | Profile/Act, Place/Venue, Event occurrence, Tour/Appearance identities | Importing every repository, screen, query, or relationship before use |
| Protocol conformance corpus | Lexicon tests, publication fixtures, authoritative-PDS cases | Extract and simplify early | Language-neutral JSON/CAR fixtures checked by Go and TypeScript | Depending on the old API, database, UI, or private services |

## Reference only unless separately justified

- Privacy threat cases, OAuth failure cases, retry/replay cases, migration downgrade lessons, and operational runbooks.
- Product vocabulary and accepted public-data authority rationale from Subcults ADRs 0007 and 0008.
- Browser and provider fixtures that can be reduced to a current Subcult OS journey.
- Historical tests that reveal a still-relevant invariant. The invariant should be rewritten as a focused OS test; the old suite is not the deliverable.

## Excluded from the initial platform core

- The existing Subcults `cmd/api/main.go` composition and legacy route aliases.
- LiveKit streaming and stream analytics.
- Trust ranking, global reputation, automated moderation, alliances, and general social-post features.
- The old React application as a whole.
- The complete migration chain, database schema, generated coverage artifacts, deployment topology, and test suite.
- Duplicate payment, upload, notification, telemetry, or account implementations where Subcult OS already has an accepted boundary.
- PDS provisioning until operational capacity, abuse handling, recovery, and invitation controls pass their own gate.
- A microservice split, message broker, or independent AppView before measured load or isolation requirements justify it.

## Extraction record template

For each accepted extraction, record:

- capability and current user journey;
- exact Subcults source revision and paths inspected;
- disposition: `adapt source`, `rewrite from contract`, `fixture only`, or `reference only`;
- copied/generated files and provenance;
- dependencies added or deliberately avoided;
- data authority and sensitive-field classification;
- focused tests, cross-language fixtures, and broader OS checks;
- migration and rollback behavior; and
- remaining behavior intentionally not imported.
