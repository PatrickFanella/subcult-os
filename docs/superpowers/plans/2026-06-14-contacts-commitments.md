# Contacts Commitments Implementation Plan

> **For agentic workers:** Execute this plan task-by-task. Recommended path:
> dispatch a fresh subagent per task, review each result with `review-quality`,
> then continue. For complex multi-agent splits, use
> `parallel-feature-development`, `team-composition-patterns`, and
> `team-communication-protocols`. Steps use checkbox (`- [ ]`) syntax for
> tracking.

**Goal:** Add private workspace contacts and commitments so organizers can remember people and track promises without exposing that memory publicly.

**Architecture:** Add workspace-scoped `contacts` and `commitments` tables, both private to workspace owner/member reads. Owners can mutate; members can read. Contacts are reusable scene memory; commitments are event/workspace promises with status and due dates. Public routes, discovery, tickets, applications, staffing, settlements, and archives must not expose contact notes or commitment notes.

**Tech Stack:** Go `net/http`, pgx/Postgres embedded `schema.sql`, React TypeScript SPA, Vitest, `make verify`.

---

## Current State

- Workspaces have members/invitations and event lifecycle surfaces.
- Participants/applications/staffing capture event-specific people, but no reusable workspace contact list exists.
- Staffing tracks operational tasks; no general commitment/promise ledger exists.
- Notification activity exists, but no contact-centric follow-up memory exists.

## File Structure

- `backend/internal/app/schema.sql`: add `contacts` and `commitments` tables plus indexes.
- `backend/internal/app/app.go`: register contacts and commitments routes.
- `backend/internal/app/contacts.go`: new contact DTOs, list/create/update handlers.
- `backend/internal/app/commitments.go`: new commitment DTOs, list/create/update handlers.
- `backend/internal/app/lifecycle_test.go`: DB-backed API, privacy, and permission tests.
- `web/src/domain.ts`: contact and commitment DTO/request types.
- `web/src/views/WorkspaceView.tsx`: workspace contact and commitment panels.
- `web/src/views/EventEditorView.tsx`: event commitment panel.
- `web/src/App.test.tsx`: frontend smoke/privacy coverage.
- `README.md` and `docs/runbooks/deployment-checklist.md`: private-memory boundary docs.

---

### Task 1: Contact Schema and Read API

**Files:**
- Modify: `backend/internal/app/schema.sql`
- Modify: `backend/internal/app/app.go`
- Create: `backend/internal/app/contacts.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`

- [x] **Step 1: Add failing backend coverage**

Add `TestWorkspaceContactsListAPI` to `backend/internal/app/lifecycle_test.go`:

```go
func TestWorkspaceContactsListAPI(t *testing.T) {
    fx := newLifecycleFixture(t)
    workspaceID := fx.workspaceID

    empty := getJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+workspaceID+"/contacts", http.StatusOK)
    if got := empty.JSON.([]any); len(got) != 0 {
        t.Fatalf("expected empty contacts, got %#v", got)
    }

    var contactID string
    if err := fx.app.db.QueryRow(t.Context(), `
        insert into contacts (workspace_id, display_name, email, phone, notes, tags, created_by_person_id)
        values ($1, 'Mira Door', 'mira@example.test', '+15555550123', 'Prefers late load-in', array['door','trusted'], $2)
        returning id
    `, workspaceID, ownerPersonID(t, fx)).Scan(&contactID); err != nil {
        t.Fatal(err)
    }

    resp := getJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+workspaceID+"/contacts", http.StatusOK)
    contacts := resp.JSON.([]any)
    if len(contacts) != 1 {
        t.Fatalf("expected one contact, got %#v", contacts)
    }
    contact := mustObject(t, contacts[0])
    if contact["id"] != contactID || contact["displayName"] != "Mira Door" || contact["email"] != "mira@example.test" || contact["notes"] != "Prefers late load-in" {
        t.Fatalf("unexpected contact: %#v", contact)
    }
    tags := contact["tags"].([]any)
    if len(tags) != 2 || tags[0] != "door" || tags[1] != "trusted" {
        t.Fatalf("unexpected tags: %#v", tags)
    }

    otherFx := newLifecycleFixture(t)
    getJSON(t, fx.app, nil, "/api/workspaces/"+workspaceID+"/contacts", http.StatusForbidden)
    getJSON(t, fx.app, otherFx.ownerCookie, "/api/workspaces/"+workspaceID+"/contacts", http.StatusForbidden)
}
```

