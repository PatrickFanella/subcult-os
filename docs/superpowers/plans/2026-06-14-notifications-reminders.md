# Notifications Activity Implementation Plan

> **For agentic workers:** Execute this plan task-by-task. Recommended path:
> dispatch a fresh subagent per task, review each result with `review-quality`,
> then continue. For complex multi-agent splits, use
> `parallel-feature-development`, `team-composition-patterns`, and
> `team-communication-protocols`. Steps use checkbox (`- [ ]`) syntax for
> tracking.

**Goal:** Add privacy-safe operational notification activity on top of the existing email outbox.

**Architecture:** Keep delivery local to the existing `email_outbox` table. Add a typed `notification_events` ledger to make notification creation idempotent, auditable, and visible to workspace operators without exposing private free-text application/staffing notes.

**Tech Stack:** Go `net/http` backend, pgx/Postgres embedded `schema.sql`, React TypeScript SPA, Vitest, `make verify`.

---

## Current State

- `email_outbox` already exists and is used for workspace invitations and ticket emails.
- `/api/dev/email-outbox` exposes the development outbox.
- Role applications can be reviewed privately; staffing items can be assigned and completed.
- There is no typed notification ledger, no idempotency key per notification rule, and no operator-facing notification activity.

## File Structure

- `backend/internal/app/schema.sql`: add `notification_events`.
- `backend/internal/app/notifications.go`: new helper functions and DTO/load handlers.
- `backend/internal/app/app.go`: register notification routes.
- `backend/internal/app/event_roles.go`: enqueue review outcome notifications.
- `backend/internal/app/staffing.go`: enqueue staffing assignment notifications.
- `backend/internal/app/lifecycle_test.go`: DB-backed tests.
- `web/src/domain.ts`: notification DTO types.
- `web/src/views/EventEditorView.tsx`: notification activity panel.
- `web/src/App.test.tsx`: frontend regression coverage.
- `README.md`: notification privacy notes.

---

### Task 1: Notification Ledger

**Files:**
- Modify: `backend/internal/app/schema.sql`
- Create: `backend/internal/app/notifications.go`
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`
- Modify: `docs/superpowers/plans/2026-06-14-notifications-reminders.md`

- [x] **Step 1: Add schema**

Add after `email_outbox`:

```sql
create table if not exists notification_events (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references workspaces(id) on delete cascade,
  event_id uuid references events(id) on delete cascade,
  recipient_email text not null,
  notification_type text not null,
  related_type text not null,
  related_id uuid,
  idempotency_key text not null unique,
  email_outbox_id uuid references email_outbox(id) on delete set null,
  subject text not null,
  preview text not null,
  status text not null default 'queued' check (status in ('queued')),
  created_by_person_id uuid references people(id),
  created_at timestamptz not null default now()
);

create index if not exists notification_events_event_created_idx
  on notification_events (event_id, created_at desc);
```

- [x] **Step 2: Add helper and DTO**

Create `backend/internal/app/notifications.go` with:

```go
package app

import (
    "context"
    "database/sql"
    "net/http"
    "strings"
    "time"

    "github.com/jackc/pgx/v5"
)

type notificationEventDTO struct {
    ID               string  `json:"id"`
    EventID          *string `json:"eventId,omitempty"`
    RecipientEmail   string  `json:"recipientEmail"`
    NotificationType string  `json:"notificationType"`
    RelatedType      string  `json:"relatedType"`
    RelatedID        *string `json:"relatedId,omitempty"`
    Subject          string  `json:"subject"`
    Preview          string  `json:"preview"`
    Status           string  `json:"status"`
    CreatedAt        string  `json:"createdAt"`
}

type enqueueNotificationParams struct {
    WorkspaceID      string
    EventID          string
    RecipientEmail   string
    NotificationType string
    RelatedType      string
    RelatedID        string
    IdempotencyKey   string
    Subject          string
    Body             string
    Preview          string
    CreatedByPersonID string
}

