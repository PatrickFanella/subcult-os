# Reminder Automation Implementation Plan

> **For agentic workers:** Execute this plan task-by-task. Recommended path:
> dispatch a fresh subagent per task, review each result with `review-quality`,
> then continue. For complex multi-agent splits, use
> `parallel-feature-development`, `team-composition-patterns`, and
> `team-communication-protocols`. Steps use checkbox (`- [ ]`) syntax for
> tracking.

**Goal:** Add private operational reminders for commitments and staffing so workspace operators can trigger due follow-up notifications safely and idempotently.

**Architecture:** Add a `reminder_events` ledger that records reminder decisions and links to existing `notification_events`/`email_outbox`. A workspace-owner sweep endpoint processes due reminders for one workspace at a time; each reminder uses a source-specific idempotency key so retries do not duplicate emails. Operators can read reminder activity privately; public routes never expose reminder data.

**Tech Stack:** Go `net/http`, pgx/Postgres embedded `schema.sql`, existing `enqueueNotification`, React TypeScript SPA, Vitest, `make verify`.

---

## Scope Boundaries

- This wave adds the idempotent reminder engine and owner-triggered sweep endpoint.
- It does **not** add a background cron daemon, external mail provider, notification preferences, or recurring schedules.
- Reminder text must not include commitment descriptions, staffing notes, application messages, contact notes, archive notes, settlement details, or template private notes.

## File Structure

- `backend/internal/app/schema.sql`: add `reminder_events` table and indexes.
- `backend/internal/app/app.go`: register reminder list/sweep routes.
- `backend/internal/app/reminders.go`: new reminder DTOs, list handler, sweep handler, commitment/staffing reminder collectors.
- `backend/internal/app/notifications.go`: optionally add helper behavior only if needed; prefer existing `enqueueNotification` unchanged.
- `backend/internal/app/lifecycle_test.go`: reminder API, idempotency, privacy, and permission tests.
- `web/src/domain.ts`: reminder DTO type.
- `web/src/views/WorkspaceView.tsx`: workspace reminder panel and owner-only sweep button.
- `web/src/views/EventEditorView.tsx`: event reminder activity panel.
- `web/src/App.test.tsx`: frontend smoke/privacy tests.
- `README.md`, `docs/runbooks/deployment-checklist.md`: reminder privacy/operator-triggered boundary.

---

### Task 1: Reminder Ledger and Read API

**Files:**
- Modify: `backend/internal/app/schema.sql`
- Modify: `backend/internal/app/app.go`
- Create: `backend/internal/app/reminders.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`
- Modify: `docs/superpowers/plans/2026-06-14-reminder-automation.md`

- [x] **Step 1: Add failing ledger coverage**

Add `TestReminderLedgerAPI` to `backend/internal/app/lifecycle_test.go`:

```go
func TestReminderLedgerAPI(t *testing.T) {
    fx := newLifecycleFixture(t)
    event := createEvent(t, fx, "Benefit Show", 20)
    eventID := mustString(t, event, "id")
    commitment := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/commitments", map[string]any{"title": "Seed reminder source"}, http.StatusOK)
    commitmentID := mustString(t, commitment.JSON, "id")

    var reminderID string
    if err := fx.app.db.QueryRow(t.Context(), `
        insert into reminder_events (
            workspace_id, event_id, source_type, source_id, reminder_type,
            recipient_email, due_at, idempotency_key, status, subject, preview, created_by_person_id
        ) values ($1, $2, 'commitment', $3, 'commitment.due', 'owner@example.test', now(), 'test-reminder-key', 'queued', 'Reminder subject', 'Reminder preview', $4)
        returning id
    `, fx.workspaceID, eventID, commitmentID, ownerPersonID(t, fx)).Scan(&reminderID); err != nil {
        t.Fatal(err)
    }

    workspaceResp := getJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/reminders", http.StatusOK)
    reminders := workspaceResp.JSON.([]any)
    if len(reminders) != 1 {
        t.Fatalf("expected one reminder, got %#v", reminders)
    }
    reminder := mustObject(t, reminders[0])
    if reminder["id"] != reminderID || reminder["sourceType"] != "commitment" || reminder["reminderType"] != "commitment.due" || reminder["subject"] != "Reminder subject" {
        t.Fatalf("unexpected reminder: %#v", reminder)
    }

    eventResp := getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/reminders", http.StatusOK)
    if got := eventResp.JSON.([]any); len(got) != 1 {
        t.Fatalf("expected one event reminder, got %#v", got)
    }

    otherFx := newLifecycleFixture(t)
    getJSON(t, fx.app, nil, "/api/workspaces/"+fx.workspaceID+"/reminders", http.StatusForbidden)
    getJSON(t, fx.app, otherFx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/reminders", http.StatusForbidden)
}
```

