# Overnight core-workflow review — 2026-11-08

The overnight pass reviewed the event, admission and account paths alongside the
new volunteer-door, availability-poll and act-tech-sheet work. It found and
reproduced three defects. The source changes are locally qualified; hosted CI
and merge state are separate from these receipts.

## Findings and repairs

| Finding | Reproduction | Repair and proof |
| --- | --- | --- |
| Pricing could commit after an owner became an organizer | Pause an event UPDATE in a disposable PostgreSQL trigger after early finance checks, demote the actor, then resume. Both price-only and mixed price/location requests returned 200, changed the event and recorded success. | [PR #243](https://git.subcult.tv/subculture-collective/subcult-os/pulls/243) rechecks finance before commit. Both cases return 403, retain the old event fields and create no success audit entry. The focused test passes under the Go race detector. |
| Auth form and URL could disagree | From `/login`, choose Create account. The form changes while the address stays on `/login`; reload opens Sign in. | [PR #244](https://git.subcult.tv/subculture-collective/subcult-os/pulls/244) uses the existing route links. Browser click opens `/signup?next=%2Fworkspace`, reload retains Create account, and Back returns to Sign in with `next` intact. |
| Native Door advertised an admitted guest as ready | Admit a synthetic guest on Android. The card showed Checked in together with Ready for entry and an enabled Checked in action. | The native Door change in this branch shows Already admitted and disables the confirmed action. A fresh Metro bundle and process restart showed an admitted guest with the action disabled alongside an unadmitted guest with Check in enabled. |

## Automated verification

The final pinned `bash scripts/dev-env.sh verify` passed with 420 web tests,
46 mobile tests and the complete disposable database gate: 521 top-level cases,
721 including nested cases, no failures or skips. Disposable test containers were
removed. The permission regression separately passed with `go test -race`.
Golangci reported zero new issues for that backend change.

A second, manually named disposable development stack used its own
`subcult_qa_overnight_20261108` database. Mail delivery, AT OAuth and projection
were disabled. Against that stack:

- `scripts/alpha-qa.sh` passed the free-ticket lifecycle, capacity conflict,
  invitation/door-role boundaries, duplicate admission and closeout counts.
- `scripts/qa-operations.sh` passed all 36 checks for invitations, workspace
  switching, contacts, commitments, staffing, templates, role applications and
  reminder boundaries.
- `scripts/qa-finance-closeout.sh` passed all 27 checks for ledger categories,
  exact cents, correction/replay, Finance removal, free closeout, private archive
  notes, reuse, templates and scoped exports.

The harnesses read only synthetic accounts’ held verification messages, consume
challenges through the normal endpoint, and remove their temporary cookie files.
They do not send live messages or contact ticket/payment providers.

## Browser and Android journeys

A fresh synthetic account was created through the browser form. The normal
verification endpoint consumed its held-email challenge, and the browser signed
in, created a workspace, drafted an event with start/end times, and published it.
A separate browser session reserved a free ticket and opened its QR ticket page.
The signed-in owner found that guest in Door.

Android 15 / SDK 54 Expo Go signed into the same disposable workspace. A process
relaunch retained the session and correct event selection. Native Door found and
admitted the browser-reserved guest. Refreshing browser Door showed Checked in
with its action disabled. Browser End of Night persisted one reservation and one
admission. Native logout followed by another process relaunch returned to the
sign-in gate.

The separate tech-sheet journey created two acts through operator controls,
saved a shared input with distinct per-act notes, changed set order and opened
the printable list. A three-act, 24-channel fixture measured about 917 CSS pixels
inside a 1,032-pixel A4 content area after correcting the first oversized layout.
Removing an act excluded its eight inputs while retaining its sheet for review;
reattaching it restored the 24-channel list. The browser had no horizontal
overflow at 390 pixels in light and dark appearances. Native Run of Show opened
the input list, reloaded all 24 channels, and reached the monitor/backline notes.

## Limits

These are synthetic development receipts, not a real event or a production
release qualification. The review did not drive an OS Save as PDF dialog, test
camera scanning against a physical device, validate an iOS build, qualify
hardware secure storage, or use a real AT/payment/mail provider. Private stage
plot checks used synthetic media adapters; production bucket policy and storage
credentials were not configured. The Android emulator does not satisfy issue
#6’s physical-device requirement.

The open-issue review retained owner decisions, pilot-demand gates, consenting
AT-identity/provider qualification and dependency links. In particular #200 still
requires the owner decision in #201; #225 requires pilot demand. Source-complete
issues whose closure is blocked by #6 were not forced closed.

Rehearsal logs are retained in the local worktree’s ignored `.cache/overnight/`
directory. Credentials and usable invitation, verification or ticket links must
not be copied into this report.
