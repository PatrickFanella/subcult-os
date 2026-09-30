# Disposable operator journey — 2026-09-30

This is an ongoing synthetic rehearsal, not completion of issue #25. Its first
browser stage found and repaired missing owner role setup. The remaining stages
are staffing/commitments, application versus assignment, participant views,
free booking/ticket/door, finance, private closeout and template reuse. Paired
existing-versus-joined time measurement and intended-user evaluation remain open.

## Runtime identity

- Source baseline: qualified PR #180, `e25312ef23807aef861579ec916bb07965289ad6`.
  The backend initially served that baseline. The membership-roster stage below
  loads its candidate into the owned rehearsal API; Vite uses the same mounted
  checkout. Published candidate revisions belong to their PR/runtime receipts.
- Compose project: `subcult-qa-operator-e25312e`, using only standalone
  `compose.dev.yml` and an ignored disposable-only override.
- Web/API preview: `http://localhost:33027`; PostgreSQL loopback binding33028.
- Database: `subcult_qa_operator_e25312e`, tmpfs storage, all 29 migrations.
  Initial people/events/outbox counts were zero.
- API health and actual database identity were checked. Mail delivery, AT OAuth
  and AT projection are disabled; no mail/announcement/provider worker is started.
- `localhost` cookies/storage are separate from the original retained preview's
  `127.0.0.1` hostname. The retained development database is not a rehearsal target.

Container IDs, configuration, flags and counts are recorded in ignored
`.cache/dev-env/operator-journey-runtime.json`. Preserve this owned disposable
stack while the rehearsal continues, then remove only its resources after receipts
are saved. Do not run mailbox readers or rehearsal scripts against retained data.

## Persisted browser stages

The synthetic owner signed up and explicitly verified through the normal browser
confirmation. The held verification link was read only for its `example.test`
recipient from the disposable DB; no messages were sent. The browser created
workspace “Operator Journey · Disposable” and event “Operator Journey · Free Night”.
The event was then published locally through its normal editor action.

The datetime-locator helper initially typed into the wrong field. Native form
validation blocked that first attempt; no event was created. The title was
corrected and the date entered with the input's native value setter and normal
input/change events before browser submission. The stored start is
`2026-11-08T00:00:00Z`. This does not qualify the native datetime widget or settle
the legacy event-zone model.

Event `523fef0d-42b3-4930-b788-0068ec077889` belongs to workspace
`68d6238f-4c1f-4d6e-b597-95a18db326ab`. It has allocation10, four saved roles
(one public), zero applications and one held verification outbox row. Role setup,
unknown-save recovery and private/public absence proof are detailed in
[participation role setup](../development/participation-role-setup.md).

This stage is real browser-to-API-to-PostgreSQL evidence for signup, workspace,
event and role setup, with explicit synthetic failure/component checks recorded
separately. It is not a full lifecycle, member, provider, mobile or deployment receipt.


## Role setup qualified; crew and public application stages

PR #181 at `6ed01556ff91b51b7997f74543b77e6802b26731` passed both hosted
checks: push11075/job20026 and PR11076/job20027. Each ran 296 web/34 mobile
tests and DB421 top-level/599 including nested, with no failures/skips. The
existing kvant runner handled both; no owned runner or credentials were created.
PR and issue #25 receipts were updated and read back.

A second synthetic account verified through the normal browser flow and accepted
the member invitation. The event editor showed read-only roles without creation
controls. A direct role-create request returned403; the role count stayed4.
The three held outbox rows cover owner verification, invitation and crew
verification. None was sent.

After normal sign-out, `/api/me` returned401. The anonymous public page accepted
one application for the public performer role. Its button changed to Submitted,
and the independent ticket form stayed blank. The disposable DB confirms one
submitted application and zero staffing items: applying did not assign work.
Acceptance-versus-assignment still needs owner review and participant proof.

The preview lost its page context during the following reservation attempt.
Corrected navigation and preview reopening reached `chrome-error://chromewebdata/`
although host API health remained200. No reservation persisted: tickets0,
applications1, staffing0 and held outbox3. Browser reservation/door proof remains
pending. The preview and disposable stack are preserved for continuation.

The invitation UI had reported “Invite sent” for its held email. The follow-up
changes that receipt to invitation creation with queued, unconfirmed delivery,
and names the action Create invite. The existing API commits the invitation and
outbox row together; it does not return a provider delivery result. No delivery
flag, provider configuration, API or database behavior changes.


