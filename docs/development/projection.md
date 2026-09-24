# AT record projection (DISC-01)

Status: allowlisted, restart-safe projection implemented 2026-09-23
(migration 000011); backfill, rebuild, reconcile and metrics added
2026-09-24 (migration 000013; see below). This implements the recommended
starting point for D10 (projection ingestion) in
[`decisions.md`](decisions.md); it does not itself accept the decision.
Publication (writing to a PDS, PUB-01) and any web/mobile discovery UI
(UX-01) remain open.

## What this is, and is not

`at_projection_records` is a read-only mirror of remote AT Protocol records
for the three admitted collections (`tv.subcult.profile`, `tv.subcult.place`,
`tv.subcult.event.occurrence`; see [ADR 0007](../adr/0007-minimal-lexicon-admission.md)
and [lexicon-contract.md](lexicon-contract.md)). It is populated only by the
`atproto-project` worker consuming an external stream; nothing in the CRUD
API or the public-preview serializer described in
[cultural-model.md](cultural-model.md) reads or writes it, and the projector
never writes into any `cultural_*` table. The two record stores stay on
opposite sides of the trust boundary in
[data-boundaries.md](data-boundaries.md): `cultural_*` is this operator's own
private write path; `at_projection_*` is an untrusted external mirror kept
only for a later reconciliation/discovery decision (PUB-01/UX-01) that has
not been made yet.

## Adapter choice

Three ways to consume repo-level AT Protocol events were compared, scoped to
"an external consumer that only needs three collections":

1. **Raw firehose (`com.atproto.sync.subscribeRepos`)** — the base relay
   subscription. Frames are CBOR-encoded CAR slices containing every
   collection on every subscribed repo; a consumer must decode CAR/MST
   blocks and filter client-side. The pinned Indigo commit
   (`v0.0.0-20260903211445-41278964ec8e`) has the `atproto/repo` and
   `atproto/data` packages needed to decode this, but there is no built-in
   server-side collection filter, so 100% of relay traffic (every
   collection, every repo) would cross the wire and get decoded before this
   consumer could discard everything except three NSIDs. That is
   disproportionate to admitting exactly three collections, and CAR/MST
   decoding is meaningfully more surface to keep correct than JSON.
2. **Jetstream (`com.bsky.jetstream`-shaped JSON websocket)** — a public,
   widely deployed relay-adjacent service that re-emits firehose commits as
   flat JSON (`{"did", "time_us", "kind", "commit": {...}}` or
   `{"kind": "account", "account": {...}}`), and accepts a
   `wantedCollections` query parameter so the *server* drops everything this
   consumer does not need before it is ever sent. Its cursor is a
   `time_us` microsecond Unix timestamp passed back as a `cursor` query
   parameter to resume. The pinned Indigo commit does not ship a Jetstream
   client, but Jetstream's wire format is public and stable enough that a
   ~140-line adapter (`atproto_projection_jetstream.go`) is the entire
   integration surface; Indigo is not required for it at all.
3. **Tap** — checked for a supported client in the pinned Indigo commit;
   none exists. Ruled out for this consumer (nothing to pin, nothing to
   review).

**Choice: Jetstream.** It is the smallest adapter that lets an upstream
service do the collection filtering instead of this process, needs no
additional Indigo surface, and its cursor model (a single opaque
monotonically-increasing string) maps directly onto the
`at_projection_cursor` table's restart contract. The raw firehose is not
ruled out permanently — a future decision might need a collection Jetstream
does not carry, or might need direct relay control — but it is not the
smallest fit today.

The stream client sits behind a narrow interface so tests never open a
network connection:

```go
type StreamSource interface {
    Next(ctx context.Context) (StreamEvent, error)
}
```

`MemoryStreamSource` (`atproto_projection.go`) is the fixed, ordered
in-memory implementation every processor test uses. `JetstreamSource`
(`atproto_projection_jetstream.go`) is the only implementation that opens a
websocket, using `golang.org/x/net/websocket` (already an indirect
dependency of the pinned Indigo module; promoted to a direct `require` here,
not upgraded). It is exercised by build/compile only; no test dials a real
Jetstream endpoint.

## Persistence

Migration `000011_at_projection.sql` adds three tables, none referenced by
any existing code path:

