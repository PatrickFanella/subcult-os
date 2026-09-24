# ADR 0007: Admit a Minimal `tv.subcult.*` Lexicon Chain for Event/Profile/Place

## Status

Proposed. This ADR does not accept D5. D5 (Lexicon admission) remains an open
decision belonging to the repository owner; merging this ADR (and the
accompanying contract and validators) is how the owner accepts it. Until then,
`tv.subcult.*` records described here are not published.

## Context

Subcult OS needs a public Lexicon for exactly three accepted journeys: a
public event occurrence with a public time and location, a creator/collective
profile, and a place/venue. [`architecture.md`](../development/architecture.md)
marks the "Culture to schema validator" contract as "Design only, blocked on
admitted Lexicon", and [`atproto-kernel.md`](../development/atproto-kernel.md)
states plainly that no `tv.subcult.*` Lexicon has been admitted.

`subcult.tv` is owned by the project owner, so `tv.subcult.*` NSIDs are
available to mint without cross-application collision risk.

### Existing schemas reviewed

- **`community.lexicon.calendar.event`** and **`community.lexicon.calendar.rsvp`**
  (Lexicon Community, `github.com/lexicon-community/lexicon`, MIT License,
  copyright 2026 Lexicon Community). `event` defines `name`, `description`,
  `createdAt`, `startsAt`, `endsAt`, a `mode` (in-person/virtual/hybrid) and
  `status` (planned/scheduled/rescheduled/cancelled/postponed) token, an array
  of `locations` (a union of an inline URI object or one of the location
  defs below), an array of associated `uris`, and `rsvpExpected`. `rsvp`
  references an event via `com.atproto.repo.strongRef` and carries a
  going/interested/notgoing `status`.
- **`community.lexicon.location.address`**, **`.geo`**, **`.fsq`**, **`.hthree`**
  (same repository and license). `address` is a street-level object
  (`street`, `locality`, `region`, `postalCode`, `country`, `name`) with no
  public/private distinction. `geo` is a `latitude`/`longitude` (as strings)
  pair, also with no bounds beyond string type. `fsq`/`hthree` are
  Foursquare-ID and H3-index location encodings, not reviewed further because
  neither is needed for a minimal venue reference.
- **`events.smokesignal.*`** (Smoke Signal). Search results describe
  `events.smokesignal.calendar.event` and `.rsvp` record shapes, but the
  stated source repository (`github.com/SmokeSignal-Events/lexicon`) returned
  HTTP 404 when fetched directly during this review, so its current schema
  text, license and exact field list could not be independently verified.
  Because the application-owned `events.smokesignal.*` namespace is not ours
  to mint into regardless, this gap does not block a recommendation; it is
  recorded so a future reviewer knows this source needs a working link before
  being relied on further.

### Assessment against the accepted journeys

The `community.lexicon` calendar/location schemas are real prior art for the
same problem and are freely reusable under their MIT license. They are not,
however, a drop-in fit for what this repository needs to admit:

1. **No profile record.** Neither `community.lexicon.calendar.*` nor
   `community.lexicon.location.*` defines a creator/collective profile
   record. A `tv.subcult.*` profile is required regardless of what is decided
   about event/location.
2. **No place/venue record with stable identity.** `community.lexicon`
   locations are inline value objects embedded in an event's `locations`
   array, not independently addressable records. This repository's operator
   model wants a place that can be referenced by strong reference from
   multiple occurrences and can carry its own provenance and evolution, which
   argues for a dedicated record.
3. **No bounds or public/private semantics.** `community.lexicon.calendar.event`
   places no `maxLength`/`maxGraphemes` bounds on `name`/`description`, no
   bound on the `locations` array, and no field distinguishing a place whose
   coordinates the operator has approved for public display from one that
   has not. `community.lexicon.location.address` includes `street`, which
   this repository's [data boundaries](../development/data-boundaries.md)
   treat as a private field that must never reach a public record.
4. **Open validation by design.** Reviewing the pinned Indigo `atproto/lexicon`
   package and the official `@atproto/lexicon` TypeScript package during this
   admission (see [the contract doc](../development/lexicon-contract.md))
   showed that official Lexicon validation intentionally accepts additive
   unknown fields on any object, by protocol design, for forward
   compatibility. Adopting a schema as-is does not, by itself, get us the
   "reject unknown private/operational fields at the public projection
   boundary" behavior `atproto-kernel.md` already commits to; that boundary
   has to be built regardless of which schema is admitted.

