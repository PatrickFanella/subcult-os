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
