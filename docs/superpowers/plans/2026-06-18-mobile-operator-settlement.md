# Mobile Operator and Settlement Implementation Plan

> **For agentic workers:** Execute this plan task-by-task. Recommended path:
> dispatch a fresh subagent per task, review each result with `review-quality`,
> then continue. For complex multi-agent splits, use
> `parallel-feature-development`, `team-composition-patterns`, and
> `team-communication-protocols`. Steps use checkbox (`- [ ]`) syntax for
> tracking.

**Goal:** Make the mobile staff experience usable during a real Event and deepen Settlement / End of Night so Hosts can close the Event with a trustworthy report, notes, and payout summary.

**Architecture:** Build on the architecture-deepened Modules from the previous sprint. Keep mobile screens thin by extending `mobile/src/modules/**` helpers, keep backend Settlement rules server-owned, and use web/mobile views as Adapters over stable Interfaces. Do not turn this into a Discovery or payment-provider sprint.

**Tech Stack:** Go API with `net/http`, `pgx`, Postgres, Docker Compose, Vite React TypeScript with Vitest, Expo SDK 54 React Native mobile, Vitest for mobile pure Module tests, Makefile verification.

---

## Scope and sequencing

This combines two product slices:

1. **Mobile operator sprint** — Workspace/Event selection, Door lookup/check-in, Run of Show, Ticket wallet, network/error states, and device QA evidence.
2. **Settlement / End of Night sprint** — Event Report, Settlement summary, adjustment notes, Archive notes, and testable closeout rules.

The sequencing is evidence-first:

1. Record current mobile/operator and End of Night QA baseline.
2. Improve mobile staff selection and Door/Run of Show reliability.
3. Add backend Settlement rule tests and small helpers.
4. Improve web End of Night / Settlement presentation without route or DTO drift.
5. Add mobile closeout read-only visibility where it helps operators.
6. Final QA pass with `make verify`, `make test-db` when available, and device/browser notes.

## File structure target

### QA and docs

- Create: `docs/qa/mobile-operator-settlement-2026-06.md` — baseline/final QA matrix and command results.
- Modify: `docs/stacks.md` only if this sprint adds a durable test or mobile QA convention.
- Modify: `README.md` only if a user-facing run command changes.

### Backend

- Modify: `backend/internal/app/events.go` — keep handlers thin; call Settlement helpers where useful.
- Create: `backend/internal/app/settlement.go` — pure Settlement / End of Night summary helpers if equivalent logic is currently embedded in handlers.
- Create: `backend/internal/app/settlement_test.go` — always-on tests for Settlement totals, no-show counts, payout adjustments, and idempotent report behavior.
- Modify: `backend/internal/app/lifecycle_test.go` — add DB-backed tests only if `TEST_DATABASE_URL` coverage is needed for a full closeout flow.

### Web

- Modify: `web/src/modules/eventEditor/eventEditorModel.ts` — add focused Settlement view-model helpers.
- Modify: `web/src/modules/eventEditor/eventEditorModel.test.ts` — cover new Settlement helpers.
- Modify: `web/src/views/EventEditorView.tsx` — consume helpers without changing routes or copy except where this plan says to clarify closeout.
- Modify: `web/src/modules/workspace/workspaceModel.ts` only if Archive/Settlement cards need shared status helpers.

### Mobile

- Modify: `mobile/app/staff.tsx` — improve Workspace/Event selection states and error recovery.
- Modify: `mobile/app/door.tsx` — improve lookup/check-in feedback and retry path.
- Modify: `mobile/app/run-of-show.tsx` — improve operator status updates and stale-order handling.
- Modify: `mobile/app/tickets.tsx` and `mobile/app/ticket.tsx` — harden wallet/pending-checkout states.
- Create or modify: `mobile/src/modules/staff/staffOperatorModel.ts` — pure helper for selected Workspace/Event readiness and empty states.
- Modify: `mobile/src/modules/runOfShow/runOfShowModel.ts` and tests — add any missing operator helper coverage.
- Modify: `mobile/src/modules/tickets/ticketJourney.ts` and tests — add wallet/arrival helper coverage if missing.
- Create: `mobile/src/modules/settlement/settlementModel.ts` — read-only closeout summary helpers if mobile needs to display Event Report / Settlement.
- Create: `mobile/src/modules/settlement/settlementModel.test.ts` — pure tests for mobile closeout display helpers.

