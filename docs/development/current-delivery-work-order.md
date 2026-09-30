# Current delivery work order

Observed 2026-09-29 on Kvant. Source and fetched `origin/main` both point to
`89b6ea9203db0bc3e42d51eb217614e8edc38061`. The working branch is
`t3code/complete-development-issues`; it was clean before this documentation
update. Gitea reports 68 open issues and no open pull requests. This is a dated
reconciliation, not a production qualification receipt.

## Merged work and remaining acceptance

| Issue | Verified current state | Next unfinished work |
| --- | --- | --- |
| [#6 IDENT-02](https://git.subcult.tv/subculture-collective/subcult-os/issues/6) | Mobile verification/recovery app links merged in PR #168 at `89b6ea9`. Current main has passing hosted run 10687. | Run the [physical-device checklist](mobile-app-links.md), including domain association, secure storage, restart, refresh, recovery revocation and logout. |
| [#7 MAIL-01](https://git.subcult.tv/subculture-collective/subcult-os/issues/7) | Resend adapter, durable outbox, signed feedback and suppression exist in source. | Qualify configured provider delivery to approved test recipients. Automated qualification keeps sending disabled. |
| [#10 AT-LIVE](https://git.subcult.tv/subculture-collective/subcult-os/issues/10) | Identity-only OAuth source and synthetic checks exist. | Qualify a consenting test identity against the configured HTTPS metadata, callback and JWKS URLs, including refresh and remote revocation. |
| [#50 LIFE-01](https://git.subcult.tv/subculture-collective/subcult-os/issues/50) | Occurrence safeguards and owner-only draft action worklist are merged. Worklist merge `8166559` has passing hosted run 10160. Destination-scoped dispatch and owner-only listing notice previews are implemented locally; no runtime adapter is installed. | Implement atomic notice approval/queuing, send-time checks, per-recipient outcomes and reconciliation. Public/provider effects and refunds retain their own authority and qualification gates. |
| [#51 OFFLINE-01](https://git.subcult.tv/subculture-collective/subcult-os/issues/51) | Synthetic merge model and separate-client process harness exist. The harness passed in this reconciliation. | Physical-device partition/reconnect, persistence, duplicate scan, revocation, device-loss and manual-fallback evidence. Do not enable offline admission from synthetic results. |
| [#54 EXPORT-01](https://git.subcult.tv/subculture-collective/subcult-os/issues/54) | PR #167 merged at `d40221f`. Both required head runs 10168 and 10169 passed at `8317ef3`; the issue still says they are queued. | Reconcile the issue's stale CI/merge status and prerequisite #25. A selected accounting-provider format still requires an actual requirement. |
| [#57 PORTALS-01](https://git.subcult.tv/subculture-collective/subcult-os/issues/57) | Participant portal implementation is merged. The issue records source completion but dependency-blocked closure. Shared merge baseline `8166559` has passing hosted run 10160. | Reconcile prerequisites without deleting dependency links. Do not add vendor fees, invoices or payouts without pilot demand and permission rules. |

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
suppression visibility. Next, implement atomic approval/queuing, send-time
authority checks, per-recipient outcomes and reconciliation.
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