## API-only operations qualification and owner invitation controls

PR #182 (`7d9563c3368e04282070df953d02375bae66c02d`) passed both hosted
checks: push11077/job20028 and PR11078/job20029, each with 296 web/34 mobile
tests and DB421/599, no failures/skips. Existing kvant runner preserved; no
owned runner or credentials created. Qualified PR body and head were read back.

With the preview connection still unavailable, `scripts/qa-operations.sh` ran
against the live disposable API/database after actual destination, tmpfs storage
and disabled provider flags were checked. All 36 steps passed: verified accounts,
workspace membership, contacts, commitment transitions, staffing creation and
assignment permissions, template creation/application, role application review,
and reminder owner/idempotency boundaries. It created separate synthetic
accounts, workspace and events; it did not advance the original browser event.
The DB now contains four people, two workspaces, three events and nine held
outbox rows. Original browser-event state remains roles4, applications1,
staffing0 and tickets0. This is API/runtime qualification, not browser evidence.

The earlier member workspace browser page exposed an invitation form even though
only owners may use its API. The follow-up isolates that form, renders it only
for owners, removes its non-owner guidance link and guards the submit handler.
It uses the shared Button and keeps a stable status region before submission.
SSR tests cover member, organizer, finance, door, crew and unresolved roles,
owner entry points, and pending controls. No server permission changes.

A focused normal-API login as the original verified crew member returned403 for
invitation creation. Invitation rows stayed2→2 and held outbox rows9→9. Cookies
were held only in the private process; the browser session was not changed.
The same preview connection problem prevents changed-form visual, keyboard and
screen-reader proof. Read-only invitation listings and legitimate member event
creation remain governed by their existing server contracts.


The owner-control candidate passed the full pinned local gate: 304 web/34 mobile
tests, backend checks/builds and DB421 top-level/599 including nested, with no
failures/skips. Complete output was retained and the disposable test DB removed.
Cross-workspace pending invitation callback recovery is not covered by this
control-visibility slice; existing server authorization remains authoritative.


## Free-ticket API rehearsal and current door authority

PR #183 at `b4369fa5a806133961a4c3dc061b7d011928c888` passed hosted
push11081/job20033 and PR11082/job20034: each ran304 web/34 mobile and
DB421/599, no failures/skips. Existing kvant runner preserved; no owned runner
or credentials created, zero repository registrations verified. PR body and
head were read back.

The unchanged `scripts/alpha-qa.sh` then reproduced a stale rehearsal assumption.
Its baseline member account received403 at door search after successfully
reserving a free ticket and checking capacity exhaustion409. AUTHORITY-01
correctly limits that capability to owner and door roles. The failure is in the
rehearsal, not evidence to widen server permissions. Its synthetic event/ticket
remain in the owned disposable DB as partial-run evidence.

The repaired script resolves the accepted member's workspace membership ID
through an owner workspace read, proves the initial door search403, and uses
the normal owner PATCH endpoint to grant role `door`. Exact-code search then
returns one ticket, check-in succeeds and repeated check-in stays idempotent.
The owner changes the role to `crew`; another check-in returns403. The final
owner end-of-night report confirms one reservation, one checked-in ticket and
zero no-shows. The free script run exited0. No paid mode/provider was invoked.
Failure assertions now avoid dumping outbox messages or ticket responses.

This is API/runtime proof from separate synthetic accounts/workspace/event, not
browser door, scanner, offline/device or production qualification. The original
browser event still has no tickets and one submitted application. The suspected
cross-workspace invitation callback leak has not been reproduced through normal
navigation: workspace switching uses full-page links. Context recovery remains
unqualified; do not present that concern as an observed navigation defect.


Final script content passed a second free-only API run after assertion-output
cleanup. `bash -n` and `shellcheck -x` passed. Full pinned local verification
passed304 web/34 mobile tests, backend checks/builds and DB421 top-level/599
including nested, no failures/skips; disposable verification DB removed.
No broader acceptance gate is closed by the repaired script.


## Membership role and access-state roster

PR #184 (`40a55c4bac0e53ce9efd06e48998342c7ef887df`) passed push11083/job20035
and PR11084/job20036: 304 web/34 mobile, DB421/599, no failures/skips. Existing
kvant runner preserved; no owned runner or credentials created, repository
registrations0. Qualified PR body and head were read back.

