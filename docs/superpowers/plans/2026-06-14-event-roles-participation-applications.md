# Event Roles Participation Applications Implementation Plan

> **For agentic workers:** Execute this plan task-by-task. Recommended path:
> dispatch a fresh subagent per task, review each result with `review-quality`,
> then continue. For complex multi-agent splits, use
> `parallel-feature-development`, `team-composition-patterns`, and
> `team-communication-protocols`. Steps use checkbox (`- [ ]`) syntax for
> tracking.

**Goal:** Let organizers define event-specific participation roles, collect public applications, review them privately, and carry accepted participant memory into the archive.

**Architecture:** Add first-class `event_roles` and `event_role_applications` tables without changing ticketing or workspace membership. Public pages expose only published role prompts and public application intake; private owner/member routes expose applications and accepted roster. Archive memory stores accepted participant summaries at closeout without publishing private answers.

**Tech Stack:** Go `net/http` backend, pgx/Postgres embedded `schema.sql`, React TypeScript SPA, Vitest, `make verify`.

---

## Current State

- Events support draft/published/end-of-night, paid/free tickets, settlement closeout, private archives, archive search, and seeded drafts.
- There are no persisted event roles or participation applications.
- Public event pages already accept ticket reservations; private event editor already manages event lifecycle/settlement/archive state.

## File Structure

- `backend/internal/app/schema.sql`: add role/application/archive participant tables and additive migrations.
- `backend/internal/app/app.go`: register event role/application routes.
- `backend/internal/app/events.go`: keep event lifecycle/archive closeout integration; add archive participant snapshot helper if needed.
- `backend/internal/app/event_roles.go`: new handlers and DTOs for roles/applications/review/roster.
- `backend/internal/app/lifecycle_test.go`: DB-backed integration tests for role setup, public application intake, review, roster, and archive snapshot.
- `web/src/domain.ts`: role/application/roster/archive participant DTOs.
- `web/src/views/EventEditorView.tsx`: role setup, application review, roster, archive participant memory.
- `web/src/views/PublicEventView.tsx`: public role application form.
- `web/src/views/WorkspaceView.tsx`: light participant/application status copy if useful.
- `web/src/App.test.tsx`: frontend smoke/regression assertions.

---

### Task 1: Event Role Definitions

