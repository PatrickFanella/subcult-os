# Staffing Commitments Implementation Plan

> **For agentic workers:** Execute this plan task-by-task. Recommended path:
> dispatch a fresh subagent per task, review each result with `review-quality`,
> then continue. For complex multi-agent splits, use
> `parallel-feature-development`, `team-composition-patterns`, and
> `team-communication-protocols`. Steps use checkbox (`- [ ]`) syntax for
> tracking.

**Goal:** Turn accepted participants and workspace members into an operational staffing board with shifts, tasks, assignment status, closeout visibility, and archive memory.

**Architecture:** Add event-scoped staffing records without changing ticketing, applications, or workspace membership. Store staffing items in `event_staffing_items`, optionally assigned to a workspace person or accepted role application; owner/member can read, owners can create/update/assign/complete, staffing mutations serialize on the event row with end-of-night, and end-of-night snapshots open staffing state into private archive memory.

**Tech Stack:** Go `net/http`, pgx/Postgres embedded `schema.sql`, React TypeScript SPA, Vitest, `make verify`.

---

## Current State

- Events support roles, public participation applications, private review, accepted participant roster, and archive participant memory.
- Event editor already loads roles, applications, participants, settlements, and archive panels.
- Workspace archive/search and seeded-draft loops exist.
- There is no operational staffing/task/shift object yet.

## File Structure

- `backend/internal/app/schema.sql`: add staffing and archive staffing tables plus additive migrations.
- `backend/internal/app/app.go`: register staffing routes.
- `backend/internal/app/staffing.go`: new handlers/DTOs for list, create, update/assign/complete, and summary helpers.
- `backend/internal/app/events.go`: include archive staffing snapshot at archive creation.
- `backend/internal/app/workspaces.go`: optionally expose staffing summary in workspace event/archive views.
- `backend/internal/app/lifecycle_test.go`: DB-backed staffing workflow tests.
- `web/src/domain.ts`: add staffing DTOs.
- `web/src/views/EventEditorView.tsx`: staffing board UI.
- `web/src/views/WorkspaceView.tsx`: staffing status summary and closeout/archive nudges.
- `web/src/App.test.tsx`: frontend smoke/regression tests.

---

### Task 1: Staffing Schema and Read API

**Files:**
- Modify: `backend/internal/app/schema.sql`
- Modify: `backend/internal/app/app.go`
- Create: `backend/internal/app/staffing.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`

- [ ] **Step 1: Add tables**

Add `event_staffing_items`:

```sql
create table if not exists event_staffing_items (
  id uuid primary key default gen_random_uuid(),
  event_id uuid not null references events(id) on delete cascade,
  title text not null,
  kind text not null check (kind in ('task', 'shift')),
  notes text not null default '',
  starts_at timestamptz,
  ends_at timestamptz,
  assigned_person_id uuid references people(id),
  assigned_application_id uuid references event_role_applications(id),
  status text not null default 'open' check (status in ('open', 'assigned', 'completed', 'cancelled')),
  created_by_person_id uuid not null references people(id),
  completed_at timestamptz,
  completed_by_person_id uuid references people(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  check (ends_at is null or starts_at is null or ends_at >= starts_at)
);
```

Add additive migrations for each column after the create block if later edits add fields.

- [ ] **Step 2: Add DTOs**

Go DTO:

```go
type eventStaffingItemDTO struct {
    ID                    string  `json:"id"`
    EventID               string  `json:"eventId"`
    Title                 string  `json:"title"`
    Kind                  string  `json:"kind"`
    Notes                 string  `json:"notes"`
    StartsAt              *string `json:"startsAt,omitempty"`
    EndsAt                *string `json:"endsAt,omitempty"`
    AssignedPersonID      *string `json:"assignedPersonId,omitempty"`
    AssignedApplicationID *string `json:"assignedApplicationId,omitempty"`
    AssigneeName          *string `json:"assigneeName,omitempty"`
    Status                string  `json:"status"`
    CreatedAt             string  `json:"createdAt"`
    UpdatedAt             string  `json:"updatedAt"`
    CompletedAt           *string `json:"completedAt,omitempty"`
    CompletedByPersonID   *string `json:"completedByPersonId,omitempty"`
}
```

