# Private Archive Lessons Learned Implementation Plan

> **For agentic workers:** Execute this plan task-by-task. Recommended path:
> dispatch a fresh subagent per task, review each result with `review-quality`,
> then continue. For complex multi-agent splits, use
> `parallel-feature-development`, `team-composition-patterns`, and
> `team-communication-protocols`. Steps use checkbox (`- [ ]`) syntax for
> tracking.

**Goal:** Turn finalized event closeout into private workspace memory: an archive record, private read API, lessons learned notes, operator UI, and a safe “seed next draft” flow.

**Architecture:** Keep public event pages, immutable reports, and settlements separate from archive memory. Add an `event_archives` root row created idempotently at end-of-night and append-only `event_archive_notes` for lessons learned; expose private owner/member reads and owner-only note/draft-seed writes through event-scoped API routes.

**Tech Stack:** Go `net/http` backend, pgx/Postgres embedded `schema.sql`, React TypeScript SPA, Vitest, `make verify`.

---

## Current State

- Events already move `draft -> published -> end_of_night`.
- End-of-night creates an immutable `event_reports` snapshot and an `event_settlements` row.
- Settlements can be adjusted and finalized by owners.
- Workspace owner/member permissions already gate private event/report/settlement reads.
- There is no durable archive object, lessons learned surface, or safe event cloning flow yet.

## File Structure

- `backend/internal/app/schema.sql`: add archive tables and additive `alter table` statements if needed.
- `backend/internal/app/app.go`: register private archive routes.
- `backend/internal/app/events.go`: implement archive creation/read/notes/seed handlers near existing event report and settlement handlers unless this file becomes too large during execution.
- `backend/internal/app/lifecycle_test.go`: add DB-backed lifecycle tests for archive creation, permissions, notes, and seed safety.
- `web/src/domain.ts`: add archive DTO types.
- `web/src/api.ts`: reuse existing API helpers and `ApiError`; only change if endpoint helper behavior is needed.
- `web/src/views/EventEditorView.tsx`: show archive status/content for closed events.
- `web/src/views/WorkspaceView.tsx`: expose closed-event archive affordances and the “seed draft” entry point.
- `web/src/App.test.tsx`: add smoke regression coverage for the archive UI.

---

### Task 1: Archive Spine