func (a *App) enqueueNotification(ctx context.Context, tx pgx.Tx, params enqueueNotificationParams) (bool, error) {
    recipient := normalizeEmail(params.RecipientEmail)
    if recipient == "" || strings.TrimSpace(params.IdempotencyKey) == "" {
        return false, nil
    }

    eventID := sql.NullString{String: params.EventID, Valid: params.EventID != ""}
    relatedID := sql.NullString{String: params.RelatedID, Valid: params.RelatedID != ""}
    actorID := sql.NullString{String: params.CreatedByPersonID, Valid: params.CreatedByPersonID != ""}
    preview := strings.TrimSpace(params.Preview)
    if len([]rune(preview)) > 240 {
        preview = string([]rune(preview)[:240])
    }

    var notificationID string
    if err := tx.QueryRow(ctx, `
        insert into notification_events (workspace_id, event_id, recipient_email, notification_type, related_type, related_id, idempotency_key, email_outbox_id, subject, preview, created_by_person_id)
        values ($1, $2, $3, $4, $5, $6, $7, null, $8, $9, $10)
        on conflict (idempotency_key) do nothing
        returning id
    `, params.WorkspaceID, eventID, recipient, params.NotificationType, params.RelatedType, relatedID, params.IdempotencyKey, params.Subject, preview, actorID).Scan(&notificationID); err != nil {
        if err == pgx.ErrNoRows {
            return false, nil
        }
        return false, err
    }

    var outboxID string
    if err := tx.QueryRow(ctx, `
        insert into email_outbox (recipient_email, subject, body, related_type, related_id)
        values ($1, $2, $3, $4, $5)
        returning id
    `, recipient, params.Subject, params.Body, params.RelatedType, relatedID).Scan(&outboxID); err != nil {
        return false, err
    }

    if _, err := tx.Exec(ctx, `update notification_events set email_outbox_id = $1 where id = $2`, outboxID, notificationID); err != nil {
        return false, err
    }
    return true, nil
}
```

- [x] **Step 3: Add list route**

Register:

```go
a.mux.HandleFunc("GET /api/events/{eventID}/notifications", a.handleListEventNotifications)
```

Implement `handleListEventNotifications`: authenticated workspace owner/member for the event can read; return latest 50 `notificationEventDTO` rows for the event ordered `created_at desc`; non-members get current private-route behavior.

- [x] **Step 4: Add types/tests**

Add TypeScript:

```ts
export interface NotificationEventDTO {
  id: string;
  eventId?: string;
  recipientEmail: string;
  notificationType: string;
  relatedType: string;
  relatedId?: string;
  subject: string;
  preview: string;
  status: 'queued';
  createdAt: string;
}
```

Add `TestNotificationLedgerAPI`: manually insert one `email_outbox` and one `notification_events` row; owner/member can list it; other workspace user/unauth forbidden; response has no free-text application message/staffing notes.

- [ ] **Step 5: Verify and commit**

Run: `gofmt -w backend/internal/app/notifications.go backend/internal/app/app.go backend/internal/app/lifecycle_test.go && make verify`

Commit:

```bash
git add backend/internal/app/schema.sql backend/internal/app/notifications.go backend/internal/app/app.go backend/internal/app/lifecycle_test.go web/src/domain.ts docs/superpowers/plans/2026-06-14-notifications-reminders.md
git commit -m "Add notification event ledger"
```

---

### Task 2: Role Review Notifications

**Files:**
- Modify: `backend/internal/app/event_roles.go`
- Modify: `backend/internal/app/lifecycle_test.go`

- [ ] **Step 1: Notify only terminal public outcomes**

In `handleReviewEventRoleApplication`, after a real status change to `accepted`, `waitlisted`, or `rejected`, enqueue one notification to `application.ApplicantEmail`. Load event title, workspace ID, role name, and applicant email inside the review transaction before enqueueing.

Idempotency key:

```go
"role_application:" + application.ID + ":" + nextStatus
```

Subject/body must include event title and role name, but **must not include** the applicant free-text message. Preview should be generic, e.g. `Application accepted for Performer`.

- [ ] **Step 2: Preserve idempotency**

If `previousStatus == nextStatus`, return the current DTO without sending a notification. If the implementation keeps the existing update/audit behavior for unchanged statuses, notification sending must still be gated on `previousStatus != nextStatus` plus the ledger idempotency key.

- [ ] **Step 3: Add tests**

Extend role-review tests to assert:
- accepted emits exactly one `notification_events` row and one outbox email;
- repeated accepted review does not duplicate;
- rejected/waitlisted emit their own notification type/key;
- submitted/under_review/withdrawn do not emit applicant outcome emails;
- notification/outbox body does not include the original application message.

- [ ] **Step 4: Verify and commit**

Run: `gofmt -w backend/internal/app/event_roles.go backend/internal/app/lifecycle_test.go && make verify`

Commit:

```bash
git add backend/internal/app/event_roles.go backend/internal/app/lifecycle_test.go
git commit -m "Notify role application outcomes"
```

---

### Task 3: Staffing Assignment Notifications

**Files:**
- Modify: `backend/internal/app/staffing.go`
- Modify: `backend/internal/app/lifecycle_test.go`

- [ ] **Step 1: Notify new assignments**

When `PATCH /api/events/{eventID}/staffing/{staffingID}` changes an item from unassigned or different assignee to a workspace person or accepted/confirmed application, enqueue a `staffing.assignment` notification only when the resulting status is `assigned` or an `open` item is promoted to `assigned`. Do not send assignment notifications for rows whose resulting status is `completed` or `cancelled`.

Recipient:
- `assigned_person_id`: person email from `people`.
- `assigned_application_id`: applicant email from `event_role_applications`.

Load event title, workspace ID, staffing title, and recipient email inside the update transaction after locking the event and staffing item.

Idempotency key:

```go
"staffing_assignment:" + staffingID + ":" + recipientEmail
```

Subject/body include event title and staffing title. Do **not** include staffing notes or application message. Reassigning to a different recipient creates a new notification; clearing assignee does not send.

- [ ] **Step 2: Add tests**

Extend staffing update tests to assert:
- assigning a workspace member queues one notification;
- repeated same assignment does not duplicate;
- assigning an accepted application queues to applicant email;
- clearing/cancelling/completing without new assignee does not queue;
- notification/outbox body excludes staffing notes and application message.

- [ ] **Step 3: Verify and commit**

Run: `gofmt -w backend/internal/app/staffing.go backend/internal/app/lifecycle_test.go && make verify`

Commit:

```bash
git add backend/internal/app/staffing.go backend/internal/app/lifecycle_test.go
git commit -m "Notify staffing assignments"
```

---

### Task 4: Operator Notification Activity UI

**Files:**
- Modify: `web/src/views/EventEditorView.tsx`
- Modify: `web/src/App.test.tsx`

- [ ] **Step 1: Load notifications**

In `EventEditorView`, load `/api/events/${event.id}/notifications` for owner/member private event views. Suppress 404/403 the same way other private panels do if needed.

- [ ] **Step 2: Render panel**

Add panel `Notification activity` showing count and latest rows with:
- recipient email;
- notification type;
- subject;
- preview;
- queued status;
- created timestamp.

Do not render full outbox body. Do not render application messages or staffing notes.

- [ ] **Step 3: Refresh after actions**

After role review status changes and staffing assignment updates, reload notifications or patch local state by refetching.

- [ ] **Step 4: Add tests**

Add App tests with mocked notifications that assert panel copy appears and private free-text does not appear.

- [ ] **Step 5: Verify and commit**

Run: `make verify`

Commit:

```bash
git add web/src/views/EventEditorView.tsx web/src/App.test.tsx
git commit -m "Show event notification activity"
```

---

### Task 5: Docs and Safety Pass

**Files:**
- Modify: `README.md`
- Modify: `docs/runbooks/deployment-checklist.md`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/App.test.tsx`

