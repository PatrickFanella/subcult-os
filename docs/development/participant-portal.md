# Participant portal

Status: implemented PORTALS-01 first slice. The authenticated participant
portal at `/participant` reads `GET /api/me/participant-portal`. It provides a
person's own staffing schedule and event-linked commitments. It does not make a
role application an assignment, create an account from an email address, or
add an application-to-person binding.

## Access boundary

The API identifies the viewer from the current authenticated session. It
returns a staffing item only when `assigned_person_id` equals that person, or
the person holds an active claimed place on it, and the person has an active
workspace membership. Removed, revoked, and expired
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
cancelled, in a workspace where the person has an active membership. That
covers both the legacy single assignee and a place the person holds through an
active claim (see open-shift self-claim below). A released or removed claim, or
a cancelled shift, drops out of the next download. Application-only and other
people's assignments never appear. A missing session returns 401 with no
calendar body.

Each timed assignment becomes one `VEVENT`:

- `UID` is `staffing-<staffing item ID>@os.subcult.tv`, so a fresh download
  with a changed time updates the same entry when imported again. A person
  holds at most one place per shift, so the UID is unique within their file.
- `DTSTAMP` and `LAST-MODIFIED` are the latest of the staffing item's and the
  event's `updated_at` and, for a claimed place, the time the person's active
  claim was made. Releasing and claiming again therefore advances them, as does
  a reschedule. `SEQUENCE` is that time in whole seconds since
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

An application becomes person-backed only through the session binding
described under open shifts. There is still no flow that claims an existing
anonymous application, and an approved application remains distinct from an
assignment. Vendor fees, invoice generation, payout
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
session. A second calendar test covers a self-claimed place with its times and
location next to a legacy assignment, its removal after release, a later
revision after claiming again, and removal when the shift is cancelled. Unit tests cover escaping, folding with multibyte text and open-ended
shifts. The web render test includes populated and denied states and the
download action and revocation note. A separate synthetic browser journey verified populated desktop and mobile
layouts, membership revocation and refresh, and logout and refresh in the
installed local preview. This is local browser evidence; it does not qualify
a physical mobile device, a provider flow, or a production deployment.

## Open-shift self-claim

Status: implemented for issue #218. An owner opens one timed shift on one event
to self-claim. Eligible participants claim and release their own places in the
portal under the shift's capacity and release cutoff.

### Shift policy

`PATCH /api/events/{eventID}/staffing/{staffingID}` accepts four owner-only
fields:

- `capacity`: places on the item, 1 to 100. Every item, including tasks and
  items from before migration 31, defaults to 1.
- `claimRoleIds`: the event roles whose approved applicants may claim.
- `releaseCutoffHours`: 0 to 720, default 24. A participant can release until
  `startsAt` minus this many hours. At that instant and later the API refuses.
- `claimsOpen`: requires a shift with start and end times and at least one
  eligible role. The database also rejects an open untimed item.

Reducing capacity below the places held returns 409.

### Who can claim

A person can claim when all of these hold inside the claim transaction:

- They are signed in and hold a current workspace membership of any role,
  including `participant`: not removed, revoked or expired. The membership row is share-locked, so a revocation
  either commits first and denies the claim, or waits for it.
- Their application for an eligible role is `accepted` or `confirmed` and is
  bound to their account. Submitted, under-review and waitlisted applications
  do not qualify.
- The event is published. Draft events and End of Night refuse claims.
- The shift is open to claims, open or assigned, and has not started.

An application is bound to an account only when a signed-in person submits it
with the same email as their verified account email. The submission sets
`applicant_person_id` and the organizer list shows "signed-in account". An
anonymous submission, or a signed-in one with a different email, stays
unbound and cannot claim even when its email text matches an account.
Earlier anonymous applications are not bound retroactively; the organizer can
withdraw one and the person can apply again while signed in.

### One capacity authority

Places held = the legacy single assignee (`assigned_person_id` or
`assigned_application_id`, at most one) plus active rows in
`event_staffing_claims`. Participant claims and releases, organizer additions
and removals, and organizer single-assignee edits all lock the event row and
then the staffing item row before counting. A final place is therefore granted
once. A partial unique index allows one active claim per person per shift.

A retried claim returns the existing place without taking another. A retried
release after success returns the released state and frees nothing further.
The item moves between `open` and `assigned` as places fill and free;
completed and cancelled items keep their status.

