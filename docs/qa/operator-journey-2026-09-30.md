# Disposable operator journey — 2026-09-30

This is an ongoing synthetic rehearsal, not completion of issue #25. Its first
browser stage found and repaired missing owner role setup. The remaining stages
are staffing/commitments, application versus assignment, participant views,
free booking/ticket/door, finance, private closeout and template reuse. Paired
existing-versus-joined time measurement and intended-user evaluation remain open.

## Runtime identity

- Source baseline: qualified PR #180, `e25312ef23807aef861579ec916bb07965289ad6`.
  Backend source remains that baseline; Vite serves the role-setup candidate from
  the same source-mounted checkout. Published candidate revision belongs to its PR.
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