---

## Task 1: Record baseline operator and closeout QA

**Files:**

- Create: `docs/qa/mobile-operator-settlement-2026-06.md`

- [ ] **Step 1: Create the QA baseline document**

Create `docs/qa/mobile-operator-settlement-2026-06.md`:

```markdown
# Mobile Operator and Settlement QA — 2026-06

## Goal

Verify mobile operator readiness and End of Night / Settlement behavior before and after the sprint.

## Environment

- Host: Linux
- Browser: Not run
- Expo/device: Not run
- API URL used by mobile: Not recorded

## Required commands

```bash
make verify
```

```bash
make test-db
```

Run `make test-db` only when `TEST_DATABASE_URL` points at a disposable Postgres database.

## Baseline command results

| Command | Result | Notes |
| --- | --- | --- |
| `make verify` | Not run | |
| `make test-db` | Not run | TEST_DATABASE_URL unavailable unless recorded otherwise |
| `make up-build` | Not run | |
| `make smoke` | Not run | |
| `make down` | Not run | |

## Mobile operator QA matrix

| Area | Path | Steps | Expected result | Baseline | Final | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| Staff shell | Expo `/staff` | Open staff mode after login | Selected Workspace and Event are clear; errors are actionable | Not run | Not run | |
| Workspace selection | Expo `/staff` | Switch Workspace | Event list and persisted selection update safely | Not run | Not run | |
| Event selection | Expo `/staff` | Switch active Event | Door, Run of Show, and dashboard point at the selected Event | Not run | Not run | |
| Door lookup | Expo `/door` | Enter valid Ticket code | Ticket details load with raw Ticket code and payment status | Not run | Not run | |
| Door check-in | Expo `/door` | Check in the same Ticket twice | First check-in succeeds; second is idempotent | Not run | Not run | |
| Run of Show | Expo `/run-of-show` | Change task status | List reorders by status/start/created/id and shows stable copy | Not run | Not run | |
| Ticket wallet | Expo `/tickets` | Open wallet with pending and checked-in Tickets | Pending/ready/checked-in states are distinguishable | Not run | Not run | |
| Poor network | Expo app | Disable API or use wrong API URL | Errors explain recovery; app does not lose selected Workspace/Event | Not run | Not run | |

## Settlement / End of Night QA matrix

| Area | Path | Steps | Expected result | Baseline | Final | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| End of Night | Web `/events/:id` | End a published Event | Report is created once and Event status becomes End of Night | Not run | Not run | |
| Report counts | Web `/events/:id` | Compare reserved/check-in/no-show counts | Counts match Tickets and Door records | Not run | Not run | |
| Settlement totals | Web `/events/:id` | Review gross/fees/net/manual adjustments | Totals are readable and internally consistent | Not run | Not run | |
| Archive notes | Web `/events/:id` | Add closeout/archive notes | Notes persist and appear in Workspace archive | Not run | Not run | |
| Mobile closeout visibility | Expo staff flow | View closed Event summary if available | Mobile shows read-only status/report cue or clear web handoff | Not run | Not run | |

## Regression log

| ID | Area | Symptom | Fix | Verification |
| --- | --- | --- | --- | --- |

## Automated test additions

| Test file | Behavior covered | Command |
| --- | --- | --- |
```

- [ ] **Step 2: Run baseline non-interactive checks**

Run:

```bash
make verify
```

Expected: PASS.

Run:

```bash
unset TEST_DATABASE_URL; make test-db
```

Expected: FAIL with `TEST_DATABASE_URL is required for DB-backed tests`.

- [ ] **Step 3: Update baseline command results**

In `docs/qa/mobile-operator-settlement-2026-06.md`, set:

```markdown
| `make verify` | Passed | Baseline non-DB verification |
| `make test-db` | Guard verified | TEST_DATABASE_URL unavailable; integration coverage not run |
```

- [ ] **Step 4: Record browser/device availability honestly**

If no browser or Expo device is available, keep the QA matrix rows as `Not run` and add notes:

```markdown
browser unavailable in agent environment
Expo/device unavailable in agent environment
```

---

## Task 2: Improve mobile staff selection resilience

**Files:**