The next source slice fixes roster information needed before owner role
management. Previously every non-owner role appeared as Member, and the private
workspace read omitted membership expiry/revocation state. The roster now labels
Owner, Organizer, Finance, Door and Crew accurately; the historical `member`
role uses the Crew label, matching its capability bundle. Unknown roles remain
unqualified. The private API derives active/expired/revoked state using database
time, with revocation taking precedence. It includes applicable expiry/revocation
instants, excludes soft-removed memberships, and marks current/scoped workspace
reads `Cache-Control: no-store`, including denied responses. No migration.

The client describes configured capabilities only for active state, shows no
workspace access for expired/revoked memberships, and treats an older response
without status as unavailable. Times are displayed in the viewer's zone with
semantic datetime values. Status is a snapshot from the last workspace read,
not an authorization grant or continuing proof. Shared DTOs keep new metadata
optional for compatibility; mobile role types now match the existing six roles.
Native rendering and owner role-change controls remain unqualified/unimplemented.

Full pinned local verification passed315 web/34 mobile, DB422 top-level/606
including nested, no failures/skips; disposable test DB removed. Eleven web
cases cover role labels, unknown/legacy data, active versus unusable authority.
The new persisted DB case covers active, future expiry, expiration at database
time, revoked, revoked-and-expired precedence, removed absence, serialization,
member denial and private cache headers on both workspace read routes.

Only the owned rehearsal API was restarted to load this source candidate. Its
Go runtime is1.26.6; PostgreSQL/tmpfs data and the retained development stack
were preserved. Normal owner API expiry/revoke actions targeted membership
`f6d4738e-aa45-498e-8f08-bf098039d887` in the separate alpha workspace
`5c36f247-6010-4669-bcd5-06bddc0ee34a`. Owner roster reads matched active,
future expiry, expired, restored active, then revoked state. Member reads were
200,200,403,200,403 respectively; expiry/revocation instants and no-store headers
matched. The original crew role remains member with no expiry/revocation.
Outbox21 stays held; original browser-event tickets0/applications1. No provider
was contacted. Recursive public-event reads omit all new membership fields.

Changed-roster visual/keyboard/SR proof remains pending the same preview
connection error. The API and fixture receipts do not replace that browser gate.


## Owner member-role assignment

PR #185 (`32a474ec77ca2e188332f9e5f40930921719c7b9`) passed both hosted
checks: push11089/job20041 and PR11090/job20042. Each passed315 web/34 mobile
and DB422 top-level/606 including nested, with no failures/skips. The existing
kvant runner was preserved; no owned runner or credentials were created and
repository runner registrations remain0. Qualified PR body/head read back.

Owners now have a dedicated Member roles page linked from their workspace.
It offers Owner, Organizer, Finance, Door and Crew with capability descriptions
before assignment. The legacy member role defaults to its effective Crew bundle;
unchanged selections do not write. Nonowners and inactive, unknown-role or
unresolved memberships remain read-only. The request contains only the selected
role and targets the membership row ID, preserving expiry/revocation metadata.
The server still decides permissions and rejects last-active-owner demotion.
No authority policy, schema or access-restoration control changed.

A synchronous pending guard fences duplicate submissions across the page.
Matched receipts trigger a fresh canonical roster read; permissions are never
applied optimistically. Denied writes clear the private roster. Unknown outcomes
fence further writes until a fresh read, without replaying the write. Known
validation/last-owner rejections remain editable. Unmounted callbacks cannot
update the departed page. Separate route instances are keyed by workspace ID.

Eleven transport cases cover encoded membership paths, role-only payloads,
metadata preservation, invalid roles, mismatched receipts, server rejections and
workspace identity. Eleven static-render cases cover all five nonowner roles,
owner choices/review, inactive/unresolved rows, unknown roles and missing private
workspace data. These do not prove browser callback timing or native interaction.
Changed-page visual, keyboard and screen-reader proof remains pending T3 preview
connection recovery. No provider, native-device or deployment claim.


A normal API rehearsal on separate synthetic alpha workspace
`3449e469-cbde-44bf-9a32-3a3e1996bb1e`, membership
`eec8aa9c-8fc2-4dbb-9c2d-3100efcd60d4`, proved role-only Door/Crew changes
preserve a future expiry, canonical refreshed roster state, nonowner change403
and last-owner demotion409. The original Crew/no-expiry state was restored through
normal owner API calls. Original browser actors/event were untouched. This proves
the existing API boundary, not the page's actual browser interaction.


