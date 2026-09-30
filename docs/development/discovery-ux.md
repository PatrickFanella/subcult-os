# Anonymous cultural discovery and reservation handoff (UX-01)

Status: anonymous discovery API and a lightweight in-page web UI implemented
2026-09-24, over the existing DISC-01 projection
([`projection.md`](projection.md)). No migration; no schema change. This
implements the recommended starting point for discovering the AT record
projection from the public web app; publication (PUB-01) and any native
mobile UI remain open.

## What this is, and is not

This reads only `at_projection_records` (an untrusted external mirror; see
[`projection.md`](projection.md)) and, for reservation handoff resolution
only, `event_public_links` and the public-safe `events.status`/
`events.public_slug` columns already exposed by the existing anonymous
`/api/public/events*` routes. It never reads `cultural_*` (the operator's
own private write path from MODEL-01) and never reads any private table
(contacts, staffing, tickets, `cultural_place_protected_details`).

## API

- `GET /api/public/discovery/occurrences` — lists projected
  `tv.subcult.event.occurrence` records whose projection status is `active`
  (`deleted`/`unavailable` records are excluded), newest-updated first.
  Optional `locality` query parameter matches an occurrence whose `place`
  strong reference resolves to a projected place record whose own
  `locality` field equals the parameter (case-insensitive, exact match), or
  whose own `name` contains the parameter as a substring (case-insensitive);
  optional `limit` (default 20, max 100) and `offset` for pagination.
- `GET /api/public/discovery/occurrences/{uri...}` — one occurrence by its
  `at://` URI. The route captures everything after the `occurrences/`
  segment as a wildcard tail (`net/http`'s `{name...}` pattern keeps
  embedded slashes literal); the client sends the URI with its `at://`
  scheme stripped (for example
  `did:plc:xyz/tv.subcult.event.occurrence/abc`) so the URI's own slashes
  never need percent-encoding. Unlike the list route, this route still
  returns a `deleted`/`unavailable` record, with `projectionStatus` set
  accordingly, so a client can show "no longer available" instead of a bare
  404 for a occurrence it already had linked.

Both routes require no session (guest access).

### Response shape

```json
{
  "uri": "at://did:plc:.../tv.subcult.event.occurrence/abc",
  "source": { "did": "did:plc:...", "uri": "at://..." },
  "name": "Signal Night",
  "startsAt": "2026-10-01T20:00:00Z",
  "timezone": "America/Chicago",
  "status": "scheduled",
  "projectionStatus": "active",
  "location": { "name": "The Venue", "locality": "Chicago", "region": "IL", "country": "US", "latitude": "41.8781", "longitude": "-87.6298" },
  "handoff": { "kind": "local", "eventSlug": "signal-night", "reservationPath": "/api/public/events/signal-night/reservations" }
}
```

`source.handle` is declared in the DTO but always absent today: the current
`at_projection_records` schema (migration 000011) stores no handle column,
only the authority `did`; a later projection enrichment could populate it
without a contract change. `location` is a safe public projection built
only from the *projected* `tv.subcult.place` record's own JSON (name,
locality, region, country, and coarse public `coordinates` when the
Lexicon record carries them) — never from `cultural_places`/
`cultural_place_protected_details`, and the admitted `tv.subcult.place`
Lexicon has no street-address or access-notes field at all, so there is
nothing of that shape to leak by construction.

## Reservation/ticket handoff resolution

For each occurrence URI, `resolveDiscoveryHandoff`
(`backend/internal/app/public_discovery_occurrences.go`) looks up
`event_public_links` for that exact `public_uri`, most recently observed
first:

- No row → `{"kind": "none", "reason": "no_mapping"}`.
- A row whose `status` is not `fresh` or `changed` (`invalid`,
  `unavailable`, `deleted`) → `{"kind": "none", "reason": "mapping_<status>"}`.
- Otherwise, load the mapped local event: if it is not `published` or has
  no `public_slug` → `{"kind": "none", "reason": "event_not_published"}`
  (or `event_not_found` if the row is somehow gone).