- Create: `mobile/src/modules/staff/staffOperatorModel.ts`
- Create: `mobile/src/modules/staff/staffOperatorModel.test.ts`
- Modify: `mobile/app/staff.tsx`
- Modify: `mobile/src/staff/selectionStore.ts` only if persistence helper coverage exposes a bug.

- [ ] **Step 1: Add pure staff operator helpers**

Create `mobile/src/modules/staff/staffOperatorModel.ts`:

```ts
import type { EventDTO, WorkspaceSummaryDTO } from '@/api/types';

export function selectedWorkspaceLabel(workspace: WorkspaceSummaryDTO | null) {
  return workspace?.name ?? 'Select a Workspace';
}

export function selectedEventLabel(event: EventDTO | null) {
  return event?.title ?? 'Select an Event';
}

export function staffSelectionReady(workspace: WorkspaceSummaryDTO | null, event: EventDTO | null) {
  return Boolean(workspace && event);
}

export function staffSelectionEmptyCopy(workspaces: WorkspaceSummaryDTO[], events: EventDTO[]) {
  if (workspaces.length === 0) return 'No Workspaces available for this account.';
  if (events.length === 0) return 'No Events available in this Workspace yet.';
  return 'Choose a Workspace and Event to begin.';
}

export function nextSelectedEvent(currentEventId: string | null, events: EventDTO[]) {
  if (currentEventId) {
    const match = events.find((event) => event.id === currentEventId);
    if (match) return match;
  }
  return events[0] ?? null;
}
```

- [ ] **Step 2: Add staff operator tests**

Create `mobile/src/modules/staff/staffOperatorModel.test.ts`:

```ts
import { describe, expect, it } from 'vitest';

import { nextSelectedEvent, selectedEventLabel, selectedWorkspaceLabel, staffSelectionEmptyCopy, staffSelectionReady } from './staffOperatorModel';

describe('staffOperatorModel', () => {
  const workspace = { id: 'workspace-1', name: 'Main Workspace', role: 'owner' } as const;
  const event = { id: 'event-1', title: 'Night Market' } as never;

  it('labels selected Workspace and Event', () => {
    expect(selectedWorkspaceLabel(workspace)).toBe('Main Workspace');
    expect(selectedWorkspaceLabel(null)).toBe('Select a Workspace');
    expect(selectedEventLabel(event)).toBe('Night Market');
    expect(selectedEventLabel(null)).toBe('Select an Event');
  });

  it('reports readiness only when Workspace and Event are selected', () => {
    expect(staffSelectionReady(workspace, event)).toBe(true);
    expect(staffSelectionReady(workspace, null)).toBe(false);
    expect(staffSelectionReady(null, event)).toBe(false);
  });

  it('chooses the persisted Event when still available', () => {
    const events = [{ id: 'event-1', title: 'One' }, { id: 'event-2', title: 'Two' }] as never[];
    expect(nextSelectedEvent('event-2', events)?.id).toBe('event-2');
    expect(nextSelectedEvent('missing', events)?.id).toBe('event-1');
    expect(nextSelectedEvent(null, [])).toBeNull();
  });

  it('keeps empty-state copy stable', () => {
    expect(staffSelectionEmptyCopy([], [])).toBe('No Workspaces available for this account.');
    expect(staffSelectionEmptyCopy([workspace], [])).toBe('No Events available in this Workspace yet.');
  });
});
```

- [ ] **Step 3: Refactor `mobile/app/staff.tsx` to use helpers**

Import the helpers:

```ts
import { nextSelectedEvent, selectedEventLabel, selectedWorkspaceLabel, staffSelectionEmptyCopy, staffSelectionReady } from '@/src/modules/staff/staffOperatorModel';
```

Use them only where current inline logic already performs equivalent labeling, empty-state, readiness, or next-selection behavior. Do not change visible copy unless replacing current copy with the exact strings in the helper.

- [ ] **Step 4: Validate**

Run:

```bash
pnpm --dir mobile run test -- src/modules/staff/staffOperatorModel.test.ts
pnpm --dir mobile run typecheck
make verify
```

Expected: PASS.

---

## Task 3: Harden mobile Door and Ticket wallet operator states

**Files:**

- Modify: `mobile/src/modules/tickets/ticketJourney.ts`
- Modify: `mobile/src/modules/tickets/ticketJourney.test.ts`
- Modify: `mobile/app/door.tsx`
- Modify: `mobile/app/tickets.tsx`
- Modify: `mobile/app/ticket.tsx`