- **`at_projection_records`** — `uri` primary key (`at://did/collection/rkey`),
  `did`, `collection`, `rkey`, `cid`, `rev`, `record` (`jsonb`, null for a
  tombstone), `size_bytes`, `status` (`active`/`deleted`/`unavailable`),
  `first_seen_at`, `updated_at`, `source_cursor` (the cursor value that most
  recently touched this row, for audit).
- **`at_projection_cursor`** — one row per source name (`jetstream` today),
  the last durably committed cursor and its timestamp.
- **`at_projection_quarantine`** — one row per rejected event: `uri` (when
  known), `reason`, a bounded/truncated `payload`, the event's `cursor`, and
  `created_at`.

## Processor semantics

`ProjectionProcessor.ProcessEvent` (`atproto_projection.go`) applies exactly
one `StreamEvent` inside a single database transaction that also advances
`at_projection_cursor`, so a crash between the record write and the cursor
update is impossible — either both land, or neither does (see
`TestProjectionCrashMidBatchThenReplayFromStoredCursor`, which injects a
failure after the row write but before commit and proves a replay from the
resumed cursor produces no duplicate and no missing row).

- **Collection allowlist**: an event for anything other than the three
  admitted NSIDs advances the cursor and returns `skipped_collection`
  without being read, stored, or quarantined
  (`TestProjectionAdmitsOnlyAllowlistedCollections`).
- **Lexicon validation**: `create`/`update` commits run through
  `atprotocol.ValidateAdmittedRecord`, the same two-pass structural-plus-
  allowlist validator MODEL-01 uses for inbound public data, so a record
  missing a required field or carrying an undeclared (potentially private)
  field is rejected the same way either path would reject it
  (`TestProjectionRejectsRecordFailingLexiconValidation`,
  `TestProjectionRejectsPrivateFieldLeak`).
- **Size bound**: a record over 64 KiB is quarantined without being parsed
  further (`TestProjectionRejectsOversizeRecord`); the quarantine `payload`
  itself is separately truncated to 4 KiB so an oversize or adversarial
  record cannot grow the quarantine table unbounded either.
- **Malformed JSON**: quarantined, not treated as a crash
  (`TestProjectionRejectsMalformedJSON`).
- **Duplicate replay**: an event with the same `(uri, cid)` as the currently
  stored row is a no-op for the record table, but the cursor still advances
  (`TestProjectionDuplicateReplayIsNoOp`).
- **Out-of-order updates**: an event whose `rev` sorts before the stored
  `rev` is ignored (the newer stored record is kept), and the cursor still
  advances (`TestProjectionOutOfOrderUpdateIsIgnored`). Jetstream's `rev`
  (TID) values are lexicographically ordered by creation time, so string
  comparison is sufficient without decoding.
- **Delete**: marks the row `deleted` and preserves `did`/`collection`/
  `rkey` provenance rather than removing the row, including when the delete
  arrives before any create was seen (a tombstone with no prior record;
  `TestProjectionDeleteMarksDeletedAndPreservesProvenance`,
  `TestProjectionDeleteBeforeCreatePreservesTombstone`).
- **Account state**: `deactivated`/`takendown`/`suspended` mark every record
  for that DID `unavailable`; `deleted` marks them `deleted` and is terminal
  — a later `active` event only reactivates rows this processor itself
  marked `unavailable`, never a row already `deleted`
  (`TestProjectionAccountUnavailableAndReactivation`,
  `TestProjectionAccountDeletionIsTerminalAndSurvivesReactivation`).
- **Reconnect**: `ProjectionProcessor.Run` retries a transient `Next` error
  with bounded exponential backoff (1s, 2s, 4s, ... capped at 30s by
  default), resuming from the last durably committed cursor; `MaxAttempts`
  bounds how many consecutive failures it tolerates before giving up and
  returning an error to its caller (`atproto-project`'s process supervisor
  is expected to restart it, exactly as the existing `-watch` workers rely
  on their own outer loop plus `restart: unless-stopped` in Compose).

## Runtime

`backend/cmd/atproto-project` follows the `atproto-revoke`/`email-deliver`
shape: default invocation prints aggregate, secret-free status (stored
cursor, record count, quarantine count) as JSON without opening a network
connection; `-run` explicitly starts consuming the configured stream and
requires `AT_PROJECTION_ENABLED=true`. Configuration is:

