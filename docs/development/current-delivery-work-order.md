# Current delivery work order

Observed 2026-09-29 on Kvant. Source and fetched `origin/main` both point to
`89b6ea9203db0bc3e42d51eb217614e8edc38061`. The working branch is
`t3code/complete-development-issues`; it was clean before this documentation
update. Gitea reports 68 open issues and no open pull requests. This is a dated
reconciliation, not a production qualification receipt.

## 2026-09-30 continuation

PR #169 merged at `375451dc97f7c78d74fed182c46f93375b48c67d` after its
exact-head push and PR checks passed. PR #170 (`4fbeaafa`) now targets `main`;
both hosted checks passed. PR #171 (`39893d69`) adds the private event access
worksheet; both hosted checks passed (runs 11019/11020), including the full
disposable database gate. These are separate heads and evidence receipts.

The merge-commit follow-up job exhausted the temporary runner daemon's 256 MiB
limit while downloading actions, before tests started. Its failed attempt was
retained, owned orphaned job containers were stopped, the ephemeral registration
credential was removed, and the daemon limit was raised to 1 GiB. Rerunning the
same merge commit produced a passing job 19978, including the full disposable DB
gate: 405 top-level tests (567 including nested), zero failures or skips. The repository queue assigned the older
#171 jobs to that temporary capacity first. No workflow, commit or shared runner
configuration changed.

PR #172 (`7c7727239bfd7acc4263a8d89003c01519676044`) passed hosted push
11037/job 19983 and PR 11038/job 19984: 273 web tests, 34 mobile tests and the
complete DB gate with 419 top-level tests (595 including nested), zero failures
or skips. Temporary runners exited successfully, were removed, and their local
registration credentials were deleted. Zero repository runner registrations
were verified. The published stack #170 → #171 → #172 → #173 → #174 remains open and linked
to this thread.

PR #173 (`66472a21e205ff0a7507d8f3713695861ebe78c0`) adds the owner-only
occurrence/venue comparison. Both hosted checks passed: PR 11043/job 19990 and
push 11044/job 19991, with 277 web/34 mobile tests and 421 top-level DB tests
(599 including nested), zero failures/skips. Temporary runner cleanup and zero
repository registrations were verified. The comparison uses one database
snapshot, separate provenance and Unknown states. Stored occurrence-specific
verification, public display and intended-audience evaluation remain open.

PR #174 (`67fb7967fdbd1e874b624e44d4c46f6452f03e80`) fixes #58's archive
permission/pending-write states. Push 11045/job 19992 and PR 11046/job 19993
passed the full hosted gate: 278 web/34 mobile tests and 421 top-level DB tests
(599 including nested), zero failures/skips. Synthetic browser checks covered
held/duplicate submits, a committed correction with a lost response, validation
and 401/403 clearing. Creation remains non-idempotent; uncertain saves require
reloading and inspecting the ledger. The archive stays private and unpublished.

PR #175 (`9af914d580336822336797953e4763740ca643eb`) passed both
hosted checks (push 11049/job 19997, PR 11050/job 19998). It fixes projected
discovery occurrence time-zone conversion and theme-aware focus/coordinate colors. Final full local gate passed: 281
web/34 mobile tests and 421 top-level DB tests (599 including nested), zero
failures/skips. Formatting tests passed under Honolulu/Tokyo viewer zones;
synthetic component-browser checks covered date rollover, fallback, keyboard
open/close/focus return and dark contrast. An observed test DB memory-cgroup OOM
was repaired with a repository-only 1 GiB cap; final resource counters had zero
OOM kills. Native/device, full journey and remaining accessibility gates stay
open. Issue #54/#57 stale CI and #50/#55/#58 source evidence were reconciled
without removing prerequisites or closing issues.

## Merged work and remaining acceptance