- Otherwise → `{"kind": "local", "eventSlug": ..., "reservationPath":
  "/api/public/events/{slug}/reservations"}`, the same reservation route
  `tickets.go` already serves.

A missing, stale, invalid, unavailable or deleted mapping always returns
`kind: "none"` with a reason; it never falls back to a different
`event_public_links` row or a different event.
`TestDiscoveryHandoffResolvesLocalReservationAndNeverCrossesEvents`
(`backend/internal/app/public_discovery_occurrences_integration_test.go`)
plants two occurrences that deliberately share the same display title, each
correctly mapped to a different local event, and asserts each resolves only
to its own event's slug.

### External ticket handoff is not reachable in this slice

The admitted `tv.subcult.event.occurrence` Lexicon
(`contracts/lexicons/tv.subcult.event.occurrence.json`) declares no ticket
URL field at all — only `name`, `description`, `profile`, `place`,
`startsAt`, `endsAt`, `allDay`, `timezone`, `status`, `createdAt`. There is
therefore nothing for an operator-approved host allowlist
(`TICKET_HANDOFF_ALLOWED_HOSTS`) to gate: this slice implements only the
local reservation handoff described above. Adding an external ticket URL
would require a Lexicon change (a new admitted field, reviewed the same way
ADR 0007 reviewed the current three collections) before an allowlisted
external handoff could exist; that is out of scope here.

## Web UI

`web/src/views/DiscoverView.tsx` adds a `DiscoveryOccurrencesSection` below
the existing published-events grid:

- A list of occurrence cards (`role="button"`, keyboard-operable: Enter/Space
  opens the detail view, matching the existing card-grid's focus/keyboard
  conventions from `journeys.md`).
- A lightweight, dependency-free coordinate plot (`web/src/modules/discovery/discoveryOccurrenceModel.ts`'s
  `projectOccurrencesToPlot`): an inline SVG with one circle per occurrence
  that has public coordinates, using a simple equirectangular projection.
  This is a visual index into the list, not a navigable/tiled map, and adds
  no new mapping dependency.
- A detail view (`role="dialog"`) showing source (the `at://` URI), status,
  timezone-qualified time and safe location, plus the resolved handoff: a
  `Reserve` link to the local `/e/{slug}` route when `handoff.kind ===
  "local"`, or an explicit "no reservation destination" message when it is
  `none`. An occurrence whose `projectionStatus` is not `active` always
  shows the unavailable message instead of a handoff, even if a stale
  `event_public_links` row would otherwise resolve one. Escape closes the
  detail view (a `document`-level `keydown` listener); clicking the
  backdrop or the `Close` button also closes it. Guest access: no session
  is read or required anywhere in this component.
- Base list layout is single-column (`grid-cols-1`), moving to
  `sm:grid-cols-2`/`xl:grid-cols-3` only at wider breakpoints, so the
  360px-narrow case never depends on a multi-column layout existing.

### Occurrence time and themed interaction — 2026-09-30

Cards and detail dialogs convert `startsAt` into the recorded event `timezone`;
they do not attach the event-zone label to the viewer's local clock. The visible
label includes the event zone and offset at that instant, so the repeated 1:30
AM at a daylight-saving fall-back has distinct offsets. A missing or unsupported
zone displays the UTC instant with **event time zone unavailable**. Invalid
start timestamps retain their original text. This applies to projected occurrence
cards/detail, not the legacy local-event time display or all-day semantics.

Formatting uses the browser's locale and its supported time-zone data through
[`Intl.DateTimeFormat`](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/DateTimeFormat/DateTimeFormat).
The card focus outline and coordinate-point fill use semantic theme tokens;
dark mode displays a light outline and points against dark surfaces.