- `AT_PROJECTION_ENABLED` (default `false`) — gates `-run`, mirroring
  `MAIL_DELIVERY_ENABLED`.
- `AT_PROJECTION_SOURCE_URL` — the Jetstream `wss://` subscribe endpoint;
  required when `AT_PROJECTION_ENABLED` is true.

The opt-in Compose profile `atproto-projection` runs
`/app/atproto-project -run` with `restart: unless-stopped`, mirroring the
existing `atproto-workers` profile:

```sh
docker compose --profile atproto-projection up -d
```

## Backfill, rebuild, reconcile (recovery)

Status: added 2026-09-24 (migration `000013_at_projection_recovery.sql`).
This addresses the stream-only limits noted below: the stream has no
history/replay beyond its own retention, and this projector previously had
no way to recover a missed window, verify itself against the authoritative
source, or handle a PDS-side account migration or deletion.

### Approved authorities

`at_projection_authorities` (`did` primary key, `approved_by_person_id`,
`approved_at`, `revoked_at`, `note`) is the allowlist gate for backfill: the
otherwise-unbounded "fetch someone else's repo" capability only ever runs
against a DID an operator has explicitly approved and not revoked.
`ApproveProjectionAuthority`/`RevokeProjectionAuthority`/
`ListApprovedProjectionAuthorities` (`atproto_backfill.go`) manage it; the
`atproto-project -approve-authority did -approved-by person-id [-note text]`
and `-revoke-authority did` command flags are the only operator surface (no
new HTTP route; this mirrors how `-run` is already the only surface for the
stream).

### Backfill

`RunProjectionBackfill(ctx, db, catalog, lister, did)` lists all three
admitted collections from `did`'s PDS through `atproto.RecordLister`
(`com.atproto.repo.listRecords`, cursor-paged, bounded page size of 100
records, bounded to 5000 records per authority per invocation, and each
listed record is bounded to the same 64 KiB `RecordListMaxRecordBytes` the
stream path uses) and feeds every listed record through the same
`ProjectionProcessor.ProcessEvent` path the stream uses, so Lexicon
validation, the collection allowlist, and quarantine are shared rather than
reimplemented. It writes its own audit cursor under a distinct
`at_projection_cursor` source row (`"backfill"`, via
`NewProjectionProcessorWithSource`) so it can never advance or overwrite the
live `"jetstream"` resume cursor. After a full listing pass, any locally
stored `active` record for that `(did, collection)` the authority no longer
lists is marked `deleted`, preserving `did`/`collection`/`rkey` provenance
(`TestRunProjectionBackfillMarksMissingRecordsDeletedPreservingProvenance`).
`did` must be a currently-approved authority
(`TestRunProjectionBackfillRejectsUnapprovedAuthority`); backfill never
implicitly approves.

The production `atproto.RecordLister` is `IdentityRecordLister`
(`backend/internal/atproto/record_list.go`): like `IdentityRecordFetcher`,
it resolves the authority's PDS endpoint through the hardened identity
directory on every call, so an account migration to a new PDS host is
transparent — the URI stays `at://did/collection/rkey`, host-independent,
and provenance keeps the DID
(`TestRunProjectionBackfillAccountMigrationKeepsProvenance`). The same
public-only, no-proxy outbound transport (`ssrf.PublicOnlyTransport`) used
elsewhere in this package refuses a PDS endpoint that resolves to a
private, loopback, or link-local address, independent of what the identity
directory itself returns
(`TestIdentityRecordListerRefusesPrivatePDSEndpoint` in
`record_list_test.go`).

Every `listRecords` page is retried up to `backfillMaxListAttempts` (3)
times before the run is recorded `failed`
(`TestRunProjectionBackfillSourceOutageRecordsFailedRun`); each listed
record is still applied through its own `ProcessEvent` transaction, so an
outage partway through a collection never rolls back records already
committed, and no partial per-authority cursor is advanced beyond what
succeeded (there is no persisted inter-run resume point for backfill at
all — every backfill invocation is a fresh full listing).

