# AT record projection (DISC-01)

Status: allowlisted, restart-safe projection implemented 2026-09-23
(migration 000012). This implements the recommended starting point for D10
(projection ingestion) in [`decisions.md`](decisions.md); it does not itself
accept the decision. Publication (writing to a PDS, PUB-01) and any
web/mobile discovery UI (UX-01) remain open.

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

Migration `000012_at_projection.sql` adds three tables, none referenced by
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

## Known limits

- `JetstreamSource` does not implement ping/pong keepalive or batching; it
  relies on `Run`'s outer reconnect/backoff loop to recover from a dropped
  connection, but that loop retries `Next` on the *same* `websocket.Conn`
  rather than dialing a fresh one at the stored cursor. In production this
  means a connection drop is detected (every `Next` call after the drop
  errors) and backed off correctly, but recovery requires the process
  supervisor to restart the command rather than reconnecting in-process.
  Building a full in-process reconnect (fresh `JetstreamSource` per retry)
  is a reasonable follow-up; it is not required by the acceptance criteria
  above, which are about bounded backoff and no cursor loss, both of which
  hold today.
- No backfill/replay-from-arbitrary-cursor tooling beyond resuming from the
  single stored cursor; there is exactly one source row
  (`ProjectionSourceName = "jetstream"`).
- `JetstreamSource` is exercised by compile-time checks only (no test opens
  a real websocket); its correctness against a live Jetstream endpoint is
  unverified.