Full pinned local verification passed337 web/34 mobile tests, backend checks and
builds, DB422 top-level/606 including nested, no failures/skips. Disposable test
DB removed. The initial TypeScript fixture-cast failure was corrected before the
complete rerun. Scoped review, edited Markdown links and diffcheck passed.


## Owner role controls qualified; finance/closeout API rehearsal

PR #186 (`dc095ffad2d7b639ed2cae5a91a5bc068bd8ec32`) passed push11093/job20046
and PR11094/job20047:337 web/34 mobile, DB422 top-level/606 including nested,
no failures/skips. Existing kvant runner preserved; no owned runner/credentials,
repository registrations0. Qualified PR body/head and issue #25 read back.

On separate disposable alpha workspace3449e469-cbde-44bf-9a32-3a3e1996bb1e,
closed free event46d57735-6a5a-403a-9689-66056e5ab73a, normal API calls retained
four finance rows: budget10000 cents, payable8000, actual3000 corrected to2500.
Current categories remain separate; ticket gross stays0. Same request-key replay
returned the same correction ID; changed payload with that key returned409.
Crew reads403, assigned Finance reads200, restored Crew reads403. No provider.

Settlement finalized with gross0; late adjustment rejected409. Private archive
was available to owner and denied anonymous401. Private note remains in archive.
Reuse seeded draft36a59dda-e344-4703-8624-af9e5dc91aad once; retry returned the
same ID. Tickets/check-ins, roles, staffing, finance lines and archive were not
copied. Private templatefc7e7cd7-16b8-4a23-b5d3-7a20843a71ae applied while
draft; after synthetic local publication, applying again returned409. Public
response omits private finance/archive/template sentinel text and fields.
The first ad-hoc lookup used ended instead of source-owned end_of_night and
stopped before mutations; corrected before proof. Do not rerun this ad-hoc script
against the finalized event; its initial conditions are no longer present.

API source remained185/Go1.26.6; frontend186. Original browser actor/event state
stays roles4/applications1/staffing0/tickets0; aggregate10people/5workspaces/
7events/3tickets, all21 outbox held. Retained development stack/data untouched.
T3 status/open retry still reaches chrome-error with no application root, while
host web/health return200. This is API proof, not joined browser, intended-user,
paired-timing, keyboard/SR/device, provider or deployment acceptance.

## Private finance read and write session

The finance panel previously rendered its editor after a failed/denied ledger
read and guarded submissions using rendered busy state. A new event-owned session
requires a valid event-scoped private read before editing, admits one write
synchronously, and validates receipt event/type/direction/amount/currency and
correction/payable references before displaying it. Denial or uncertain write
outcomes clear private lines/draft and fence further writes until a fresh read;
refresh never replays the mutation. Known400/409 rejections remain editable and
retain an unchanged request key. Successful writes retain existing manual-record
and category-separation behavior. No payment execution or authority policy change.

Keyed inner panels isolate event lifetimes; permission loss unmounts the private
panel. Layout cleanup invalidates session and view lifetimes, including reactivation
before an old response arrives. Loading does not claim the ledger is empty.
Shared Button and a persistent atomic status region cover save/recovery controls.
Nineteen transport/session cases cover pending read/write fences, identity,
400/409 retry,401/403/500 denial/uncertainty, disconnects, mismatched receipts and
departed/restarted lifetimes. Five static-render cases cover absent permission,
loading, unavailable recovery, confirmed empty data and retained payable history.
These fixtures do not prove actual React callback timing or browser/SR interaction.


Full pinned local gate passed361 web/34 mobile tests, backend checks/builds,
DB422 top-level/606 including nested, no failures/skips; test DB removed.
Initial key-generator typing and fixture React-import/401-refresh failures were
corrected before the final complete rerun. All attempts retained as local logs.
Scoped review, edited Markdown links and diffcheck passed. Browser proof remains
pending; no live provider/deployment/retained-state action.


## Finance panel qualified; repeatable free API rehearsal

PR #187 (`1cdd075f4c7eed9344a92e55ab89317cdb07538e`) passed push11097/job20050
and PR11098/job20051:361 web/34 mobile, DB422 top-level/606 including nested,
no failures/skips. Existing kvant runner preserved; no owned runner/credentials,
repository registrations0. Qualified PR body/head/base and issue #25 read back.

