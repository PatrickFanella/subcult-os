# Admitted Lexicon contract

Status: proposed by [ADR 0007](../adr/0007-minimal-lexicon-admission.md).
D5 in [`decisions.md`](decisions.md) remains open until the repository owner
accepts that ADR. This document describes the three record Lexicons under
`contracts/lexicons/` and the corpus at `contracts/atproto-lexicon.fixtures.json`
that both the Go and TypeScript validators run.

## Why a wire-format allowlist is enforced outside the Lexicon schema

Official Lexicon validation (both the pinned Indigo `atproto/lexicon`
package and the official `@atproto/lexicon` TypeScript package) intentionally
allows additive, undeclared object fields to pass validation; this is a
deliberate protocol design choice for forward compatibility, not a bug in
either library. `docs/development/atproto-kernel.md`'s Lexicon boundary
already commits to rejecting unknown private/operational fields "at the
public projection boundary" rather than relying on the wire schema for that.
Both validators in this repository (`backend/internal/atproto/lexicon.go`
and `web/src/atprotoLexiconConformance.test.ts`) therefore run two checks per
record: the official structural validation, then a recursive walk that
rejects any object key the admitted schema's own `properties` map did not
declare. The allowlist is derived directly from the same Lexicon JSON
documents, not maintained as a second hand-written list.

The two implementations also disagree on `datetime` strictness: Indigo
requires an explicit UTC offset by default, `@atproto/lexicon` does not. This
repository's own public time policy (below) requires the explicit offset, so
the TypeScript validator adds the same regex check Indigo already applies by
default, instead of relying on the SDK default.

## `tv.subcult.profile`

File: `contracts/lexicons/tv.subcult.profile.json`. Record key: `literal:self`
(one profile record per repository, matching the `app.bsky.actor.profile`
convention).

| Field | Required | Type / bounds |
| --- | --- | --- |
| `displayName` | yes | string, 1-100 graphemes, max 500 bytes |
| `description` | no | string, 0-500 graphemes, max 3000 bytes |
| `createdAt` | yes | datetime, RFC 3339 UTC instant with explicit offset |