The single assignee is a person either directly (`assigned_person_id`) or
through an assigned application bound to their account. Both count as that
person, so they cannot claim, be added or be assigned a second place on the
same shift.

Overlap rule: a person cannot hold two places on the same event whose
`[startsAt, endsAt)` windows overlap. Windows that only touch are allowed.
Cancelled and untimed items never conflict. The rule applies to self-claims,
organizer additions and organizer assignments, by person or by application.
When an owner moves a held shift's start or end, or reopens a completed or
cancelled shift, every current holder is checked again. A conflict returns 409
with the number of affected holders, never their names.

Cancelling a shift ends its claimed places. Each is recorded as removed by the
cancelling owner, with a `staffing.claim_removed` audit entry whose reason is
`shift_cancelled`. A reopened shift therefore starts without claimed places;
only the single assignee, if any, stays.

### Organizer override

Owners can add a current workspace member to a place
(`POST .../staffing/{staffingID}/claims`) and remove any place
(`DELETE .../claims/{claimID}`). The override skips the role policy and the
release cutoff. Capacity, duplicate and overlap rules still apply. An addition
queues the existing `staffing.assignment` notification.

Only owners manage staffing, so only owners receive the `claims` list (who
holds each place) in staffing responses. Other workspace roles get
`filledCount` and an empty `claims` list. A holder is named by display name,
or "Member" when they have none; their email is never included.

### Participant routes and returned fields

- `GET /api/me/open-shifts` lists claimable shifts for the person's eligible
  roles and every shift they currently claim.
- `POST /api/me/open-shifts/{staffingID}/claim` and `.../release`.
- `POST /api/me/open-shifts/{staffingID}/release-request` is the organizer
  contact action after the cutoff. Before the cutoff it returns 409, since the
  participant can release the place directly. It queues a private
  notification to each workspace owner, at most once per place and owner, and
  releases nothing. Repeat requests write no further notification or audit
  row. The participant never receives an owner's address.

An open shift returns only its ID, event ID and title, title, start and end,
participant requirements, `remainingPlaces`, whether the viewer holds a place,
whether they can still release it, `organizerManaged`, and the release
deadline. `organizerManaged` is true when the viewer's place is the single
assignment; the portal then says "Assigned by the organizer; contact them to
change it" and offers neither release nor a release request. It never returns
who else holds a place, their contacts, operator notes, application data or
finance data. A held place appears in the existing assignment list with status
`assigned`, also while other places on the shift stay open. Responses use
`Cache-Control: private, no-store`.

### Not in this slice

Reminders and lifecycle notices go to every holder of a place: the single
assignee and each active claim holder who is still a current member, also
while other places on the shift stay open. An item nobody holds reminds the
owners that it is unstaffed. The event archive still records only the single
assignee's name. Recurring series and a separate volunteer platform remain out
of scope.

Known limitation: a claimed place stays active, and keeps counting against
capacity, after its holder's membership is removed, revoked or expires, or
after their application is withdrawn or rejected. The holder can no longer
release it, and reminders and notices skip them. An owner removes the place
from the staffing panel. Ending these places automatically needs a hook in
both membership routes plus a sweep for expiry, with lock ordering that does
not deadlock against claims, so it is left for a later slice.

### Verification

`staffing_claims_integration_test.go` covers four racing writers for the final
place (two participant claims, an organizer assignment and an organizer
addition) over four shifts, concurrent claim retries, pending, under-review,
waitlisted, unbound and wrong-role applications, draft and closed events,
membership revocation, the overlap rule, release one microsecond before, at
and after the cutoff, the release request, organizer override and capacity
limits, portal privacy, claimants as lifecycle-notice recipients, and an upgrade from a version-30 database with a
legacy assignee. `staffing_claims_review_integration_test.go` covers reminders
and notices for holders of partly filled shifts, application-based
assignments in duplicate and overlap checks, owner-only holder identities,
the overlap recheck on a move or reopen, ending claims on cancellation, the
`organizerManaged` flag, release requests before and after the cutoff, and the
database refusing claim and eligible-role rows that mix events. Web render tests cover the claim, full, release, after-cutoff
and pending states and the organizer panel. The repository has no browser
journey harness, so no browser journey was run for this slice.