Read-only exports on the earlier synthetic finalized event passed owner200,
Crew403 and anonymous401 for CSV, Markdown and printable HTML, with no-store,
UTF-8, retained4 history rows, current budget10000/payable8000/actual2500 cents,
superseded actual exclusion and UTC timestamp. Unrelated archive/template note
sentinels are absent. Initial ad-hoc assertions used an incorrect CSV row name
and false rather than its blank noncurrent flag; corrected against source before
proof. No data mutation or browser Print/PDF qualification.

The new finance-closeout-qa harness turns the precondition-dependent ad-hoc run
into fresh-record normal API proof. Shell and Python entrypoints both enforce
existing disposable-target guards. Private temp cookies are removed on exit;
output contains named checks and bounded errors, not tokens/capabilities/DTOs.
Fresh verified owner/crew actors cover ledger corrections/key replay, Finance
permission removal, free door/closeout, private notes, seed retry/no copied data,
private template reuse and scoped CSV/Markdown/HTML exports with exact cents/UTC.
See [the harness guide](../qa/finance-closeout-rehearsal.md).

The first run stopped401 before workspace creation: urllib uses localhost.local
for a single-label host, while curl stored host-only cookies under localhost.
The helper now normalizes only those host-only cookies for a localhost API.
Both subsequent localhost runs passed all27 checks. Six unsafe wrapper/helper
invocations (remote API, missing opt-in, retained DB name) rejected before cookie
or API use. Bash syntax, ShellCheck with sourced files and Python syntax passed.
All attempts retained. Original browser event remains roles4/applications1/
staffing0/tickets0. After these runs:16people/7workspaces/11events/5tickets and
all31 outbox rows held. Owned runtime has only API/PG/web, mail/OAuth/projection
flags false, backend source185/Go1.26.6. Retained development data/stack untouched.
No live provider/device/deployment or broader acceptance claim.


A third27-check run through127.0.0.1 also passed. Read-only persisted checks show
three distinct fresh rehearsal workspaces, each with2 members/2 events/4 finance
history rows/1 ticket/1 private archive. Post-run aggregate18people/8workspaces/
13events/6tickets, all35 outbox held. The original browser vector is unchanged.
Full pinned local verification passed361 web/34 mobile, DB422 top-level/606
including nested, no failures/skips; disposable gate DB removed. This gate is
separate from the retained owned rehearsal DB. Edited Markdown links, syntax,
ShellCheck and scoped review passed; no broader acceptance closure.


## Exact event start preservation during unrelated edits

PR #188 (`fee51e77b254dcc43cbbe284f0b3dc294426aaf6`) passed both hosted
checks: push11101/job20055 and PR11102/job20056, with361 web/34 mobile tests
and DB422 top-level/606 including nested, no failures/skips. The existing kvant
runner was preserved; no owned runner or credentials, repository registrations0.
Qualified PR body/head/base and issue #25 receipts were read back.

Both event editors reconstructed every saved start from minute-level local text.
That truncated server seconds/fractions and could select the other instant during
a repeated local hour. On a published synthetic event with one reserved ticket,
the actual web and mobile builders converted2026-11-01T07:30:45.123456Z into
2026-11-01T06:30:00.000Z under America/Chicago. Both unrelated description
PATCH requests returned409. An exact-start control succeeded200.

Hydrated forms now retain the original timestamp privately. An unchanged displayed
start uses that original value; a changed input uses the existing parser. The
metadata never enters the write payload. Both actual fixed builders passed normal
API description edits200 and retained the exact start and one reservation.
The first post-fix rehearsal reused an identical description and correctly hit
the server's no-op400; distinct per-client descriptions resolved the harness
assertion without changing server validation. Supplemental actual-model runs
passed under Chicago, New York, Honolulu and Tokyo. Five regression cases per
client cover fractional precision, repeated-hour instants, offset input, deliberate
changes and invalid source fallback. The red gate failed the three preservation
cases before the implementation; the complete green pinned gate passed366 web/
39 mobile and DB422/606, with no failures/skips and disposable gate DB removed.

This is model and owned synthetic API evidence. Actual browser/native editing,
legacy event-zone persistence and deliberate ambiguous/nonexistent wall-time
selection remain open. No live provider, deployment or retained-data action.