No other field is admitted. Contact details, workspace membership, role and
any operational identifier are never present; those stay in the private
Identity/Operations tables per
[`data-boundaries.md`](data-boundaries.md#private-classes) and
[`architecture.md`](architecture.md#bounded-modules).

## `tv.subcult.place`

File: `contracts/lexicons/tv.subcult.place.json`. Record key: `tid`.

| Field | Required | Type / bounds |
| --- | --- | --- |
| `name` | yes | string, 1-140 graphemes, max 600 bytes |
| `locality` | no | string, 0-100 graphemes, max 400 bytes (public city/town) |
| `region` | no | string, 0-100 graphemes, max 400 bytes |
| `country` | no | string, exactly 2 characters (ISO 3166-1 alpha-2) |
| `coordinates` | no | `#geo` object: `latitude`/`longitude` strings, 1-20 characters each |
| `createdAt` | yes | datetime, RFC 3339 UTC instant with explicit offset |

**Public location semantics.** Only a public venue name, a public
city/region, and coarse coordinates may appear, and only when the operator
has explicitly approved that specific place record for public coordinates.
There is no schema-level flag for that approval: the schema only ever
receives a place that the operator's publish action has already decided is
public, the same way every other AT record here only exists once a human has
approved the transfer (`data-boundaries.md`'s public payload rule). A street
address, postal code, door code, staffing detail, or any pricing/money field
is never admitted here; the `address` field set used by
`community.lexicon.location.address` (`street`, `postalCode`) was
deliberately not adopted for exactly this reason (see ADR 0007).

## `tv.subcult.event.occurrence`

File: `contracts/lexicons/tv.subcult.event.occurrence.json`. Record key: `tid`.

| Field | Required | Type / bounds |
| --- | --- | --- |
| `name` | yes | string, 1-200 graphemes, max 1000 bytes |
| `description` | no | string, 0-3000 graphemes, max 12000 bytes |
| `profile` | yes | `#strongRef` to the authoring `tv.subcult.profile` record |
| `place` | no | `#strongRef` to a `tv.subcult.place` record |
| `startsAt` | yes | datetime, RFC 3339 UTC instant with explicit offset |
| `endsAt` | no | datetime, RFC 3339 UTC instant with explicit offset |
| `allDay` | no | boolean |
| `timezone` | no | string, max 64 bytes (IANA time zone database identifier) |
| `status` | no | string, open `knownValues`: `scheduled`, `rescheduled`, `postponed`, `cancelled`; default `scheduled` |
| `createdAt` | yes | datetime, RFC 3339 UTC instant with explicit offset |

`#strongRef` is defined locally in this file (`uri` + `cid`) rather than
referencing the unbundled `com.atproto.repo.strongRef` lexicon, so the
admitted chain has no dependency outside `contracts/lexicons/`.

**Public time semantics.**

- `startsAt`/`endsAt` are always UTC instants in RFC 3339 form with an
  explicit offset (`Z` or `+HH:MM`/`-HH:MM`); a bare local time with no
  offset is rejected by both validators (see above).
- `timezone`, when present, is the IANA identifier for the local wall-clock
  zone attendees should read `startsAt`/`endsAt` in. It is optional at the
  wire level, but the publication path must set it whenever a local wall
  time (not just the UTC instant) matters to attendees, most notably when
  `allDay` is true.
- **All-day representation:** when `allDay` is `true`, `startsAt`/`endsAt`
  carry the UTC instants of local midnight at the start and end of the
  covered local calendar date range, and `timezone` identifies which local
  calendar that is. There is no separate date-only field.
- **TBA (time to be announced) representation:** there is no explicit "TBA"
  marker in this schema. `startsAt` is a required field because a *public*
  occurrence, by the accepted journey's own definition, has a public time.
  An event whose time is not yet decided is not published as a
  `tv.subcult.event.occurrence` record; it remains a private operator draft
  (per `architecture.md`'s event-identity model) until a real start time
  exists to publish.
- `status`'s `knownValues` are declared open (not a closed `enum`) on
  purpose: a client encountering a status value it does not recognize must
  degrade gracefully rather than treat the whole record as invalid, matching
  how AT Protocol expects `knownValues` string fields to evolve. Neither
  validator in this repository rejects an unrecognized `status` value.

**Public location semantics.** `place` is an optional strong reference to a
`tv.subcult.place` record; when absent, the occurrence has no public venue
(for example, a fully virtual or venue-TBA-but-time-confirmed event). There
is no inline location object on the occurrence itself, keeping the "which
place fields are public" decision entirely inside `tv.subcult.place`.

**Never present on this record:** ticket price or inventory, reservation or
check-in state, staffing, contact details, settlement, or the private
operator event's internal identifiers. Those stay in the private Operations
module (`architecture.md`'s bounded module table) and are related to the
public occurrence only through the private mapping state `architecture.md`
already describes under "Event identity".

## Version evolution rule

Additive optional fields only. A new optional property, a new open
`knownValues` member, or a wholly new record type (a new NSID) are backward
compatible. Any of the following is a breaking change and requires a new
NSID (for example `tv.subcult.event.occurrence2`) rather than mutating the
existing one in place:

- narrowing or removing an existing field, tightening an existing bound, or
  changing a field's type or `format`;
- converting an optional field to required, or a `knownValues` (open) field
  to a closed `enum`;
- changing the record `key` scheme (for example from `tid` to a versioned
  key), which would also invalidate every existing record's address.

This repository has not needed a versioned record-key scheme for any
admitted NSID yet; the rule above states the choice (new NSID) so a future
breaking change has an unambiguous default instead of an ad hoc decision.

## Corpus

`contracts/atproto-lexicon.fixtures.json` holds named valid/invalid cases per
NSID, each invalid case carrying a `reason` (`missing required`,
`over-bound string`, `bad datetime`, `bad at-uri syntax`, `unknown field`,
`private field leak`, or `wrong $type`). Both
`backend/internal/atproto/lexicon_conformance_test.go` and
`web/src/atprotoLexiconConformance.test.ts` load this file and the same
`contracts/lexicons/*.json` documents and must agree on every case's
valid/invalid result.

## Verification

```bash
cd backend && go test ./internal/atproto -run TestSharedLexiconConformanceFixture -count=1
cd web && pnpm run test -- atprotoLexiconConformance
```