TypeScript DTO mirrors this shape with `kind: 'task' | 'shift'` and `status: 'open' | 'assigned' | 'completed' | 'cancelled'`.

- [ ] **Step 3: Register and implement list route**

Register:

```go
a.mux.HandleFunc("GET /api/events/{eventID}/staffing", a.handleListEventStaffing)
```

Implement `handleListEventStaffing` in `staffing.go`: load event, require workspace owner/member, return rows ordered by `status`, `starts_at nulls last`, `created_at`, `id`. Join to `people` and `event_role_applications` for `assigneeName` using person display/email or applicant name. Missing event returns 404; non-member/unauth uses current private route forbidden behavior.

- [ ] **Step 4: Add tests**

Add `TestEventStaffingListAPI`: empty list returns `[]`; owner/member can read; non-member/unauth forbidden; seeded direct rows serialize nullable fields correctly.

- [ ] **Step 5: Verify and commit**

Run: `gofmt -w backend/internal/app/app.go backend/internal/app/staffing.go backend/internal/app/lifecycle_test.go && make verify`

Commit:

```bash
git add backend/internal/app/schema.sql backend/internal/app/app.go backend/internal/app/staffing.go backend/internal/app/lifecycle_test.go web/src/domain.ts docs/superpowers/plans/2026-06-14-staffing-commitments.md
git commit -m "Add event staffing read model"
```

---

### Task 2: Create Staffing Items

**Files:**
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/staffing.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`

- [ ] **Step 1: Register create route**

```go
a.mux.HandleFunc("POST /api/events/{eventID}/staffing", a.handleCreateEventStaffing)
```

- [ ] **Step 2: Implement request validation**

Request:

```go
type createEventStaffingRequest struct {
    Title    string  `json:"title"`
    Kind     string  `json:"kind"`
    Notes    string  `json:"notes"`
    StartsAt *string `json:"startsAt"`
    EndsAt   *string `json:"endsAt"`
}
```

Rules: owner-only; event must be `draft` or `published` (closed events reject with 409); title trim non-empty; kind only `task`/`shift`; notes trim and max 2000 runes; optional starts/ends parse RFC3339 and `endsAt >= startsAt` when both set.

- [ ] **Step 3: Insert and audit**

Insert with `status='open'`, `created_by_person_id=actorID`, timestamps default. Audit `staffing.created` with metadata only `eventId`, `staffingItemId`, `kind`, `status`; do not include notes.

- [ ] **Step 4: Tests**

Add `TestEventStaffingCreateAPI`: owner creates task and shift; member forbidden create but can list; empty title, invalid kind, inverted dates, too-long notes bad request; closed event create conflict; audit excludes notes.

- [ ] **Step 5: Verify and commit**

Run: `gofmt -w backend/internal/app/app.go backend/internal/app/staffing.go backend/internal/app/lifecycle_test.go && make verify`

Commit:

```bash
git add backend/internal/app/app.go backend/internal/app/staffing.go backend/internal/app/lifecycle_test.go web/src/domain.ts
git commit -m "Create event staffing items"
```

---

### Task 3: Assign and Complete Staffing Items

**Files:**
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/staffing.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`

- [ ] **Step 1: Register patch route**

```go
a.mux.HandleFunc("PATCH /api/events/{eventID}/staffing/{staffingID}", a.handleUpdateEventStaffing)
```

- [ ] **Step 2: Implement update semantics**

Request:

