# Venue and event access information worksheets

Issue #55 (`ACCESS-INFO`) now has a private venue and event source slice. The owner
expansion decision permits source development ahead of rollout prerequisites.
This does not establish venue conditions, partner demand, public accessibility
claims or accessibility-user evaluation. Those acceptance items remain open.

Owners open **Venue access worksheets** from the workspace to choose an existing
venue or create a minimal named reference. This owner-only index includes names
and IDs, never protected addresses or legacy access notes. Creating a name does
not assert facilities or publish a listing. Venue observations remain distinct
from event-specific observations. Neither old notes nor linked occurrence places
automatically populate either worksheet.

Owners open **Access worksheet** from the saved event editor, or use
`/events/{eventId}/access-info`. Six topics always have an explicit state:
step-free entry, step-free bathrooms, seating, sensory conditions, transit and
access contact. Unknown is the default and remains distinct from No or Not
available. Venue assertions are not inherited into this event worksheet.

A known value requires a bounded source reference, source kind and review time.
Source kinds are organizer assertion, external reference and either
venue observation or event-specific observation, according to worksheet scope. These labels describe the recorded provenance; none is
independent certification. Text topics also require conditions/details. Review
and optional review-by fields use UTC, preserving an unchanged hydrated instant.
Future review times are rejected. A review-by deadline must follow its review.

An elapsed explicit deadline makes the effective value Unknown and marks the
record as needing review. The original assertion and its provenance remain in
history. Review time alone does not expire a record. Worksheet checks and each
projected revision include the server evaluation time; Refresh information
checks current expiry and newer corrections. There is no background venue
verification or inference from old text, photographs or previous events.

## Occurrence venue comparison

The event worksheet includes **Compare venue and event information**. Owners
choose an occurrence and see its actual linked venue beside the event's six
recorded topics. Missing venue links and missing topic assertions remain
Unknown. Source kind, review time and expiry remain visible in each column;
expired original values retain their history/provenance and display effective
Unknown. A venue assertion is never promoted into the event worksheet.

The owner-only comparison reads occurrence time/status/revision, linked venue
name and both worksheets from one repeatable-read database snapshot, using one
expiry evaluation time. Owner membership is rechecked after that transaction
before returning private data. The occurrence selector has a separate minimal
owner-only endpoint; it excludes creator identity, description and public record
metadata. Responses are private and not cacheable.

**Refresh comparison** reloads the occurrence choices and both worksheets.
Selecting another occurrence immediately clears the prior comparison; late
responses cannot replace the newer selection. A failed refresh clears the old
comparison, and permission loss clears the entire parent worksheet, draft,
history and comparison. The displayed occurrence revision and comparison time
identify what was read. Conditions can change after that read.

This is a read-only comparison, not a saved occurrence-specific verification.
Event information remains event-wide. Comparing matching values does not prove
that those facilities or arrangements apply to this occurrence. There is no
copy, confirmation, certification or public sharing action in this view.

## Corrections and private boundaries

Each update appends a topic revision and an atomic audit entry. It requires the
exact current revision and a reason; concurrent corrections yield one new
revision and a conflict for the stale writer. Choosing Unknown withdraws the
current assertion and clears assertion details/source dates in the new record.
The prior record remains in owner-only history, twenty revisions per page.

A stable UUID request key binds scope, resource, topic, active owner and normalized
payload. Exact replay returns the original recorded revision, even after a later
correction. Derived expiry and evaluation time may advance. A changed binding
conflicts. The client retains the key after an uncertain response and does not
replace a newer displayed revision with an older replay.

Reads and writes require current owner membership. Writes lock the venue or event and
recheck the owner before committing. Read/replay/write responses recheck access
before returning private data and use `Cache-Control: private, no-store`.
Permission loss clears the worksheet, draft and history in the client. Request
keys, fingerprints and reviewer identity are excluded from response DTOs.

Personal accommodation requests are a separate future feature. This worksheet
has no request model, attendee linkage, diagnosis field, notification or public
sharing control. Do not enter personal requests in the plain-text source or
conditions fields. Individual requests remain private communication outside
this worksheet.

The worksheet does not enter the general Event DTO, anonymous event endpoints,
public archive serializer or AT Protocol records. There is no public display or automatic inheritance in this slice. Explicit
occurrence-scoped review of a pinned venue revision against an event arrangement,
public wording/correction routes, privacy/retention decisions for any future
personal request feature, and meaningful accessibility-user evaluation remain
unfinished #55 acceptance work.

## API and schema

- `GET /api/events/{eventId}/access-info/occurrences`: minimal owner-only
  occurrence choices, ordered by start time and ID.
- `GET /api/events/{eventId}/access-info/occurrences/{occurrenceId}/comparison`:
  the occurrence, event worksheet and nullable linked venue worksheet, from one
  snapshot with a shared `evaluatedAt`; wrong event/occurrence binding is 404.
- `GET /api/events/{eventId}/access-info`: six current event entries and server
  evaluation time; topics without history have revision zero and Unknown.
- `POST /api/events/{eventId}/access-info/{topic}`: append one private revision;
  `expectedRevision` and `requestKey` are required. Returns 201 for a new record,
  200 for exact replay, 409 for stale revision or changed request binding.
- `GET /api/events/{eventId}/access-info/{topic}/history?before={revision}`:
  descending history, twenty records per page and optional `nextBefore` cursor.

- `GET /api/workspaces/{workspaceId}/venue-access?after={placeId}`: owner-only
  venue names and IDs, at most 100 per page with an optional `nextAfter` cursor.
- `POST /api/workspaces/{workspaceId}/venue-access`: create a named reference
  with a required UUID `requestKey`. Exact owner/workspace/name replay returns
  the same venue; changed binding conflicts. Creation and audit are atomic.
- `/api/workspaces/{workspaceId}/places/{placeId}/access-info` and its topic
  update/history routes use the same revision contract with venue scope.

Migration 28 adds `event_access_revisions` without changing event, ticket, notice
or mail data. Migration 29 extends that ledger with an exclusive event-or-place scope,
separate place/topic revision sequences and source/scope checks. It preserves
existing event revision identities and adds the venue-reference replay ledger.
The physical table name is retained to preserve the earlier migration/history.
API and workers must use matching schema-29 binaries. Older
binaries reject the newer ledger; a rollback needs a qualified pre-migration
recovery path. Development backups and synthetic browser fixtures are local,
ignored artifacts. Automated checks use a separate disposable database.

Verification must cover validation, unknown and expiry semantics, owner/member/
outsider/revocation boundaries, exact/conflicting replay, concurrent correction,
audit rollback, history pagination, upgrade/replay preservation and public
absence. Full `make verify` and the disposable database gate remain required.
Browser evidence proves the synthetic editor journey; it does not establish
actual venue accessibility or usability with the intended audience.
