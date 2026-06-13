# First Event Lifecycle Tracer Bullet Implementation Plan

> **For agentic workers:** Execute this plan task-by-task. Recommended path:
> dispatch a fresh subagent per task, review each result with `review-quality`,
> then continue. For complex multi-agent splits, use
> `parallel-feature-development`, `team-composition-patterns`, and
> `team-communication-protocols`. Steps use checkbox (`- [ ]`) syntax for
> tracking.

**Goal:** Build the first end-to-end subcult-os lifecycle: signup/login, multi-member Workspace, Event publication, guest free Ticket reservation, mobile Door Check-In, End of Night, and private Event Report.

**Architecture:** Keep the first slice intentionally monolithic and boring: Go HTTP API, PostgreSQL persistence, Vite React frontend, server-rendered JSON contracts, and logged email outbox records instead of real email delivery. The first slice should prove domain flow and data boundaries without Stripe, Contacts, Event Roles, public discovery, archive publishing, or advanced permissions.

**Tech Stack:** Go 1.26 standard library HTTP server, PostgreSQL 17, `github.com/jackc/pgx/v5/pgxpool`, Vite React TypeScript, Vitest, Docker Compose.

---

## Domain scope locked by `CONTEXT.md`

Implement only the resolved first cuts:

- First Tracer Bullet: `Create Event → publish Public Event Page → reserve free Ticket → Door Check-In → End of Night → Event Report`.
- Multi-member Workspaces from the start.
- Owner + Member only.
- Real email/password authentication.
- Minimal email invitations, logged/stored email outbox allowed.
- Guest Checkout allowed for attendee reservations.
- Reservation data: email required, Display Name optional.
- Door: mobile-first, manual lookup first, one check-in per Ticket, no re-entry.
- Public Event Page: direct link only, no discovery index.
- Ticketing: one free Ticket Type, one allocation number, closes when full.
- No Stripe, Contacts, Event Roles, public profiles, public archive, deletion, cancellation, or rescheduling in this tracer bullet.

---

## File structure

### Backend files

- Modify: `backend/go.mod` — add `pgx/v5` dependency.
- Replace: `backend/internal/app/app.go` — app construction, routing, middleware, shared response helpers.
- Create: `backend/internal/app/config.go` — environment config parsing.
- Create: `backend/internal/app/db.go` — PostgreSQL pool setup and migration runner.
- Create: `backend/internal/app/schema.sql` — first-slice schema.
- Create: `backend/internal/app/auth.go` — signup/login/session handling and password hashing.
- Create: `backend/internal/app/workspaces.go` — Workspace create/current/list, invitations, member removal.
- Create: `backend/internal/app/events.go` — draft Event create/update/list/publish/end-of-night/report.
- Create: `backend/internal/app/tickets.go` — public reservation, ticket lookup, check-in.
- Create: `backend/internal/app/audit.go` — minimal Audit Trail writer.
- Create: `backend/internal/app/email_outbox.go` — stored outbound email records.
- Replace: `backend/internal/app/app_test.go` — keep smoke test against the HTTP app.
- Create: `backend/internal/app/lifecycle_test.go` — end-to-end acceptance test using a real PostgreSQL database when `TEST_DATABASE_URL` is set.
- Modify: `backend/cmd/app/main.go` — start HTTP server instead of printing greeting.

### Frontend files

- Replace: `web/src/App.tsx` — route-by-path minimal SPA shell.
- Create: `web/src/api.ts` — typed API client with cookie credentials.
- Create: `web/src/domain.ts` — frontend DTO types matching backend JSON.
- Create: `web/src/views/AuthView.tsx` — signup/login.
- Create: `web/src/views/WorkspaceView.tsx` — current Workspace, members, invites, event list.
- Create: `web/src/views/EventEditorView.tsx` — create/edit/publish/end-of-night/report.
- Create: `web/src/views/PublicEventView.tsx` — public direct-link Event page and guest reservation.
- Create: `web/src/views/DoorView.tsx` — mobile-first manual lookup/check-in.
- Create: `web/src/views/TicketView.tsx` — guest ticket confirmation.
- Replace: `web/src/App.test.tsx` — smoke tests for route rendering and important labels.
- Modify: `web/src/styles.css` — minimal responsive layout and mobile Door affordances.

### Config/docs files

- Modify: `docker-compose.yml` — pass `APP_ENV`, `DATABASE_URL`, `SESSION_SECRET`, and `PUBLIC_WEB_URL` to the API service.
- Modify: `.env.example` — add API/auth/email-outbox variables.
- Modify: `README.md` — document first-slice local flow and useful URLs.
- Optional after implementation: create `docs/adr/0004-first-slice-auth-and-email-outbox.md` if the team wants to preserve why email/password and logged email outbox were chosen.

