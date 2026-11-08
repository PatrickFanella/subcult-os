# Availability polls (#223)

Owners and organizers can attach a private availability poll to a draft event.
The event editor has an Availability polls link and a results grid. Each poll
has a prompt, two to twelve options, an optional closing time, and an option to
show respondent names. Names are hidden from other respondents by default.
An option stores an explicit start instant, optional end and IANA timezone.

The organizer enters dates in the browser’s local timezone. The participant
page displays each option in both the viewer’s local time and the stored
organizer timezone, including the zone abbreviation across DST transitions.
Spring-forward local times that do not exist are rejected rather than silently
shifted. A repeated local hour uses the browser’s first occurrence; the API
accepts explicit offset/UTC instants for either occurrence. There is no calendar
sync, recurrence, contact matching, invitation mail or external publication.

Creation returns a sharing URL once: `/poll/{pollID}#token`. Keep it private.
The database stores only a hash. The fragment travels in `X-Poll-Link`, never a
request URL or Referrer header. The private web route and public poll API send
noindex instructions. API responses are `private, no-store`, and the client
omits session cookies and session refresh. A link reveals the prompt, options,
answer totals and selected option, without event/workspace details. When names
are enabled, it also reveals names and answers; private notes remain restricted
to the organizer and the respondent who owns them.

The respondent’s browser generates a random edit capability before saving,
retains only that capability in local storage and presents it in `X-Poll-Edit`.
Names, notes and answers are not placed in browser storage. Replaying the same
capability updates the same response, including after an uncertain response;
another capability cannot edit it. Deleting browser storage loses edit access.
If storage is denied, edit access lasts only in the current tab and the page
says so. A name is self-asserted and does not prove control of a person, contact,
performer account or membership.

Response writes are serialized on the poll, have a shared database budget of
30 attempts per minute per poll, and are limited to 200 responses. Request
bodies are capped at 16 KiB. Names are limited to 80 characters, notes to 1000,
and each response must answer every option with yes, if need be or no. Closing
a poll, reaching its closing time, choosing an option or publishing the event
stops new responses and edits. Existing results remain readable through the
link until the poll or event is deleted.

Use option is disabled while the editor has unsaved changes. It calls the
existing event PATCH with `pollOptionId`: the server locks the event, checks
current coordinator access and draft status, copies the selected start/end,
closes the poll and audits the event update in one transaction. A published
event returns a conflict pointing to the lifecycle/reschedule flow. Poll
selection never edits public occurrences or emits an AT record. Optional event
ends are stored as UTC instants and must follow the start. Unrelated web edits
preserve the exact end instant, including sub-millisecond precision.

Coordinators can close or delete polls while the event is a draft. Deletion
requires confirmation in the UI and cascades to options, responses and answers.
Event deletion has the same cascade. Audit metadata records IDs and actions,
not respondent answers or bearer capabilities. Event templates and archive reuse
do not copy polls or edit capabilities. There is no automatic consent or
marketing grant. See [data lifecycle](data-lifecycle.md) for retention.

Migration 36 adds the private poll tables and optional `events.ends_at`. Older
binaries refuse schema 36; a rollback needs the pre-migration backup or a
compatible forward fix. Do not downgrade the migration ledger. Tests must use
the disposable test database through `bash scripts/dev-env.sh verify` and the
synthetic browser journey; never involve real respondents or live mail.
