# Private event-to-public-occurrence links (LINK-01)

Status: data model, preview/attach/list/detach/refresh API implemented
2026-09-23 (migration 000009). This is an operator-private feature; it does
not add or change any public/anonymous endpoint or record.

## What this is

A workspace member who has published a `tv.subcult.event.occurrence` AT
record elsewhere (through PUB-01, another client, or by hand) can record,
inside their own workspace, which local private `events` row that public
record corresponds to. The relationship lives entirely on the operator
side: it is never projected into a public record and never read by the
anonymous public endpoints (`GET /api/public/events*`,
`GET .../occurrences/{id}/public-preview`). See
[`data-boundaries.md`](data-boundaries.md) for the general private/public
split this follows.

## Table

`event_public_links` (migration `000009_event_public_links.sql`):

- `workspace_id`, `event_id` — composite `(event_id, workspace_id)` foreign
  key back to `events (id, workspace_id)`, matching migration 000008's
  pattern, so a cross-workspace reference is rejected by PostgreSQL itself.
- `public_uri`, `observed_cid` — the exact AT-URI and CID observed at
  attach or refresh time. `unique (event_id, public_uri)` makes repeated
  attach of the same URI to the same event idempotent at the database
  level, not only in application code.
- `authority_did` — the repo DID of the identity that authored the public
  record, resolved and verified through `backend/internal/atproto` at
  attach/refresh time (not merely parsed out of the URI).
- `status` — one of `fresh`, `changed`, `unavailable`, `deleted`. Set by the
  refresh operation; a freshly attached link starts `fresh`.
- `last_error`, `observed_at`, `last_checked_at` — freshness bookkeeping
  updated only by refresh.
- `created_by_person_id` — the workspace member who attached the link.

Local operations never read this table on their hot path: event editing,
occurrence CRUD, ticketing and staffing queries have no join to
`event_public_links`, and a change to this table's `status` changes nothing
about the local event. A link whose public record changed, became
unavailable, or was deleted is only ever a review signal surfaced through
this table's own rows; it cannot silently rewrite operational facts (per
`data-boundaries.md`).

## API

All routes are workspace-scoped and go through the existing
`requireWorkspaceRole(r, event.WorkspaceID, "owner", "member")` gate used by
the MODEL-01 cultural routes; a person who is not a member of the event's
workspace gets `403 Forbidden`, matching those routes exactly (this
repository authorizes by loading the event first and checking role by its
resolved workspace, not by returning a workspace-conditioned 404).

- `POST /api/events/{eventID}/public-links/preview` — body `{"publicUri"}`.
  Parses the URI (must be a DID-authority `tv.subcult.event.occurrence`
  record URI), resolves the authority's identity and fetches the record
  through the injectable `App.recordFetcher`
  (`atprotocol.RecordFetcher`/`IdentityRecordFetcher`), which applies the
  same public-only outbound-policy HTTP client and hardened identity
  directory as AT OAuth (`backend/internal/atproto`). The fetched record is
  validated against the admitted Lexicon catalog
  (`atprotocol.ValidateAdmittedRecord`) before any field is trusted.
  Returns the source identity (DID/handle), CID and the event-relevant
  fields (`name`, `description`, `startsAt`, `endsAt`, `allDay`,
  `timezone`, `status`). **Never persists anything.**
- `POST /api/events/{eventID}/public-links` — body `{"publicUri"}`. Runs the
  same resolve-and-validate path as preview, then persists. Repeating the
  same `(event, publicUri)` attach is idempotent: an
  `on conflict (event_id, public_uri) do update` no-op returns the
  pre-existing row (`200`) rather than inserting a duplicate or silently
  re-observing a new CID; use refresh to update freshness deliberately.
- `GET /api/events/{eventID}/public-links` — lists links for the event,
  newest first.
- `DELETE /api/events/{eventID}/public-links/{linkID}` — detaches; `404` if
  the link does not belong to that event.
- `POST /api/events/{eventID}/public-links/{linkID}/refresh` — re-fetches
  the stored `public_uri` through the same fetcher and updates only this
  row's `status`/`observed_cid`/`last_checked_at`/`last_error` columns:
  - fetch succeeds with the same CID → `fresh`;
  - fetch succeeds with a different CID → `changed`, `observed_cid` and
    `observed_at` move to the new value;
  - fetch fails with `atprotocol.RecordNotFoundError` → `deleted`;
  - fetch fails any other way (identity resolution failure, network
    failure, non-200 upstream) → `unavailable`, with `last_error` set to
    the failure detail. `observed_cid` is left untouched in both failure
    cases, since nothing new was actually observed.

Every attach/detach/refresh is recorded through the existing
`App.audit` mechanism (`event_public_link.attached` /
`.detached` / `.refreshed`).

## Fetcher boundary

`backend/internal/atproto/record_fetch.go` adds `RecordFetcher`, an
interface with a single production implementation,
`IdentityRecordFetcher`, that resolves the URI's authority through a
hardened `identity.Directory` and issues one
`com.atproto.repo.getRecord` request against the resolved PDS endpoint
using the same `publicOnlyHTTPClient`/`hardenIdentityDirectory` helpers AT
OAuth already uses. `App.recordFetcher` is set to
`atprotocol.NewIdentityRecordFetcher()` in production and swapped for a
fixture implementation in tests
(`backend/internal/app/event_public_links_integration_test.go`); no test in
this repository makes a live network call to exercise this feature.

## Known limits

- No UI. This slice is API-only.
- `authority_did` is trusted from the resolved identity at attach/refresh
  time; it is not re-verified against any later identity rotation until
  the next refresh.
- Refresh is manual (an explicit `POST .../refresh`); there is no
  background scheduler that refreshes links automatically.
- The preview/attach `publicUri` must reference a
  `tv.subcult.event.occurrence` record; other admitted collections are
  rejected, matching this slice's scope.