Regression checks cover event-zone date rollover, daylight-saving gaps/repeated
hours and unknown-zone fallback. The same 13 formatting/model tests passed under
Honolulu and Tokyo viewer zones. T3 Code on Linux (Chrome 152.0.7977.130,
Electron 44.4.2; 1402 × 876 CSS pixels) mounted the actual
`DiscoveryOccurrencesSection` with intercepted synthetic public-list responses:
Chicago and Tokyo cards/detail showed different event-local dates/times, an
invalid zone showed the UTC fallback, Enter opened detail, Escape closed it and
returned focus, and the dark keyboard focus outline/coordinate fill were read
from computed styles. The fixture root and fetch interception were removed and
the original page restored. This is component-fixture browser evidence; it does
not qualify an external projection feed, the complete discovery journey, native
mobile, all-day records or the remaining dialog accessibility behavior.

### Contracts

`PublicDiscoveryOccurrenceDTO` (and its nested `PublicDiscoverySourceDTO`/
`PublicDiscoveryLocationDTO`/`PublicDiscoveryHandoffDTO`) is declared in
`contracts/api.schema.json` with a `forbidden` list covering the private
field classes from `data-boundaries.md` (ticket/contact email, street
address, access notes, staff assignment, settlement detail, invitation
token, OAuth token, member role), and matching TypeScript interfaces exist
in both `web/src/domain.ts` and `mobile/src/api/types.ts` so
`scripts/check-contracts.mjs` enforces the same shape on both clients even
though only the web client renders it in this slice.

## Testing

- Go: `backend/internal/app/public_discovery_occurrences_integration_test.go`
  seeds real projection rows through `ProjectionProcessor.ProcessEvent` (the
  same validated ingestion path DISC-01 tests use, not a hand-inserted row)
  and exercises: list excludes deleted, detail still returns a deleted
  record with its status, safe location surfaces from the place record, a
  missing mapping returns `none`/`no_mapping`, a mapping made `unavailable`
  by refresh returns `none`, the two-occurrences-share-a-title handoff test
  described above, and a privacy-sentinel test
  (`TestDiscoveryRoutesNeverLeakPrivacySentinels`) that plants the shared
  `plantPrivacySentinels` fixture (contacts, staffing notes, ticket
  email/display name, protected place street address/access notes) and
  asserts none of those sentinels appear in either discovery route's
  response body.
- Web: `web/src/modules/discovery/discoveryOccurrenceModel.test.ts` (pure
  formatting/handoff/plot-projection logic) and
  `web/src/views/DiscoverView.discoveryOccurrences.test.tsx` (loading,
  empty, error, populated-list, keyboard open, click open, close, local
  handoff, unavailable-handoff, deleted/unavailable-projection, and
  coordinate-plot-omits-locationless-occurrences states), following this
  repository's existing convention (see `IdentityActionView.test.tsx`) of
  mocking React's hooks so a function component can be called directly and
  its returned element tree walked for handler props, rather than using
  jsdom/testing-library.

## Known limits

- The `document`-level Escape-to-close `keydown` listener and any real CSS
  media-query behavior at a 360px viewport are not exercised by the vitest
  component tests above, since this repository's existing component-test
  convention renders via `renderToString`/direct function-call rather than
  jsdom; both are verifiable only in a real browser and are not claimed to
  be covered here.
- List/detail location resolution does one extra query per occurrence to
  load its referenced place record (no batched join); acceptable at this
  slice's scale, a known limit for a larger catalog.
- `locality` filtering on the list route joins the occurrence's `place`
  strong reference to its projected place record and compares that place's
  own `locality` field (case-insensitive, exact match), with a fallback
  substring match on the occurrence's own `name`; it is not a geocoded or
  bounding-box query. No bounding-box parameter is implemented in this
  slice — the acceptance criterion's "optional bounding box or locality
  filter" is satisfied via locality only, and pagination is offset/limit.
- External ticket handoff is not reachable in this slice; see the section
  above. `TICKET_HANDOFF_ALLOWED_HOSTS` is not implemented since there is
  no Lexicon field for it to gate yet.
- `source.handle` is always absent; the projection schema stores no handle
  column today.
- No native mobile UI ships in this slice, only the matching TypeScript
  contract used by the contract checker.