- [x] **Step 2: Add contacts schema**

Append to `schema.sql` after workspace tables:

```sql
create table if not exists contacts (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references workspaces(id) on delete cascade,
  display_name text not null,
  email text,
  phone text,
  notes text not null default '',
  tags text[] not null default '{}',
  created_by_person_id uuid not null references people(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  check (trim(display_name) <> '')
);

create index if not exists contacts_workspace_name_idx on contacts (workspace_id, lower(display_name));
create unique index if not exists contacts_workspace_email_idx on contacts (workspace_id, lower(email)) where email is not null and email <> '';
```

- [x] **Step 3: Register route and DTO**

In `app.go`:

```go
a.mux.HandleFunc("GET /api/workspaces/{workspaceID}/contacts", a.handleListContacts)
```

Create `contacts.go` with:

```go
package app

import (
    "net/http"
    "time"
)

type contactDTO struct {
    ID          string   `json:"id"`
    WorkspaceID string   `json:"workspaceId"`
    DisplayName string   `json:"displayName"`
    Email       string   `json:"email,omitempty"`
    Phone       string   `json:"phone,omitempty"`
    Notes       string   `json:"notes"`
    Tags        []string `json:"tags"`
    CreatedAt   string   `json:"createdAt"`
    UpdatedAt   string   `json:"updatedAt"`
}
```

Implement `handleListContacts`: require workspace `owner` or `member`; query contacts for `workspace_id`; order by `lower(display_name), created_at`; return `[]contactDTO`. Use `pgx.ErrNoRows` only if you add single-contact helpers; list should return `[]`.

- [x] **Step 4: Add TypeScript DTO**

In `web/src/domain.ts`:

```ts
export interface ContactDTO {
  id: string;
  workspaceId: string;
  displayName: string;
  email?: string;
  phone?: string;
  notes: string;
  tags: string[];
  createdAt: string;
  updatedAt: string;
}
```

- [ ] **Step 5: Verify and commit**

Run: `gofmt -w backend/internal/app/app.go backend/internal/app/contacts.go backend/internal/app/lifecycle_test.go && make verify`

Commit:

```bash
git add backend/internal/app/schema.sql backend/internal/app/app.go backend/internal/app/contacts.go backend/internal/app/lifecycle_test.go web/src/domain.ts docs/superpowers/plans/2026-06-14-contacts-commitments.md
git commit -m "Add workspace contacts read model"
```

---

### Task 2: Contact Create and Update APIs