- [ ] **Step 1: Add Ticket operator helpers**

In `mobile/src/modules/tickets/ticketJourney.ts`, add helpers if they do not already exist:

```ts
export function ticketLookupInput(value: string) {
  return value.trim();
}

export function doorCheckInButtonLabel(checkingIn: boolean, checkedIn: boolean) {
  if (checkingIn) return 'Checking in…';
  return checkedIn ? 'Checked in' : 'Check in';
}

export function ticketWalletEmptyCopy(hasLoaded: boolean) {
  return hasLoaded ? 'No Tickets saved to this device yet.' : 'Loading Tickets…';
}
```

Preserve raw Ticket code behavior. Do not strip spaces/hyphens unless backend lookup is changed and tested in the same task.

- [ ] **Step 2: Add tests for Ticket operator helpers**

In `mobile/src/modules/tickets/ticketJourney.test.ts`, add:

```ts
it('keeps lookup input raw except outer whitespace', () => {
  expect(ticketLookupInput(' ABC 123 ')).toBe('ABC 123');
});

it('labels Door check-in actions', () => {
  expect(doorCheckInButtonLabel(false, false)).toBe('Check in');
  expect(doorCheckInButtonLabel(true, false)).toBe('Checking in…');
  expect(doorCheckInButtonLabel(false, true)).toBe('Checked in');
});
```

- [ ] **Step 3: Refactor Door and wallet screens**

Use helper functions in:

- `mobile/app/door.tsx`
- `mobile/app/tickets.tsx`
- `mobile/app/ticket.tsx`

Keep visible copy stable unless the helper encodes the exact current copy.

- [ ] **Step 4: Validate**

Run:

```bash
pnpm --dir mobile run test -- src/modules/tickets/ticketJourney.test.ts
pnpm --dir mobile run typecheck
make verify
```

Expected: PASS.

---

## Task 4: Improve Run of Show operator reliability

**Files:**

- Modify: `mobile/src/modules/runOfShow/runOfShowModel.ts`
- Modify: `mobile/src/modules/runOfShow/runOfShowModel.test.ts`
- Modify: `mobile/app/run-of-show.tsx`

- [ ] **Step 1: Add status transition helper**

In `mobile/src/modules/runOfShow/runOfShowModel.ts`, add:

```ts
export function applyRunOfShowStatusUpdate<T extends { id: string; status: string }>(items: T[], itemId: string, status: T['status']) {
  return sortRunOfShowItems(items.map((item) => (item.id === itemId ? { ...item, status } : item)));
}
```

If the existing item type uses a narrower status union, use that union instead of `string`.

- [ ] **Step 2: Add transition tests**

In `mobile/src/modules/runOfShow/runOfShowModel.test.ts`, add:

```ts
it('re-sorts after a status update', () => {
  const items = [
    { id: 'a', status: 'completed', startsAt: '2026-06-19T20:00:00Z', createdAt: '2026-06-01T00:00:00Z' },
    { id: 'b', status: 'open', startsAt: '2026-06-19T21:00:00Z', createdAt: '2026-06-01T00:00:00Z' },
  ] as never[];

  expect(applyRunOfShowStatusUpdate(items, 'a', 'open' as never).map((item) => item.id)).toEqual(['a', 'b']);
});
```

- [ ] **Step 3: Use helper in screen update path**

In `mobile/app/run-of-show.tsx`, replace inline map/sort logic after status update with `applyRunOfShowStatusUpdate`.

- [ ] **Step 4: Validate**

Run:

```bash
pnpm --dir mobile run test -- src/modules/runOfShow/runOfShowModel.test.ts
pnpm --dir mobile run typecheck
make verify
```

Expected: PASS.

---

## Task 5: Deepen backend Settlement helpers

**Files:**

- Create: `backend/internal/app/settlement.go`
- Create: `backend/internal/app/settlement_test.go`
- Modify: `backend/internal/app/events.go`

- [ ] **Step 1: Extract pure Settlement summary helpers**

Create `backend/internal/app/settlement.go`:

