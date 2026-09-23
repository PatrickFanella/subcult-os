# Operations Panels QA — 2026-09-23

Issue #5, second and fourth acceptance bullets: verify contacts, commitments,
staffing, templates, roles/applications, reminder boundaries, workspace
invitations and workspace switching at the API level, and record failed
cases/inherited gaps as bounded follow-ups.

## Scope and evidence type

This is API-level evidence captured with `curl`/`psql`/`python` against a real
running Docker Compose stack (Go API + Postgres) built from this worktree. It
is **not** browser or native-device evidence. No Chromium/Playwright session
and no Expo/device rehearsal were run for this slice. The core lifecycle
(workspace/event/reservation/door/End of Night) was already qualified with a
headed Chromium rehearsal in an earlier entry; that is not repeated here.

## Served revision and environment

- Revision: `c97f07a58a70c2b20090cc6809bc056329f84342` (`git rev-parse HEAD` in this worktree)
- Compose project: `subcult-os-qual` (isolated; did not touch any other Compose project or container on the host)
- Ports: `POSTGRES_PORT=45432 API_PORT=48080 WEB_PORT=48079`
- Database: `POSTGRES_DB=subcult_qa_operations_c97f07a` (disposable, named `subcult_qa_*` per the existing rehearsal-script guard)
- Stack build: `make up-build` (fresh image build from this worktree's sources)
- Stack health: `make smoke` passed (API `/api/health` and web root both became reachable)

## Exact command

```bash
export COMPOSE_PROJECT_NAME=subcult-os-qual POSTGRES_PORT=45432 API_PORT=48080 WEB_PORT=48079
export POSTGRES_DB=subcult_qa_operations_c97f07a
make up-build
make smoke
export API_URL=http://localhost:48080
export QA_DATABASE_URL="postgres://app:change-me@localhost:45432/subcult_qa_operations_c97f07a?sslmode=disable"
export QA_DISPOSABLE_DATABASE=1
bash scripts/qa-operations.sh
docker compose -p subcult-os-qual down -v
```

`scripts/qa-operations.sh` (new; same conventions as `scripts/alpha-qa.sh`) is
also runnable as `make operations-qa` once the environment variables above are
exported.

## Result

The script ran twice against the same fresh stack build; both runs produced
identical output: **36 passed, 0 failed**, exit code 0.

## Per-step results

| # | Step | Result |
| --- | --- | --- |
| 1 | create owner account | PASS |
| 2 | create member account | PASS |
| 3 | owner creates workspace | PASS |
| 4 | owner invites member | PASS |
| 5 | member accepts invitation | PASS |
| 6 | resolve member personId via `/api/me` | PASS |
| 7 | member switches into invited workspace (`GET /api/workspaces/{id}`) | PASS |
| 8 | owner's `/api/workspaces/current` resolves to the created workspace | PASS |
| 9 | owner creates draft event | PASS |
| 10 | create contact | PASS |
| 11 | list contacts returns the created contact | PASS |
| 12 | update contact (PATCH displayName/tags) | PASS |
| 13 | create commitment linked to event and contact | PASS |
| 14 | commitment update rejects an invalid status (400) | PASS |
| 15 | mark commitment done sets `completedAt`/`completedByPersonId` | PASS |
| 16 | create overdue open commitment for reminder sweep | PASS |
| 17 | create far-future open commitment that must not trigger a reminder | PASS |
| 18 | member cannot create staffing items (403, owner-only boundary) | PASS |
| 19 | owner creates staffing item | PASS |
| 20 | owner assigns staffing item to member (`assignedPersonId`, status `assigned`) | PASS |
| 21 | member cannot PATCH staffing items (403, owner-only boundary) | PASS |
| 22 | create event template | PASS |
| 23 | create draft event to receive the template | PASS |
| 24 | apply template to draft event (title/allocation copied) | PASS |
| 25 | template application conflicts (409) once the event is published | PASS |
| 26 | create public role | PASS |
| 27 | publish event so public roles/applications are reachable | PASS |
| 28 | public applicant submits role application | PASS |
| 29 | role application review rejects an invalid status (400) | PASS |
| 30 | owner accepts role application | PASS |
| 31 | member cannot decide role applications (403, owner-only boundary) | PASS |
| 32 | member cannot sweep reminders (403, owner-only boundary) | PASS |
| 33 | no reminders exist before the overdue commitment is swept | PASS |
| 34 | owner sweeps reminders; creates exactly one reminder for the overdue commitment | PASS |
| 35 | reminder list contains only the overdue commitment's reminder (far-future commitment excluded) | PASS |
| 36 | repeat sweep does not resend the same reminder (idempotency boundary) | PASS |

### Reminder boundary detail

The reminder sweep (`POST /api/workspaces/{id}/reminders/sweep`) is gated by:
role (workspace owner only — members get 403), by state (only `open`
commitments whose `dueAt` has passed produce a reminder; a commitment due 30
days out produced none), and by idempotency (`idempotency_key` uniqueness in
`reminder_events`/`notification_events`, verified by an immediate repeat sweep
returning `createdCount: 0`). The API does not expose a separate recipient
consent flag; recipient selection falls back from the commitment owner to
workspace owners with a verified membership row, which this rehearsal did not
need to exercise as a negative case since the workspace always has an owner.

## Follow-ups (bounded, not fixed here)

1. **No API-level consent/opt-out flag for reminder recipients.**
   - Reproduction: read `backend/internal/app/reminders.go` `commitmentReminderRecipients`/`staffingReminderRecipients` — they resolve an email from workspace membership or an accepted role application with no explicit consent or notification-preference check.
   - Expected vs actual: the acceptance bullet asks to confirm reminders are not sent "without the required consent or state"; the only implemented gate is *state* (due date, role, open status). There is no recipient-level consent record to test.
   - Suggested scope: either confirm with product that state-gating is the intended full boundary (and update the acceptance criteria wording), or add an explicit opt-out/consent column plus a negative test. Small (design decision + one migration + one handler check) if pursued.

2. **Workspace "switching" has no server-side session state.**
   - Reproduction: `backend/internal/app/workspaces.go` `handleCurrentWorkspace` always returns the member's most-recently-created workspace; there is no endpoint that persists an "active workspace" selection. The web client (`web/src/modules/workspace/workspaceLoaders.ts`) implements switching purely client-side via `GET /api/workspaces/{id}`.
   - Expected vs actual: this is not a defect — it is documented client behavior — but it means "workspace switching" cannot be verified as a server-tracked state transition, only as an authorized read of a specific workspace by ID. Recorded here so a future reviewer does not look for a missing switch endpoint.
   - Suggested scope: none required; documentation note only.

3. **Broader operations-panel browser/device coverage remains open.**
   - Reproduction: this rehearsal is API-only (`curl`/`psql`), matching the existing `alpha-qa.sh`/`fake-event-qa.sh` pattern.
   - Expected vs actual: contacts/commitments/staffing/templates/roles/reminders UI panels in `web/src` have not been driven through a real browser in this slice, and no mobile/Expo staff-flow coverage was added.
   - Suggested scope: a follow-up headed-browser rehearsal (Playwright/Chromium) covering the operations panels, similar in shape to the existing lifecycle rehearsal referenced in the 2026-09-20 execution-log entry. Medium — requires UI selectors research per panel.

No product defects were found while exercising this slice; all owner-only
authorization boundaries, status-transition validation, and idempotent-sweep
behavior matched the code as read.
