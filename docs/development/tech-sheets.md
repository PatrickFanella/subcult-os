# Private act tech sheets and input lists

Issue #224 attaches one private tech sheet to each act credited as a performer
on an event occurrence. The event’s input list combines the currently credited
acts in set order. A duplicate credit across listings does not duplicate the act.

Owners and organizers edit sheets in the event editor. Create or choose an act
profile under **Manage bill / copy sheets**, then attach it to a dated listing.
If the event has no listing, the explicit **Create draft listing for bill** action
uses the saved event title and time. It does not publish the listing.

Each sheet holds up to 64 ordered inputs, monitor needs, backline brought and
backline needed. An input records its label, mic/DI/line source, mic or DI,
stand, phantom-power requirement and notes. Move acts to change set order; move
inputs within a sheet to change their channel order. Give inputs the same shared
key across acts only when they use one physical channel. The combined list keeps
each act’s requirements and notes visible, including conflicting phantom power.
The sound person must check changes before repatching.

Active workspace members with `operate` can read `/events/{eventID}/input-list`.
The web staffing board and native run-of-show screen link to it. Native shows the
channel list and monitor/backline notes; sheet editing, stage-plot viewing and
printing are in the web view. `owner` and `organizer` can edit. Every request
rechecks membership; revocation denies the next request. Participant identities
and anonymous door links grant no tech-sheet access.

**Open printable input list** opens an authenticated snapshot in a new tab. Use
browser Print / Save as PDF; the A4 layout is designed for a three-act, 24-channel
bill with short notes. Long notes, large fonts or a larger bill can require more
pages. Printed, downloaded and already displayed copies cannot be recalled when
membership changes; keep them private.

## Versions, removal and reuse

Every sheet update and stage-plot upload carries `expectedVersion`; creation uses
zero. A stale version returns 409 without replacing another edit. The input-list
order has its own version. Reload after a conflict and reapply the intended edit.

Removing an act deletes its performer credits on this event, excludes it from
the combined list and retains its sheet for coordinator review. Crew cannot read
the retained sheet or its plot. Reattaching the same act restores the sheet.
Deleting a sheet requires explicit confirmation and its current version.

Templates and archive reuse do not copy sheets. The separate, confirmed copy
action can copy notes and inputs from another event in the same workspace into a
draft. Only matching acts already credited on the target are copied; existing
sheets are never overwritten. Stage-plot objects are excluded. There is no
participant self-edit link, drawing tool or equipment inventory.

## Private stage-plot storage

Configure `MEDIA_S3_PRIVATE_BUCKET` alongside the existing S3 credentials. It must
be a separate bucket from `MEDIA_S3_BUCKET`, and its policy must deny anonymous
reads and public ACLs. Do not expose it through `MEDIA_PUBLIC_BASE_URL`, a CDN or
a public bucket policy. The application refuses the same bucket for artwork and
stage plots; storage policy remains the operator’s responsibility.

Uploads accept sniffed PNG/JPEG only, at most 4 MiB, 8192 pixels per side and 20
megapixels. Save the sheet first. Uploads compare the version before storage and
again before attachment. The API streams images through a member-authorized
private/no-store response; it never returns an object-store URL or signing token.
Failed attachment, replacement, removal and confirmed deletion attempt private
object cleanup. If cleanup fails, the unreferenced object stays private; include
it in the bucket’s normal retention reconciliation. Deleting an entire event or
workspace cascades database rows, so that reconciliation must also remove orphan
objects. There is no automatic public publication of plots.

Migration 37 adds only `event_act_tech_sheets`, `event_act_inputs` and
`event_input_list_orders`, with composite workspace foreign keys. Public event,
publication and projection DTOs do not read them. Apply migrations before the new
API; do not delete these tables to roll back a populated environment. Restore the
previous application only with its compatible schema/readiness boundary, or
roll forward. Back up the private bucket with the database.

## Verification

Run `bash scripts/dev-env.sh verify` for the pinned complete local gate and the
separate disposable database suite. The tech-sheet tests cover shared channels,
ordering, version races, permission/revocation, cross-workspace constraints,
retention, confirmed deletion, explicit copy and private image cleanup. The
shared privacy sentinel fixture plants monitor, input, backline and private
object-key values and scans anonymous responses and public occurrence previews.
No automated check should use production images, credentials or mail delivery.
