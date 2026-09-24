# Cultural model (MODEL-01)

Status: data model, workspace-scoped CRUD API and public-preview projection
implemented 2026-09-23 (migration 000008). This implements the recommended
starting points for D6 (public cultural model) and D7 (operator/public event
relationship) in [`decisions.md`](decisions.md); it does not itself accept
either decision. Publication (writing to a PDS), projection/discovery
ingestion and any web/mobile UI remain open in DISC-01, PUB-01 and UX-01.

## Tables

All tables below are workspace-scoped and carry a composite foreign key back
to their parent's `(id, workspace_id)` pair, so a cross-workspace reference
is rejected by PostgreSQL itself, not only by the application layer.

- **`cultural_profiles`** — a creator or collective. `kind` is one of
  `creator`, `collective`, `act`; "act" is a profile kind rather than a
  separate table, per D6's recommended starting point. `public_uri`/
  `public_cid` are nullable and unset until a later publish.
- **`cultural_places`** — public place/venue fields only: `name`,
  `locality`, `region`, `country`, and `coordinates_public` plus
  `public_latitude`/`public_longitude` (both null unless
  `coordinates_public` is true; a check constraint enforces this pairing).
  These mirror `tv.subcult.place`'s admitted field set exactly.
- **`cultural_place_protected_details`** — one row per place, holding
  `street_address` and `access_notes`. This is a separate table, not just
  separate columns, so the public serializer's SQL never even joins it; the
  existing event `location_display` free-text field is left alone rather
  than duplicated, since places are a new, independently addressable
  concept the way `data-boundaries.md` and ADR 0007 describe.
- **`event_occurrences`** — a public occurrence that relates to exactly one
  existing private operator `events` row (D7). One operator event may relate
  to more than one occurrence (the same show cross-listed under two
  collectives); the foreign key is on `occurrence.event_id`, not the other
  direction. Carries `starts_at`/`ends_at` (`timestamptz`), `all_day`,
  `timezone` (IANA identifier, informational — the instant is always stored
  as UTC), `status` (`scheduled`/`rescheduled`/`postponed`/`cancelled`), an
  optional `place_id`, and nullable `public_uri`/`public_cid` for a later
  publish.
- **`event_occurrence_profiles`** — multi-host attribution: a join table
  from an occurrence to one or more profiles, each with a `role`
  (`host`/`performer`/`collective`) and a `sort_order` for display order.

## Public/protected split

A place's public columns (`cultural_places`) and protected columns
(`cultural_place_protected_details`) are physically separate tables. The
public projection code (`backend/internal/app/cultural_public_projection.go`)
only ever queries the public columns of `cultural_places`; it has no
reference to the protected table at all, so there is no field to
accidentally forget to strip. The operator-facing CRUD API
(`cultural_places.go`) does join both tables, since an authorized workspace
member is allowed to see and edit the protected fields.

## Time and DST

`starts_at`/`ends_at` are `timestamptz` (an absolute UTC instant);
`timezone` is an informational IANA identifier for the local wall-clock zone
attendees should read the instant in, matching the admitted
`tv.subcult.event.occurrence` fields. Because the instant, not a naive wall
time, is authoritative, a daylight-saving transition inside an occurrence's
span is represented correctly by construction:
`TestEventOccurrenceDSTChicagoRoundTrip` stores a 2026-03-07 23:00 to
2026-03-08 04:00 America/Chicago occurrence (crossing the US spring-forward
boundary), and asserts the round-tripped instant, the local wall-clock hours
in each zone abbreviation (CST before, CDT after), and the real elapsed
duration (4h, not the 5h a naive wall-clock subtraction would produce).

## Reschedule vs. tickets

Rescheduling an occurrence (`PATCH .../occurrences/{id}`) only ever updates
`event_occurrences` columns (name/description/place/time/status). Tickets
and reservations live in the `tickets` table, keyed by the private operator
`event_id`, which an occurrence update never touches — there is no code path
from the occurrence handler to `tickets` at all.
`TestEventOccurrenceRescheduleDoesNotChangeTickets` reserves a ticket, then
reschedules the occurrence, and asserts the ticket's id, code and status are
byte-for-byte unchanged. Updating `startsAt`/`endsAt` to a different instant
also flips `status` to `rescheduled` automatically (unless the caller sets an
explicit `status`).