None of this is a criticism of `community.lexicon`; it is simply optimized
for general calendaring, not for a schema that must ship with strict field
allowlists and an explicit public/private location cut from day one.

### Recommendation

**Author a minimal `tv.subcult.*` chain**, directly informed by
`community.lexicon.calendar.event`/`.rsvp` and `community.lexicon.location.*`
(same overall record shape and time/location vocabulary) but not copied from
them line-for-line, so that:

- a profile record can exist at all (`tv.subcult.profile`);
- a place/venue is its own addressable, strongly-referenceable record with a
  narrow public-only field set (`tv.subcult.place`);
- the event occurrence (`tv.subcult.event.occurrence`) can declare explicit
  string/array bounds and a strict allowlist from the start, and reference
  the profile and place records by a self-contained strong reference rather
  than depending on an unbundled `com.atproto.repo.strongRef` lexicon.

RSVP, the `mode` (in-person/virtual/hybrid) axis, and multi-location events
are intentionally deferred; none of the three accepted journeys requires them
yet, and adding them later is an additive Lexicon change (new optional
fields or a new record type), not a breaking one.

## Decision

Admit the following record Lexicons, defined in `contracts/lexicons/`:

- `tv.subcult.profile` — minimal public creator/collective profile.
- `tv.subcult.place` — minimal public place/venue, public-location fields only.
- `tv.subcult.event.occurrence` — minimal public event occurrence referencing
  a profile (required) and a place (optional) by strong reference.

Field allowlists, bounds, time and location semantics, and the version
evolution rule are specified in
[`docs/development/lexicon-contract.md`](../development/lexicon-contract.md).
The same valid/invalid record corpus
(`contracts/atproto-lexicon.fixtures.json`) is validated by both the pinned
Indigo `atproto/lexicon` package (`backend/internal/atproto`) and the
official `@atproto/lexicon` TypeScript package (`web/src`), each additionally
enforcing the field-allowlist and explicit-datetime-offset boundary that the
official validators leave open by protocol design.

This ADR proposes admission; it does not itself flip D5 to Accepted. The
repository owner accepts D5 by merging this ADR (and the contract/validators
it describes), per the framing in issue #11.

## Consequences

- `docs/development/atproto-kernel.md`'s Lexicon boundary section and
  `docs/development/decisions.md`'s D5 row point here instead of restating
  the schema decision inline.
- The "Culture to schema validator" contract in `architecture.md` can move
  from "Design only" once this ADR is accepted, but publication, projection
  ingestion and reconciliation (D7–D10) remain separate, still-open work.
- Because `community.lexicon` was reviewed but not adopted wholesale, no
  MIT-licensed file was copied into this repository; the three admitted
  Lexicon JSON documents are original text.
- If Smoke Signal's lexicon repository becomes reachable again, it should be
  reviewed before any RSVP or multi-location extension is proposed, since its
  field choices may already have shaken out real interoperability lessons.
- No database, migration or runtime handler changed in this slice; admission
  is contract, validators and documentation only.

## Evidence

- `community.lexicon.calendar.event` and `.rsvp`, `community.lexicon.location.address`
  and `.geo` fetched from `raw.githubusercontent.com/lexicon-community/lexicon/main/...`
  on 2026-09-23; MIT License confirmed from the same repository's `LICENSE` file.
- `github.com/SmokeSignal-Events/lexicon` returned HTTP 404 on 2026-09-23;
  only indirect search-result descriptions of its record shape are available,
  and they are not relied on for any decision above.
- Indigo `atproto/lexicon` package inspected at the pinned commit
  (`github.com/bluesky-social/indigo@v0.0.0-20260903211445-41278964ec8e`) in
  the local module cache; confirmed present, confirmed `knownValues` is
  advisory-only (open) while `enum` is closed, and confirmed default
  `ValidateRecord` datetime parsing requires an explicit UTC offset.
- `@atproto/lexicon@0.7.6` inspected via `npm pack`; confirmed its default
  datetime format validator is lenient about a missing UTC offset, which is
  why the TypeScript validator in `web/src/atprotoLexiconConformance.test.ts`
  additionally enforces the explicit-offset policy itself.