```go
type updateEventStaffingRequest struct {
    Title                 *string `json:"title"`
    Notes                 *string `json:"notes"`
    StartsAt              *string `json:"startsAt"`
    ClearStartsAt         bool    `json:"clearStartsAt"`
    EndsAt                *string `json:"endsAt"`
    ClearEndsAt           bool    `json:"clearEndsAt"`
    AssignedPersonID      *string `json:"assignedPersonId"`
    AssignedApplicationID *string `json:"assignedApplicationId"`
    ClearAssignee         bool    `json:"clearAssignee"`
    Status                *string `json:"status"`
}
```

Rules: owner-only; begin a transaction; lock the event row `FOR UPDATE` first, then lock item row `FOR UPDATE` so staffing mutations serialize with end-of-night archive snapshots; reject closed events for any actual mutation; exactly one assignee source may be set at a time; `clearAssignee` clears both assignment fields and moves `assigned -> open` unless a status is explicitly provided; `assignedPersonId` must be active workspace member; `assignedApplicationId` must belong to same event and have status `accepted` or `confirmed`; setting an assignee moves `open -> assigned` unless explicit status is `completed` or `cancelled`; `clearStartsAt`/`clearEndsAt` explicitly clear nullable times; invalid status returns 400.

Completion idempotency: first transition into `completed` sets `completed_at` and `completed_by_person_id`; repeated `completed -> completed` preserves original completion fields and does not write a duplicate audit entry. `cancelled` clears completion fields only when transitioning away from completed.

- [ ] **Step 3: Audit**

Audit `staffing.updated` with metadata `eventId`, `staffingItemId`, `previousStatus`, `status`, `assignedPersonId`, `assignedApplicationId`. Do not include notes.

- [ ] **Step 4: Tests**

Add `TestEventStaffingUpdateAPI`: assign workspace member, clear assignee, assign accepted participant application, reject submitted/rejected application, reject cross-event application, complete item, repeated complete preserves `completedAt`, cancel item, clear shift times, member forbidden, closed event update conflict, audit privacy.

- [ ] **Step 5: Verify and commit**

Run: `gofmt -w backend/internal/app/app.go backend/internal/app/staffing.go backend/internal/app/lifecycle_test.go && make verify`

Commit:

```bash
git add backend/internal/app/app.go backend/internal/app/staffing.go backend/internal/app/lifecycle_test.go web/src/domain.ts
git commit -m "Assign and complete staffing items"
```

---

### Task 4: Event Editor Staffing Board

**Files:**
- Modify: `web/src/views/EventEditorView.tsx`
- Modify: `web/src/App.test.tsx`

- [ ] **Step 1: Load staffing**

When an event loads, fetch `/api/events/${event.id}/staffing`. Keep 404/error behavior consistent with roles/applications. Store `staffingItems` state.

- [ ] **Step 2: Render board**

Add a `Staffing board` panel showing counts by `open`, `assigned`, `completed`, `cancelled`, then rows grouped by task/shift. Each row shows title, notes, time window, status, and assignee name or `Unassigned`.

- [ ] **Step 3: Owner create form**

Add a compact owner-facing form: title, kind, notes, startsAt, endsAt. POST to create endpoint and append returned item. Hide mutation controls for non-owners using the same event workspace role state used by archive actions.

- [ ] **Step 4: Owner assignment and status controls**

For each item, owner can assign to either an active workspace member or an accepted/confirmed participant application, clear the assignee, mark `completed`, or mark `cancelled`; PATCH item and replace it in state. Use labels without participant emails in the UI. Load accepted participant roster from the existing participants endpoint and workspace members from the already-loaded event workspace role response.

- [ ] **Step 5: Tests and commit**

Extend `web/src/App.test.tsx`: staffing panel visible; counts render; owner sees create/assign/complete controls; member does not see mutation controls.

Run: `make verify`

Commit:

```bash
git add web/src/views/EventEditorView.tsx web/src/App.test.tsx
git commit -m "Add event staffing board UI"
```

---

### Task 5: Workspace and Archive Staffing Memory

