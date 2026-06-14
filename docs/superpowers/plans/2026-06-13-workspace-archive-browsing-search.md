# Workspace Archive Browsing Search Implementation Plan

> **For agentic workers:** Execute this plan task-by-task. Recommended path:
> dispatch a fresh subagent per task, review each result with `review-quality`,
> then continue. For complex multi-agent splits, use
> `parallel-feature-development`, `team-composition-patterns`, and
> `team-communication-protocols`. Steps use checkbox (`- [ ]`) syntax for
> tracking.

**Goal:** Make private event archives discoverable as workspace memory with a workspace archive index, search-lite, clearer archive detail state, and an explicit learning loop from closed event to next draft.

**Architecture:** Keep archives private and workspace-scoped. Add a summary-only workspace archive index API backed by `event_archives` joined to `events`, then progressively expose that index in `WorkspaceView` with server-side search and links into the existing `EventEditorView` archive detail panel.

**Tech Stack:** Go `net/http` backend, pgx/Postgres embedded `schema.sql`, React TypeScript SPA, Vitest, `make verify`.

---

## Current State

- Event detail already supports `GET /api/events/{eventID}/archive`.
- Owners can append archive lessons and seed one idempotent next draft.
- `event_archives.seeded_event_id` exists and is additively migrated.
- Workspace view currently lists events and has a small closed-event archive CTA, but no archive index or search.
- Public archives, SEO, media, performer/vendor credits, and cross-workspace discovery remain out of scope.

## File Structure

- `backend/internal/app/app.go`: register workspace archive list route.
- `backend/internal/app/events.go`: keep event archive detail helpers; expose `seededEventId` in archive DTO.
- `backend/internal/app/workspaces.go`: implement workspace-scoped archive index handler near existing workspace/event list handlers.
- `backend/internal/app/lifecycle_test.go`: add DB-backed API tests for archive index, search, permissions, and seeded draft fields.
- `web/src/domain.ts`: add archive summary DTO and `seededEventId` to archive detail DTO.
- `web/src/views/WorkspaceView.tsx`: add archive index state, cards, search input, empty states, and loop guidance. Existing frontend tests mock `React.useState` positionally, so each UI task must update the affected state arrays or extract pure rendering helpers before adding many hooks.
- `web/src/views/EventEditorView.tsx`: polish archive detail with seeded-draft metadata and workspace archive back-link.
- `web/src/App.test.tsx`: add frontend smoke coverage for archive index/search/detail loop copy.

---

### Task 1: Workspace Archive Index API