A `RecordLister` may return `atproto.ErrProjectionCursorGap` for a resumed
listing it cannot honor. `RunProjectionBackfill` records that attempt as a
`gap` run and immediately retries once as a fresh listing, recording a
second run for the actual result
(`TestRunProjectionBackfillCursorGapTriggersFreshBackfill`). The bundled
`IdentityRecordLister` never returns this error itself (a listing always
restarts at `cursor=""` since backfill does not persist a per-authority
resume cursor); it exists for a future incremental backfill mode and for
fixture-driven tests today.

### Rebuild and compare

`RunProjectionRebuild(ctx, db, lister)` lists every three-collection record
from every approved, non-revoked authority into an in-memory shadow map
(never written to the database) and compares it against
`at_projection_records`, reporting `ProjectionDiff`: `missing_uris` (the
authority has it, the table does not), `extra_uris` (the table has an
active row the authority no longer lists), `cid_mismatch_uris` (both have
it, active, different CID) and `status_mismatch_uris` (the authority has it
active, the table's row exists but is not active). The diff carries URIs
only — URIs are public identifiers (`at://did/collection/rkey`); no record
body ever appears in a diff, a run's `counts`, or command output
(`TestRunProjectionRebuildReportsDiff`).

### Reconcile

`RunProjectionReconcile(ctx, db, catalog, lister)` rebuilds the same shadow
state and then applies it: `missing`/`cid_mismatch`/`status_mismatch` URIs
are written through `ProcessEvent` (so validation and the allowlist still
apply) and `extra` URIs are marked `deleted`. It only ever touches records
for currently-approved authorities, because the shadow state and the
comparison set are both scoped to `ListApprovedProjectionAuthorities`
(`TestRunProjectionReconcileAppliesDiffOnlyForApprovedAuthorities`).

### Run ledger

Every backfill/rebuild/reconcile invocation writes one
`at_projection_runs` row (`kind`, `authority` — null for a multi-authority
rebuild/reconcile pass, `started_at`, `finished_at`, `outcome` —
`running`/`completed`/`failed`/`gap`, `counts` — small integer aggregates
only, `error`). `LastProjectionRuns` returns the most recent row per kind
for status reporting.

### Metrics

`RunProjectionMetrics(ctx, db)` (surfaced as the default, no-flag
`atproto-project` output) reports: the stored jetstream cursor and its
derived `stream_lag_seconds` (now minus the cursor's `time_us` instant,
`nil` if the cursor is empty or not a Jetstream-shaped numeric string),
`record_count`, `quarantined_count`, `records_by_status` (grouped counts),
`last_runs` (one summary per kind), and `approved_authority_count`. No
record body, DID-scoped record content, or email address ever appears in
this structure or its JSON encoding
(`TestRunProjectionMetricsExcludesPrivateData`); this is not exposed as an
HTTP route in this change, since the command's own JSON output already
meets the "secret-free aggregate status" bar the rest of this worker's
surface uses.

## Known limits

- `JetstreamSource` does not implement ping/pong keepalive or batching.
  `RunWithConnector` detects a dropped connection when `Next` errors, closes
  the failed source, waits with bounded exponential backoff and dials a
  fresh `JetstreamSource` at the last committed cursor
  (`TestProjectionRunWithConnectorRedialsAtCommittedCursor`). A database
  error stops the loop instead of reconnecting.
- `JetstreamSource` is exercised by compile-time checks only (no test opens
  a real websocket); its correctness against a live Jetstream endpoint is
  unverified. The recovery tooling above does not change this; `IdentityRecordLister`
  is exercised only against `httptest` fixtures, never a live PDS.
- Backfill always performs a full (cursor-reset) listing per invocation;
  there is no persisted per-authority incremental resume cursor, so
  `ErrProjectionCursorGap` handling is exercised by fixture tests only —
  the bundled `IdentityRecordLister` never has an occasion to return it.
  Backfill is also not wired into stream-side gap detection: a stream
  outage longer than Jetstream's own retention window is not automatically
  detected or corrected by a backfill trigger; an operator must run
  `-backfill`/`-reconcile` manually today.
- `RunProjectionRebuild`/`RunProjectionReconcile` hold the entire shadow
  state for every approved authority in memory for the duration of one
  call; this is bounded per authority (`backfillMaxRecordsPerAuthority`,
  5000) but not bounded in aggregate across many authorities.
- Metrics are command-output only in this change; no session-gated
  operator HTTP route exposes them yet.