**Files:**
- Modify: `backend/internal/app/schema.sql`
- Modify: `backend/internal/app/events.go`
- Modify: `backend/internal/app/staffing.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`
- Modify: `web/src/views/WorkspaceView.tsx`
- Modify: `web/src/views/EventEditorView.tsx`
- Modify: `web/src/App.test.tsx`

- [ ] **Step 1: Add archive staffing table**

```sql
create table if not exists event_archive_staffing_items (
  id uuid primary key default gen_random_uuid(),
  archive_id uuid not null references event_archives(id) on delete cascade,
  source_staffing_item_id uuid not null,
  title text not null,
  kind text not null check (kind in ('task', 'shift')),
  status text not null check (status in ('open', 'assigned', 'completed', 'cancelled')),
  assignee_name text,
  created_at timestamptz not null default now(),
  unique (archive_id, source_staffing_item_id)
);
```

- [ ] **Step 2: Snapshot only on archive creation**

In `events.go`, when `ensureEventArchive` returns `created == true`, call `snapshotArchiveStaffingItems`. End-of-night already locks the event row; Task 3 requires staffing mutations to lock that same row, so closeout and staffing updates serialize. Snapshot all non-cancelled staffing items with title/kind/status/assignee name. Do not snapshot notes. Do not mutate archive staffing on end-of-night retries once archive exists.

- [ ] **Step 3: Expose archive staffing memory**

Extend `EventArchiveDTO` with `staffingItems: EventArchiveStaffingItemDTO[]`; load rows ordered by status/title. TypeScript mirrors the DTO.

- [ ] **Step 4: Workspace closeout signal data source**

Extend the workspace event list response with summary counts to avoid frontend N+1 requests. Add optional fields to `EventDTO`/Go `eventDTO`: `staffingOpenCount`, `staffingAssignedCount`, `staffingCompletedCount`, `staffingCancelledCount`. In `handleListEvents`, left join an aggregate subquery grouped by event_id. For single-event detail, either populate the same counts from a helper query or omit only if TypeScript marks them optional; prefer helper so editor/workspace agree.

- [ ] **Step 5: Workspace closeout signal UI**

In `WorkspaceView.tsx`, show staffing status copy on event cards when closed or published: unresolved staffing remains before closeout, all staffing complete, or no staffing items.

- [ ] **Step 6: UI archive memory**

In `EventEditorView.tsx` archive panel, render `Staffing memory` with non-note fields only. Include nudge: unresolved items should inform next draft planning.

- [ ] **Step 7: Tests, review, commit**

Add backend tests: archive snapshot includes staffing items once, excludes notes, retries do not append/mutate after staffing changes; duplicate titles preserved via source ID; workspace event list exposes staffing counts without N+1 behavior assumptions. Add frontend test for archive staffing memory and workspace staffing status copy.

Run @oracle review for snapshot idempotency/privacy, then `make verify`.

Commit:

```bash
git add backend/internal/app/schema.sql backend/internal/app/events.go backend/internal/app/staffing.go backend/internal/app/lifecycle_test.go web/src/domain.ts web/src/views/WorkspaceView.tsx web/src/views/EventEditorView.tsx web/src/App.test.tsx
git commit -m "Add archive staffing memory"
```

---

## Final Verification

- [ ] Run `make verify`.
- [ ] Run `git status --short --branch` and confirm clean tree.
- [ ] Run `git log --oneline -10` and confirm five staffing commits.

## Out of Scope

- Public volunteer self-scheduling.
- Notifications/reminders.
- Recurring staffing templates.
- Drag-and-drop scheduling.
- Payroll or payout integration.
- Copying private staffing notes into archive memory or public pages.

## Self-Review

- Spec coverage: plan covers schema/read, create, assign/complete, editor UI, workspace/archive memory.
- Placeholder scan: no TODO/TBD placeholders remain; each slice has concrete routes, rules, tests, and commands.
- Type consistency: `EventStaffingItemDTO` and archive staffing DTO names align across Go/TS/routes.