| Issue | Verified current state | Next unfinished work |
| --- | --- | --- |
| [#6 IDENT-02](https://git.subcult.tv/subculture-collective/subcult-os/issues/6) | Mobile verification/recovery app links merged in PR #168 at `89b6ea9`. Current main has passing hosted run 10687. | Run the [physical-device checklist](mobile-app-links.md), including domain association, secure storage, restart, refresh, recovery revocation and logout. |
| [#7 MAIL-01](https://git.subcult.tv/subculture-collective/subcult-os/issues/7) | Resend adapter, durable outbox, signed feedback and suppression exist in source. | Qualify configured provider delivery to approved test recipients. Automated qualification keeps sending disabled. |
| [#10 AT-LIVE](https://git.subcult.tv/subculture-collective/subcult-os/issues/10) | Identity-only OAuth source and synthetic checks exist. | Qualify a consenting test identity against the configured HTTPS metadata, callback and JWKS URLs, including refresh and remote revocation. |
| [#50 LIFE-01](https://git.subcult.tv/subculture-collective/subcult-os/issues/50) | Occurrence safeguards and owner-only draft action worklist are merged. Worklist merge `8166559` has passing hosted run 10160. Destination-scoped dispatch, guarded listing notices, per-recipient outcomes and private owner observations merged in #169 at `375451dc`; both head checks passed and merge follow-up job 19978 passed after runner repair. Open #170 clarifies queue state and refreshes the owner worklist; both head checks passed. Generic runtime adapters remain absent; notices use the guarded mail worker. | Qualify live notice delivery and operator reconciliation controls, then the wider coordinated lifecycle. Public/provider effects and refunds retain their own authority and qualification gates. |
| [#51 OFFLINE-01](https://git.subcult.tv/subculture-collective/subcult-os/issues/51) | Synthetic merge model and separate-client process harness exist. The harness passed in this reconciliation. | Physical-device partition/reconnect, persistence, duplicate scan, revocation, device-loss and manual-fallback evidence. Do not enable offline admission from synthetic results. |
| [#54 EXPORT-01](https://git.subcult.tv/subculture-collective/subcult-os/issues/54) | PR #167 merged at `d40221f`. Both required head runs 10168 and 10169 passed at `8317ef3`; the issue now records their successful checks and merge. | Finish prerequisite #25 and accounting-user qualification. A selected accounting-provider format still requires an actual requirement. |
| [#57 PORTALS-01](https://git.subcult.tv/subculture-collective/subcult-os/issues/57) | Participant portal implementation is merged. The issue records source completion but dependency-blocked closure. Shared merge baseline `8166559` has passing hosted run 10160. | Complete prerequisites #25/#36; stale CI wording is corrected and dependency links remain. Do not add vendor fees, invoices or payouts without pilot demand and permission rules. |

The following hosted results were read from Gitea's exact-commit status API:
[main run 10687](https://git.subcult.tv/subculture-collective/subcult-os/actions/runs/10687),
[worklist/portal baseline run 10160](https://git.subcult.tv/subculture-collective/subcult-os/actions/runs/10160),
and [export runs 10168](https://git.subcult.tv/subculture-collective/subcult-os/actions/runs/10168)
and [10169](https://git.subcult.tv/subculture-collective/subcult-os/actions/runs/10169).
These results supersede old queued-run notes; they do not establish deployment.

## Execution order

1. Recover the unmerged development-environment slice described below. Verify
   setup, preview, seeding, watch/restart, isolation and the complete local gates
   before using it for the next issue batch.
2. Finish #6's device qualification and #7/#10's provider qualification with the
   specified identities, recipients and exact configured endpoints. Preserve
   source-complete authority and consent issues while their prerequisites remain
   open.
3. Once #19's prerequisites are complete, implement publication authorization
   and exact payload preview. Keep repository scopes separate from identity-only
   grants and recheck authority before execution.
4. Implement #20's durable publication intents and unknown-outcome
   reconciliation, then #21's pending/conflict/recovery UI and #22's independent
   calendar reader proof.
5. Qualify the joined operator journey (#25), accessibility (#26), security
   (#27), operations (#28), recovery (#29) and isolated deployment candidate
   (#30) before protected-pilot and cutover gates.

The [expansion work order](../superpowers/plans/2026-09-24-expansion-50-71.md)
also permits independent source work ahead of rollout prerequisites. #50's
dispatch infrastructure now includes destination-scoped claims, explicit
dispatch approval, current owner/revision checks and synthetic adapter outcome
tests. No runtime adapter is installed. The local continuation adds private listing
notice previews with exact saved
revision/CID checks, templated content, deduplicated operational recipients and
suppression visibility. Atomic approval/queuing, send-time authority and
relationship guards, and per-recipient outcomes now exist locally. Desktop synthetic notice approval and persistent owner-review journeys passed after browser automation recovered. Next, qualify
live delivery. A private owner review log now records notes with server-observed
recipient outcomes and explains held, retrying, accepted, quarantined and
withheld states. It does not retry messages or resolve uncertain provider
acceptance. Provider-backed reconciliation remains unfinished.
Refunds remain separate from event cancellation.

## Worktree audit and selected recovery

Read-only inspection found eight registered worktrees. Ahead/behind counts are
relative to fetched main `89b6ea9`; dirty state is observed, not inferred from
the branch name.

| Worktree | Head | Ahead / behind main | Observed state |
| --- | --- | --- | --- |
| Base checkout (`main`) | `5790c7a` | 1 / 95 | Modified `t3.json` and untracked `.claude/`; preserved. |
| `dev-environments` | `5790c7a` | 1 / 95 | Clean; contains the unmerged T3 development-environment commit. |
| Current `t3code-124282a2` | `89b6ea9` | 0 / 0 | Clean before this task; recovery and documentation edits now belong here. |
| Detached `t3code-2cc801da` | `da81a00` | 0 / 102 | Clean; head already contained in main. |
| `t3code-cad82a67` | `8317ef3` | 0 / 3 | Clean; finance work already merged. |
| `t3code-e6dba5a4` | `c97f07a` | 0 / 156 | Clean; qualification stack already merged. |
| Claude workflow `wf_72566bfc-c74-1` | `0fa2efa` | 0 / 92 | Clean; announcement implementation already merged. |
| Temporary merge-verification worktree | `966a111` | 0 / 174 | Clean; head already contained in main. |

The next recoverable commit is
`5790c7a4ca87fc3312459179d43035c766b05612`, not the announcement or finance
worktree. It adds `t3.json`, `compose.dev.yml`, `scripts/dev-env.sh`,
`scripts/dev-seed.mjs`, the Vite API proxy target and operating instructions.
The base checkout's local fallback changes only Setup Worktree; main lacks the
repository scripts used by Start Dev and the other project actions.

The commit was recovered into the current branch, preserving
all other worktrees. Current-source corrections mount the Lexicon contracts
read-only, share the installed T3 launcher's host test lock, and retain test
database logs under ignored `.cache/dev-env/` before removing the disposable
test container. The test container now matches the shared recipe's 768 MiB
memory limit and one CPU. The new mock-only regression checks separate checkout project
identity, exclusion of inherited deployment Compose settings, failure exit
propagation and test-only cleanup, including a failed log capture.

### Recovered workflow qualification

`bash scripts/dev-env.sh setup` passed. A stable rerun of `start` exited 0 and
served the worktree's preview on a Docker-assigned loopback port. `seed` passed
twice, retaining one Development Collective workspace and one draft Development
Night event. Browser sign-in reached the operator home with that workspace and
event. Authenticated synthetic occurrence/credit creation and public-preview
validation returned 200. This is local development evidence, not publication.

`watch` restarted the API after an owned temporary backend comment was added;
the container start timestamp changed and Compose reported the restart. The
comment was removed and the source diff checked. The watcher and this task's
preview containers were stopped; the synthetic development database and caches
remain for the next development session.

The original 512 MiB test database failed twice. Kernel cgroup OOM records and
retained PostgreSQL logs confirm killed database processes, not an accepted
application regression. Live observation measured approximately 507 MiB usage
against that 512 MiB limit. The shared recipe's 768 MiB limit had already passed
the baseline; the host had approximately 19 GiB available before the bounded
increase. Failed verification and database logs were retained locally.

After correction, `bash scripts/dev-env.sh verify` exited 0: complete
`make verify` followed by complete disposable `make test-db`, with 252 web,
34 mobile and 385 top-level database tests passing; zero database failures or
skips. ShellCheck, the mock-only development-helper regression, JavaScript
syntax validation, project-action JSON validation and documentation link checks
also passed. The successful test database was removed and its logs retained in
ignored `.cache/dev-env/test-db.log`.

The setup hook has been validated as configuration and its command has run
manually. No new T3 thread creation was exercised. Hosted CI for this recovered
change remains separate from the passing main baseline.

## Local evidence and environment limits

The installed T3 launcher ran `make verify` in an isolated source snapshot at
the revision above, exiting 0. This includes 252 passing web tests, 34 passing
mobile module tests, Go tests/vet/build, client type checks, shared contracts,
web build and Compose/template validation. It does not include database-backed
tests, physical-device behavior or live provider calls.

The complete `make test-db` baseline also passed in the installed launcher's
isolated test environment: 385 top-level passing tests, zero failures and zero
skips. The test PostgreSQL container used temporary storage and was removed.

`node scripts/offline-door-experiment.mjs` exited 0. Two distinct client
processes used separate temporary stores; reconnect produced `accepted`,
`duplicate_check_in` and `duplicate_operation`, with expired and revoked work
rejected or held for review. Fixture hash:
`5ea990313c2bd845d19a7cbe4feb6360c3c97958e782a3df5b80d4900e36e54e`.
Temporary fixture stores were removed by the harness.

T3 device inventory exposes only this Linux host. Android is unavailable
because no SDK is configured; iOS simulators require macOS/Xcode. Neither
inventory nor module tests satisfy #6's physical-device criterion. Use the
[device qualification procedure](mobile-app-links.md) on an available test
device and record its actual build and observed outcomes.

No issue state, dependency, provider setting or deployed runtime was changed
by this reconciliation.

## 2026-09-30 overnight continuation

PR [#169](https://git.subcult.tv/subculture-collective/subcult-os/pulls/169)
remains open at qualified head `b0172c0`. Hosted push run 10947 and PR run
10948 passed the full baseline and disposable database gate. The follow-up
branch `t3code/lifecycle-outcomes-worklist` preserves that head and distinguishes
unapproved drafts from completed local notice queues. It refreshes the worklist
after approval and guards against stale responses following event changes.

The next source slice is #55's access-information model under the owner's
expansion development decision. Start with an owner-only event worksheet with
explicit unknown states, source/review dates, conservative expiry and correction
history. Venue inheritance, public display and accessibility-user evaluation
remain separate unfinished acceptance work. Personal accommodation requests
are excluded; no demand, venue verification or user-study evidence is inferred.
Provider and deployment gates remain unchanged.

### ACCESS-INFO private event worksheet source

The owner-only event worksheet now implements six explicit topics, unknown
states, source/review times, conservative expiry, immutable corrections and
bounded history. See [its contract](event-access-information.md). Full local verification passed: 272 web/34 mobile tests and 412 top-level
database passes (588 with subtests), zero failures/skips. A focused race gate
also passed. Real desktop edit, expiry, correction, withdrawal, reload, access
denial and uncertain-response recovery checks passed with synthetic data. Database
checks cover permission loss, stale/concurrent correction, exact/conflicting
replay, rollback, pagination, upgrade/replay and public absence. This is a source
slice under the owner's expansion decision; no public accessibility claim,
partner demand or accessibility-user evidence is inferred. Venue assertion
history and event verification/public display remain next acceptance work.

### Discovery keyboard follow-up — 2026-09-30

PR #175 remains open with successful push and PR CI. The reproduced occurrence
dialog focus escape has a bounded native-dialog follow-up. Screen-reader and
physical-device qualification remain separate; both native device platforms are
unavailable on this host. See `discovery-ux.md` and the execution log for the
component fixture and final local/hosted receipts.

### Qualified native dialog and next discovery controls — 2026-09-30

PR #176 (`2f3c0e6adae2bcb062014b6c96dc22e52a7d663a`) is open and passed
push 11053/job 20001 and PR 11054/job 20002: 282 web/34 mobile tests,
421 top-level DB tests (599 including nested), no failures/skips. Owned runner
cleanup and zero repository registrations were verified. The remaining map-point
focus defect was reproduced and repaired in the next slice, alongside named
grouped points, result-status markup, public browse copy and themed search focus.
That slice passed the full local gate with 284 web/34 mobile tests and the same
421/599 DB tests. Browser component checks and limits are recorded in
`discovery-ux.md`. Assistive-technology and physical-device qualification remain
open; this work does not authorize live feeds, publication or mail delivery.

### Discovery controls qualified; long-content blocker repaired — 2026-09-30

PR #177 (`ade02d9a1d5a84454e4dd63ca0ace1eb7bf1f143`) passed push
11057/job 20006 and PR 11058/job 20007. Each ran 284 web/34 mobile tests
and 421 top-level DB tests (599 including nested), no failures/skips. Owned
runner cleanup and zero registrations verified; #26 receipt updated.

The follow-up fixes unbroken text expanding a 360px discovery viewport and
pushing the modal Close control out of reach. Browser iframe proof now shows no
horizontal overflow for the original long occurrence fixture, long published
fields and error text; vertical modal scrolling and native keyboard Close,
Reserve and Escape were verified. Final local gate passed 284 web/34 mobile
tests and 421/599 DB tests. This advances #26's long-content/reflow acceptance
for discovery only; physical devices, screen readers and full journeys remain
separate. See `discovery-ux.md` for exact geometry and evidence boundaries.

### Discovery reflow qualified; public booking context repaired — 2026-09-30

PR #178 (`702d2f8b6793632d1240866f2ac9d3d6edba9005`) passed push
11061/job 20010 and PR 11062/job 20011 with 284 web/34 mobile tests and
421/599 DB tests, no failures/skips. Owned runner cleanup and zero repository
registrations were verified; #26 evidence was reconciled.

The next public booking slice scopes guest state to one mounted slug and hides
forms until a matching event read completes. It prevents stale write callbacks
and paid redirects after that view is removed, including A → B → A. Final local
gate passed 286 web/34 mobile tests and 421/599 DB tests, no failures/skips.
Actual Strict Mode component fixtures verified delayed responses, duplicate
submit guards and current success/pending states with seven simulated writes.
See [the booking-context record](public-booking-context.md). Real backend booking
journeys, responsive booking forms, legacy event-zone semantics, assistive
technology, native devices and live providers remain separate gates.

### Public booking context qualified; narrow reflow repaired — 2026-09-30

PR #179 (`2d0dca74cc8f204f0394b451b5e244c09176715d`) passed push
11065/job 20015 and PR 11066/job 20016 with 286 web/34 mobile tests and
421/599 DB tests, no failures/skips. Owned runner cleanup and zero repository
registrations were verified; PR and issue #26 receipts were read back.

The next slice fixes unbroken text overflowing public booking forms at 360px.
Actual scoped iframe proof now fits the original long fixture, normal fields,
confirmation holder/email and long role/read failures. Native keyboard input,
Tab and Enter reach the controls; light/dark focus screenshots inspected.
Full local gate passed 286 web/34 mobile tests and 421/599 DB tests, no
failures/skips. See [the updated booking report](public-booking-context.md).
The joined backend operator journey (#25), event-clock semantics, assistive
technology and physical/native/provider qualification remain unfinished.

### Booking reflow qualified; joined operator rehearsal started — 2026-09-30

PR #180 (`e25312ef23807aef861579ec916bb07965289ad6`) passed PR11069/job20019
and push11070/job20020: 286 web/34 mobile and DB421/599, no failures/skips.
Existing kvant runner preserved; no owned credentials/runner created, zero
repository registrations verified. PR and issue #26 receipts read back.

The separate disposable operator environment now has real browser-created
identity, workspace and published free event state. Its first workflow blocker
was missing owner role setup. The follow-up panel defaults to private and joins
the existing owner API to the editor; final local gate passed 296 web/34 mobile
and DB421/599, no failures/skips. Actual and synthetic recovery evidence is
recorded in [role setup](participation-role-setup.md).
Continue [the operator rehearsal](../qa/operator-journey-2026-09-30.md) through
applications/assignment, staffing, guest/door, finance, closeout and template reuse.
Issue #25 remains partial; paired timing, intended users, physical/native devices,
screen readers and provider/deployment gates remain separate.


### Role setup qualified; invitation receipt corrected — 2026-09-30

PR #181 (`6ed01556ff91b51b7997f74543b77e6802b26731`) passed push11075/job20026
and PR11076/job20027: 296 web/34 mobile and DB421/599, no failures/skips.
Existing kvant runner preserved; no owned runner or credentials created. PR and
issue #25 receipts were read back. Normal invitation acceptance and member
read-only role UI passed; an unauthorized creation returned403 with roles4→4.
An anonymous public application persisted as submitted, with staffing0.

The next small correction replaces “Invite sent” with a created/queued receipt
and unconfirmed delivery. The existing transactional API supplies creation and
queue evidence only. The joined rehearsal and browser connection limit are
recorded in [the operator journey](../qa/operator-journey-2026-09-30.md).
Continue owner application review, staffing/commitments, participant views, free
ticket/door, finance, closeout/template reuse and paired timing qualification.

The invitation-copy candidate passed full pinned local verification: 296 web/34
mobile tests, backend checks/builds and the complete disposable DB gate (exit0).
The test database was removed. Its DB output capture was truncated by the tool
output budget, so this receipt does not derive complete DB test counts from it.
Changed-copy browser and screen-reader proof is pending the preview connection
recovery; no live email was sent. Hosted qualification follows publication.


### Invitation receipt qualified; owner controls corrected — 2026-09-30

PR #182 (`7d9563c3368e04282070df953d02375bae66c02d`) passed push11077/job20028
and PR11078/job20029: 296 web/34 mobile tests, DB421/599, no failures/skips.
Existing kvant runner preserved; no owned credentials or runner created; qualified
PR body and head were read back. The disposable API operations rehearsal passed
36/36 using separate synthetic accounts/events and held mail.

The next UI correction removes the owner-only invitation form and its guidance
entry point from non-owner roles, with a matching submit guard. Focused API member
creation was rejected403 with invitations2→2 and outbox9→9. Scope and evidence
limits are recorded in [the ongoing operator journey](../qa/operator-journey-2026-09-30.md).
Full local verification passed: 304 web/34 mobile, DB421/599, no failures/skips;
test DB removed. Changed-form browser/SR proof still requires
preview connection recovery. Continue the original browser event through owner
application review, assignment/participant views, ticket/door, finance and reuse.


### Owner invitation controls qualified; door rehearsal repaired — 2026-09-30

PR #183 (`b4369fa5a806133961a4c3dc061b7d011928c888`) passed push11081/job20033
and PR11082/job20034: 304 web/34 mobile, DB421/599, no failures/skips. Existing
kvant runner preserved; qualified PR body/head read back, no owned runner needed.

The free-ticket API rehearsal found a stale baseline-member door assumption.
The current server correctly rejected it403. The repaired script proves that
denial, grants the synthetic membership role `door` through the owner API,
checks search/idempotent check-in, then changes the role to `crew` and proves
write denial403. Final report is reserved1/checked-in1/no-shows0; script exit0.
No provider or browser/device claim. See [the operator report](../qa/operator-journey-2026-09-30.md).
The suspected invitation context leak is unconfirmed through normal navigation,
which uses full-page workspace links. Continue joined browser qualification
when the preview connection recovers; owner role-management UI is also not yet
qualified by this API-only role grant. Full local verification passed304 web/34
mobile, DB421/599, no failures/skips; disposable test DB removed. Final script
rehearsal, Bash syntax and ShellCheck with sourced files also passed.


### Door rehearsal qualified; membership access roster added — 2026-09-30

PR #184 (`40a55c4bac0e53ce9efd06e48998342c7ef887df`) passed push11083/job20035
and PR11084/job20036: 304 web/34 mobile, DB421/599, no failures/skips. Existing
kvant runner preserved; qualified PR body/head read back, no owned runner needed.

The next slice makes roster roles and server-derived access status truthful
before owner role-management UI. Private current/scoped workspace reads expose
active/expired/revoked state and timestamps, use no-store and omit removed rows.
No migration or authority mutation change. Web labels all six roles accurately;
inactive/unknown membership does not imply usable permissions. Shared mobile
role types match the server. Full local gate315 web/34 mobile, DB422/606, no
failures/skips. Normal owner expiry/revoke API proof passed in the separate
alpha workspace, with original actor/event state preserved and all21 mail held.
See [the operator report](../qa/operator-journey-2026-09-30.md) for scope and limits.
Continue owner role-management controls and the joined browser work when preview
connection recovers; finance, private closeout/reuse, paired timings, native
devices, screen readers and provider/deployment gates remain separate.


### Membership roster qualified; owner role controls added — 2026-09-30

PR #185 passed both hosted checks at32a474ec77ca2e188332f9e5f40930921719c7b9:
push11089/job20041, PR11090/job20042,315 web/34 mobile, DB422/606, no failures
or skips. Existing kvant runner preserved; qualified PR body/head read back.

The next slice adds a dedicated owner Member roles page using the existing
role-only API. Capability review precedes assignment; canonical refresh follows
a matched receipt. Nonowners and inactive/unknown memberships are read-only.
Denied responses clear private state; unknown write outcomes require refresh
without replay. No expiry/revocation restoration or authority-policy change.
Transport and static-render tests keep browser interaction proof separate.
See [the operator report](../qa/operator-journey-2026-09-30.md).
Continue joined browser qualification when preview recovers, then finance,
private closeout/reuse and paired timings. Native devices, screen readers,
providers and deployment remain separate gates.


Full pinned local verification passed337 web/34 mobile tests, backend checks and
builds, DB422 top-level/606 including nested, no failures/skips. Disposable test
DB removed. The initial TypeScript fixture-cast failure was corrected before the
complete rerun. Scoped review, edited Markdown links and diffcheck passed.


### Owner roles qualified; finance access/write handling — 2026-09-30

PR #186 atdc095ffad2d7b639ed2cae5a91a5bc068bd8ec32 passed both hosted checks,
push11093/job20046 and PR11094/job20047:337 web/34 mobile, DB422/606, no
failures/skips. Existing kvant runner preserved; qualified body/head read back.
Separate normal API finance/private-closeout/reuse rehearsal passed; original
browser actors/event unchanged and all21mailheld. See [the operator report](../qa/operator-journey-2026-09-30.md).

The next source slice requires a successful private ledger read before editing,
serializes writes synchronously, matches receipts to the event and submitted
record, clears denial/uncertainty state and requires refresh without replay.
Known validation/conflict rejections remain editable with unchanged-key retry.
Keyed panel/lifetime guards isolate departed responses. Session/transport and
static-render evidence remains separate from the blocked browser gate.
Continue the original joined browser workflow when preview connection recovers;
paired timings, intended-user, keyboard/SR/native and provider/deployment gates
remain open. The ad-hoc API rehearsal is already finalized; use fresh synthetic
records for a repeat rather than rerunning its precondition-dependent script.


Full pinned local gate passed361 web/34 mobile tests, backend checks/builds,
DB422 top-level/606 including nested, no failures/skips; test DB removed.
Initial key-generator typing and fixture React-import/401-refresh failures were
corrected before the final complete rerun. All attempts retained as local logs.
Scoped review, edited Markdown links and diffcheck passed. Browser proof remains
pending; no live provider/deployment/retained-state action.


### Finance session qualified; fresh API rehearsal — 2026-09-30

PR #187 passed both hosted checks at1cdd075f4c7eed9344a92e55ab89317cdb07538e,
push11097/job20050 and PR11098/job20051:361 web/34 mobile, DB422/606, no
failures/skips. Existing kvant runner preserved; qualified body/head read back.

The next slice adds `make finance-closeout-qa`, a repeatable normal API journey
with fresh verified synthetic actors and records, guarded disposable targets,
private temp cookies and bounded failure output. Two localhost runs passed27
checks each; unsafe targets rejected at both entrypoints. No retained application
or original browser-vector changes. See [the harness guide](../qa/finance-closeout-rehearsal.md)
and [operator evidence](../qa/operator-journey-2026-09-30.md).

Continue joined browser/paired-timing qualification when preview recovers. This
harness does not satisfy physical-device, provider, browser Print/PDF or actual
accounting-user gates, and does not justify enabling offline admission.


A third27-check run through127.0.0.1 also passed. Read-only persisted checks show
three distinct fresh rehearsal workspaces, each with2 members/2 events/4 finance
history rows/1 ticket/1 private archive. Post-run aggregate18people/8workspaces/
13events/6tickets, all35 outbox held. The original browser vector is unchanged.
Full pinned local verification passed361 web/34 mobile, DB422 top-level/606
including nested, no failures/skips; disposable gate DB removed. This gate is
separate from the retained owned rehearsal DB. Edited Markdown links, syntax,
ShellCheck and scoped review passed; no broader acceptance closure.


### Preserve exact starts while editing — 2026-09-30

PR #188 passed both hosted checks atfee51e77b254dcc43cbbe284f0b3dc294426aaf6
(push11101/job20055, PR11102/job20056;361 web/34 mobile, DB422/606).

The next slice preserves the original hydrated event start in web/mobile forms
when minute-level start text is unchanged. Actual old builders shifted a Chicago
repeated-hour instant and lost seconds; reserved-event description edits failed409.
Fixed builders passed200 and kept the exact start/reservation. Full pinned local
gate passed366 web/39 mobile and DB422/606. See the operator evidence for red/green
checks, four viewer-zone model runs and API rehearsal limitations.

Continue hosted qualification, then the first bounded unfinished slice. Joined
browser/native editing, legacy event-zone persistence and deliberate DST gap/fold
selection remain open; this fix does not qualify those gates.


### Browser recovery; staffing identity — 2026-09-30

The stack through #188 is merged; #189 targets main with the exact-start fix.
Its hosted jobs11160/20126 and11161/20127 remain queued behind merge runs.
The preview recovered and the real description-only web save preserved the exact
start/reservation. Original owner/browser application acceptance and task setup
then exposed stale participant options and membership-vs-person staffing identity.

The staffing candidate adds personId to the private roster, uses explicitly active
person identities for member assignment, and refreshes participants on application
updates. Real browser assignment and under-review/accepted option removal/return
passed. Full pinned local gate passed373 web/39 mobile, DB422/606. Continue hosted
monitoring, then the preserved joined journey. Crew portal proof awaits the correct
synthetic credential; no password reset. Free booking/door, commitments, shifts,
closeout/reuse, paired timings and native/provider gates remain open.


### Crew portal and free admission browser stages — 2026-09-30

#189/#190 are merged; observed main717b79a matches the qualified local files.
Hosted checks remain queued on original/updated heads and main. Normal recovery
of the synthetic Crew fixture unblocked its portal: assigned task/requirements
visible, operator note absent. Original owner/browser event now has one completed
private commitment and one browser-reserved/browser-checked-in free ticket.
Persisted staffing1 remains assigned to Crew; all40 outbox rows held.

The next source slice fixes unconfirmed recovery-delivery copy in web/mobile.
Full pinned local gate passed375 web/39 mobile and DB422/606, no failures/skips.
The corrected web generic notice was checked in the real browser. Continue hosted
monitoring and then timed shifts, finance/closeout and private archive/template
reuse in the preserved original event. Paired timings, intended-user and real
device/provider/deployment gates remain open; offline admission stays disabled.


### User-selected Subcults terminal foundation — 2026-09-30

The old `subcults` CSS at `93a13af` is now the shared visual reference: monospace,
square frames, purple actions, black/charcoal dark surfaces and neon accents.
Light/Dark/System preferences remain. Web bundles Space Mono with its license;
native themed styles currently use platform monospace. Shared tokens and the
design gallery are updated. Exact native font loading remains a device task.

Token contrast and the final full pinned local gate passed375 web/39 mobile
tests and the complete disposable database gate (426/610), with no failures or
skips. The gate database was removed. T3 preview currently resolves to a
browser connection error despite host HTTP200, so rendered visual review is
pending. Finish publication and hosted checks, restore preview qualification,
then resume the preserved original-event workflow stages.