- [x] **Step 2: Add schema**

Append after `notification_events`:

```sql
create table if not exists reminder_events (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references workspaces(id) on delete cascade,
  event_id uuid references events(id) on delete cascade,
  source_type text not null check (source_type in ('commitment', 'staffing')),
  source_id uuid not null,
  reminder_type text not null check (reminder_type in ('commitment.due', 'staffing.upcoming', 'staffing.unassigned')),
  recipient_email text not null,
  due_at timestamptz not null,
  idempotency_key text not null unique,
  notification_event_id uuid references notification_events(id) on delete set null,
  status text not null default 'queued' check (status in ('queued')),
  subject text not null,
  preview text not null,
  created_by_person_id uuid references people(id),
  created_at timestamptz not null default now()
);

create index if not exists reminder_events_workspace_created_idx on reminder_events (workspace_id, created_at desc);
create index if not exists reminder_events_event_created_idx on reminder_events (event_id, created_at desc);
create index if not exists reminder_events_source_idx on reminder_events (source_type, source_id);
```

- [x] **Step 3: Register read routes and DTO**

In `app.go`:

```go
a.mux.HandleFunc("GET /api/workspaces/{workspaceID}/reminders", a.handleListWorkspaceReminders)
a.mux.HandleFunc("GET /api/events/{eventID}/reminders", a.handleListEventReminders)
```

In `reminders.go`, add `reminderEventDTO` with fields: `id`, `workspaceId`, optional `eventId`, `sourceType`, `sourceId`, `reminderType`, `recipientEmail`, `dueAt`, optional `notificationEventId`, `status`, `subject`, `preview`, `createdAt`.

Handlers: workspace owner/member can list workspace reminders; event owner/member can list event reminders. Limit 100, order `created_at desc`.

- [x] **Step 4: Add TS DTO**

In `web/src/domain.ts`, add `ReminderEventDTO` matching JSON.

- [x] **Step 5: Verify and commit**

Run: `gofmt -w backend/internal/app/app.go backend/internal/app/reminders.go backend/internal/app/lifecycle_test.go && make verify`

Commit:

```bash
git add backend/internal/app/schema.sql backend/internal/app/app.go backend/internal/app/reminders.go backend/internal/app/lifecycle_test.go web/src/domain.ts docs/superpowers/plans/2026-06-14-reminder-automation.md
git commit -m "Add reminder event ledger"
```

---

### Task 2: Reminder Sweep Core

