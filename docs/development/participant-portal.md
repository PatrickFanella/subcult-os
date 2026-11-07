# Participant portal

Status: implemented PORTALS-01 first slice. The authenticated participant
portal at `/participant` reads `GET /api/me/participant-portal`. It provides a
person's own staffing schedule and event-linked commitments. It does not make a
role application an assignment, create an account from an email address, or
add an application-to-person binding.

## Access boundary

The API identifies the viewer from the current authenticated session. It
returns a staffing item only when `assigned_person_id` equals that person and
the person has an active workspace membership. Removed, revoked, and expired
memberships are excluded in the query and rechecked before the response is
written. Reassigning an item changes what the next request returns.

`assigned_application_id` is intentionally excluded. An event role application
stores a submitted name, email, message, and review status. Even an accepted
application with text matching an account email does not prove control of that
account and does not authorize portal access.

The response uses `Cache-Control: private, no-store`. The web view clears its
previous portal state before a refresh and after a failed request, so an
authorization failure has no previously loaded assignment content to render.
The server-side rendered regression covers the denied state. A synthetic browser
journey in the installed development preview verified that membership revocation
followed by Refresh removes the loaded assignment and commitment, and logout
followed by Refresh shows an unauthorized error without those records.

## Returned fields

For each person-backed staffing assignment, the API returns only event ID and
title, staffing item ID, title, kind, start/end times, status, and participant
requirements. For a
commitment, it returns only ID, event ID/title, title, due time, and status.
Commitments appear only when their owner is the current person and their event
also has a current person-backed assignment.

The portal does not return staffing notes, application names/emails/messages or
statuses, contacts, commitment descriptions, ticket data, finance/settlement
data, payment-provider references, payout data, membership roles, or audit
metadata. The workspace header links authenticated users to `/participant`.

## Shift calendar download

`GET /api/me/participant-portal/shifts.ics` returns the portal's assignments as
an RFC 5545 calendar attachment (`text/calendar; charset=utf-8`,
`Content-Disposition: attachment`, `Cache-Control: private, no-store`). It uses
the same loader, session check and membership recheck as the JSON route, so it
contains exactly the assignments the portal shows: person-backed, not
cancelled, in a workspace where the person has an active membership.
Application-only and other people's assignments never appear. A missing session
returns 401 with no calendar body. Assignments that become visible in the portal
through a later assignment path appear in the download without further changes.

Each timed assignment becomes one `VEVENT`:

- `UID` is `staffing-<staffing item ID>@os.subcult.tv`, so a fresh download
  with a changed time updates the same entry when imported again.
- `DTSTAMP` and `LAST-MODIFIED` are the later of the staffing item's and the
  event's `updated_at`. `SEQUENCE` is that time in whole seconds since
  2026-01-01 UTC, so any later revision has a higher sequence. Two edits within
  the same second share a sequence. Edits that change only private fields, such
  as operator notes, also advance these values without revealing the change.
- `DTSTART` and `DTEND` are written in UTC. An assignment without a start time
  is omitted. One with a start but no end has no `DTEND`, which RFC 5545 treats
  as ending at its start.
- `SUMMARY` is the assignment and event title. `DESCRIPTION` holds the event
  title, assignment kind and participant requirements. `LOCATION` is the
  event's `location_display`. Nothing else is included: no operator notes,
  other people's names or contacts, ticket, finance or payout data.

Text is escaped per RFC 5545, lines end in CRLF and are folded at 75 octets
without splitting a UTF-8 character. The serializer is in
`backend/internal/app/ical.go`; there is no third-party dependency.

The portal shows a **Download shifts (.ics)** action and states that a
downloaded file cannot be revoked remotely. A cancelled or reassigned shift is
left out of the next download, but a calendar client keeps entries it already
imported until the person removes them. This is a one-time file export, not a
subscription feed.

## Participant requirements

Staffing items have a separate `participant_requirements` field. It defaults to
an empty string, is capped at 2,000 characters, and is written only by an
event owner through the existing staffing create and update routes. Operator
`notes` remain private and are never reused as participant requirements.

The participant portal returns requirements only for an active,
person-backed assignment belonging to the authenticated account. Application
assignments, assignments for another person, revoked memberships, contacts,
ticket and settlement data remain excluded.

There is no verified account-claim flow for existing applications. Any future
binding must require a deliberate, verified claim and preserve that an
application is not an assignment. Vendor fees, invoice generation, payout
records, and payout reminders remain outside this slice until a pilot shows a
need and defines the authority and finance boundaries.

## Verification scope

The backend integration test includes authenticated person assignment and own
commitment visibility, application-only exclusion even when its email matches,
private-field sentinels, reassignment on a later request, revoked membership,
the registered HTTP route, session middleware, and cache control. The calendar
integration test parses the download and covers owner isolation, untimed and
cancelled exclusion, a reschedule that keeps the UID and advances `SEQUENCE`
and `LAST-MODIFIED`, revoked membership, response headers, and 401 without a
session. Unit tests cover escaping, folding with multibyte text and open-ended
shifts. The web render test includes populated and denied states and the
download action and revocation note. A separate synthetic browser journey verified populated desktop and mobile
layouts, membership revocation and refresh, and logout and refresh in the
installed local preview. This is local browser evidence; it does not qualify
a physical mobile device, a provider flow, or a production deployment.