**Files:**
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/workspaces.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`
- Modify: `docs/superpowers/plans/2026-06-13-workspace-archive-browsing-search.md`

- [x] **Step 1: Write failing API test**

Add `TestWorkspaceArchiveIndexAPI` to `backend/internal/app/lifecycle_test.go`:

```go
func TestWorkspaceArchiveIndexAPI(t *testing.T) {
    fx := newLifecycleFixture(t)
    event := createEvent(t, fx, "Night Market", 4)
    eventID := mustString(t, event, "id")
    publishEvent(t, fx, eventID)
    postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)
    postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/archive/notes", map[string]any{"body": "Move doors earlier."}, http.StatusOK)

    resp := getJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/archives", http.StatusOK)
    archives := resp.JSON.([]any)
    if len(archives) != 1 {
        t.Fatalf("expected one archive, got %#v", resp.JSON)
    }
    archive := mustObject(t, archives[0])
    if archive["eventId"] != eventID || archive["title"] != "Night Market" || int(archive["noteCount"].(float64)) != 1 {
        t.Fatalf("unexpected archive summary: %#v", archive)
    }
    if archive["id"] == "" || archive["reportId"] == "" || archive["settlementId"] == "" || archive["createdAt"] == "" || archive["updatedAt"] == "" {
        t.Fatalf("missing archive refs/timestamps: %#v", archive)
    }

    getJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/archives", http.StatusOK)
    getJSON(t, fx.app, nil, "/api/workspaces/"+fx.workspaceID+"/archives", http.StatusForbidden)
    other := newLifecycleFixture(t)
    getJSON(t, fx.app, other.memberCookie, "/api/workspaces/"+fx.workspaceID+"/archives", http.StatusForbidden)
}
```

- [x] **Step 2: Register route**

In `backend/internal/app/app.go`, add:

```go
a.mux.HandleFunc("GET /api/workspaces/{workspaceID}/archives", a.handleListWorkspaceArchives)
```

- [x] **Step 3: Add Go DTO**

In `backend/internal/app/workspaces.go` or `events.go`, add:

```go
type workspaceArchiveSummaryDTO struct {
    ID           string  `json:"id"`
    EventID      string  `json:"eventId"`
    Title        string  `json:"title"`
    StartsAt     string  `json:"startsAt"`
    LocationDisplay string `json:"locationDisplay"`
    NoteCount    int     `json:"noteCount"`
    ReportID     string  `json:"reportId"`
    SettlementID string  `json:"settlementId"`
    SeededEventID *string `json:"seededEventId,omitempty"`
    CreatedAt    string  `json:"createdAt"`
    UpdatedAt    string  `json:"updatedAt"`
}
```

- [x] **Step 4: Implement list handler**

Use `requireWorkspaceRole(r, workspaceID, "owner", "member")`. Query `event_archives` joined to `events` for the workspace, ordered newest `events.starts_at desc, event_archives.created_at desc`, with `limit 100`. Scan nullable `seeded_event_id` through `sql.NullString` and assign `SeededEventID: nullableString(seededEventID)`. Do not include note bodies in this summary route.

- [x] **Step 5: Add TypeScript DTO**

In `web/src/domain.ts`, add:

```ts
export interface WorkspaceArchiveSummaryDTO {
  id: string;
  eventId: string;
  title: string;
  startsAt: string;
  locationDisplay: string;
  noteCount: number;
  reportId: string;
  settlementId: string;
  seededEventId?: string;
  createdAt: string;
  updatedAt: string;
}
```

- [ ] **Step 6: Verify and commit**

Run: `gofmt -w backend/internal/app/app.go backend/internal/app/workspaces.go backend/internal/app/lifecycle_test.go && make verify`

Commit:

```bash
git add backend/internal/app/app.go backend/internal/app/workspaces.go backend/internal/app/lifecycle_test.go web/src/domain.ts docs/superpowers/plans/2026-06-13-workspace-archive-browsing-search.md
git commit -m "Add workspace archive index API"
```

---

### Task 2: Workspace Archive Section

**Files:**
- Modify: `web/src/views/WorkspaceView.tsx`
- Modify: `web/src/App.test.tsx`

- [ ] **Step 1: Add failing frontend smoke test**

In `web/src/App.test.tsx`, add a workspace rendering test that mocks archive summaries and asserts `Workspace archive`, `Night Market`, `1 note`, and `Open archive` are visible.

- [ ] **Step 2: Load archives with workspace events**

In `WorkspaceView.tsx`, add `archives` state of type `WorkspaceArchiveSummaryDTO[]`. When a workspace is loaded, fetch `/api/workspaces/${workspace.id}/archives` alongside `/events`. Update existing positional `useState` test fixtures in `web/src/App.test.tsx` to account for new state, or extract an `ArchiveSection` pure component and test it directly.

- [ ] **Step 3: Render archive section**

Below the existing active event list, render a `Workspace archive` section with cards showing title, starts date, location, note count, seeded draft status, and an `Open archive` link to `/events/{eventId}`.

- [ ] **Step 4: Preserve owner-only seed action**

Keep `Seed next draft` visible only for `workspace.role === 'owner'`. If `seededEventId` exists, show `Open seeded draft` instead of seed button. After a successful seed POST, either refetch `/api/workspaces/${workspace.id}/archives` or patch the local archive summary with `seededEventId: seeded.id` so double-click/retry state is reflected immediately.

- [ ] **Step 5: Verify and commit**

Run: `make verify`

Commit:

```bash
git add web/src/views/WorkspaceView.tsx web/src/App.test.tsx
git commit -m "Add workspace archive section"
```

---

### Task 3: Archive Search-Lite

**Files:**
- Modify: `backend/internal/app/workspaces.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/views/WorkspaceView.tsx`
- Modify: `web/src/App.test.tsx`

- [ ] **Step 1: Add failing backend search test**

Extend `TestWorkspaceArchiveIndexAPI` or add `TestWorkspaceArchiveIndexSearch` to cover:

- `?q=night` matches event title.
- `?q=doors` matches archive note body.
- `?q=%` and `?q=_` do not behave as unescaped wildcards.
- A note-body match does not include note bodies in the summary response.
- `?q=missing` returns empty array.
- Search remains workspace-scoped.

- [ ] **Step 2: Implement server-side `q`**

Normalize with `strings.TrimSpace(strings.ToLower(r.URL.Query().Get("q")))`; reject queries longer than 120 characters with `400` and `search query is too long`. Empty query returns latest archives. Non-empty query matches `lower(events.title)`, `lower(events.public_description)`, `lower(events.location_display)`, or an `exists` subquery over `event_archive_notes.body` using privacy-safe substring matching: `position($query in lower(column)) > 0`. Do not use raw `LIKE '%' || $query || '%'` because `%` and `_` become wildcards. Keep `limit 100`.

- [ ] **Step 3: Add frontend search input**

In `WorkspaceView.tsx`, add `archiveQuery`, `archiveSearching`, and a small form in the archive section. Submit reloads archives from `/api/workspaces/${workspace.id}/archives?q=${encodeURIComponent(archiveQuery)}`. Add reset button that clears query and reloads unfiltered archives.

- [ ] **Step 4: Add frontend regression assertions**

Extend App test coverage to assert the archive search input placeholder, search button, and empty-search copy render.

- [ ] **Step 5: Verify and commit**

Run: `make verify`

Commit:

```bash
git add backend/internal/app/workspaces.go backend/internal/app/lifecycle_test.go web/src/views/WorkspaceView.tsx web/src/App.test.tsx
git commit -m "Add private archive search"
```

---

### Task 4: Archive Detail Polish

**Files:**
- Modify: `backend/internal/app/events.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`
- Modify: `web/src/views/EventEditorView.tsx`
- Modify: `web/src/App.test.tsx`

- [ ] **Step 1: Expose `seededEventId` in archive detail DTO**

Add `SeededEventID *string json:"seededEventId,omitempty"` to `eventArchiveDTO`, load `seeded_event_id` with `sql.NullString`, assign through existing `nullableString`, and add `seededEventId?: string` to `EventArchiveDTO` in TypeScript.

- [ ] **Step 2: Add backend assertion**

After seed-draft API test creates a draft, GET `/api/events/{sourceEventID}/archive` and assert `seededEventId` equals the seeded draft ID.

- [ ] **Step 3: Polish archive panel**

In `EventEditorView.tsx`, show archive created/updated timestamps, report/settlement refs as small metadata, a `Back to workspace archive` link using `/workspace?workspaceId=${event.workspaceId}`, and if `seededEventId` exists show `Open seeded draft` instead of `Seed next draft`. After a successful seed POST, either refetch `/api/events/${event.id}/archive` or patch local archive state with `seededEventId: seeded.id` so the detail panel changes immediately.

- [ ] **Step 4: Verify and commit**

Run: `make verify`

Commit:

```bash
git add backend/internal/app/events.go backend/internal/app/lifecycle_test.go web/src/domain.ts web/src/views/EventEditorView.tsx web/src/App.test.tsx
git commit -m "Polish archive detail state"
```

---

### Task 5: Operator Learning Loop

**Files:**
- Modify: `web/src/views/WorkspaceView.tsx`
- Modify: `web/src/views/EventEditorView.tsx`
- Modify: `web/src/App.test.tsx`
- Modify: `README.md` if the user-facing workflow docs need one short note

- [ ] **Step 1: Add loop guidance copy**

In `WorkspaceView.tsx`, add a small guidance panel near the archive section:

- If no archives: `Closed events will become private workspace memory here.`
- If archives with no notes: `Open an archive and capture the first lesson.`
- If archives with notes and no seeded draft: `Use lessons to seed the next draft.`
- If seeded draft exists: `Review the seeded draft before publishing.`

- [ ] **Step 2: Add detail nudges**

In `EventEditorView.tsx`, when an archive has zero notes, show `Capture one lesson before seeding the next draft.` When it has notes, show `Use these notes while planning the next event.` Do not copy private notes into draft fields.

- [ ] **Step 3: Add frontend tests**

Update `web/src/App.test.tsx` to assert at least one guidance state and the no-copy privacy copy.

- [ ] **Step 4: Final verification and commit**

Run: `make verify`

Commit:

```bash
git add web/src/views/WorkspaceView.tsx web/src/views/EventEditorView.tsx web/src/App.test.tsx README.md
git commit -m "Clarify archive learning loop"
```

---

## Final Verification

- [ ] Run `make verify`.
- [ ] Run `git status --short --branch` and confirm the working tree is clean.
- [ ] Run `git log --oneline -8` and confirm the five archive browsing commits are present.

## Out of Scope

- Public archive pages or public search.
- Full-text search, trigram indexes, pagination, or ranking.
- Cross-workspace search.
- Copying private lesson notes into event drafts.
- Media, credits, performer/vendor profiles, or venue pages.

## Self-Review

- Spec coverage: the plan covers archive index, workspace archive UI, search-lite, archive detail polish, and operator learning loop.
- Placeholder scan: each task has concrete routes, files, commands, and expected behavior.
- Type consistency: `WorkspaceArchiveSummaryDTO`, `EventArchiveDTO.seededEventId`, route paths, and permission semantics are consistent across backend, frontend, and tests.
