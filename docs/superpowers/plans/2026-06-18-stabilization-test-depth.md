# Stabilization and Test Depth Sprint Implementation Plan

> **For agentic workers:** Execute this plan task-by-task. Recommended path:
> dispatch a fresh subagent per task, review each result with `review-quality`,
> then continue. For complex multi-agent splits, use
> `parallel-feature-development`, `team-composition-patterns`, and
> `team-communication-protocols`. Steps use checkbox (`- [ ]`) syntax for
> tracking.

**Goal:** Stabilize the architecture-deepened app through focused manual QA, regression fixes, and stronger automated tests around the new Modules and critical Event lifecycle.

**Architecture:** This sprint is not a feature sprint. Keep product behavior stable, use the new Deep Modules as test seams, and fix only regressions found by QA or test coverage gaps. Prefer small commits that either add evidence, fix a discovered regression, or strengthen a test harness.

**Tech Stack:** Go API with `net/http`, `pgx`, Postgres, Docker Compose, Vite React TypeScript with Vitest, Expo SDK 54 React Native mobile, Makefile verification.

---

## Scope and sequencing

This combines the **Stabilization sprint** and **Test depth sprint** into one evidence-first sprint:

1. Establish a manual QA matrix and runbook for the full Event lifecycle.
2. Add always-on unit tests around backend Modules where DB tests currently skip.
3. Add a real DB-backed integration test command so skipped lifecycle tests can run intentionally.
4. Add mobile pure-module tests for new mobile Modules.
5. Run web/mobile/API smoke passes, file regressions as checklist items, and fix only concrete failures.
6. End with `make verify` plus a documented QA result.

## File structure target

### Documentation and QA artifacts

- Create: `docs/qa/stabilization-test-depth-2026-06.md` — sprint QA matrix, manual test routes, observed results, and fixed regressions.
- Modify: `README.md` — only if the sprint changes the recommended local verification or DB-backed test commands.
- Modify: `docs/stacks.md` — only if the sprint adds a new persistent test convention.

### Backend tests

- Create: `backend/internal/app/test_fixtures_test.go` — shared non-DB helper builders for Module tests if duplication appears while adding tests.
- Modify: `backend/internal/app/event_lifecycle_test.go` — add always-on unit coverage for lifecycle rule edge cases not requiring Postgres.
- Modify: `backend/internal/app/ticket_journey_test.go` — add always-on unit coverage for reservation/payment/check-in edge cases.
- Modify: `backend/internal/app/media_storage_test.go` — add non-DB tests for unavailable media storage and public URL behavior if not already covered.
- Modify: `Makefile` — add an explicit DB-backed test target if the repo does not already have one.

### Web tests

- Modify: `web/src/App.test.tsx` — add one or two high-value shell tests only if manual QA finds a route regression.
- Modify focused Module tests under `web/src/modules/**` only when QA finds a missed behavior rule.

### Mobile tests

- Create: `mobile/package.json` test script changes only if Vitest is added.
- Create: `mobile/vitest.config.ts` — mobile pure-module test config, if Vitest is not already configured for mobile.
- Create: `mobile/src/modules/events/eventEditModel.test.ts` — pure tests for Event edit payload/readiness/image selection helpers.
- Create: `mobile/src/modules/runOfShow/runOfShowModel.test.ts` — pure tests for Run of Show ordering/status helper behavior.
- Create: `mobile/src/modules/tickets/ticketJourney.test.ts` — pure tests for Ticket display/wallet helper behavior.
- Create: `mobile/src/modules/discovery/discoveryModel.test.ts` — pure tests for discovery copy/query/result helpers.
- Modify: `Makefile` — include mobile tests in `verify` only if they are fast and stable.

---

## Task 1: Create stabilization QA matrix

**Files:**

- Create: `docs/qa/stabilization-test-depth-2026-06.md`

- [ ] **Step 1: Add the QA document**

Create `docs/qa/stabilization-test-depth-2026-06.md` with this content:

