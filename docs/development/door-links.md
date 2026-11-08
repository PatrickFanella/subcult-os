# Volunteer door links (#222)

An active workspace owner can create and revoke a named bearer link for one
published event under **Volunteer door links** in the event editor. A link
admits guests without an account. Organizers, finance, door, crew and
participants cannot create, list or revoke links. Normal signed-in Door access
keeps its existing permission rules.

The owner chooses a local start and end time, at most seven days apart. The API
stores UTC instants, a label, the creator and a token hash. The complete URL is
shown once. Share it privately: possession is admission authority; v1 does not
add a PIN or permission to sell, refund, edit the event or read other operator
data. This is a bounded online capability rather than a workspace invitation.

The URL uses `/door-link/{eventID}#token`. Its fragment is never sent as a URL
or referrer; the client presents it in `X-Door-Link` to
`/api/door-links/{eventID}/tickets` and `/check-ins`. The client omits cookies,
session refresh and caching. It does not persist the token or roster in browser
storage. The fragment remains available for reload; anyone using that browser
can reopen the link until expiry or revocation. Remove the tab/history entry
when handing over a shared device. The private web route and API are noindex.

Each request checks the event, hash, start (inclusive), end (exclusive) and
revocation. Closed events and events with a cancelled cultural listing deny
access with the same generic response. New check-ins recheck authority after
the ticket lock. A transaction holds a share lock on the link and event until
commit, so a completed revocation cannot be followed by a waiting check-in.
Two simultaneous scans produce one admission and one `409 already checked in`.

Search requires three characters, matches names and codes literally, never
searches email, and returns at most 20 admission-only records. Search and
check-in share a database-backed budget of 60 requests per link per minute.
Responses are `private, no-store`; requests, headers and tokens are not logged.
A request budget bounds enumeration speed; it does not make a bearer link safe
to publish. No offline roster or camera scanner is implemented by this change.

New link check-ins set only `checked_in_by_door_link_id`, with a composite
foreign key enforcing the ticket event. Person check-ins keep their existing
actor column. The migration permits historical rows with no person actor; it
never invents an actor for old attendance. An audit row stores the link ID,
event and ticket code without guest email. Owner door records show each link
label and count after revocation. End-of-night reports snapshot these counts.
No link or attribution is included in public event or AT record DTOs.

Migration 35 is additive and runs before the API accepts link requests. It
retains revoked and expired link metadata with attendance for attribution.
Event deletion removes its links along with tickets. Template/archive reuse
does not copy links. An older binary refuses schema 35; rollback requires the
pre-migration backup or a compatible forward fix. Do not downgrade the ledger.

Verification uses `bash scripts/dev-env.sh verify`: the full repository checks
and a separate disposable PostgreSQL gate. Dedicated database tests cover the
role matrix, windows, revocation, concurrent admission, waiting-ticket
revocation, scope, literal/bounded search, rate reset and report attribution.
The shared privacy fixture plants link labels and tokens, checks public/AT
preview absence and captures secret-free request logs. Browser journeys must
use synthetic events and at least two links; no real audience should be used
for qualification.
