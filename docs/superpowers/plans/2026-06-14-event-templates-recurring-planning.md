# Event Templates Recurring Planning Implementation Plan

> **For agentic workers:** Execute this plan task-by-task. Recommended path:
> dispatch a fresh subagent per task, review each result with `review-quality`,
> then continue. For complex multi-agent splits, use
> `parallel-feature-development`, `team-composition-patterns`, and
> `team-communication-protocols`. Steps use checkbox (`- [ ]`) syntax for
> tracking.

**Goal:** Let organizers save repeatable event planning templates and apply them to draft events without exposing private planning memory publicly.

**Architecture:** Add workspace-scoped `event_templates` as private planning records. Templates copy only explicit event-planning fields plus optional private notes for organizers; they do not copy tickets, applications, participants, staffing assignments, settlements, archives, notifications, contacts, or commitments. Owners can mutate templates; owners/members can read and apply to draft events.

**Tech Stack:** Go `net/http`, pgx/Postgres embedded `schema.sql`, React TypeScript SPA, Vitest, `make verify`.

---

## File Structure

- `backend/internal/app/schema.sql`: add `event_templates`.
- `backend/internal/app/app.go`: register template routes.
- `backend/internal/app/event_templates.go`: template DTOs, list/create/update/delete/apply handlers.
- `backend/internal/app/events.go`: optionally add `templateId`/`templateName` to event DTO after applying template.
- `backend/internal/app/lifecycle_test.go`: backend API, permission, apply, and privacy tests.
- `web/src/domain.ts`: template DTO and request types.
- `web/src/views/WorkspaceView.tsx`: workspace template list/create/edit controls.
- `web/src/views/EventEditorView.tsx`: apply-template controls for draft events and save-template-from-event action.
- `web/src/App.test.tsx`: frontend smoke/privacy tests.
- `README.md`, `docs/runbooks/deployment-checklist.md`: private template boundary.

---

### Task 1: Template Read Model

**Files:**
- Modify: `backend/internal/app/schema.sql`
- Modify: `backend/internal/app/app.go`
- Create: `backend/internal/app/event_templates.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`
- Modify: `docs/superpowers/plans/2026-06-14-event-templates-recurring-planning.md`

- [x] **Step 1: Add failing list coverage**

Add `TestEventTemplatesListAPI` to `backend/internal/app/lifecycle_test.go`:

```go
func TestEventTemplatesListAPI(t *testing.T) {
    fx := newLifecycleFixture(t)

    empty := getJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/event-templates", http.StatusOK)
    if got := empty.JSON.([]any); len(got) != 0 {
        t.Fatalf("expected empty templates, got %#v", got)
    }

    var templateID string
    if err := fx.app.db.QueryRow(t.Context(), `
        insert into event_templates (
            workspace_id, name, title, public_description, location_display,
            ticket_allocation, pricing_mode, ticket_price_cents, ticket_currency,
            private_notes, created_by_person_id
        ) values ($1, 'Monthly Market', 'Night Market', 'Public copy', 'The Hall', 40, 'fixed', 1500, 'usd', 'Private run-of-show', $2)
        returning id
    `, fx.workspaceID, ownerPersonID(t, fx)).Scan(&templateID); err != nil {
        t.Fatal(err)
    }

    resp := getJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/event-templates", http.StatusOK)
    templates := resp.JSON.([]any)
    if len(templates) != 1 {
        t.Fatalf("expected one template, got %#v", templates)
    }
    tmpl := mustObject(t, templates[0])
    if tmpl["id"] != templateID || tmpl["name"] != "Monthly Market" || tmpl["title"] != "Night Market" || tmpl["privateNotes"] != "Private run-of-show" {
        t.Fatalf("unexpected template: %#v", tmpl)
    }

    otherFx := newLifecycleFixture(t)
    getJSON(t, fx.app, nil, "/api/workspaces/"+fx.workspaceID+"/event-templates", http.StatusForbidden)
    getJSON(t, fx.app, otherFx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/event-templates", http.StatusForbidden)
}
```

- [x] **Step 2: Add schema**

Append after `events` or near planning tables:

```sql
create table if not exists event_templates (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references workspaces(id) on delete cascade,
  name text not null,
  title text not null,
  public_description text not null default '',
  location_display text not null default '',
  ticket_allocation integer not null default 0 check (ticket_allocation >= 0),
  pricing_mode text not null default 'free' check (pricing_mode in ('free', 'fixed')),
  ticket_price_cents integer not null default 0 check (ticket_price_cents >= 0),
  ticket_currency text not null default 'usd',
  private_notes text not null default '',
  created_by_person_id uuid not null references people(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  check (trim(name) <> ''),
  check (trim(title) <> '')
);

create index if not exists event_templates_workspace_name_idx on event_templates (workspace_id, lower(name));
```