```markdown
# Stabilization and Test Depth QA — 2026-06

## Goal

Verify that the architecture-deepened Modules preserve the current Event lifecycle across web, API, and mobile, then record the automated tests added to prevent regressions.

## Required commands

Run before and after fixes:

```bash
make verify
```

Run when `TEST_DATABASE_URL` is available:

```bash
make test-db
```

## Manual QA matrix

| Area | Path | Steps | Expected result | Result | Notes |
| --- | --- | --- | --- | --- | --- |
| Web auth | `/login` | Sign up, log out, log in | Current Workspace loads without session errors | Not run | |
| Workspace | `/workspace` | Switch Workspace, view Events, open Event editor | Event cards keep old status copy: Draft/Live/Closed | Not run | |
| Event create | `/events/new` | Create draft with title, description, location, allocation | Draft Event saves and returns to editor | Not run | |
| Event publish | `/events/:id` | Publish complete draft | Public Event Page link works | Not run | |
| Public Event Page | `/e/:slug` | Reserve a free Ticket | Ticket page opens with raw Ticket code | Not run | |
| Door Record | `/door` | Look up Ticket and check in twice | Second check-in is idempotent | Not run | |
| End of Night | `/events/:id` | End the night after check-in | Event Report, Archive, Settlement sections load | Not run | |
| Discovery | `/discover` | Search published Events | Event Discovery remains active and copy is unchanged | Not run | |
| Mobile attendee | Expo app | Open discovery, Event detail, reserve/check Ticket | Attendee flow works without session header regressions | Not run | |
| Mobile staff | Expo app | Select Workspace/Event, use Door and Run of Show | Operator flow works and ordering is stable | Not run | |

## Regression log

| ID | Found in area | Symptom | Fix commit | Verification |
| --- | --- | --- | --- | --- |

## Automated test additions

| Test file | Behavior covered | Command |
| --- | --- | --- |
```

- [ ] **Step 2: Verify the document renders as plain Markdown**

Run:

```bash
test -f docs/qa/stabilization-test-depth-2026-06.md
```

Expected: command exits with status `0`.

- [ ] **Step 3: Commit the QA matrix**

Run:

```bash
git add docs/qa/stabilization-test-depth-2026-06.md
git commit -m "docs: add stabilization qa matrix"
```

Expected: one docs-only commit.

---

## Task 2: Add explicit DB-backed backend test target

**Files:**

- Modify: `Makefile`
- Modify: `docs/stacks.md`
- Modify: `docs/qa/stabilization-test-depth-2026-06.md`

- [ ] **Step 1: Add a DB-backed test target**

Add this target to `Makefile` near `test-backend`:

```make
test-db: ## Run DB-backed Go integration tests when TEST_DATABASE_URL is set
	@if [ -z "$${TEST_DATABASE_URL}" ]; then \
		echo "TEST_DATABASE_URL is required for DB-backed tests"; \
		exit 1; \
	fi
	cd backend && go test ./internal/app -run 'TestFirstEventLifecycleCurrentCreatePublishFreeDoorEndOfNightFlow|TestTicketReservationCurrentCapacityAndDoorRules|TestRunMigrationsCreatesEventsTable' -count=1 -v
```

Also add `test-db` to the `.PHONY` line.

- [ ] **Step 2: Run the target without DB URL to verify the guard**

Run:

```bash
unset TEST_DATABASE_URL; make test-db
```

Expected: FAIL with `TEST_DATABASE_URL is required for DB-backed tests`.

- [ ] **Step 3: Document the target**

In `docs/stacks.md`, add this under the verification section:

```markdown
- `make test-db` runs DB-backed Go integration tests and requires `TEST_DATABASE_URL`. These tests cover the full Event lifecycle, Ticket capacity/Door behavior, and schema loading. They are intentionally separate from `make verify` so local verification remains fast and does not require a database fixture.
```

- [ ] **Step 4: Update the QA matrix command status**

In `docs/qa/stabilization-test-depth-2026-06.md`, keep the `make test-db` command and add this note below it:

```markdown
If no disposable Postgres test database is available, record `Not run — TEST_DATABASE_URL unavailable` in the manual QA notes instead of silently treating skipped DB tests as coverage.
```

- [ ] **Step 5: Verify and commit**

Run:

```bash
make verify
```

Expected: PASS.

Commit:

```bash
git add Makefile docs/stacks.md docs/qa/stabilization-test-depth-2026-06.md
git commit -m "test: add explicit db integration target"
```

---

## Task 3: Strengthen always-on backend Module tests

**Files:**

- Modify: `backend/internal/app/event_lifecycle_test.go`
- Modify: `backend/internal/app/ticket_journey_test.go`
- Modify: `backend/internal/app/media_storage_test.go`

- [ ] **Step 1: Add Event lifecycle edge tests**

In `backend/internal/app/event_lifecycle_test.go`, add tests that do not touch the database:

```go
func TestEventLifecyclePublishedReservationGates(t *testing.T) {
	lifecycle := eventLifecycle{Status: eventStatusPublished, ReservationCount: 2}

	if lifecycle.CanChangeTitle("Old", "New") {
		t.Fatal("published events should not allow title changes")
	}
	if lifecycle.CanChangeStartsAt("2026-06-19T20:00:00Z", "2026-06-19T21:00:00Z") {
		t.Fatal("published events with reservations should not allow startsAt changes")
	}
	if lifecycle.CanChangeTicketAllocation(1) {
		t.Fatal("allocation cannot drop below reservations")
	}
	if lifecycle.CanChangePricing("free", 0, "fixed", 2000) {
		t.Fatal("pricing cannot change after reservations")
	}
}

func TestEventLifecycleDraftAllowsPreparationChanges(t *testing.T) {
	lifecycle := eventLifecycle{Status: eventStatusDraft, ReservationCount: 0}

	if !lifecycle.CanEdit() {
		t.Fatal("draft events should be editable")
	}
	if !lifecycle.CanChangeTitle("Old", "New") {
		t.Fatal("draft events should allow title changes")
	}
	if !lifecycle.CanChangeTicketAllocation(1) {
		t.Fatal("draft events should allow allocation changes")
	}
}
```

- [ ] **Step 2: Add Ticket journey precedence tests**

In `backend/internal/app/ticket_journey_test.go`, add:

```go
func TestTicketJourneyCapacityPrecedenceForPaidEvents(t *testing.T) {
	if !ticketJourneyIsFull(1, 1) {
		t.Fatal("expected event to be full")
	}
	if ticketJourneyCanReservePublic("fixed", 1, 1) {
		t.Fatal("fixed-price full event should not allow public free reservation")
	}
}

func TestTicketJourneyCheckInRules(t *testing.T) {
	if !ticketJourneyCanCheckIn("free") {
		t.Fatal("free tickets should be check-in eligible")
	}
	if !ticketJourneyCanCheckIn("paid") {
		t.Fatal("paid tickets should be check-in eligible")
	}
	if ticketJourneyCanCheckIn("pending") {
		t.Fatal("pending tickets should not be check-in eligible")
	}
}
```

- [ ] **Step 3: Add media storage non-DB tests**

In `backend/internal/app/media_storage_test.go`, add tests for public URL generation that do not require Postgres:

```go
func TestS3MediaStoragePublicURLCleansObjectKey(t *testing.T) {
	storage := &s3MediaStorage{publicBaseURL: "https://cdn.example.test/media"}

	got := storage.publicURL("events//image.png")
	want := "https://cdn.example.test/media/events/image.png"
	if got != want {
		t.Fatalf("publicURL() = %q, want %q", got, want)
	}
}
```

If `publicURL` is not currently a method, add this private method to `backend/internal/app/media_storage.go` and call it from the upload path:

```go
func (s *s3MediaStorage) publicURL(objectKey string) string {
	return strings.TrimRight(s.publicBaseURL, "/") + "/" + path.Clean(objectKey)
}
```

- [ ] **Step 4: Run focused backend tests**

Run:

```bash
cd backend && go test ./internal/app -run 'EventLifecycle|TicketJourney|MediaStorage' -count=1 -v
```

Expected: PASS without requiring `TEST_DATABASE_URL` for the new unit tests.

- [ ] **Step 5: Run full verification and commit**

Run:

```bash
make verify
```

Expected: PASS.

Commit:

```bash
git add backend/internal/app/event_lifecycle_test.go backend/internal/app/ticket_journey_test.go backend/internal/app/media_storage_test.go backend/internal/app/media_storage.go
git commit -m "test: deepen backend module coverage"
```

---

## Task 4: Add mobile pure-module test harness

**Files:**

- Modify: `mobile/package.json`
- Create: `mobile/vitest.config.ts`
- Create: `mobile/src/modules/events/eventEditModel.test.ts`
- Create: `mobile/src/modules/runOfShow/runOfShowModel.test.ts`
- Create: `mobile/src/modules/tickets/ticketJourney.test.ts`
- Create: `mobile/src/modules/discovery/discoveryModel.test.ts`
- Modify: `Makefile`

- [ ] **Step 1: Add Vitest to mobile dev dependencies**

Run:

```bash
pnpm --dir mobile add -D vitest
```

Expected: `mobile/package.json` and the lockfile update.

- [ ] **Step 2: Add the mobile test script**

In `mobile/package.json`, add this script:

```json
"test": "vitest run"
```

- [ ] **Step 3: Add mobile Vitest config**

Create `mobile/vitest.config.ts`:

```ts
import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    environment: 'node',
    include: ['src/modules/**/*.test.ts'],
  },
});
```

- [ ] **Step 4: Add Event edit model tests**

Create `mobile/src/modules/events/eventEditModel.test.ts` with tests for payload and readiness:

```ts
import { describe, expect, it } from 'vitest';

import { buildEventEditPayload, defaultEventEditForm, eventEditReadinessWarnings } from './eventEditModel';

describe('eventEditModel', () => {
  it('builds a fixed-price payload with cents and uppercase currency', () => {
    const form = {
      ...defaultEventEditForm(),
      title: ' Night Market ',
      description: ' Doors at eight ',
      location: ' Warehouse ',
      startsAt: '2026-06-19 20:00',
      ticketAllocation: '25',
      pricingMode: 'fixed',
      ticketPrice: '12.50',
      currency: 'usd',
    };

    expect(buildEventEditPayload(form)).toMatchObject({
      title: 'Night Market',
      description: 'Doors at eight',
      location: 'Warehouse',
      ticketAllocation: 25,
      pricingMode: 'fixed',
      ticketPriceCents: 1250,
      currency: 'USD',
    });
  });

  it('reports readiness warnings for missing publish fields', () => {
    expect(eventEditReadinessWarnings(defaultEventEditForm())).toEqual([
      'Add a title.',
      'Add a start time.',
      'Add a location.',
      'Add a public description.',
    ]);
  });
});
```

If the actual exported names differ, adapt the imports to the names in `mobile/src/modules/events/eventEditModel.ts` and keep the assertions equivalent.

- [ ] **Step 5: Add Run of Show model tests**

Create `mobile/src/modules/runOfShow/runOfShowModel.test.ts` with tests mirroring web ordering:

```ts
import { describe, expect, it } from 'vitest';

import { sortRunOfShowItems } from './runOfShowModel';

describe('runOfShowModel', () => {
  it('sorts by status, start time, created time, then id', () => {
    const items = [
      { id: 'done', status: 'completed', startsAt: '2026-06-19T19:00:00Z', createdAt: '2026-06-01T00:00:00Z' },
      { id: 'late-open', status: 'open', startsAt: null, createdAt: '2026-06-01T00:00:00Z' },
      { id: 'early-open', status: 'open', startsAt: '2026-06-19T20:00:00Z', createdAt: '2026-06-01T00:00:00Z' },
      { id: 'assigned', status: 'assigned', startsAt: '2026-06-19T18:00:00Z', createdAt: '2026-06-01T00:00:00Z' },
    ];

    expect(sortRunOfShowItems(items as never).map((item) => item.id)).toEqual(['early-open', 'late-open', 'assigned', 'done']);
  });
});
```

- [ ] **Step 6: Add Ticket and Discovery model smoke tests**

Create focused tests that assert current labels/copy exactly. Use the exported helper names from `mobile/src/modules/tickets/ticketJourney.ts` and `mobile/src/modules/discovery/discoveryModel.ts`; if a helper is not exported, export the smallest pure helper needed by the screen.

For `mobile/src/modules/discovery/discoveryModel.test.ts`, include this assertion shape:

```ts
expect(discoveryEmptyMessage(false)).toBe('Published Events will appear here when hosts share them.');
expect(discoveryEmptyMessage(true)).toBe('No Events match that search yet.');
```

For `mobile/src/modules/tickets/ticketJourney.test.ts`, include this assertion shape:

```ts
expect(ticketPaymentLabel('free')).toBe('Free');
expect(ticketPaymentLabel('pending')).toBe('Pending');
expect(ticketPaymentLabel('paid')).toBe('Paid');
```

- [ ] **Step 7: Wire mobile tests into Makefile**

Add target:

```make
test-mobile: ## Run mobile pure module tests
	pnpm --dir mobile run test
```

Add `test-mobile` to `.PHONY` and change the `test` aggregate to include it if the mobile suite runs in under five seconds:

```make
test: test-backend test-web test-mobile ## Run automated tests
```

- [ ] **Step 8: Verify and commit**

Run:

```bash
pnpm --dir mobile run test
pnpm --dir mobile run typecheck
make verify
```

Expected: PASS.

Commit:

```bash
git add mobile/package.json mobile/vitest.config.ts mobile/src/modules Makefile pnpm-lock.yaml
git commit -m "test: add mobile module test harness"
```