- [ ] **Step 1: Strengthen safety tests**

Add regression assertions that notification APIs and UI never expose:
- role application `message`;
- staffing `notes`;
- archive note bodies beyond existing archive routes;
- settlement internals.

- [ ] **Step 2: Document notification rules**

Update README shipped-slice section with notification activity. Update deployment checklist with notification privacy boundary: outbox rows contain recipient email and body; public discovery never exposes notification data; production delivery needs real mail provider review.

- [ ] **Step 3: Final verify and commit**

Run: `make verify`

Commit:

```bash
git add README.md docs/runbooks/deployment-checklist.md backend/internal/app/lifecycle_test.go web/src/App.test.tsx
git commit -m "Document notification safety boundaries"
```

---

## Final Verification

- [ ] Run `make verify`.
- [ ] Run `git status --short --branch` and confirm the working tree is clean.
- [ ] Run `git log --oneline -10` and confirm the five notification commits are present.

## Out of Scope

- Real SMTP/provider delivery.
- User notification preferences.
- Background scheduler, cron reminders, or manual reminder actions.
- SMS/push notifications.
- Public notification feeds.
- Sending role application free-text or staffing notes in generic activity panels.

## Self-Review

- Spec coverage: plan covers typed ledger, review notifications, staffing assignment notifications, operator activity UI, docs/safety.
- Placeholder scan: all slices define concrete routes, schema, fields, and verification commands.
- Type consistency: `NotificationEventDTO`, `notificationEventDTO`, `notification_events`, and notification type strings are consistent.