```go
package app

type settlementTotals struct {
	TicketsReserved int
	TicketsCheckedIn int
	NoShows int
	GrossCents int
	FeeCents int
	NetCents int
}

func buildSettlementTotals(ticketsReserved int, ticketsCheckedIn int, grossCents int, feeCents int) settlementTotals {
	noShows := ticketsReserved - ticketsCheckedIn
	if noShows < 0 {
		noShows = 0
	}
	netCents := grossCents - feeCents
	return settlementTotals{
		TicketsReserved: ticketsReserved,
		TicketsCheckedIn: ticketsCheckedIn,
		NoShows: noShows,
		GrossCents: grossCents,
		FeeCents: feeCents,
		NetCents: netCents,
	}
}
```

If equivalent structs already exist in `events.go`, move only the pure arithmetic into this helper and keep DTO shapes unchanged.

- [ ] **Step 2: Add Settlement tests**

Create `backend/internal/app/settlement_test.go`:

```go
package app

import "testing"

func TestBuildSettlementTotals(t *testing.T) {
	totals := buildSettlementTotals(10, 7, 25000, 800)

	if totals.TicketsReserved != 10 || totals.TicketsCheckedIn != 7 || totals.NoShows != 3 {
		t.Fatalf("unexpected attendance totals: %#v", totals)
	}
	if totals.GrossCents != 25000 || totals.FeeCents != 800 || totals.NetCents != 24200 {
		t.Fatalf("unexpected money totals: %#v", totals)
	}
}

func TestBuildSettlementTotalsDoesNotReturnNegativeNoShows(t *testing.T) {
	totals := buildSettlementTotals(2, 3, 0, 0)

	if totals.NoShows != 0 {
		t.Fatalf("no-shows = %d, want 0", totals.NoShows)
	}
}
```

- [ ] **Step 3: Use helper in End of Night code**

In `backend/internal/app/events.go`, find the End of Night / report generation arithmetic and replace local no-show/net calculations with `buildSettlementTotals`. Do not change response JSON names or database writes.

- [ ] **Step 4: Validate**

Run:

```bash
cd backend && go test ./internal/app -run 'Settlement|EventLifecycle' -count=1 -v
make verify
```

Expected: PASS.

---

## Task 6: Improve web Settlement / End of Night view-model coverage

**Files:**

- Modify: `web/src/modules/eventEditor/eventEditorModel.ts`
- Modify: `web/src/modules/eventEditor/eventEditorModel.test.ts`
- Modify: `web/src/views/EventEditorView.tsx`

- [ ] **Step 1: Add Settlement display helpers**

In `web/src/modules/eventEditor/eventEditorModel.ts`, add helpers if not already present:

```ts
import type { EventReportDTO, EventSettlementDTO } from '../../domain';

export function settlementAttendanceSummary(report: EventReportDTO | null) {
  if (!report) return 'End the night to generate the Event Report.';
  return `${report.ticketsCheckedIn} checked in · ${report.noShows} no-shows`;
}

export function settlementNetSummary(settlement: EventSettlementDTO | null) {
  if (!settlement) return 'Settlement will appear after End of Night.';
  return formatSignedMoney(settlement.netProceedsCents, settlement.currency);
}
```

If `EventSettlementDTO` uses different property names, use the existing names from `web/src/domain.ts`.

- [ ] **Step 2: Add exact helper tests**

In `web/src/modules/eventEditor/eventEditorModel.test.ts`, add tests for:

```ts
expect(settlementAttendanceSummary(null)).toBe('End the night to generate the Event Report.');
expect(settlementAttendanceSummary({ ticketsCheckedIn: 7, noShows: 3 } as never)).toBe('7 checked in · 3 no-shows');
```

Add a net-summary assertion using the current `formatSignedMoney` output.

- [ ] **Step 3: Refactor the view carefully**

In `web/src/views/EventEditorView.tsx`, use the helpers only for existing Settlement / Report display text. Preserve routes, status labels, editor-specific lifecycle copy, and form behavior.

- [ ] **Step 4: Validate**

Run:

```bash
pnpm --dir web run test -- eventEditor
pnpm --dir web run format
make verify
```

Expected: PASS.

---

## Task 7: Add mobile closeout read-only helpers

**Files:**

- Create: `mobile/src/modules/settlement/settlementModel.ts`
- Create: `mobile/src/modules/settlement/settlementModel.test.ts`
- Modify: `mobile/app/event-dashboard.tsx` or `mobile/app/staff.tsx` only if a read-only closed Event cue already belongs there.

