# Event access information worksheet

Issue #55 (`ACCESS-INFO`) now has a private event-level source slice. The owner
expansion decision permits source development ahead of rollout prerequisites.
This does not establish venue conditions, partner demand, public accessibility
claims or accessibility-user evaluation. Those acceptance items remain open.

Owners open **Access worksheet** from the saved event editor, or use
`/events/{eventId}/access-info`. Six topics always have an explicit state:
step-free entry, step-free bathrooms, seating, sensory conditions, transit and
access contact. Unknown is the default and remains distinct from No or Not
available. Venue assertions are not inherited into this event worksheet.

A known value requires a bounded source reference, source kind and review time.
The three source kinds are organizer assertion, event-specific observation and
external reference. These labels describe the recorded provenance; none is
independent certification. Text topics also require conditions/details. Review
and optional review-by fields use UTC, preserving an unchanged hydrated instant.
Future review times are rejected. A review-by deadline must follow its review.

An elapsed explicit deadline makes the effective value Unknown and marks the
record as needing review. The original assertion and its provenance remain in
history. Review time alone does not expire a record. Worksheet checks and each
projected revision include the server evaluation time; Refresh information
checks current expiry and newer corrections. There is no background venue
verification or inference from old text, photographs or previous events.

## Corrections and private boundaries

Each update appends a topic revision and an atomic audit entry. It requires the
exact current revision and a reason; concurrent corrections yield one new
revision and a conflict for the stale writer. Choosing Unknown withdraws the
current assertion and clears assertion details/source dates in the new record.
The prior record remains in owner-only history, twenty revisions per page.

A stable UUID request key binds event, topic, active owner and normalized
payload. Exact replay returns the original recorded revision, even after a later
correction. Derived expiry and evaluation time may advance. A changed binding
conflicts. The client retains the key after an uncertain response and does not
replace a newer displayed revision with an older replay.

Reads and writes require current owner membership. Writes lock the event and
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
public archive serializer or AT Protocol records. There is no public display or
venue-level ledger in this slice. Venue assertions versus event verification,
public wording/correction routes, privacy/retention decisions for any future
personal request feature, and meaningful accessibility-user evaluation remain
unfinished #55 acceptance work.

## API and schema

- `GET /api/events/{eventId}/access-info`: six current event entries and server
  evaluation time; topics without history have revision zero and Unknown.
- `POST /api/events/{eventId}/access-info/{topic}`: append one private revision;
  `expectedRevision` and `requestKey` are required. Returns 201 for a new record,
  200 for exact replay, 409 for stale revision or changed request binding.
- `GET /api/events/{eventId}/access-info/{topic}/history?before={revision}`:
  descending history, twenty records per page and optional `nextBefore` cursor.

Migration 28 adds `event_access_revisions` without changing event, ticket, notice
or mail data. API and workers must use matching schema-28 binaries. Older
binaries reject the newer ledger; a rollback needs a qualified pre-migration
recovery path. Development backups and synthetic browser fixtures are local,
ignored artifacts. Automated checks use a separate disposable database.

Verification must cover validation, unknown and expiry semantics, owner/member/
outsider/revocation boundaries, exact/conflicting replay, concurrent correction,
audit rollback, history pagination, upgrade/replay preservation and public
absence. Full `make verify` and the disposable database gate remain required.
Browser evidence proves the synthetic editor journey; it does not establish
actual venue accessibility or usability with the intended audience.