**Files:**
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/contacts.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`

- [ ] **Step 1: Add failing mutation coverage**

Add `TestWorkspaceContactsMutationAPI`:

```go
func TestWorkspaceContactsMutationAPI(t *testing.T) {
    fx := newLifecycleFixture(t)
    workspaceID := fx.workspaceID

    created := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+workspaceID+"/contacts", map[string]any{
        "displayName": "  Mira Door  ",
        "email": " MIRA@EXAMPLE.TEST ",
        "phone": "  +15555550123 ",
        "notes": "  Do not publish this note.  ",
        "tags": []string{" door ", "trusted", "door", ""},
    }, http.StatusOK)
    contact := mustObject(t, created.JSON)
    if contact["displayName"] != "Mira Door" || contact["email"] != "mira@example.test" || contact["phone"] != "+15555550123" || contact["notes"] != "Do not publish this note." {
        t.Fatalf("unexpected normalized contact: %#v", contact)
    }
    tags := contact["tags"].([]any)
    if len(tags) != 2 || tags[0] != "door" || tags[1] != "trusted" {
        t.Fatalf("unexpected normalized tags: %#v", tags)
    }

    contactID := contact["id"].(string)
    postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+workspaceID+"/contacts", map[string]any{
        "displayName": "Duplicate Mira",
        "email": "mira@example.test",
    }, http.StatusConflict)

    updated := patchJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+workspaceID+"/contacts/"+contactID, map[string]any{
        "displayName": "Mira Lead",
        "clearEmail": true,
        "clearPhone": true,
        "notes": "Still private",
        "tags": []string{"lead"},
    }, http.StatusOK)
    updatedContact := mustObject(t, updated.JSON)
    if updatedContact["displayName"] != "Mira Lead" || updatedContact["email"] != nil || updatedContact["phone"] != nil || updatedContact["notes"] != "Still private" {
        t.Fatalf("unexpected updated contact: %#v", updatedContact)
    }

    postJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+workspaceID+"/contacts", map[string]any{"displayName": "Member"}, http.StatusForbidden)
    patchJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+workspaceID+"/contacts/"+contactID, map[string]any{"displayName": "Member"}, http.StatusForbidden)
    postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+workspaceID+"/contacts", map[string]any{"displayName": ""}, http.StatusBadRequest)

    rawAudit := auditMetadataForAction(t, fx.app.db, "contact.created") + auditMetadataForAction(t, fx.app.db, "contact.updated")
    if strings.Contains(rawAudit, "Do not publish this note") || strings.Contains(rawAudit, "Still private") || strings.Contains(rawAudit, "mira@example.test") || strings.Contains(rawAudit, "+15555550123") || strings.Contains(rawAudit, "Mira") || strings.Contains(rawAudit, "trusted") {
        t.Fatalf("audit metadata leaked notes: %s", rawAudit)
    }
}
```

Add this audit helper in `lifecycle_test.go` if it is not already present:

```go
func auditMetadataForAction(t *testing.T, db *pgxpool.Pool, action string) string {
    t.Helper()
    rows, err := db.Query(t.Context(), `select metadata::text from audit_entries where action = $1 order by created_at asc`, action)
    if err != nil {
        t.Fatal(err)
    }
    defer rows.Close()

    var combined strings.Builder
    for rows.Next() {
        var metadata string
        if err := rows.Scan(&metadata); err != nil {
            t.Fatal(err)
        }
        combined.WriteString(metadata)
        combined.WriteString("\n")
    }
    if err := rows.Err(); err != nil {
        t.Fatal(err)
    }
    return combined.String()
}
```

Ensure `lifecycle_test.go` imports `strings` and `github.com/jackc/pgx/v5/pgxpool` if the helper requires them.

- [ ] **Step 2: Register mutation routes**

In `app.go`:

```go
a.mux.HandleFunc("POST /api/workspaces/{workspaceID}/contacts", a.handleCreateContact)
a.mux.HandleFunc("PATCH /api/workspaces/{workspaceID}/contacts/{contactID}", a.handleUpdateContact)
```

- [ ] **Step 3: Implement request types and normalization**

In `contacts.go`:

```go
type createContactRequest struct {
    DisplayName string   `json:"displayName"`
    Email       string   `json:"email"`
    Phone       string   `json:"phone"`
    Notes       string   `json:"notes"`
    Tags        []string `json:"tags"`
}