- [x] **Step 3: Register list route and DTO**

In `app.go`:

```go
a.mux.HandleFunc("GET /api/workspaces/{workspaceID}/event-templates", a.handleListEventTemplates)
```

Create `event_templates.go` with `eventTemplateDTO` fields: `id`, `workspaceId`, `name`, `title`, `publicDescription`, `locationDisplay`, `ticketAllocation`, `pricingMode`, `ticketPriceCents`, `ticketCurrency`, `privateNotes`, `createdAt`, `updatedAt`. Implement `handleListEventTemplates` for workspace owner/member reads, ordered by `lower(name), created_at`.

- [x] **Step 4: Add TS DTO**

In `web/src/domain.ts` add `EventTemplateDTO` matching JSON camelCase.

- [ ] **Step 5: Verify and commit**

Run: `gofmt -w backend/internal/app/app.go backend/internal/app/event_templates.go backend/internal/app/lifecycle_test.go && make verify`

Commit:

```bash
git add backend/internal/app/schema.sql backend/internal/app/app.go backend/internal/app/event_templates.go backend/internal/app/lifecycle_test.go web/src/domain.ts docs/superpowers/plans/2026-06-14-event-templates-recurring-planning.md
git commit -m "Add event template read model"
```

---

### Task 2: Template Mutations

**Files:**
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/event_templates.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`

- [ ] **Step 1: Add failing mutation coverage**

Add `TestEventTemplatesMutationAPI`:

```go
func TestEventTemplatesMutationAPI(t *testing.T) {
    fx := newLifecycleFixture(t)
    created := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/event-templates", map[string]any{
        "name": "  Monthly Market  ",
        "title": "  Night Market  ",
        "publicDescription": "  Public copy  ",
        "locationDisplay": "  The Hall  ",
        "ticketAllocation": 40,
        "pricingMode": "fixed",
        "ticketPriceCents": 1500,
        "ticketCurrency": " USD ",
        "privateNotes": "Private setup note",
    }, http.StatusOK)
    tmpl := mustObject(t, created.JSON)
    if tmpl["name"] != "Monthly Market" || tmpl["title"] != "Night Market" || tmpl["ticketCurrency"] != "usd" || tmpl["privateNotes"] != "Private setup note" {
        t.Fatalf("unexpected created template: %#v", tmpl)
    }
    templateID := tmpl["id"].(string)

    updated := patchJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/event-templates/"+templateID, map[string]any{
        "name": "Market v2",
        "pricingMode": "free",
        "ticketPriceCents": 0,
        "privateNotes": "Still private",
    }, http.StatusOK)
    if mustObject(t, updated.JSON)["name"] != "Market v2" || mustObject(t, updated.JSON)["pricingMode"] != "free" {
        t.Fatalf("unexpected updated template: %#v", updated.JSON)
    }

    postJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/event-templates", map[string]any{"name": "member", "title": "member"}, http.StatusForbidden)
    patchJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/event-templates/"+templateID, map[string]any{"name": "member"}, http.StatusForbidden)
    postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/event-templates", map[string]any{"name": "", "title": "x"}, http.StatusBadRequest)
    postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/event-templates", map[string]any{"name": "bad", "title": "bad", "pricingMode": "fixed", "ticketPriceCents": 49}, http.StatusBadRequest)

    rawAudit := auditMetadataForAction(t, fx.app.db, "event_template.created") + auditMetadataForAction(t, fx.app.db, "event_template.updated")
    if strings.Contains(rawAudit, "Private setup note") || strings.Contains(rawAudit, "Still private") || strings.Contains(rawAudit, "Night Market") {
        t.Fatalf("audit metadata leaked template content: %s", rawAudit)
    }

    doJSON(t, http.MethodDelete, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/event-templates/"+templateID, nil, http.StatusNoContent)
    getJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/event-templates", http.StatusOK)
}
```

- [ ] **Step 2: Register mutation routes**

```go
a.mux.HandleFunc("POST /api/workspaces/{workspaceID}/event-templates", a.handleCreateEventTemplate)
a.mux.HandleFunc("PATCH /api/workspaces/{workspaceID}/event-templates/{templateID}", a.handleUpdateEventTemplate)
a.mux.HandleFunc("DELETE /api/workspaces/{workspaceID}/event-templates/{templateID}", a.handleDeleteEventTemplate)
```

- [ ] **Step 3: Implement request types and validation**

Create request requires `name`, `title`; update uses pointer fields for partial update. Rules: owner-only; trim all string fields; name/title required when supplied; publicDescription/location/privateNotes max 2000 runes; ticketAllocation >= 0; pricingMode `free|fixed`; fixed price must be >= 50 cents; free price forced/validated as 0; currency lowercased and defaults to `usd`.

- [ ] **Step 4: Implement mutations and audit**

Create/update/delete only affect rows in same workspace. Audit `event_template.created`, `event_template.updated`, `event_template.deleted` with `{templateId, workspaceId}` only. Never audit private notes or public copy.

- [ ] **Step 5: Add TS request types, verify, commit**

Add `CreateEventTemplateRequestDTO` and `UpdateEventTemplateRequestDTO`. Run `make verify` and commit:

```bash
git add backend/internal/app/app.go backend/internal/app/event_templates.go backend/internal/app/lifecycle_test.go web/src/domain.ts
git commit -m "Add event template mutations"
```

---

### Task 3: Apply Template to Draft

**Files:**
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/event_templates.go`
- Modify: `backend/internal/app/events.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`