---

## Task 5: Run manual stabilization pass and fix concrete regressions

**Files:**

- Modify: `docs/qa/stabilization-test-depth-2026-06.md`
- Modify: exact app files only when a regression is reproduced.

- [ ] **Step 1: Start the stack**

Run:

```bash
make up-build
```

Expected: Docker Compose starts the backend, web, and database services.

- [ ] **Step 2: Record service URLs**

Run:

```bash
make urls
```

Expected: local web/API URLs are printed. Copy the URLs into `docs/qa/stabilization-test-depth-2026-06.md` under a new `## Environment` section.

- [ ] **Step 3: Run the web lifecycle QA matrix**

In the browser, execute these rows in order and update the `Result` column:

1. Web auth.
2. Workspace.
3. Event create.
4. Event publish.
5. Public Event Page.
6. Door Record.
7. End of Night.
8. Discovery.

For every failed row, add an entry to the regression log with a concrete symptom and reproduction route.

- [ ] **Step 4: Run mobile operator smoke if Expo is available**

Run:

```bash
make dev-mobile
```

Expected: Expo starts using the current Makefile mobile API URL. In the Expo app, run the Mobile attendee and Mobile staff rows. Record `Passed`, `Failed`, or `Not run — Expo/device unavailable`.

- [ ] **Step 5: Fix only reproduced regressions**

For each failed QA row, make the smallest code change that restores pre-refactor behavior. Add or update a focused test before the fix when the regression is in a pure Module. Use this commit format per regression:

```bash
git add <changed-files>
git commit -m "fix: stabilize <area> regression"
```

- [ ] **Step 6: Stop the stack**

Run:

```bash
make down
```

Expected: Docker Compose services stop cleanly.

---

## Task 6: Final evidence review

**Files:**

- Modify: `docs/qa/stabilization-test-depth-2026-06.md`
- Modify: `README.md` or `docs/stacks.md` only if commands changed.

- [ ] **Step 1: Run all verification commands**

Run:

```bash
make verify
```

Expected: PASS.

If `TEST_DATABASE_URL` is available, run:

```bash
make test-db
```

Expected: PASS.

- [ ] **Step 2: Update automated test additions table**

In `docs/qa/stabilization-test-depth-2026-06.md`, fill the automated test additions table with every test file added this sprint. Example row:

```markdown
| `mobile/src/modules/events/eventEditModel.test.ts` | Event edit payload/readiness helpers | `pnpm --dir mobile run test` |
```

- [ ] **Step 3: Run final quality review**

Ask review to check:

```text
Review the stabilization/test-depth sprint. Verify that manual QA findings are recorded, tests cover new Module Interfaces rather than view internals, DB-backed tests are explicit, and no broad feature work or UX drift slipped into the sprint.
```

- [ ] **Step 4: Fix review findings**

Apply only concrete fixes from the review. If a finding requires a product decision, add it to the QA document under `## Follow-up candidates` with a one-sentence reason.

- [ ] **Step 5: Final sprint commit**

Run:

```bash
git add docs/qa/stabilization-test-depth-2026-06.md README.md docs/stacks.md
git commit -m "docs: record stabilization test evidence"
```

Skip this commit if those files have no changes after prior task commits.

---

## Acceptance criteria

- `make verify` passes at the end of the sprint.
- `make test-db` exists and fails loudly when `TEST_DATABASE_URL` is missing.
- DB-gated lifecycle tests are not mistaken for normal `make verify` coverage.
- Backend lifecycle, Ticket journey, and media storage have always-on unit coverage for important rules.
- Mobile pure Modules have a working test harness and focused tests.
- Manual QA matrix records pass/fail/not-run status for web, API, and mobile flows.
- Every fixed regression has a reproduction note and an automated test where practical.
- No new product feature work is mixed into the sprint.

## Risks and guardrails

- **Risk:** The sprint becomes a feature sprint. **Guardrail:** only fix reproduced regressions or missing tests.
- **Risk:** DB tests silently skip. **Guardrail:** add `make test-db` with a hard `TEST_DATABASE_URL` guard.
- **Risk:** Mobile test harness gets tangled with native runtime APIs. **Guardrail:** test only pure `mobile/src/modules/**` helpers in Node.
- **Risk:** Manual QA is performed but not reusable. **Guardrail:** record exact routes, commands, and results in `docs/qa/stabilization-test-depth-2026-06.md`.