- [ ] **Step 1: Add mobile Settlement display helpers**

Create `mobile/src/modules/settlement/settlementModel.ts`:

```ts
export function mobileCloseoutStatusLabel(status: string) {
  return status === 'end_of_night' ? 'Closed out' : 'Open';
}

export function mobileCloseoutHandoffCopy(status: string) {
  return status === 'end_of_night'
    ? 'Review the full Event Report and Settlement on web.'
    : 'Close the Event from web when the room is done.';
}
```

- [ ] **Step 2: Add tests**

Create `mobile/src/modules/settlement/settlementModel.test.ts`:

```ts
import { describe, expect, it } from 'vitest';

import { mobileCloseoutHandoffCopy, mobileCloseoutStatusLabel } from './settlementModel';

describe('settlementModel', () => {
  it('labels closed Events for operator handoff', () => {
    expect(mobileCloseoutStatusLabel('end_of_night')).toBe('Closed out');
    expect(mobileCloseoutStatusLabel('published')).toBe('Open');
    expect(mobileCloseoutHandoffCopy('end_of_night')).toBe('Review the full Event Report and Settlement on web.');
  });
});
```

- [ ] **Step 3: Add the cue only if it is low-risk**

If `mobile/app/event-dashboard.tsx` already shows Event status, use these helpers there for a small read-only cue. If adding the cue would require layout work, skip view changes and keep this as tested prep for the next UI pass.

- [ ] **Step 4: Validate**

Run:

```bash
pnpm --dir mobile run test -- src/modules/settlement/settlementModel.test.ts
pnpm --dir mobile run typecheck
make verify
```

Expected: PASS.

---

## Task 8: Final QA pass and evidence update

**Files:**

- Modify: `docs/qa/mobile-operator-settlement-2026-06.md`

- [ ] **Step 1: Run final verification**

Run:

```bash
make verify
```

Expected: PASS.

If a disposable DB is available:

```bash
make test-db
```

Expected: PASS.

If unavailable, record `Not run — TEST_DATABASE_URL unavailable`.

- [ ] **Step 2: Run stack smoke**

Run:

```bash
make up-build
make smoke
make down
```

Expected: PASS.

- [ ] **Step 3: Update QA doc**

Fill the `Final` column for every row that was actually tested. If browser/device are unavailable, leave `Final` as `Not run` and add the unavailable reason.

Add automated test rows for every new test file:

```markdown
| `mobile/src/modules/staff/staffOperatorModel.test.ts` | Staff Workspace/Event selection helpers | `pnpm --dir mobile run test` |
| `backend/internal/app/settlement_test.go` | Settlement totals and no-show rules | `cd backend && go test ./internal/app -run Settlement -count=1 -v` |
```

- [ ] **Step 4: Run final review**

Ask review:

```text
Review the mobile operator and Settlement sprint. Verify mobile changes improve operator reliability without route/API/UX drift, Settlement rules remain server-owned, tests cover Module Interfaces, QA evidence is honest, and no broad Discovery/payment-provider work slipped in.
```

- [ ] **Step 5: Fix review findings**

Fix only concrete review findings. Product questions go under `## Follow-up candidates` in the QA doc.

---

## Acceptance criteria

- `make verify` passes.
- Mobile pure-module tests cover staff selection, Door/Ticket helper behavior, Run of Show status updates, and any mobile closeout helper introduced.
- Backend always-on tests cover Settlement totals and no-show rules.
- End of Night / Settlement JSON and routes remain stable.
- Door lookup keeps raw Ticket code semantics unless backend normalization is explicitly added and tested.
- QA doc records what was actually run, with browser/Expo unavailable if not tested.
- No Discovery scope expansion and no payment-provider-specific language added to Settlement core rules.

## Risks and guardrails

- **Risk:** Mobile polish becomes a redesign. **Guardrail:** only improve operator reliability and evidence-backed friction.
- **Risk:** Settlement logic drifts into web/mobile. **Guardrail:** server owns arithmetic; clients display summaries.
- **Risk:** Ticket code formatting breaks manual Door lookup. **Guardrail:** preserve raw code lookup and test it.
- **Risk:** Manual QA gets overstated. **Guardrail:** use `Not run` for unavailable browser/device paths.
