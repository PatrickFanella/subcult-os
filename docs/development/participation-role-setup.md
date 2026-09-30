# Participation role setup

## Owner workflow

The joined operator rehearsal found an editor gap: owners could review role
applications and assign tasks/shifts, but could not create a participation role
through the web interface. The existing owner-only role endpoint already supports
this operation. `EventRoleSetupPanel` now joins that step to the event editor.

Owners of the matching workspace can add roles while the event is open. Members
and closed-event views retain the readable role list without creation controls.
Each role shows its description, public/private visibility, capacity and active
state. Role creation does not assign a task, accept an application or add a
workspace member. The existing review and staffing workflows remain separate.

New roles default to private. The request explicitly sends `public: false`
because the existing API defaults to public when that property is omitted.
Checking “Accept public applications” changes the action to “Add public
application role” and explains that the role/description appear on the published
event page. Successful creation resets the draft to private. Operator notes
belong in the staffing board, separate from participant-facing descriptions.

Capacity is a nonnegative whole number within the existing PostgreSQL integer
range; zero means no limit. Descriptions use a 2000-Unicode-character limit,
matching the existing role-edit contract. Names and descriptions are trimmed.
This slice adds no endpoint, schema migration, provider or permission authority.

## Write and context boundaries

The editor keys the panel by event, owner permission and closed state. Cleanup
invalidates callbacks in the removal commit; A → B → a fresh A and permission
changes cannot apply an old success to the current role list. Displayed role
records must match the panel's event ID.

A synchronous guard prevents repeated POSTs before disabled fields render. A
400 preserves the draft for correction. A 401/403 clears fields and hides the
panel's retained role list, with a reload path to recheck access. Other errors
and mismatched success responses are treated as uncertain outcomes: controls
stay disabled until the owner reloads and inspects the saved list. Creation is
not idempotent; the UI never automatically replays it.

## Local verification — 2026-09-30

Final `bash scripts/dev-env.sh verify` passed 296 web/34 mobile tests, backend
checks/builds and the full disposable DB gate: 421 top-level tests, 599 including
nested tests, no failures/skips. The disposable verification DB was removed.
New tests cover explicit private payloads, capacity/storage boundaries, Unicode
description length, read-only controls and exclusion of another event's roles.

Actual T3 browser controls against the separate disposable operator environment
created private and explicitly public roles. An intercepted held request was
forwarded to the real API once, then its committed response was lost. Repeated
submits produced one request/row; reload recovered exactly one role and reset
the draft. Two additional intercepted responses verified 400 correction and
403 clearing without writes. Strict Mode component fixtures with two synthetic
POST responses verified A → B → A and permission-change callback invalidation;
no creation callback ran after removal and no held fixture responses remained.

Four roles are persisted in the disposable rehearsal DB, one public. Anonymous
public-role reads return only that role; the private description sentinel and
other private roles remain absent. No application, staffing assignment or live
provider action has yet been qualified by this slice.

The actual editor route at a measured 360 × 800 CSS-pixel iframe fit long role
names/descriptions: page width345/345, panel311/311, role cards261/261. Explicit
light/dark screenshots were inspected:
`browser-screenshot-localhost-munzk7ro-ae333957.png` and
`browser-screenshot-localhost-munzk7wg-b97767e0.png`. Tab advanced the frame's
active element, but both browser documents reported `hasFocus() === false`;
visual keyboard-focus proof is therefore unqualified. No source focus defect
is inferred from that inactive-window observation. Frames, roots, interception
and helper globals were removed and the original editor/system appearance restored.

Detailed receipts remain in ignored `.cache/dev-env/operator-role-setup-*`.
Hosted CI, actual member/closed-state browser checks, the full joined operator
journey, physical devices and screen-reader speech remain separate gates.