**Files:**
- Create: `backend/internal/app/event_roles.go`
- Modify: `backend/internal/app/schema.sql`
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`
- Modify: `docs/superpowers/plans/2026-06-14-event-roles-participation-applications.md`

- [x] **Step 1: Add backend test `TestEventRoleDefinitionsAPI`**

Create an event as owner. Assert `GET /api/events/{eventID}/roles` returns `[]`. Assert owner can `POST /api/events/{eventID}/roles` with `{ name: "Performer", description: "Play a 20-minute set.", capacity: 3, public: true }`. Assert response contains id, eventId, name, description, capacity, public, active, createdAt, updatedAt. Assert member can read but cannot create. Assert public-role list on unpublished event returns 404/empty according to public event conventions; published event public role endpoint returns only public+active roles.

- [x] **Step 2: Add schema**

Add `event_roles`:

```sql
create table if not exists event_roles (
  id uuid primary key default gen_random_uuid(),
  event_id uuid not null references events(id) on delete cascade,
  name text not null,
  description text not null default '',
  capacity integer not null default 0 check (capacity >= 0),
  public boolean not null default true,
  active boolean not null default true,
  created_by_person_id uuid not null references people(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
```

- [x] **Step 3: Register routes**

Add:

```go
a.mux.HandleFunc("GET /api/events/{eventID}/roles", a.handleListEventRoles)
a.mux.HandleFunc("POST /api/events/{eventID}/roles", a.handleCreateEventRole)
a.mux.HandleFunc("GET /api/public/events/{slug}/roles", a.handleListPublicEventRoles)
```

- [x] **Step 4: Implement DTOs/handlers**

Use owner/member private read; owner-only create. Validate trimmed name non-empty, capacity >= 0, description trimmed. Public route loads published event by slug and returns only `public=true and active=true` roles ordered by created_at asc.

- [x] **Step 5: Add TypeScript DTO**

Add `EventRoleDTO` with id, eventId, name, description, capacity, public, active, createdAt, updatedAt.

- [x] **Step 6: Verify and commit**

Run: `gofmt -w backend/internal/app/app.go backend/internal/app/event_roles.go backend/internal/app/lifecycle_test.go && make verify`

Commit: `git commit -m "Add event role definitions"`

---

### Task 2: Public Role Applications

**Files:**
- Modify: `backend/internal/app/schema.sql`
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/event_roles.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`
- Modify: `web/src/views/PublicEventView.tsx`
- Modify: `web/src/App.test.tsx`

- [ ] **Step 1: Add backend test `TestPublicRoleApplicationsAPI`**

Create/publish event and public role. POST to `/api/public/events/{slug}/role-applications` with `{ roleId, applicantName, applicantEmail, message }`. Assert status `submitted`, trimmed fields, role/event IDs, timestamps. Assert unpublished events, inactive/private roles, role-event mismatch, invalid email, empty name, and >2000-char message are rejected. Assert audit action `role_application.submitted`.

- [ ] **Step 2: Add schema**

Add `event_role_applications`:

```sql
create table if not exists event_role_applications (
  id uuid primary key default gen_random_uuid(),
  event_id uuid not null references events(id) on delete cascade,
  role_id uuid not null references event_roles(id) on delete cascade,
  applicant_name text not null,
  applicant_email text not null,
  message text not null default '',
  status text not null default 'submitted' check (status in ('submitted','under_review','accepted','waitlisted','rejected','withdrawn','confirmed')),
  reviewed_by_person_id uuid references people(id),
  reviewed_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create unique index if not exists event_role_applications_active_email_idx
  on event_role_applications (event_id, role_id, lower(applicant_email))
  where status in ('submitted','under_review','accepted','waitlisted','confirmed');
```

- [ ] **Step 3: Register public submit route**

Add:

```go
a.mux.HandleFunc("POST /api/public/events/{slug}/role-applications", a.handleSubmitPublicRoleApplication)
```

- [ ] **Step 4: Implement validation and insert**

Accept only published event slug + public active role for that event. Validate applicant name/email, message <= 2000 runes. Insert submitted application and audit without requiring login. Reject duplicate active applications for the same event/role/email with `409 application already submitted`; withdrawn/rejected applications do not block later resubmission. Public submission audit metadata must exclude applicant name, email, and message; include only `eventId`, `roleId`, and `applicationId`.

- [ ] **Step 5: Add public UI**

In `PublicEventView.tsx`, load public roles, show “Apply to participate” cards/forms, submit application, and show a success message. Keep ticket reservation flow unchanged.

- [ ] **Step 6: Verify and commit**

Run: `make verify`

Commit: `git commit -m "Add public role applications"`

---

### Task 3: Private Application Review

**Files:**
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/event_roles.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`
- Modify: `web/src/views/EventEditorView.tsx`
- Modify: `web/src/App.test.tsx`

- [ ] **Step 1: Add backend test `TestEventRoleApplicationReviewAPI`**

Assert owner/member can list applications, but only owner can update status. Assert allowed transitions to `under_review`, `accepted`, `waitlisted`, `rejected`, `withdrawn`, `confirmed`; invalid status rejected. Define capacity-consuming statuses as `accepted` and `confirmed`; role capacity cannot be exceeded when capacity > 0. Include a test with one accepted and one confirmed application at capacity to ensure another accept/confirm returns `409 role capacity reached`. Include a test that `accepted -> confirmed` at full capacity succeeds, and that idempotent `accepted -> accepted` / `confirmed -> confirmed` does not falsely exceed capacity. All status changes set reviewed_by/reviewed_at and audit `role_application.reviewed`.

- [ ] **Step 2: Register routes**

Add:

```go
a.mux.HandleFunc("GET /api/events/{eventID}/role-applications", a.handleListEventRoleApplications)
a.mux.HandleFunc("PATCH /api/events/{eventID}/role-applications/{applicationID}", a.handleReviewEventRoleApplication)
```

- [ ] **Step 3: Implement list/review**

Private list is owner/member. Review is owner-only. Status update locks the target `event_roles` row with `for update` first, then locks the application row in the same transaction before counting existing capacity-consuming applications (`accepted`, `confirmed`). This serializes concurrent accepts/confirms for the same role. When counting existing capacity-consuming applications, exclude the target application ID, then add one only if the next status is `accepted` or `confirmed`; this allows `accepted -> confirmed` and idempotent accepted/confirmed updates at full capacity. If the next status is not capacity-consuming, skip the capacity check. Audit metadata must exclude applicant message and email; include eventId, roleId, applicationId, previousStatus, nextStatus.

- [ ] **Step 4: Add editor UI**

In `EventEditorView.tsx`, load roles and applications. Render an “Applications” panel with applicant, role, message, status, and owner-only status buttons/select.

- [ ] **Step 5: Verify and commit**

Run: `make verify`

Commit: `git commit -m "Add role application review"`

---

### Task 4: Accepted Participant Roster

**Files:**
- Modify: `backend/internal/app/event_roles.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`
- Modify: `web/src/views/EventEditorView.tsx`
- Modify: `web/src/views/WorkspaceView.tsx`
- Modify: `web/src/App.test.tsx`

- [ ] **Step 1: Add backend test `TestEventParticipantRosterAPI`**

After accepting/confirming applications, assert `GET /api/events/{eventID}/participants` returns accepted+confirmed participants grouped/ordered by role, excludes submitted/rejected/waitlisted, owner/member only, and includes no private message text.

- [ ] **Step 2: Register/implement roster route**

Add `GET /api/events/{eventID}/participants`. Return `EventParticipantDTO` rows with applicationId, roleId, roleName, applicantName, applicantEmail, status, updatedAt. Do not include application message.

- [ ] **Step 3: Add editor/workspace roster UI**

Show a “Participant roster” panel in `EventEditorView.tsx`; add closed/open event copy in `WorkspaceView.tsx` summarizing participant count if already loaded or available through event detail only.

- [ ] **Step 4: Verify and commit**

Run: `make verify`

Commit: `git commit -m "Add event participant roster"`

---

### Task 5: Archive Participant Memory

**Files:**
- Modify: `backend/internal/app/schema.sql`
- Modify: `backend/internal/app/events.go`
- Modify: `backend/internal/app/event_roles.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`
- Modify: `web/src/views/EventEditorView.tsx`
- Modify: `web/src/App.test.tsx`

- [ ] **Step 1: Add backend test `TestArchiveCapturesParticipantMemory`**

Accept/confirm applications, close event, then assert archive detail includes participant memory: roleName, participantName, status. Assert application messages/emails are not in archive participant memory. Assert end-of-night retry is idempotent.

- [ ] **Step 2: Add archive participant schema**

Add:

```sql
create table if not exists event_archive_participants (
  id uuid primary key default gen_random_uuid(),
  archive_id uuid not null references event_archives(id) on delete cascade,
  source_application_id uuid not null references event_role_applications(id) on delete cascade,
  role_name text not null,
  participant_name text not null,
  status text not null check (status in ('accepted','confirmed')),
  created_at timestamptz not null default now(),
  unique (archive_id, source_application_id)
);
```

- [ ] **Step 3: Snapshot participants at end-of-night**

Extend `ensureEventArchive` or add `ensureEventArchiveID` so end-of-night can obtain an archive ID. Then call `snapshotArchiveParticipants(ctx, tx, eventID, archiveID)` in both end-of-night paths: the existing-report/idempotent retry branch and the first-close branch. The snapshot inserts accepted/confirmed applications joined to roles into `event_archive_participants` with `source_application_id` and `on conflict (archive_id, source_application_id) do nothing`. This preserves duplicate human names and keeps retries idempotent.

- [ ] **Step 4: Return participant memory in archive DTO**

Extend `EventArchiveDTO` with `participants: EventArchiveParticipantDTO[]` and load ordered by roleName, participantName.

- [ ] **Step 5: Show archive participant memory**

In archive detail panel, render “Participant memory” and list names by role/status. Keep private application messages out.

- [ ] **Step 6: Verify, review, and commit**

Run: `make verify`, then route review through @oracle for privacy/permission/idempotency before commit.

Commit: `git commit -m "Add archive participant memory"`

---

## Final Verification

- [ ] Run `make verify`.
- [ ] Run `git status --short --branch` and confirm clean tree.
- [ ] Run `git log --oneline -10` and confirm five role/application commits.

## Out of Scope

- Public performer/vendor profile pages.
- Messaging applicants by email.
- Attachments, portfolios, media uploads, contracts, payouts per role, or scheduling shifts.
- Role templates shared across events.
- Public archive pages.

## Self-Review

- Spec coverage: role definitions, public application intake, private review, roster, and archive participant memory are covered.
- Placeholder scan: no task depends on unspecified future work; validation/status/privacy rules are explicit.
- Type consistency: DTO names use `EventRoleDTO`, `EventRoleApplicationDTO`, `EventParticipantDTO`, and `EventArchiveParticipantDTO` consistently.