## Ownership and cross-workspace rejection

Every handler loads the target row by ID first, then calls the existing
`requireWorkspaceRole(r, row.WorkspaceID, ...)` helper (the same one
`events.go`/`commitments.go` use) against the row's *actual* workspace, not
a workspace ID taken from the URL. Attaching a profile or place from a
different workspace to an occurrence, or creating an occurrence naming a
foreign-workspace place, is rejected at the application layer (400) and
would additionally fail at the database layer via the composite foreign
keys described above. `TestCulturalModelRejectsCrossWorkspaceAccess`
exercises both the read/write 403 case and the profile/place
cross-workspace-attachment 400 case.

## Public serializer and its sentinels

`backend/internal/app/cultural_public_projection.go` builds the exact
`tv.subcult.profile`, `tv.subcult.place` and `tv.subcult.event.occurrence`
record shapes from the public-only columns above, then validates each
through `atproto.ValidateAdmittedRecord` (the same validator
`lexicon_conformance_test.go` exercises), which runs both Indigo's
structural Lexicon validation and a recursive field-allowlist check. A
struct field that doesn't exist in the admitted schema cannot be added to
these record types without also updating the Lexicon contract, and an
extra/renamed field fails validation immediately.

`TestOccurrencePublicPreviewNeverLeaksProtectedOrOperationalData` seeds a
protected street address, protected access notes, a $42.00 paid ticket and a
staffing note, then asserts none of those literal strings (nor the ticket
amount) appear anywhere in the serialized `GET .../public-preview` response,
while the legitimately public venue/profile/occurrence names do appear.

### Pre-publication placeholder strong references

`tv.subcult.event.occurrence`'s `profile`/`place` fields are *required*
strong references (`uri` + `cid`) at the wire-format level. Nothing in this
slice writes to a PDS, so `public_uri`/`public_cid` are null for every row.
`strongRefFor` in `cultural_public_projection.go` falls back to a fixed,
schema-valid at-uri/cid pair (reused from
`contracts/atproto-lexicon.fixtures.json`) purely so `/public-preview` can
prove the projection is schema-shaped; it is documented in code as a
placeholder, never treated as a live reference, and the DTOs returned by the
CRUD endpoints keep `publicUri`/`publicCid` as `null` unless the row itself
has actually been published (which nothing here does).

## API surface

All routes require an authenticated session and workspace/event membership
via the existing `requireWorkspaceRole` gate:

- `GET/POST /api/workspaces/{workspaceID}/profiles`,
  `PATCH /api/workspaces/{workspaceID}/profiles/{profileID}`
- `GET/POST /api/workspaces/{workspaceID}/places`,
  `PATCH /api/workspaces/{workspaceID}/places/{placeID}`
- `GET/POST /api/events/{eventID}/occurrences`,
  `PATCH /api/events/{eventID}/occurrences/{occurrenceID}`
- `POST /api/events/{eventID}/occurrences/{occurrenceID}/credits`,
  `DELETE .../credits/{profileID}` (attach/detach multi-host credits)
- `GET /api/events/{eventID}/occurrences/{occurrenceID}/public-preview`
  (read-only; 400 if the occurrence has no credited profile yet, since
  `tv.subcult.event.occurrence.profile` is required)

No web or mobile UI ships in this slice; `contracts/api.schema.json` and
`web/src/domain.ts`/`mobile/src/api/types.ts` are unchanged because no DTO
was added to the shared contract schema.

## Known limits

- The admitted Lexicon catalog is compiled into the binary from
  `backend/internal/atproto/lexicons/`, a copy of `contracts/lexicons/`
  that `TestEmbeddedLexiconsMatchContracts` keeps byte-identical. Set
  `LEXICON_CONTRACT_DIR` only to validate against a different directory
  during development; production needs no contracts directory on disk.
- Publication itself (writing any record to a PDS), projection/ingestion
  into a discovery index, reconciliation, and any web/mobile UI for
  profiles/places/occurrences are all still open (DISC-01/PUB-01/UX-01).