---

## Backend JSON contracts

Use these stable DTO names in backend tests and frontend `domain.ts`:

```ts
export type WorkspaceRole = 'owner' | 'member';
export type EventStatus = 'draft' | 'published' | 'end_of_night';

export interface CurrentUserDTO {
  id: string;
  email: string;
  displayName: string | null;
  workspaces: WorkspaceSummaryDTO[];
}

export interface WorkspaceSummaryDTO {
  id: string;
  name: string;
  role: WorkspaceRole;
}

export interface EventDTO {
  id: string;
  workspaceId: string;
  title: string;
  startsAt: string;
  publicDescription: string;
  locationDisplay: string;
  ticketAllocation: number;
  reservedCount: number;
  checkedInCount: number;
  status: EventStatus;
  publicSlug: string | null;
  publicUrl: string | null;
}

export interface TicketDTO {
  id: string;
  eventId: string;
  email: string;
  displayName: string | null;
  code: string;
  status: 'reserved' | 'checked_in';
  checkedInAt: string | null;
}

export interface EventReportDTO {
  id: string;
  eventId: string;
  title: string;
  startsAt: string;
  publicUrl: string;
  ticketAllocation: number;
  ticketsReserved: number;
  ticketsCheckedIn: number;
  noShows: number;
  generatedAt: string;
  generatedByMemberEmail: string;
}
```

---

## API endpoints

Implement these endpoints only:

```text
POST   /api/auth/signup
POST   /api/auth/login
POST   /api/auth/logout
GET    /api/me

POST   /api/workspaces
GET    /api/workspaces/current
POST   /api/workspaces/{workspaceID}/invitations
POST   /api/invitations/{token}/accept
DELETE /api/workspaces/{workspaceID}/members/{memberID}

GET    /api/workspaces/{workspaceID}/events
POST   /api/workspaces/{workspaceID}/events
GET    /api/events/{eventID}
PATCH  /api/events/{eventID}
POST   /api/events/{eventID}/publish
POST   /api/events/{eventID}/end-of-night
GET    /api/events/{eventID}/report

GET    /api/public/events/{slug}
POST   /api/public/events/{slug}/reservations
GET    /api/tickets/{code}

GET    /api/events/{eventID}/door/tickets?query={emailOrNameOrCode}
POST   /api/events/{eventID}/door/check-ins
```

---

### Task 1: Backend database foundation

**Files:**
- Modify: `backend/go.mod`
- Create: `backend/internal/app/config.go`
- Create: `backend/internal/app/db.go`
- Create: `backend/internal/app/schema.sql`
- Replace: `backend/internal/app/app.go`
- Replace: `backend/internal/app/app_test.go`

- [ ] **Step 1: Add pgx dependency**

Run:

```bash
cd backend && go get github.com/jackc/pgx/v5/pgxpool@latest
```

Expected: `backend/go.mod` and `backend/go.sum` include `github.com/jackc/pgx/v5`.

- [ ] **Step 2: Write the app smoke test first**

Replace `backend/internal/app/app_test.go` with:

```go
package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	app := NewTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}
```

- [ ] **Step 3: Run the failing backend smoke test**

Run:

```bash
cd backend && go test ./internal/app -run TestHealth -count=1
```

Expected: FAIL because `NewTestApp` and `Handler` do not exist.

- [ ] **Step 4: Implement minimal app shell**

Create `backend/internal/app/config.go`:

```go
package app

import "os"

type Config struct {
	AppEnv       string
	DatabaseURL  string
	SessionSecret string
	PublicWebURL string
	Addr         string
}

func LoadConfig() Config {
	return Config{
		AppEnv:       env("APP_ENV", "development"),
		DatabaseURL:  env("DATABASE_URL", ""),
		SessionSecret: env("SESSION_SECRET", "dev-session-secret-change-me"),
		PublicWebURL: env("PUBLIC_WEB_URL", "http://localhost:5173"),
		Addr:         env("API_ADDR", ":8080"),
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
```

Create `backend/internal/app/db.go`:

```go
package app

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schemaFS embed.FS

func OpenDB(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	if databaseURL == "" {
		return nil, nil
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return nil
	}
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	_, err = pool.Exec(ctx, string(schema))
	return err
}
```

Create `backend/internal/app/schema.sql` with Task 2's schema as the first real migration target; for this task it can contain only:

```sql
create table if not exists schema_migrations (
  id text primary key,
  applied_at timestamptz not null default now()
);
```

Replace `backend/internal/app/app.go`:

```go
package app

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	config Config
	db     *pgxpool.Pool
	mux    *http.ServeMux
}

func New(config Config, db *pgxpool.Pool) *App {
	a := &App{config: config, db: db, mux: http.NewServeMux()}
	a.routes()
	return a
}

func NewTestApp(t *testing.T) *App {
	t.Helper()
	return New(Config{AppEnv: "test", PublicWebURL: "http://example.test", SessionSecret: "test-secret"}, nil)
}

func (a *App) Handler() http.Handler { return a.mux }

func (a *App) routes() {
	a.mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
```

- [ ] **Step 5: Verify Task 1**

Run:

```bash
cd backend && go test ./internal/app -run TestHealth -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit Task 1**

```bash
git add backend/go.mod backend/go.sum backend/internal/app/app.go backend/internal/app/app_test.go backend/internal/app/config.go backend/internal/app/db.go backend/internal/app/schema.sql
git commit -m "feat: add backend app foundation"
```

---

### Task 2: Schema and lifecycle acceptance test

**Files:**
- Modify: `backend/internal/app/schema.sql`
- Create: `backend/internal/app/lifecycle_test.go`

- [ ] **Step 1: Replace schema with first-slice tables**

Replace `backend/internal/app/schema.sql` with tables for:

```sql
create extension if not exists pgcrypto;

create table if not exists people (
  id uuid primary key default gen_random_uuid(),
  email text not null unique,
  display_name text,
  password_hash text not null,
  created_at timestamptz not null default now()
);

create table if not exists sessions (
  id uuid primary key default gen_random_uuid(),
  person_id uuid not null references people(id) on delete cascade,
  token_hash text not null unique,
  expires_at timestamptz not null,
  created_at timestamptz not null default now()
);

create table if not exists workspaces (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  created_at timestamptz not null default now()
);

create table if not exists workspace_members (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references workspaces(id) on delete cascade,
  person_id uuid not null references people(id) on delete cascade,
  role text not null check (role in ('owner', 'member')),
  removed_at timestamptz,
  created_at timestamptz not null default now(),
  unique (workspace_id, person_id)
);

create table if not exists workspace_invitations (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references workspaces(id) on delete cascade,
  email text not null,
  role text not null default 'member' check (role = 'member'),
  token_hash text not null unique,
  accepted_at timestamptz,
  invited_by_person_id uuid not null references people(id),
  created_at timestamptz not null default now()
);