type updateContactRequest struct {
    DisplayName *string  `json:"displayName"`
    Email       *string  `json:"email"`
    Phone       *string  `json:"phone"`
    Notes       *string  `json:"notes"`
    Tags        []string `json:"tags"`
    ClearEmail  bool     `json:"clearEmail"`
    ClearPhone  bool     `json:"clearPhone"`
}
```

Rules: owner-only; display name trimmed and required on create and when supplied on update; email trimmed/lowercased, optional, and must contain `@` if present; phone trimmed optional; notes trimmed and max 2000 runes; tags trimmed, deduped case-insensitively, max 20 tags, max 40 runes each. Empty email/phone on create means null. On update, omitted fields mean no change; `clearEmail`/`clearPhone` clear values; reject `clearEmail && email != nil` and `clearPhone && phone != nil` with `400`. Duplicate non-empty email within the same workspace returns `409 Conflict`, not `500`.

- [ ] **Step 4: Implement create/update**

Create inserts contact and audits `contact.created` with metadata `{contactId, workspaceId}` only. Update verifies contact belongs to workspace, applies only supplied fields/clear flags, and audits `contact.updated` with metadata `{contactId, workspaceId}` only. Do not write notes, email, phone, tags, or display name into audit metadata.

- [ ] **Step 5: Verify and commit**

Run: `gofmt -w backend/internal/app/app.go backend/internal/app/contacts.go backend/internal/app/lifecycle_test.go && make verify`

Commit:

```bash
git add backend/internal/app/app.go backend/internal/app/contacts.go backend/internal/app/lifecycle_test.go web/src/domain.ts
git commit -m "Add workspace contact mutations"
```

---

### Task 3: Commitment Schema and APIs

**Files:**
- Modify: `backend/internal/app/schema.sql`
- Modify: `backend/internal/app/app.go`
- Create: `backend/internal/app/commitments.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`

- [ ] **Step 1: Add failing commitment coverage**

Add `TestCommitmentsAPI`:

```go
func TestCommitmentsAPI(t *testing.T) {
    fx := newLifecycleFixture(t)
    event := createEvent(t, fx, "Benefit Show", 20)
    eventID := mustString(t, event, "id")
    dueAt := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)

    created := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/commitments", map[string]any{
        "eventId": eventID,
        "title": "Confirm projector",
        "description": "Private vendor detail",
        "dueAt": dueAt,
        "ownerPersonId": ownerPersonID(t, fx),
    }, http.StatusOK)
    commitment := mustObject(t, created.JSON)
    if commitment["title"] != "Confirm projector" || commitment["status"] != "open" || commitment["eventId"] != eventID {
        t.Fatalf("unexpected commitment: %#v", commitment)
    }

    workspaceList := getJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/commitments", http.StatusOK)
    if got := workspaceList.JSON.([]any); len(got) != 1 {
        t.Fatalf("expected one workspace commitment, got %#v", got)
    }
    eventList := getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/commitments", http.StatusOK)
    if got := eventList.JSON.([]any); len(got) != 1 {
        t.Fatalf("expected one event commitment, got %#v", got)
    }

    commitmentID := commitment["id"].(string)
    updated := patchJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/commitments/"+commitmentID, map[string]any{"status": "done"}, http.StatusOK)
    doneCommitment := mustObject(t, updated.JSON)
    if doneCommitment["status"] != "done" || doneCommitment["completedAt"] == nil || doneCommitment["completedByPersonId"] == nil {
        t.Fatalf("expected done status: %#v", updated.JSON)
    }
    firstCompletedAt := doneCommitment["completedAt"]
    secondDone := patchJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/commitments/"+commitmentID, map[string]any{"status": "done"}, http.StatusOK)
    if mustObject(t, secondDone.JSON)["completedAt"] != firstCompletedAt {
        t.Fatalf("expected idempotent done to preserve completion time: %#v", secondDone.JSON)
    }
    reopened := patchJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/commitments/"+commitmentID, map[string]any{"status": "open"}, http.StatusOK)
    reopenedCommitment := mustObject(t, reopened.JSON)
    if reopenedCommitment["status"] != "open" || reopenedCommitment["completedAt"] != nil || reopenedCommitment["completedByPersonId"] != nil {
        t.Fatalf("expected reopen to clear completion fields: %#v", reopened.JSON)
    }
    cancelled := patchJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/commitments/"+commitmentID, map[string]any{"status": "cancelled"}, http.StatusOK)
    cancelledCommitment := mustObject(t, cancelled.JSON)
    if cancelledCommitment["status"] != "cancelled" || cancelledCommitment["completedAt"] != nil || cancelledCommitment["completedByPersonId"] != nil {
        t.Fatalf("expected cancel to clear completion fields: %#v", cancelled.JSON)
    }

    postJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/commitments", map[string]any{"title": "member"}, http.StatusForbidden)
    otherFx := newLifecycleFixture(t)
    getJSON(t, fx.app, otherFx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/commitments", http.StatusForbidden)

    otherEvent := createEvent(t, otherFx, "Other workspace event", 5)
    otherContact := postJSON(t, otherFx.app, otherFx.ownerCookie, "/api/workspaces/"+otherFx.workspaceID+"/contacts", map[string]any{"displayName": "Other Contact"}, http.StatusOK)
    postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/commitments", map[string]any{"title": "bad event", "eventId": mustString(t, otherEvent, "id")}, http.StatusBadRequest)
    postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/commitments", map[string]any{"title": "bad contact", "contactId": mustObject(t, otherContact.JSON)["id"]}, http.StatusBadRequest)
    postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/commitments", map[string]any{"title": "bad owner", "ownerPersonId": ownerPersonID(t, otherFx)}, http.StatusBadRequest)
    patchJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/commitments/"+commitmentID, map[string]any{"eventId": mustString(t, otherEvent, "id")}, http.StatusBadRequest)
    patchJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/commitments/"+commitmentID, map[string]any{"contactId": mustObject(t, otherContact.JSON)["id"]}, http.StatusBadRequest)
    patchJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/commitments/"+commitmentID, map[string]any{"ownerPersonId": ownerPersonID(t, otherFx)}, http.StatusBadRequest)

    rawAudit := auditMetadataForAction(t, fx.app.db, "commitment.created") + auditMetadataForAction(t, fx.app.db, "commitment.updated")
    if strings.Contains(rawAudit, "Private vendor detail") || strings.Contains(rawAudit, "Confirm projector") {
        t.Fatalf("audit metadata leaked commitment description: %s", rawAudit)
    }
}
```

- [ ] **Step 2: Add commitments schema**

Append:

```sql
create table if not exists commitments (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references workspaces(id) on delete cascade,
  event_id uuid references events(id) on delete cascade,
  contact_id uuid references contacts(id) on delete set null,
  title text not null,
  description text not null default '',
  due_at timestamptz,
  status text not null default 'open' check (status in ('open', 'done', 'cancelled')),
  owner_person_id uuid references people(id),
  created_by_person_id uuid not null references people(id),
  completed_at timestamptz,
  completed_by_person_id uuid references people(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  check (trim(title) <> '')
);

create index if not exists commitments_workspace_status_due_idx on commitments (workspace_id, status, due_at nulls last, created_at desc);
create index if not exists commitments_event_idx on commitments (event_id, status, due_at nulls last);
```

- [ ] **Step 3: Register routes and DTOs**

In `app.go`:

```go
a.mux.HandleFunc("GET /api/workspaces/{workspaceID}/commitments", a.handleListWorkspaceCommitments)
a.mux.HandleFunc("POST /api/workspaces/{workspaceID}/commitments", a.handleCreateCommitment)
a.mux.HandleFunc("PATCH /api/workspaces/{workspaceID}/commitments/{commitmentID}", a.handleUpdateCommitment)
a.mux.HandleFunc("GET /api/events/{eventID}/commitments", a.handleListEventCommitments)
```

In `domain.ts`, add `CommitmentDTO`, `CreateCommitmentRequestDTO`, and `UpdateCommitmentRequestDTO` with fields matching schema and JSON camelCase. The update request uses optional fields plus explicit clear booleans:

```ts
export interface UpdateCommitmentRequestDTO {
  title?: string;
  description?: string;
  dueAt?: string;
  eventId?: string;
  contactId?: string;
  ownerPersonId?: string;
  status?: 'open' | 'done' | 'cancelled';
  clearDueAt?: boolean;
  clearEvent?: boolean;
  clearContact?: boolean;
  clearOwner?: boolean;
}
```

- [ ] **Step 4: Implement handlers**

Rules: owner/member can list; owner only creates/updates. Create and update validate event/contact/owner person belong to the same workspace if supplied. Update supports partial updates with pointer fields and booleans `clearDueAt`, `clearEvent`, `clearContact`, `clearOwner`; reject conflicts such as `clearDueAt && dueAt != nil`, `clearEvent && eventId != nil`, `clearContact && contactId != nil`, or `clearOwner && ownerPersonId != nil`. Completing (`status: 'done'`) is idempotent: preserve first `completedAt/completedByPersonId` if already done. Reopening/cancelling from `done` clears `completedAt` and `completedByPersonId`, matching staffing semantics. Audit `commitment.created`/`commitment.updated` with IDs/status only, never title or description.

- [ ] **Step 5: Verify and commit**

Run: `gofmt -w backend/internal/app/app.go backend/internal/app/commitments.go backend/internal/app/lifecycle_test.go && make verify`

Commit:

```bash
git add backend/internal/app/schema.sql backend/internal/app/app.go backend/internal/app/commitments.go backend/internal/app/lifecycle_test.go web/src/domain.ts
git commit -m "Add workspace commitments"
```

---

### Task 4: Workspace and Event UI Surfaces

**Files:**
- Modify: `web/src/views/WorkspaceView.tsx`
- Modify: `web/src/views/EventEditorView.tsx`
- Modify: `web/src/App.test.tsx`

- [ ] **Step 1: Add frontend fixtures/tests**

Extend `web/src/App.test.tsx` to cover:

- Workspace renders `Contacts` panel with contact name/tags and private note copy.
- Workspace renders `Commitments` panel with open/done counts and due date.
- Event editor renders event-specific commitments.
- Owner controls are visible for owner workspace, hidden for member workspace.

- [ ] **Step 2: Add WorkspaceView state and loaders**

In `WorkspaceView.tsx`, load `/api/workspaces/${workspace.id}/contacts` and `/api/workspaces/${workspace.id}/commitments` alongside existing workspace data. 403 should hide panels; other errors use the page error state.

- [ ] **Step 3: Render contact panel**

Render `Contacts` with empty state `No contacts yet. Add people you want to remember across events.` Show display name, email/phone if present, tags, and notes. Owner-only create/edit form can be simple: displayName, email, phone, tags comma-separated, notes.

- [ ] **Step 4: Render commitment panel**

Render `Commitments` with status filters/copy. Owner-only create form: title, description, dueAt, optional event select from loaded events. Owner-only status buttons: mark done, reopen, cancel. Do not render commitment descriptions in public routes.

- [ ] **Step 5: Add EventEditor commitment panel**

In `EventEditorView.tsx`, load `/api/events/${event.id}/commitments`. Render event-specific commitments with owner-only create/status controls. Reuse workspace members and event title context; do not duplicate contact editor here.

- [ ] **Step 6: Verify and commit**

Run: `make verify`

Commit:

```bash
git add web/src/views/WorkspaceView.tsx web/src/views/EventEditorView.tsx web/src/App.test.tsx
git commit -m "Add contacts and commitments UI"
```

---

### Task 5: Privacy Hardening and Documentation

**Files:**
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/App.test.tsx`
- Modify: `README.md`
- Modify: `docs/runbooks/deployment-checklist.md`

- [ ] **Step 1: Strengthen backend privacy tests**

Add assertions that public discovery, public event detail, ticket lookup, archive summaries, notification list, report, settlement, and dev outbox routes do not expose contact notes or commitment descriptions unless explicitly expected on private contacts/commitments routes.

- [ ] **Step 2: Strengthen frontend privacy tests**

Assert contact notes and commitment descriptions appear only in private workspace/editor sections and not in `/discover` or public event page render tests.

- [ ] **Step 3: Document private-memory boundary**

In `README.md`, add shipped-slice copy: `Private contacts and commitments help organizers remember scene relationships and promises; they are workspace-only and never shown on public discovery/event pages.`

In `docs/runbooks/deployment-checklist.md`, add: `Contacts/commitments may contain sensitive free text; review logs, notification templates, and public routes before production launch.`

- [ ] **Step 4: Final verification and commit**

Run: `make verify`

Commit:

```bash
git add backend/internal/app/lifecycle_test.go web/src/App.test.tsx README.md docs/runbooks/deployment-checklist.md
git commit -m "Document contacts privacy boundaries"
```

---

## Out of Scope

- Public profiles, public contact pages, follows, CRM import/export.
- Automatic contact creation from applications/tickets.
- Reminder scheduling or recurring commitments.
- Cross-workspace contacts.
- Contact dedupe/merge beyond unique email per workspace.

## Final Verification

- [ ] Run `make verify`.
- [ ] Run `git status --short --branch` and confirm clean.
- [ ] Run `git log --oneline -10` and confirm five contacts/commitments commits.

## Self-Review

- Spec coverage: contacts read/write, commitments read/write, UI surfaces, privacy/docs are all assigned to concrete tasks.
- Placeholder scan: no TBDs; each task includes exact files, route names, commands, and behavior.
- Type consistency: `ContactDTO`, `CommitmentDTO`, create/update request names, and route paths are consistent across backend/frontend.