- [ ] **Step 1: Add failing apply coverage**

Add `TestEventTemplateApplyToDraftAPI`:

```go
func TestEventTemplateApplyToDraftAPI(t *testing.T) {
    fx := newLifecycleFixture(t)
    tmpl := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/event-templates", map[string]any{
        "name": "Monthly Market",
        "title": "Template Title",
        "publicDescription": "Template public copy",
        "locationDisplay": "Template Hall",
        "ticketAllocation": 55,
        "pricingMode": "fixed",
        "ticketPriceCents": 1500,
        "ticketCurrency": "usd",
        "privateNotes": "Do not copy publicly",
    }, http.StatusOK)
    templateID := mustString(t, tmpl.JSON, "id")

    draft := createEvent(t, fx, "Old Title", 5)
    eventID := mustString(t, draft, "id")
    applied := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/apply-template", map[string]any{"templateId": templateID}, http.StatusOK)
    event := mustObject(t, applied.JSON)
    if event["title"] != "Template Title" || event["publicDescription"] != "Template public copy" || event["locationDisplay"] != "Template Hall" || int(event["ticketAllocation"].(float64)) != 55 || event["pricingMode"] != "fixed" || int(event["ticketPriceCents"].(float64)) != 1500 {
        t.Fatalf("template was not applied to draft: %#v", event)
    }
    if raw, _ := json.Marshal(event); strings.Contains(string(raw), "Do not copy publicly") {
        t.Fatalf("private template notes leaked into event DTO: %s", raw)
    }

    postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/apply-template", map[string]any{"templateId": templateID}, http.StatusOK)
    published := publishEvent(t, fx, eventID)
    postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+mustString(t, published, "id")+"/apply-template", map[string]any{"templateId": templateID}, http.StatusConflict)
    postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/apply-template", map[string]any{"templateId": templateID}, http.StatusForbidden)

    otherFx := newLifecycleFixture(t)
    otherTemplate := postJSON(t, otherFx.app, otherFx.ownerCookie, "/api/workspaces/"+otherFx.workspaceID+"/event-templates", map[string]any{"name": "Other", "title": "Other"}, http.StatusOK)
    otherDraft := createEvent(t, fx, "Still Mine", 5)
    postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+mustString(t, otherDraft, "id")+"/apply-template", map[string]any{"templateId": mustString(t, otherTemplate.JSON, "id")}, http.StatusNotFound)
}
```

- [ ] **Step 2: Register apply route**

```go
a.mux.HandleFunc("POST /api/events/{eventID}/apply-template", a.handleApplyEventTemplate)
```

- [ ] **Step 3: Implement apply semantics**

Request: `{ templateId: string }`. Owner-only for the event workspace. Begin tx, lock event row `FOR UPDATE`, reject non-draft events with `409`, load template from same workspace or `404`, and compute reserved ticket count inside the transaction using `payment_status <> 'cancelled'`. Preserve existing event safety invariants from `handleUpdateEvent`: if reserved count is greater than zero, applying a template must not change `pricing_mode`, `ticket_price_cents`, or `ticket_currency`; template `ticket_allocation` must not be below reserved count. Update only these event fields: title, public_description, location_display, ticket_allocation, pricing_mode, ticket_price_cents, ticket_currency, updated_at. Do not create tickets, roles, staffing, contacts, commitments, archive rows, notifications, or private notes. Audit `event_template.applied` with `{templateId, eventId, workspaceId}` only. Commit before reloading with pool-backed helpers, or return the updated row from the transaction; do not call a pool query that can block on the locked event row before commit. Return normal `EventDTO`.