**Files:**
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/reminders.go`
- Modify: `backend/internal/app/lifecycle_test.go`

- [ ] **Step 1: Add failing sweep coverage**

Add `TestReminderSweepAPIEmptyAndPermissions`:

```go
func TestReminderSweepAPIEmptyAndPermissions(t *testing.T) {
    fx := newLifecycleFixture(t)
    resp := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/reminders/sweep", map[string]any{"now": time.Now().UTC().Format(time.RFC3339)}, http.StatusOK)
    result := mustObject(t, resp.JSON)
    if int(result["createdCount"].(float64)) != 0 || int(result["skippedCount"].(float64)) != 0 {
        t.Fatalf("unexpected empty sweep result: %#v", result)
    }
    postJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/reminders/sweep", map[string]any{}, http.StatusForbidden)
    otherFx := newLifecycleFixture(t)
    postJSON(t, fx.app, otherFx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/reminders/sweep", map[string]any{}, http.StatusForbidden)
}
```

- [ ] **Step 2: Register sweep route**

```go
a.mux.HandleFunc("POST /api/workspaces/{workspaceID}/reminders/sweep", a.handleSweepWorkspaceReminders)
```

- [ ] **Step 3: Add sweep request/result types**

Request: optional `{ now: string }` for tests; if omitted use `time.Now().UTC()`. Result: `{ createdCount, skippedCount }`. Reject invalid `now` with `400`.

- [ ] **Step 4: Implement atomic helper**

Implement `createReminderAndNotification(ctx, tx, params)` that:

1. Normalizes recipient email; returns skipped if empty.
2. Inserts `reminder_events` first using `on conflict (idempotency_key) do nothing returning id`.
3. If conflict, returns skipped without sending email.
4. Calls `a.enqueueNotification(ctx, tx, ...)` with the **same** idempotency key used for `reminder_events.idempotency_key`.
5. Updates `reminder_events.notification_event_id` by selecting `notification_events.id` with that same idempotency key.
6. Keeps reminder, notification, and outbox writes inside the same transaction.

Do not call `enqueueNotification` before claiming the reminder idempotency key.

`skippedCount` is an in-memory sweep result only; skipped decisions are not inserted into `reminder_events`. Count idempotency conflicts as skipped. Count empty/invalid recipients as skipped. Do not count “no matching source rows” as skipped. `createdCount` counts newly inserted `reminder_events` rows that successfully create/link a notification.

- [ ] **Step 5: Verify and commit**

Run `make verify` and commit:

```bash
git add backend/internal/app/app.go backend/internal/app/reminders.go backend/internal/app/lifecycle_test.go
git commit -m "Add reminder sweep core"
```

---

### Task 3: Commitment Reminders

**Files:**
- Modify: `backend/internal/app/reminders.go`
- Modify: `backend/internal/app/lifecycle_test.go`

- [ ] **Step 1: Add failing commitment reminder coverage**

Add `TestReminderSweepCommitments`:

```go
func TestReminderSweepCommitments(t *testing.T) {
    fx := newLifecycleFixture(t)
    dueAt := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
    created := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/commitments", map[string]any{
        "title": "Confirm projector",
        "description": "Private projector vendor note",
        "dueAt": dueAt,
    }, http.StatusOK)
    commitmentID := mustString(t, created.JSON, "id")

    first := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/reminders/sweep", map[string]any{"now": time.Now().UTC().Format(time.RFC3339)}, http.StatusOK)
    if int(mustObject(t, first.JSON)["createdCount"].(float64)) != 1 || int(mustObject(t, first.JSON)["skippedCount"].(float64)) != 0 {
        t.Fatalf("expected first sweep to create one reminder: %#v", first.JSON)
    }
    repeated := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/reminders/sweep", map[string]any{"now": time.Now().UTC().Format(time.RFC3339)}, http.StatusOK)
    if int(mustObject(t, repeated.JSON)["createdCount"].(float64)) != 0 || int(mustObject(t, repeated.JSON)["skippedCount"].(float64)) != 1 {
        t.Fatalf("expected repeated sweep to skip duplicate reminder: %#v", repeated.JSON)
    }

    reminders := getJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/reminders", http.StatusOK).JSON.([]any)
    if len(reminders) != 1 {
        t.Fatalf("expected one idempotent commitment reminder, got %#v", reminders)
    }
    reminder := mustObject(t, reminders[0])
    if reminder["sourceType"] != "commitment" || reminder["sourceId"] != commitmentID || reminder["reminderType"] != "commitment.due" {
        t.Fatalf("unexpected reminder: %#v", reminder)
    }
    rawReminder, _ := json.Marshal(reminder)
    if strings.Contains(string(rawReminder), "Private projector vendor note") {
        t.Fatalf("reminder leaked commitment description: %s", rawReminder)
    }

    // Dev outbox is a development inspection route, not a workspace-private boundary.
    // This assertion only verifies reminder email copy excludes commitment free text.
    outbox := getJSON(t, fx.app, fx.ownerCookie, "/api/dev/email-outbox", http.StatusOK)
    rawOutbox, _ := json.Marshal(outbox.JSON)
    if strings.Contains(string(rawOutbox), "Private projector vendor note") {
        t.Fatalf("outbox leaked commitment description: %s", rawOutbox)
    }
}
```

- [ ] **Step 2: Implement commitment collector**

In `runWorkspaceReminderSweep`, select commitments where `workspace_id=$1`, `status='open'`, `due_at is not null`, `due_at <= now`. Recipient rule for MVP: send to active workspace owners. If `owner_person_id` exists and is an active workspace member with email, send to that person only; otherwise send to all active owners. Idempotency key: `commitment_due:{commitmentID}:{recipientEmail}`.

Subject/preview/body may include commitment title and due date, but never description/contact notes/template notes.

- [ ] **Step 3: Done/cancelled exclusion test**

Extend tests to mark commitment `done` and `cancelled`; sweep again and assert no new reminders.

- [ ] **Step 4: Verify and commit**

Run `make verify` and commit:

```bash
git add backend/internal/app/reminders.go backend/internal/app/lifecycle_test.go
git commit -m "Add commitment due reminders"
```

---

### Task 4: Staffing Reminders

**Files:**
- Modify: `backend/internal/app/reminders.go`
- Modify: `backend/internal/app/lifecycle_test.go`

- [ ] **Step 1: Add failing staffing reminder coverage**

Add `TestReminderSweepStaffing`:

```go
func TestReminderSweepStaffing(t *testing.T) {
    fx := newLifecycleFixture(t)
    event := createEvent(t, fx, "Benefit Show", 20)
    eventID := mustString(t, event, "id")
    publishEvent(t, fx, eventID)
    startsAt := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)
    staffing := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/staffing", map[string]any{
        "title": "Door lead",
        "kind": "shift",
        "notes": "Private staffing note",
        "startsAt": startsAt,
    }, http.StatusOK)
    staffingID := mustString(t, staffing.JSON, "id")

    first := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/reminders/sweep", map[string]any{"now": time.Now().UTC().Format(time.RFC3339)}, http.StatusOK)
    if int(mustObject(t, first.JSON)["createdCount"].(float64)) != 1 || int(mustObject(t, first.JSON)["skippedCount"].(float64)) != 0 {
        t.Fatalf("expected first staffing sweep to create one reminder: %#v", first.JSON)
    }
    repeated := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/reminders/sweep", map[string]any{"now": time.Now().UTC().Format(time.RFC3339)}, http.StatusOK)
    if int(mustObject(t, repeated.JSON)["createdCount"].(float64)) != 0 || int(mustObject(t, repeated.JSON)["skippedCount"].(float64)) != 1 {
        t.Fatalf("expected repeated staffing sweep to skip duplicate reminder: %#v", repeated.JSON)
    }

    reminders := getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/reminders", http.StatusOK).JSON.([]any)
    if len(reminders) != 1 {
        t.Fatalf("expected one idempotent staffing reminder, got %#v", reminders)
    }
    reminder := mustObject(t, reminders[0])
    if reminder["sourceType"] != "staffing" || reminder["sourceId"] != staffingID || reminder["reminderType"] != "staffing.unassigned" {
        t.Fatalf("unexpected staffing reminder: %#v", reminder)
    }
    rawReminder, _ := json.Marshal(reminder)
    if strings.Contains(string(rawReminder), "Private staffing note") {
        t.Fatalf("reminder leaked staffing notes: %s", rawReminder)
    }
}
```

- [ ] **Step 2: Implement staffing collector**

Select staffing items in workspace events with `events.status = 'published'`, item `status in ('open','assigned')`, `starts_at is not null`, and `starts_at <= now + interval '24 hours'`. Do not create reminders for draft or `end_of_night` events. For unassigned open items, send `staffing.unassigned` reminders to active workspace owners. For assigned items, send `staffing.upcoming` to assigned person email or accepted/confirmed application applicant email. Idempotency keys:

- `staffing_unassigned:{staffingID}:{recipientEmail}`
- `staffing_upcoming:{staffingID}:{recipientEmail}`

Subject/preview/body may include event title, staffing title, and start time. Never include staffing notes or application message.

- [ ] **Step 3: Exclusion tests**

Assert draft and `end_of_night` events do not create staffing reminders. Assert completed/cancelled staffing items do not create reminders. Assert repeated sweep does not duplicate outbox/notification/reminder rows and returns `createdCount: 0`, `skippedCount` equal to duplicate candidates.

- [ ] **Step 4: Verify and commit**

Run `make verify` and commit:

```bash
git add backend/internal/app/reminders.go backend/internal/app/lifecycle_test.go
git commit -m "Add staffing reminders"
```

---

### Task 5: Reminder UI, Docs, and Privacy

**Files:**
- Modify: `web/src/domain.ts`
- Modify: `web/src/views/WorkspaceView.tsx`
- Modify: `web/src/views/EventEditorView.tsx`
- Modify: `web/src/App.test.tsx`
- Modify: `README.md`
- Modify: `docs/runbooks/deployment-checklist.md`
- Modify: `backend/internal/app/lifecycle_test.go`

- [ ] **Step 1: Frontend DTO and panels**

Add `ReminderEventDTO` if not already added. Workspace view loads `/api/workspaces/{id}/reminders`, shows `Reminder activity`, and owner-only `Run reminder sweep` button. Event editor loads `/api/events/{eventID}/reminders` and shows event-specific reminder activity.

- [ ] **Step 2: Frontend tests**

Assert reminder panels render for private workspace/editor views, owner sees sweep button, member does not. Assert public discovery/public event/ticket views do not show reminder data.

- [ ] **Step 3: Backend privacy tests**

Extend private-memory/privacy tests so public routes, reports, settlements, archives, contacts, commitments, templates, and notification list do not expose reminder-only private sentinel text except reminder routes.

- [ ] **Step 4: Docs**

README shipped-slice copy: `Owner-triggered reminder sweeps create private notification activity for due commitments and upcoming staffing without exposing free-text notes publicly.` Deployment checklist: `Reminder sweeps use email outbox rows and may contain recipient emails; review reminder copy, idempotency keys, and future scheduler credentials before production.`

- [ ] **Step 5: Verify and commit**

Run `make verify` and commit:

```bash
git add web/src/domain.ts web/src/views/WorkspaceView.tsx web/src/views/EventEditorView.tsx web/src/App.test.tsx README.md docs/runbooks/deployment-checklist.md backend/internal/app/lifecycle_test.go
git commit -m "Add reminder activity UI and docs"
```

---

## Out of Scope

- Background cron/scheduler daemon.
- User notification preferences or unsubscribe flows.
- SMS/push delivery.
- Recurring commitments.
- Public reminder visibility.

## Final Verification

- [ ] Run `make verify`.
- [ ] Run `git status --short --branch` and confirm clean.
- [ ] Run `git log --oneline -10` and confirm five reminder commits.

## Self-Review

- Spec coverage: reminder ledger, sweep core, commitment/staffing collectors, UI, and docs are all covered.
- Placeholder scan: no TBDs; exact routes, files, tests, idempotency keys, and privacy rules are specified.
- Type consistency: `ReminderEventDTO`, reminder event fields, route paths, and reminder types match across tasks.