create table if not exists events (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references workspaces(id) on delete cascade,
  title text not null,
  starts_at timestamptz not null,
  public_description text not null,
  location_display text not null,
  ticket_allocation integer not null check (ticket_allocation >= 0),
  status text not null default 'draft' check (status in ('draft', 'published', 'end_of_night')),
  public_slug text unique,
  published_at timestamptz,
  ended_at timestamptz,
  created_by_person_id uuid not null references people(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists tickets (
  id uuid primary key default gen_random_uuid(),
  event_id uuid not null references events(id) on delete cascade,
  email text not null,
  display_name text,
  code text not null unique,
  status text not null default 'reserved' check (status in ('reserved', 'checked_in')),
  checked_in_at timestamptz,
  checked_in_by_person_id uuid references people(id),
  created_at timestamptz not null default now()
);

create table if not exists event_reports (
  id uuid primary key default gen_random_uuid(),
  event_id uuid not null unique references events(id) on delete cascade,
  generated_by_person_id uuid not null references people(id),
  generated_at timestamptz not null default now(),
  snapshot jsonb not null
);

create table if not exists audit_entries (
  id uuid primary key default gen_random_uuid(),
  actor_person_id uuid references people(id),
  action text not null,
  subject_type text not null,
  subject_id uuid,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create table if not exists email_outbox (
  id uuid primary key default gen_random_uuid(),
  recipient_email text not null,
  subject text not null,
  body text not null,
  related_type text not null,
  related_id uuid,
  created_at timestamptz not null default now()
);
```

- [ ] **Step 2: Write acceptance test skeleton**

Create `backend/internal/app/lifecycle_test.go`:

```go
package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestFirstEventLifecycle(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("set TEST_DATABASE_URL to run lifecycle acceptance test")
	}

	ctx := t.Context()
	db, err := OpenDB(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil { t.Fatal(err) }
	defer db.Close()
	if err := RunMigrations(ctx, db); err != nil { t.Fatal(err) }

	app := New(Config{AppEnv: "test", PublicWebURL: "http://public.test", SessionSecret: "test-secret"}, db)
	ownerCookie := postJSON(t, app, nil, "/api/auth/signup", map[string]any{"email":"owner@example.test","password":"secret1234","displayName":"Owner"}).Cookie
	workspace := postJSON(t, app, ownerCookie, "/api/workspaces", map[string]any{"name":"Signal Collective"}).JSON
	workspaceID := workspace["id"].(string)

	invite := postJSON(t, app, ownerCookie, "/api/workspaces/"+workspaceID+"/invitations", map[string]any{"email":"member@example.test"}).JSON
	memberCookie := postJSON(t, app, nil, "/api/auth/signup", map[string]any{"email":"member@example.test","password":"secret1234","displayName":"Door"}).Cookie
	postJSON(t, app, memberCookie, "/api/invitations/"+invite["token"].(string)+"/accept", map[string]any{})

	event := postJSON(t, app, ownerCookie, "/api/workspaces/"+workspaceID+"/events", map[string]any{"title":"Night Market","startsAt":"2026-07-01T20:00:00Z","publicDescription":"Free community event.","locationDisplay":"Warehouse District","ticketAllocation":2}).JSON
	eventID := event["id"].(string)
	published := postJSON(t, app, ownerCookie, "/api/events/"+eventID+"/publish", map[string]any{}).JSON
	slug := published["publicSlug"].(string)

	ticket := postJSON(t, app, nil, "/api/public/events/"+slug+"/reservations", map[string]any{"email":"guest@example.test","displayName":"Guest"}).JSON
	postJSON(t, app, memberCookie, "/api/events/"+eventID+"/door/check-ins", map[string]any{"code":ticket["code"].(string)})
	report := postJSON(t, app, ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}).JSON

	if report["ticketsReserved"].(float64) != 1 || report["ticketsCheckedIn"].(float64) != 1 || report["noShows"].(float64) != 0 {
		t.Fatalf("unexpected report counts: %#v", report)
	}
}

type testResponse struct { Cookie *http.Cookie; JSON map[string]any }

func postJSON(t *testing.T, app *App, cookie *http.Cookie, path string, payload map[string]any) testResponse {
	t.Helper()
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil { req.AddCookie(cookie) }
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code < 200 || rec.Code >= 300 { t.Fatalf("POST %s got %d: %s", path, rec.Code, rec.Body.String()) }
	var decoded map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil { t.Fatal(err) }
	var cookieOut *http.Cookie
	for _, c := range rec.Result().Cookies() { if c.Name == "subcult_session" { cookieOut = c } }
	return testResponse{Cookie: cookieOut, JSON: decoded}
}
```

- [ ] **Step 3: Run acceptance test and verify expected failure**

Run:

```bash
cd backend && TEST_DATABASE_URL="$DATABASE_URL" go test ./internal/app -run TestFirstEventLifecycle -count=1
```

Expected: FAIL because auth/workspace/event/ticket routes are not implemented.

- [ ] **Step 4: Commit Task 2**

```bash
git add backend/internal/app/schema.sql backend/internal/app/lifecycle_test.go
git commit -m "test: define first lifecycle acceptance path"
```

---

### Task 3: Authentication and sessions

**Files:**
- Create: `backend/internal/app/auth.go`
- Modify: `backend/internal/app/app.go`

- [ ] **Step 1: Implement auth route registration**

Add these calls inside `routes()` in `backend/internal/app/app.go`:

```go
a.mux.HandleFunc("POST /api/auth/signup", a.handleSignup)
a.mux.HandleFunc("POST /api/auth/login", a.handleLogin)
a.mux.HandleFunc("POST /api/auth/logout", a.handleLogout)
a.mux.HandleFunc("GET /api/me", a.handleMe)
```

- [ ] **Step 2: Create auth implementation**

Create `backend/internal/app/auth.go` with password hashing via `crypto/subtle` and `sha256` for the first slice. Use `bcrypt` later before production if desired; this tracer bullet only requires real, non-mocked auth.

Required functions/signatures:

```go
func (a *App) handleSignup(w http.ResponseWriter, r *http.Request)
func (a *App) handleLogin(w http.ResponseWriter, r *http.Request)
func (a *App) handleLogout(w http.ResponseWriter, r *http.Request)
func (a *App) handleMe(w http.ResponseWriter, r *http.Request)
func (a *App) requirePersonID(r *http.Request) (string, bool)
func hashPassword(password string) string
func verifyPassword(password, hash string) bool
func newToken() (string, string)
```

Behavior:

- `POST /api/auth/signup` validates email contains `@`, password length >= 8, creates `people`, creates session, sets `subcult_session` HttpOnly SameSite=Lax cookie, returns `CurrentUserDTO`.
- `POST /api/auth/login` verifies password, sets session cookie, returns `CurrentUserDTO`.
- `POST /api/auth/logout` deletes current session if present and expires cookie.
- `GET /api/me` returns current user and their active Workspaces or 401.

- [ ] **Step 3: Run auth-relevant test**

Run:

```bash
cd backend && TEST_DATABASE_URL="$DATABASE_URL" go test ./internal/app -run TestFirstEventLifecycle -count=1
```

Expected: FAIL later at `/api/workspaces`, not signup.

- [ ] **Step 4: Commit Task 3**

```bash
git add backend/internal/app/app.go backend/internal/app/auth.go
git commit -m "feat: add email password authentication"
```

---

### Task 4: Workspaces, members, invitations, and email outbox

**Files:**
- Create: `backend/internal/app/workspaces.go`
- Create: `backend/internal/app/email_outbox.go`
- Create: `backend/internal/app/audit.go`
- Modify: `backend/internal/app/app.go`

- [ ] **Step 1: Register Workspace routes**

Add inside `routes()`:

```go
a.mux.HandleFunc("POST /api/workspaces", a.handleCreateWorkspace)
a.mux.HandleFunc("GET /api/workspaces/current", a.handleCurrentWorkspace)
a.mux.HandleFunc("POST /api/workspaces/{workspaceID}/invitations", a.handleCreateInvitation)
a.mux.HandleFunc("POST /api/invitations/{token}/accept", a.handleAcceptInvitation)
a.mux.HandleFunc("DELETE /api/workspaces/{workspaceID}/members/{memberID}", a.handleRemoveMember)
```

- [ ] **Step 2: Implement helpers and handlers**

Create `workspaces.go` with:

```go
type workspaceDTO struct { ID, Name, Role string }
type memberDTO struct { ID, Email, DisplayName, Role string }

func (a *App) requireWorkspaceRole(r *http.Request, workspaceID string, allowed ...string) (personID string, role string, ok bool)
func (a *App) handleCreateWorkspace(w http.ResponseWriter, r *http.Request)
func (a *App) handleCurrentWorkspace(w http.ResponseWriter, r *http.Request)
func (a *App) handleCreateInvitation(w http.ResponseWriter, r *http.Request)
func (a *App) handleAcceptInvitation(w http.ResponseWriter, r *http.Request)
func (a *App) handleRemoveMember(w http.ResponseWriter, r *http.Request)
```

Behavior:

- Creating a Workspace makes the current Person Owner.
- Invitation route requires Owner and returns a plaintext `token` in JSON for dev/testing while storing only `token_hash`.
- Invitation writes an email outbox record with subject `You're invited to Signal Collective on subcult-os` and body containing `/invite/{token}`.
- Accept invitation requires logged-in Person whose email matches invitation email, adds active Member role, sets `accepted_at`, records audit.
- Remove Member requires Owner, cannot remove the last Owner, sets `removed_at`.

Create `email_outbox.go`:

```go
func (a *App) enqueueEmail(ctx context.Context, recipient, subject, body, relatedType, relatedID string) error
```

Create `audit.go`:

```go
func (a *App) audit(ctx context.Context, actorPersonID, action, subjectType, subjectID string, metadata map[string]any) error
```

- [ ] **Step 3: Run acceptance test**

Run:

```bash
cd backend && TEST_DATABASE_URL="$DATABASE_URL" go test ./internal/app -run TestFirstEventLifecycle -count=1
```

Expected: FAIL later at `/api/workspaces/{workspaceID}/events`, not invitations.

- [ ] **Step 4: Commit Task 4**

```bash
git add backend/internal/app/app.go backend/internal/app/workspaces.go backend/internal/app/email_outbox.go backend/internal/app/audit.go
git commit -m "feat: add multi-member workspaces"
```

---

### Task 5: Events, publication, end of night, and report

**Files:**
- Create: `backend/internal/app/events.go`
- Modify: `backend/internal/app/app.go`

- [ ] **Step 1: Register Event routes**

Add inside `routes()`:

```go
a.mux.HandleFunc("GET /api/workspaces/{workspaceID}/events", a.handleListEvents)
a.mux.HandleFunc("POST /api/workspaces/{workspaceID}/events", a.handleCreateEvent)
a.mux.HandleFunc("GET /api/events/{eventID}", a.handleGetEvent)
a.mux.HandleFunc("PATCH /api/events/{eventID}", a.handleUpdateEvent)
a.mux.HandleFunc("POST /api/events/{eventID}/publish", a.handlePublishEvent)
a.mux.HandleFunc("POST /api/events/{eventID}/end-of-night", a.handleEndOfNight)
a.mux.HandleFunc("GET /api/events/{eventID}/report", a.handleGetReport)
```

- [ ] **Step 2: Implement Event behavior**

Create `events.go` with:

```go
type eventDTO struct {
	ID string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	Title string `json:"title"`
	StartsAt string `json:"startsAt"`
	PublicDescription string `json:"publicDescription"`
	LocationDisplay string `json:"locationDisplay"`
	TicketAllocation int `json:"ticketAllocation"`
	ReservedCount int `json:"reservedCount"`
	CheckedInCount int `json:"checkedInCount"`
	Status string `json:"status"`
	PublicSlug *string `json:"publicSlug"`
	PublicURL *string `json:"publicUrl"`
}

func (a *App) handleListEvents(w http.ResponseWriter, r *http.Request)
func (a *App) handleCreateEvent(w http.ResponseWriter, r *http.Request)
func (a *App) handleGetEvent(w http.ResponseWriter, r *http.Request)
func (a *App) handleUpdateEvent(w http.ResponseWriter, r *http.Request)
func (a *App) handlePublishEvent(w http.ResponseWriter, r *http.Request)
func (a *App) handleEndOfNight(w http.ResponseWriter, r *http.Request)
func (a *App) handleGetReport(w http.ResponseWriter, r *http.Request)
```

Behavior:

- Owner and Member can create/update draft Event details.
- Owner only can publish and run End of Night.
- Publish requires title, startsAt, publicDescription, locationDisplay, ticketAllocation > 0, generates stable `public_slug`, records audit.
- Published edits allow description/location always, startsAt only while reserved count is zero, allocation only if not below reserved count.
- End of Night creates or returns one private report snapshot with title/date/public URL/allocation/reserved/checked-in/no-shows/generatedAt/generatedByMemberEmail, updates Event status, records audit.

- [ ] **Step 3: Run acceptance test**

Run:

```bash
cd backend && TEST_DATABASE_URL="$DATABASE_URL" go test ./internal/app -run TestFirstEventLifecycle -count=1
```

Expected: FAIL later at public reservation or door check-in.

- [ ] **Step 4: Commit Task 5**

```bash
git add backend/internal/app/app.go backend/internal/app/events.go
git commit -m "feat: add first event lifecycle"
```

---

### Task 6: Public reservations, tickets, and Door Check-In

**Files:**
- Create: `backend/internal/app/tickets.go`
- Modify: `backend/internal/app/app.go`

- [ ] **Step 1: Register Ticket routes**

Add inside `routes()`:

```go
a.mux.HandleFunc("GET /api/public/events/{slug}", a.handlePublicEvent)
a.mux.HandleFunc("POST /api/public/events/{slug}/reservations", a.handleReserveTicket)
a.mux.HandleFunc("GET /api/tickets/{code}", a.handleGetTicket)
a.mux.HandleFunc("GET /api/events/{eventID}/door/tickets", a.handleDoorTicketSearch)
a.mux.HandleFunc("POST /api/events/{eventID}/door/check-ins", a.handleDoorCheckIn)
```

- [ ] **Step 2: Implement reservation and check-in behavior**

Create `tickets.go` with:

```go
type ticketDTO struct {
	ID string `json:"id"`
	EventID string `json:"eventId"`
	Email string `json:"email"`
	DisplayName *string `json:"displayName"`
	Code string `json:"code"`
	Status string `json:"status"`
	CheckedInAt *string `json:"checkedInAt"`
}

func (a *App) handlePublicEvent(w http.ResponseWriter, r *http.Request)
func (a *App) handleReserveTicket(w http.ResponseWriter, r *http.Request)
func (a *App) handleGetTicket(w http.ResponseWriter, r *http.Request)
func (a *App) handleDoorTicketSearch(w http.ResponseWriter, r *http.Request)
func (a *App) handleDoorCheckIn(w http.ResponseWriter, r *http.Request)
```

Behavior:

- Public Event returns only published Events by slug and includes `remainingTickets` and `isFull`.
- Reservation requires email, optional displayName, refuses when allocation is full, creates one Ticket, writes audit with no actor, enqueues ticket email, returns TicketDTO and ticket URL.
- Ticket code is unguessable using `crypto/rand` and URL-safe base64.
- Door search requires Owner or Member and matches lowercased email, display name, or exact code.
- Check-In requires Owner or Member, refuses tickets from other Events, returns already-checked-in state on second attempt without creating re-entry, records audit.

- [ ] **Step 3: Run acceptance test**

Run:

```bash
cd backend && TEST_DATABASE_URL="$DATABASE_URL" go test ./internal/app -run TestFirstEventLifecycle -count=1
```

Expected: PASS.

- [ ] **Step 4: Run all backend tests**

```bash
cd backend && go test ./...
```

Expected: PASS. The database acceptance test may skip if `TEST_DATABASE_URL` is unset.

- [ ] **Step 5: Commit Task 6**

```bash
git add backend/internal/app/app.go backend/internal/app/tickets.go
git commit -m "feat: add free ticket reservations and door check-in"
```

---

### Task 7: HTTP server entrypoint and Docker config

**Files:**
- Modify: `backend/cmd/app/main.go`
- Modify: `docker-compose.yml`
- Modify: `.env.example`

- [ ] **Step 1: Replace backend main**

Replace `backend/cmd/app/main.go`:

```go
package main

import (
	"context"
	"log"
	"net/http"

	"git.subcult.tv/PatrickFanella/subcult-os/internal/app"
)

func main() {
	ctx := context.Background()
	config := app.LoadConfig()
	db, err := app.OpenDB(ctx, config.DatabaseURL)
	if err != nil { log.Fatal(err) }
	if db != nil { defer db.Close() }
	if err := app.RunMigrations(ctx, db); err != nil { log.Fatal(err) }

	server := app.New(config, db)
	log.Printf("subcult-os api listening on %s", config.Addr)
	log.Fatal(http.ListenAndServe(config.Addr, server.Handler()))
}
```

- [ ] **Step 2: Update Docker/environment**

In `docker-compose.yml`, add API environment values:

```yaml
      APP_ENV: ${APP_ENV:-development}
      API_ADDR: :8080
      SESSION_SECRET: ${SESSION_SECRET:-dev-session-secret-change-me}
      PUBLIC_WEB_URL: ${PUBLIC_WEB_URL:-http://localhost:5173}
```

In `.env.example`, add:

```env
APP_ENV=development
SESSION_SECRET=dev-session-secret-change-me
PUBLIC_WEB_URL=http://localhost:5173
DATABASE_URL=postgres://app:change-me@localhost:5432/app?sslmode=disable
TEST_DATABASE_URL=postgres://app:change-me@localhost:5432/app?sslmode=disable
```

- [ ] **Step 3: Verify backend build and compose config**

Run:

```bash
make build-backend
make compose-config
```

Expected: both pass.

- [ ] **Step 4: Commit Task 7**

```bash
git add backend/cmd/app/main.go docker-compose.yml .env.example
git commit -m "feat: run subcult-os api server"
```

---

### Task 8: Frontend API client and domain types

**Files:**
- Create: `web/src/domain.ts`
- Create: `web/src/api.ts`

- [ ] **Step 1: Add domain types**

Create `web/src/domain.ts` using the DTO definitions from the Backend JSON contracts section.

- [ ] **Step 2: Add API client**

Create `web/src/api.ts`:

```ts
export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(path, {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers ?? {}),
    },
  });
  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(typeof data.error === 'string' ? data.error : `Request failed: ${response.status}`);
  }
  return data as T;
}

export function postJSON<T>(path: string, body: unknown): Promise<T> {
  return api<T>(path, { method: 'POST', body: JSON.stringify(body) });
}

export function patchJSON<T>(path: string, body: unknown): Promise<T> {
  return api<T>(path, { method: 'PATCH', body: JSON.stringify(body) });
}
```

- [ ] **Step 3: Type-check**

Run:

```bash
pnpm --dir web run format
```

Expected: PASS.

- [ ] **Step 4: Commit Task 8**

```bash
git add web/src/domain.ts web/src/api.ts
git commit -m "feat: add frontend api client"
```

---

### Task 9: Frontend first-slice screens

**Files:**
- Replace: `web/src/App.tsx`
- Create: `web/src/views/AuthView.tsx`
- Create: `web/src/views/WorkspaceView.tsx`
- Create: `web/src/views/EventEditorView.tsx`
- Create: `web/src/views/PublicEventView.tsx`
- Create: `web/src/views/DoorView.tsx`
- Create: `web/src/views/TicketView.tsx`
- Modify: `web/src/styles.css`

- [ ] **Step 1: Route shell**

Replace `App.tsx` with a minimal path router:

```tsx
import { AuthView } from './views/AuthView';
import { DoorView } from './views/DoorView';
import { EventEditorView } from './views/EventEditorView';
import { PublicEventView } from './views/PublicEventView';
import { TicketView } from './views/TicketView';
import { WorkspaceView } from './views/WorkspaceView';

export default function App() {
  const path = window.location.pathname;
  if (path.startsWith('/e/')) return <PublicEventView slug={path.split('/')[2] ?? ''} />;
  if (path.startsWith('/tickets/')) return <TicketView code={path.split('/')[2] ?? ''} />;
  if (path.startsWith('/door/')) return <DoorView eventId={path.split('/')[2] ?? ''} />;
  if (path.startsWith('/events/')) return <EventEditorView eventId={path.split('/')[2] ?? ''} />;
  if (path === '/login' || path === '/signup') return <AuthView />;
  return <WorkspaceView />;
}
```

- [ ] **Step 2: Implement views**

Each view should be small and form-driven:

- `AuthView`: signup/login forms; after success `window.location.href = '/'`.
- `WorkspaceView`: load `/api/me`; if no Workspace, create Workspace form; otherwise show current Workspace, invite-by-email form, Member list if returned, and Event list with create button.
- `EventEditorView`: load/edit Event, publish button for Owners, public URL after publish, Door link, End of Night button, report summary.
- `PublicEventView`: load public Event by slug, show first publication fields, reserve form with email required and displayName optional, full state when no tickets remain.
- `DoorView`: mobile-first search input, result cards, check-in button, already-checked-in state.
- `TicketView`: load Ticket by code and show Ticket code/status.

- [ ] **Step 3: Add mobile Door CSS**

In `styles.css`, add reusable classes or Tailwind-compatible global styles so the Door form has large touch targets:

```css
button, input, textarea {
  font: inherit;
}

.door-action {
  min-height: 3.5rem;
  width: 100%;
  border-radius: 1rem;
  font-weight: 800;
}
```

- [ ] **Step 4: Run frontend type-check**

```bash
pnpm --dir web run format
```

Expected: PASS.

- [ ] **Step 5: Commit Task 9**

```bash
git add web/src/App.tsx web/src/views web/src/styles.css
git commit -m "feat: add first lifecycle frontend"
```

---

### Task 10: Frontend smoke tests and documentation

**Files:**
- Replace: `web/src/App.test.tsx`
- Modify: `README.md`

- [ ] **Step 1: Replace frontend smoke tests**

Replace `web/src/App.test.tsx` with tests that set `window.history` and render route labels:

```tsx
import { renderToString } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import App from './App';

describe('App routes', () => {
  it('renders the workspace route by default', () => {
    window.history.pushState({}, '', '/');
    expect(renderToString(<App />)).toContain('Workspace');
  });

  it('renders the public event route', () => {
    window.history.pushState({}, '', '/e/night-market');
    expect(renderToString(<App />)).toContain('Reserve');
  });

  it('renders the mobile door route', () => {
    window.history.pushState({}, '', '/door/event-1');
    expect(renderToString(<App />)).toContain('Door');
  });
});
```

- [ ] **Step 2: Document local flow in README**

Add a section:

```markdown
## First lifecycle slice

The first product slice proves:

1. Owner signs up and creates a Workspace.
2. Owner invites a Member by email; development builds store invitation email in `email_outbox`.
3. Owner creates and publishes a direct-link Public Event Page.
4. Guest reserves a free Ticket with email and optional display name.
5. Member runs mobile-friendly Door Check-In by manual lookup/code.
6. Owner runs End of Night and views the private Event Report.

Run locally:

```bash
make up-build
```

Then open:

- Web: http://localhost:5173
- API health: http://localhost:8080/api/health
```

- [ ] **Step 3: Run full verification**

Run:

```bash
make verify
```

Expected: PASS.

- [ ] **Step 4: Commit Task 10**

```bash
git add web/src/App.test.tsx README.md
git commit -m "test: cover first lifecycle frontend routes"
```

---

## Self-review checklist

- Spec coverage: every first cut from `CONTEXT.md` is represented in tasks above.
- No-goals preserved: no Stripe, Contacts, Event Roles, public profiles, public archive, deletion, cancellation, rescheduling, QR scanning, re-entry, waitlist, or discovery index.
- Acceptance path: backend `TestFirstEventLifecycle` proves the full domain path with real persistence when `TEST_DATABASE_URL` is set.
- Verification: final implementation must pass `make verify` per `AGENTS.md`.