**Files:**
- Modify: `backend/internal/app/schema.sql`
- Modify: `backend/internal/app/events.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `docs/superpowers/plans/2026-06-13-private-archive-lessons-learned.md`

- [x] **Step 1: Add failing lifecycle coverage**

Add a test named `TestFirstEventLifecycleCreatesArchiveAtEndOfNight` in `backend/internal/app/lifecycle_test.go`:

```go
func TestFirstEventLifecycleCreatesArchiveAtEndOfNight(t *testing.T) {
    fx := newLifecycleFixture(t)
    event := createEvent(t, fx, "Night Market", 4)
    eventID := mustString(t, event, "id")
    publishEvent(t, fx, eventID)

    postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)

    var archiveID string
    var status string
    var noteCount int
    if err := fx.app.db.QueryRow(t.Context(), `
        select id, status, note_count
        from event_archives
        where event_id = $1
    `, eventID).Scan(&archiveID, &status, &noteCount); err != nil {
        t.Fatal(err)
    }
    if archiveID == "" || status != "private" || noteCount != 0 {
        t.Fatalf("unexpected archive row: id=%q status=%q noteCount=%d", archiveID, status, noteCount)
    }

    postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)
    var count int
    if err := fx.app.db.QueryRow(t.Context(), `select count(*) from event_archives where event_id = $1`, eventID).Scan(&count); err != nil {
        t.Fatal(err)
    }
    if count != 1 {
        t.Fatalf("expected exactly one archive row, got %d", count)
    }
}
```

- [ ] **Step 2: Run the failing test**

Run: `cd backend && go test ./internal/app -run TestFirstEventLifecycleCreatesArchiveAtEndOfNight -count=1`

Expected: FAIL because `event_archives` does not exist.

- [x] **Step 3: Add `event_archives` schema**

In `backend/internal/app/schema.sql`, after `event_settlements`, add:

```sql
create table if not exists event_archives (
  id uuid primary key default gen_random_uuid(),
  event_id uuid not null unique references events(id) on delete cascade,
  report_id uuid not null references event_reports(id) on delete cascade,
  settlement_id uuid not null references event_settlements(id) on delete cascade,
  status text not null default 'private' check (status in ('private')),
  note_count integer not null default 0 check (note_count >= 0),
  created_by_person_id uuid not null references people(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
```

- [x] **Step 4: Create archive row in end-of-night transaction**

Add a small helper in `events.go` so both the existing-report/idempotent branch and the new-report branch can create a missing archive row:

```go
func (a *App) ensureEventArchive(ctx context.Context, tx pgx.Tx, eventID string, reportID string, actorID string) error {
    _, err := tx.Exec(ctx, `
        insert into event_archives (event_id, report_id, settlement_id, created_by_person_id)
        values ($1, $2, (select id from event_settlements where event_id = $1), $3)
        on conflict (event_id) do nothing
    `, eventID, reportID, actorID)
    return err
}
```

In `handleEndOfNight`, call this helper after the report and settlement exist. In the existing-report branch, call it before returning the existing snapshot. In the new-report branch, call it after inserting the report and settlement.

If the current local variable names differ, adapt only names, not behavior: use the report ID, current event ID, and current actor person ID.

The helper replaces this inline shape:

```go
if err := a.ensureEventArchive(r.Context(), tx, event.ID, reportID, actorID); err != nil {
    writeError(w, http.StatusInternalServerError, "could not create archive")
    return
}
```

- [ ] **Step 5: Run verification and commit**

Run: `gofmt -w backend/internal/app/events.go backend/internal/app/lifecycle_test.go && make verify`

Expected: PASS.

Commit:

```bash
git add backend/internal/app/schema.sql backend/internal/app/events.go backend/internal/app/lifecycle_test.go docs/superpowers/plans/2026-06-13-private-archive-lessons-learned.md
git commit -m "Add private event archive records"
```

---

### Task 2: Private Archive Read API

**Files:**
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/events.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`

- [ ] **Step 1: Add failing API coverage**

Add `TestFirstEventLifecycleArchiveAPI` to `backend/internal/app/lifecycle_test.go`:

```go
func TestFirstEventLifecycleArchiveAPI(t *testing.T) {
    fx := newLifecycleFixture(t)
    event := createEvent(t, fx, "Night Market", 4)
    eventID := mustString(t, event, "id")

    getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/archive", http.StatusNotFound)

    publishEvent(t, fx, eventID)
    postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)

    ownerResp := getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/archive", http.StatusOK)
    archive := mustObject(t, ownerResp.JSON)
    if archive["eventId"] != eventID || archive["status"] != "private" || int(archive["noteCount"].(float64)) != 0 {
        t.Fatalf("unexpected archive response: %#v", archive)
    }
    if archive["reportId"] == "" || archive["settlementId"] == "" || archive["createdAt"] == "" || archive["updatedAt"] == "" {
        t.Fatalf("archive response missing references/timestamps: %#v", archive)
    }
    notes, ok := archive["notes"].([]any)
    if !ok || len(notes) != 0 {
        t.Fatalf("expected empty notes array, got %#v", archive["notes"])
    }

    getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/archive", http.StatusOK)
    getJSON(t, fx.app, nil, "/api/events/"+eventID+"/archive", http.StatusForbidden)
}
```

- [ ] **Step 2: Add DTOs**

In `backend/internal/app/events.go`, add:

```go
type eventArchiveDTO struct {
    ID           string                 `json:"id"`
    EventID      string                 `json:"eventId"`
    ReportID     string                 `json:"reportId"`
    SettlementID string                 `json:"settlementId"`
    Status       string                 `json:"status"`
    NoteCount    int                    `json:"noteCount"`
    Notes        []eventArchiveNoteDTO  `json:"notes"`
    CreatedAt    string                 `json:"createdAt"`
    UpdatedAt    string                 `json:"updatedAt"`
}

type eventArchiveNoteDTO struct {
    ID                string `json:"id"`
    ArchiveID         string `json:"archiveId"`
    Body              string `json:"body"`
    CreatedByPersonID string `json:"createdByPersonId"`
    CreatedAt         string `json:"createdAt"`
}
```

- [ ] **Step 3: Register and implement GET route**

Add in `routes()`:

```go
a.mux.HandleFunc("GET /api/events/{eventID}/archive", a.handleGetArchive)
```

Implement `handleGetArchive` using the same auth/membership pattern as `handleGetReport` and `handleGetSettlement`: owner/member can read, unauth and non-members follow the current private-route behavior (`403`), and missing archive is `404`. Query `event_archives` by `event_id`; return `notes: []` for this task. `reportId` and `settlementId` are non-null strings because archive rows are only created after report and settlement rows exist.

- [ ] **Step 4: Add TypeScript DTOs**

In `web/src/domain.ts`, add:

```ts
export interface EventArchiveDTO {
  id: string;
  eventId: string;
  reportId: string;
  settlementId: string;
  status: 'private';
  noteCount: number;
  notes: EventArchiveNoteDTO[];
  createdAt: string;
  updatedAt: string;
}

export interface EventArchiveNoteDTO {
  id: string;
  archiveId: string;
  body: string;
  createdByPersonId: string;
  createdAt: string;
}
```

- [ ] **Step 5: Run verification and commit**

Run: `gofmt -w backend/internal/app/app.go backend/internal/app/events.go backend/internal/app/lifecycle_test.go && make verify`

Expected: PASS.

Commit:

```bash
git add backend/internal/app/app.go backend/internal/app/events.go backend/internal/app/lifecycle_test.go web/src/domain.ts
git commit -m "Add private archive API"
```

---

### Task 3: Lessons Learned Notes

**Files:**
- Modify: `backend/internal/app/schema.sql`
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/events.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`

- [ ] **Step 1: Add failing note coverage**

Add `TestFirstEventLifecycleArchiveNotes`:

```go
func TestFirstEventLifecycleArchiveNotes(t *testing.T) {
    fx := newLifecycleFixture(t)
    event := createEvent(t, fx, "Night Market", 4)
    eventID := mustString(t, event, "id")
    publishEvent(t, fx, eventID)
    postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)

    postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/archive/notes", map[string]any{"body": "Move doors earlier."}, http.StatusOK)
    resp := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/archive/notes", map[string]any{"body": "Keep card reader charged."}, http.StatusOK)
    archive := mustObject(t, resp.JSON)
    if int(archive["noteCount"].(float64)) != 2 {
        t.Fatalf("expected two notes, got %#v", archive)
    }
    notes := archive["notes"].([]any)
    if mustObject(t, notes[0])["body"] != "Move doors earlier." || mustObject(t, notes[1])["body"] != "Keep card reader charged." {
        t.Fatalf("notes not ordered by creation: %#v", notes)
    }

    reloaded := getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/archive", http.StatusOK)
    reloadedNotes := mustObject(t, reloaded.JSON)["notes"].([]any)
    if len(reloadedNotes) != 2 || mustObject(t, reloadedNotes[0])["body"] != "Move doors earlier." || mustObject(t, reloadedNotes[1])["body"] != "Keep card reader charged." {
        t.Fatalf("GET archive did not return persisted notes in order: %#v", reloadedNotes)
    }

    postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/archive/notes", map[string]any{"body": "   "}, http.StatusBadRequest)
    postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/archive/notes", map[string]any{"body": "member note"}, http.StatusForbidden)
}
```

- [ ] **Step 2: Add notes schema**

In `schema.sql`, after `event_archives`, add:

```sql
create table if not exists event_archive_notes (
  id uuid primary key default gen_random_uuid(),
  archive_id uuid not null references event_archives(id) on delete cascade,
  body text not null,
  created_by_person_id uuid not null references people(id),
  created_at timestamptz not null default now()
);
```

- [ ] **Step 3: Register POST route and request DTO**

Add route:

```go
a.mux.HandleFunc("POST /api/events/{eventID}/archive/notes", a.handleCreateArchiveNote)
```

Add request type:

```go
type createArchiveNoteRequest struct {
    Body string `json:"body"`
}
```

- [ ] **Step 4: Implement owner-only append**

Implement `handleCreateArchiveNote` with these rules:

1. Require authenticated workspace `owner` for the event.
2. Trim body with `strings.TrimSpace`.
3. Reject empty body with `400` and message `note body is required`.
4. Return `404` if archive does not exist.
5. Insert one row into `event_archive_notes`.
6. Increment `event_archives.note_count` and `updated_at` in the same transaction.
7. Audit `archive.note_created` with subject type `event_archive`.
8. Update `handleGetArchive` to load `event_archive_notes` ordered by `created_at asc, id asc` instead of always returning an empty notes array.
9. Return the full archive DTO including notes ordered by `created_at asc, id asc`.

- [ ] **Step 5: Run verification and commit**

Run: `gofmt -w backend/internal/app/app.go backend/internal/app/events.go backend/internal/app/lifecycle_test.go && make verify`

Expected: PASS.

Commit:

```bash
git add backend/internal/app/schema.sql backend/internal/app/app.go backend/internal/app/events.go backend/internal/app/lifecycle_test.go web/src/domain.ts
git commit -m "Add archive lessons learned notes"
```

---

### Task 4: Operator Archive UI

**Files:**
- Modify: `web/src/views/EventEditorView.tsx`
- Modify: `web/src/views/WorkspaceView.tsx`
- Modify: `web/src/App.test.tsx`

- [ ] **Step 1: Extend frontend test**

In `web/src/App.test.tsx`, extend the end-of-night/event-editor regression setup so mocked state includes an archive DTO:

```ts
const archive = {
  id: 'archive-1',
  eventId: event.id,
  reportId: 'report-1',
  settlementId: 'settlement-1',
  status: 'private',
  noteCount: 1,
  notes: [
    {
      id: 'note-1',
      archiveId: 'archive-1',
      body: 'Move doors earlier.',
      createdByPersonId: 'person-1',
      createdAt: '2026-06-14T03:10:00.000Z',
    },
  ],
  createdAt: '2026-06-14T03:00:00.000Z',
  updatedAt: '2026-06-14T03:10:00.000Z',
};
```

Assert rendered output contains `Private archive`, `Lessons learned`, `Move doors earlier.`, and `1 note`.

- [ ] **Step 2: Add EventEditor archive state**

In `EventEditorView.tsx`, add `EventArchiveDTO` state, a note form string, and loading/submitting flags. When `event.status === 'end_of_night'`, fetch `/api/events/${event.id}/archive`; suppress 404 using `ApiError` like settlement loading.

- [ ] **Step 3: Render archive panel**

Render a panel below settlement closeout:

- title `Private archive`
- status `private workspace memory`
- note count using singular/plural text
- list note bodies and timestamps
- owner-facing textarea and `Add lesson` button

- [ ] **Step 4: Add note submission**

On submit, POST to `/api/events/${event.id}/archive/notes` with `{ body: archiveNoteBody.trim() }`, replace archive state with response, and clear the textarea.

- [ ] **Step 5: Add workspace affordance**

In `WorkspaceView.tsx`, mark `end_of_night` events with copy `Archive ready after closeout` and link to the existing event editor route. Do not create a new route in this task.

- [ ] **Step 6: Run verification and commit**

Run: `make verify`

Expected: PASS.

Commit:

```bash
git add web/src/views/EventEditorView.tsx web/src/views/WorkspaceView.tsx web/src/App.test.tsx
git commit -m "Add private archive UI"
```

---

### Task 5: Seed Next Draft From Archive

**Files:**
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/events.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`
- Modify: `web/src/views/EventEditorView.tsx`
- Modify: `web/src/views/WorkspaceView.tsx`
- Modify: `web/src/App.test.tsx`

- [ ] **Step 1: Add failing seed safety test**

Add `TestFirstEventLifecycleArchiveSeedsDraftWithoutPrivateData`:

```go
func TestFirstEventLifecycleArchiveSeedsDraftWithoutPrivateData(t *testing.T) {
    fx := newLifecycleFixture(t)
    event := createEventWithPricing(t, fx, "Night Market", 40, "fixed", 1500, "usd")
    eventID := mustString(t, event, "id")
    publishEvent(t, fx, eventID)
    postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)
    postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/archive/notes", map[string]any{"body": "Private lesson"}, http.StatusOK)

    resp := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/archive/seed-draft", map[string]any{}, http.StatusOK)
    draft := mustObject(t, resp.JSON)
    draftID := draft["id"].(string)
    if draftID == eventID || draft["status"] != "draft" || draft["title"] != "Night Market" {
        t.Fatalf("unexpected seeded draft: %#v", draft)
    }
    if draft["publicUrl"] != nil {
        t.Fatalf("seeded draft must not have public URL: %#v", draft)
    }

    getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+draftID+"/archive", http.StatusNotFound)
    postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/archive/seed-draft", map[string]any{}, http.StatusForbidden)
}
```

- [ ] **Step 2: Register route**

Add:

```go
a.mux.HandleFunc("POST /api/events/{eventID}/archive/seed-draft", a.handleSeedDraftFromArchive)
```

- [ ] **Step 3: Implement owner-only seed**

Implement `handleSeedDraftFromArchive` with these rules:

1. Require owner membership for the source event workspace.
2. Require source event has an archive row; otherwise 404.
3. Insert a new `events` row in the same workspace with copied safe public planning fields only: `title`, `public_description`, `location_display`, `ticket_allocation`, `pricing_mode`, `ticket_price_cents`, `ticket_currency`.
4. Set `status='draft'`, `public_slug=null`, `published_at=null`, `ended_at=null`, and `created_by_person_id=actorID`.
5. Set `starts_at` to source `starts_at + interval '7 days'` as a harmless draft placeholder.
6. Do not copy tickets, report, settlement, archive, notes, audit entries, or public slug.
7. Audit `archive.seed_draft_created` with source event and new draft IDs.
8. Return the normal event DTO for the new draft.

- [ ] **Step 4: Add frontend action**

In `EventEditorView.tsx`, add a `Seed next draft` button in the archive panel. POST to the seed route and show a success message containing the new draft title. Keep navigation optional; do not add a router change.

- [ ] **Step 5: Add workspace copy**

In `WorkspaceView.tsx`, for closed events, show copy: `Use the private archive to seed the next draft from the event editor.`

- [ ] **Step 6: Run final verification and commit**

Run: `make verify`

Expected: PASS.

Commit:

```bash
git add backend/internal/app/app.go backend/internal/app/events.go backend/internal/app/lifecycle_test.go web/src/domain.ts web/src/views/EventEditorView.tsx web/src/views/WorkspaceView.tsx web/src/App.test.tsx
git commit -m "Seed event drafts from archives"
```

---

## Final Verification

- [ ] Run `make verify`.
- [ ] Run `git status --short --branch` and confirm the working tree is clean.
- [ ] Run `git log --oneline -8` and confirm the five archive-wave commits are present.

## Out of Scope

- Public archive pages.
- Search/indexing across archived events.
- Media uploads, credits, performers, vendors, or venue pages.
- Reopening finalized settlements.
- Copying private notes into new drafts.
- Publishing seeded drafts automatically.

## Self-Review

- Spec coverage: the plan covers archive root creation, private read API, lessons learned notes, operator UI, and safe draft seeding.
- Placeholder scan: no task depends on unspecified future work; each task has concrete routes, schema, tests, and commands.
- Type consistency: archive DTO and note DTO names are consistent across Go, TypeScript, API responses, and tests.