Add a test assertion that a draft with an existing non-cancelled ticket cannot apply a template that changes pricing or lowers allocation below reserved count. Use direct ticket insertion helpers if needed; this protects against template apply bypassing normal event update safety.

- [ ] **Step 4: Add optional event template metadata only if cheap**

If adding `templateId/templateName` to `eventDTO` complicates list/detail queries, skip it for this wave. The acceptance criterion is the applied draft content and audit, not persistent template linkage.

- [ ] **Step 5: Verify and commit**

Run `make verify` and commit:

```bash
git add backend/internal/app/app.go backend/internal/app/event_templates.go backend/internal/app/events.go backend/internal/app/lifecycle_test.go web/src/domain.ts
git commit -m "Apply templates to draft events"
```

---

### Task 4: Template UI

**Files:**
- Modify: `web/src/api.ts`
- Modify: `web/src/views/WorkspaceView.tsx`
- Modify: `web/src/views/EventEditorView.tsx`
- Modify: `web/src/App.test.tsx`

- [ ] **Step 1: Add frontend test coverage**

Extend `App.test.tsx` to assert:

- Workspace view renders `Event templates` panel and private notes for owner/member.
- Owner sees create/edit/delete controls; member does not.
- Event editor draft view renders `Apply template` control.
- Published/closed event editor does not show apply-template control.

- [ ] **Step 2: Workspace template panel**

Add `deleteJSON<T = unknown>(path: string)` to `web/src/api.ts` or call `api(path, { method: 'DELETE' })` directly. Load `/api/workspaces/${workspace.id}/event-templates` with workspace private data. Reset template state when workspace changes/creates, just like contacts/commitments. Render template cards with name, title, location, pricing, allocation, private notes. Owner-only create/edit/delete form.

- [ ] **Step 3: Event editor apply control**

Load current workspace templates in `EventEditorView` when editing a draft event. Owner-only apply form posts to `/api/events/${event.id}/apply-template`, updates `effective` and form state from returned event, and shows success. Do not render/apply for new unsaved, published, or closed events.

- [ ] **Step 4: Save-template-from-event action**

For owner on any saved event, add `Save as template` action that POSTs template create using the event's current planning fields. Default name can be `${event.title} template`. Private notes start empty; do not copy archive notes/staffing/application/contact/commitment data.

- [ ] **Step 5: Verify and commit**

Run `make verify` and commit:

```bash
git add web/src/api.ts web/src/views/WorkspaceView.tsx web/src/views/EventEditorView.tsx web/src/App.test.tsx
git commit -m "Add event template UI"
```

---

### Task 5: Privacy Hardening and Docs

**Files:**
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/App.test.tsx`
- Modify: `README.md`
- Modify: `docs/runbooks/deployment-checklist.md`

- [ ] **Step 1: Backend privacy tests**

Seed a template with sentinel private notes and assert public discovery, public event, ticket lookup, report, settlement, archive, notification list, contacts, and commitments do not expose `private_notes` unless calling template routes.

- [ ] **Step 2: Frontend privacy tests**

Assert `/discover`, public event, and ticket render paths do not show template private notes. Assert private workspace/editor template panels do show them.

- [ ] **Step 3: Docs**

README shipped-slice copy: `Event templates let organizers repeat event planning fields privately; template notes are workspace-only and never copied to public event pages.` Deployment checklist: `Review template private notes and copied event fields before production; templates must not copy applications, staffing notes, contacts, commitments, settlement, or archive notes.`

- [ ] **Step 4: Verify and commit**

Run `make verify` and commit:

```bash
git add backend/internal/app/lifecycle_test.go web/src/App.test.tsx README.md docs/runbooks/deployment-checklist.md
git commit -m "Document event template privacy boundaries"
```

---

## Out of Scope

- Recurring calendar series tables.
- Automatic scheduled event creation.
- Copying role definitions, staffing items, contacts, commitments, applications, tickets, archive notes, or settlement data from templates.
- Public template marketplace.
- Template sharing across workspaces.

## Final Verification

- [ ] Run `make verify`.
- [ ] Run `git status --short --branch` and confirm clean.
- [ ] Run `git log --oneline -10` and confirm five template commits.

## Self-Review

- Spec coverage: read model, mutations, apply-to-draft, UI, privacy/docs are all covered.
- Placeholder scan: no TBDs; each task has exact routes/files/commands.
- Type consistency: `EventTemplateDTO`, request DTOs, route names, and audit action names are consistent.
