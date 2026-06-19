# Architecture Deepening Implementation Plan

> **For agentic workers:** Execute this plan task-by-task. Recommended path:
> dispatch a fresh subagent per task, review each result with `review-quality`,
> then continue. For complex multi-agent splits, use
> `parallel-feature-development`, `team-composition-patterns`, and
> `team-communication-protocols`. Steps use checkbox (`- [ ]`) syntax for
> tracking.

**Goal:** Deepen the current web frontend, Go API, and Expo mobile app so Event lifecycle, Workspace operations, Tickets, auth, media, Run of Show, discovery, and schema rules have smaller Interfaces, stronger Locality, and higher Leverage.

**Architecture:** This is a staged refactor, not a rewrite. First create shared test fixtures and characterization tests around current behaviour, then introduce Deep Modules behind existing routes/screens, then migrate callers one slice at a time. Keep public HTTP routes and visible UX stable unless a task explicitly narrows accidental scope such as Event Discovery.

**Tech Stack:** Go 1.x API with `net/http`, `pgx`, Postgres, Vite React TypeScript web, Expo SDK 54 React Native mobile, Vitest, Go tests, Makefile verification.

---

## Scope and sequencing

Do not implement all ten findings in one commit. Use these stages:

1. **Safety net:** characterization tests and shared fixtures.
2. **Contracts and seams:** shared DTO contract, auth/session Adapter, schema authority.
3. **Core domain depth:** Event lifecycle, Ticket/Payment/Door Record, media storage.
4. **Operator surfaces:** Workspace home, Event Operations Record editor, Run of Show.
5. **Scope cleanup:** Event Discovery isolation.

Each task below should end with `make verify` unless it explicitly says to run a narrower command first.

## File structure target

### Backend

- Create: `backend/internal/app/event_lifecycle.go` — Event Status invariants, commands, derived flags, transition checks.
- Create: `backend/internal/app/event_lifecycle_test.go` — unit tests for lifecycle rules.
- Create: `backend/internal/app/ticket_journey.go` — Ticket, Payment, and Door Record state derivation plus reservation/check-in orchestration helpers.
- Create: `backend/internal/app/ticket_journey_test.go` — unit tests for Ticket journey rules.
- Create: `backend/internal/app/session_exchange.go` — canonical session request/response extraction and emission.
- Create: `backend/internal/app/session_exchange_test.go` — tests for cookie/header/bearer compatibility.
- Create: `backend/internal/app/media_storage.go` — startup-created media storage Adapter Interface and concrete S3/MinIO Adapter.
- Create: `backend/internal/app/media_storage_test.go` — fake Adapter tests for upload failure/success.
- Create: `backend/internal/app/discovery.go` — Public Event Page direct-link vs Event Discovery rules.
- Modify: `backend/internal/app/app.go` — wire new Modules and keep routes stable.
- Modify: `backend/internal/app/events.go` — delegate lifecycle, Settlement/archive/report helpers incrementally.
- Modify: `backend/internal/app/tickets.go` — delegate Ticket journey logic.
- Modify: `backend/internal/app/auth.go` — delegate session transport logic.
- Modify: `backend/internal/app/media.go` — delegate storage construction/upload.
- Modify: `backend/internal/app/db.go` — make schema authority explicit.
- Modify: `backend/internal/app/schema.sql` and/or `migrations/0001_init.sql` — remove misleading split or move toward ordered migrations.

### Shared contracts

- Create: `contracts/api.schema.json` — source of truth for DTO shape used by backend, web, and mobile.
- Create: `scripts/check-contracts.mjs` — validates generated/static TypeScript DTOs against the schema names expected in clients.
- Modify: `web/src/domain.ts` — split or annotate generated DTO sections by Module.
- Modify: `mobile/src/api/types.ts` — align DTOs with `contracts/api.schema.json`.
- Modify: `package`/Makefile verification if needed to run contract checks from `make verify`.

### Web

- Create: `web/src/modules/routing/routes.ts` — route parsing Interface for web.
- Create: `web/src/modules/eventLifecycle/eventLifecycle.ts` — client display helpers from backend lifecycle flags.
- Create: `web/src/modules/workspace/workspaceModel.ts` — Workspace home view model and loader orchestration.
- Create: `web/src/modules/eventEditor/eventEditorModel.ts` — Event Operations Record editor orchestration.
- Create: `web/src/modules/tickets/ticketJourney.ts` — Ticket/Payment/Door Record display helpers.
- Create: `web/src/modules/discovery/discoveryModel.ts` — Event Discovery isolation.
- Modify: `web/src/App.tsx`, `web/src/views/WorkspaceView.tsx`, `web/src/views/EventEditorView.tsx`, `web/src/views/PublicEventView.tsx`, `web/src/views/TicketView.tsx`, `web/src/views/DoorView.tsx`, `web/src/views/DiscoverView.tsx`.
- Test: `web/src/App.test.tsx` plus focused tests next to new Modules where the existing test setup allows.

### Mobile

- Create: `mobile/src/modules/session/sessionAdapter.ts` — canonical mobile session Adapter over secure/local storage.
- Create: `mobile/src/modules/storage/persistedStore.ts` — reusable persistence Adapter.
- Create: `mobile/src/modules/events/eventEditModel.ts` — Event edit payload/readiness/image orchestration.
- Create: `mobile/src/modules/runOfShow/runOfShowModel.ts` — Run of Show ordering, parsing, and status rules.
- Create: `mobile/src/modules/tickets/ticketJourney.ts` — Ticket display and wallet state helpers.
- Create: `mobile/src/modules/discovery/discoveryModel.ts` — discovery isolation.
- Modify: `mobile/src/api/client.ts`, `mobile/src/auth/AuthContext.tsx`, `mobile/src/auth/sessionCookieStore.ts`, `mobile/src/staff/selectionStore.ts`, `mobile/src/tickets/walletStore.ts`.
- Modify: `mobile/app/event-edit.tsx`, `mobile/app/run-of-show.tsx`, `mobile/app/ticket.tsx`, `mobile/app/door.tsx`, `mobile/app/index.tsx`.

### Documentation

- Modify: `CONTEXT.md` only if a new canonical domain term is introduced. Prefer existing terms: Event, Workspace, Public Event Page, Event Discovery, Ticket, Door Record, Settlement, Run of Show, Event Operations Record.
- Modify: `docs/stacks.md` if verification or schema conventions change.
- Create: ADR only if the team chooses to keep discovery ahead of the First Discovery Cut or rejects one of these deepening candidates for a load-bearing reason.

---

## Task 1: Add characterization tests for existing critical flows

**Findings covered:** safety net for all ten findings.

**Files:**

- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/App.test.tsx`
- Create or modify: mobile tests if the mobile test harness already exists; otherwise document manual checks in this task's commit message and add the harness in a later task.

- [ ] **Step 1: Write backend characterization tests**

Add tests that prove current behaviour before refactoring:

```go
func TestEventLifecycleCurrentFlowCreatePublishEndOfNight(t *testing.T) {
    // Arrange: create Person, Workspace, Draft Event.
    // Act: publish Event, reserve free Ticket, check in Ticket, run End of Night.
    // Assert: Event Status is end_of_night, Event Report has reserved/check-in counts, Archive exists when Settlement exists.
}

func TestTicketReservationCurrentCapacityAndDoorRules(t *testing.T) {
    // Arrange: create a Published free Event with Ticket Allocation 1.
    // Act: reserve one Ticket, attempt second reservation, check in first Ticket twice.
    // Assert: second reservation returns conflict, second check-in is idempotent or returns the existing checked-in Ticket according to current behaviour.
}
```

- [ ] **Step 2: Write web characterization tests**

Add assertions around current routing and large-view behaviour:

```tsx
it('routes direct-link Public Event Page paths to the public event view', async () => {
  window.history.pushState({}, '', '/e/test-event');
  render(<App />);
  expect(await screen.findByText(/loading/i)).toBeInTheDocument();
});

it('keeps the Workspace home usable when optional private panels are forbidden', async () => {
  // Mock contacts/commitments as 403 and assert the Event list still renders.
});
```

- [ ] **Step 3: Run narrow checks**

Run:

```bash
go test ./...
pnpm --dir web run test
```

Expected: tests pass before refactoring.

- [ ] **Step 4: Run full verification**

Run:

```bash
make verify
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/app/lifecycle_test.go web/src/App.test.tsx
git commit -m "test: characterize core event flows"
```

---

## Task 2: Make schema authority explicit

**Findings covered:** 10.

**Files:**

- Modify: `backend/internal/app/db.go:11-38`
- Modify: `backend/internal/app/schema.sql`
- Modify: `migrations/0001_init.sql`
- Modify: `docs/stacks.md:155-161`

- [ ] **Step 1: Decide alpha schema authority**

For this repo's current alpha state, choose embedded idempotent schema as the explicit authority. Do not introduce a migration tool yet unless production data exists.

- [ ] **Step 2: Remove misleading migration split**

Change `migrations/0001_init.sql` to a pointer file that makes the authority clear:

```sql
-- Alpha schema authority lives in backend/internal/app/schema.sql.
-- This placeholder exists so the repository has an obvious future home for
-- ordered migrations when destructive changes, backfills, or production data
-- require them.
```

- [ ] **Step 3: Add a backend test that schema loads cleanly**

Add a test that calls `RunMigrations` against the test database path already used by existing backend tests and asserts a representative table exists, such as `events`.

- [ ] **Step 4: Update docs**

Update `docs/stacks.md` to say: alpha source of truth is `backend/internal/app/schema.sql`; `migrations/` is reserved for the future ordered migration tool.

- [ ] **Step 5: Verify and commit**

Run:

```bash
go test ./...
make verify
```

Commit:

```bash
git add backend/internal/app/db.go backend/internal/app/schema.sql migrations/0001_init.sql docs/stacks.md
git commit -m "docs: clarify alpha schema authority"
```

---

## Task 3: Create shared API contract checks

**Findings covered:** 4.

**Files:**

- Create: `contracts/api.schema.json`
- Create: `scripts/check-contracts.mjs`
- Modify: `web/src/domain.ts`
- Modify: `mobile/src/api/types.ts`
- Modify: `Makefile` if needed

- [ ] **Step 1: Add contract schema for shared DTOs**

Create `contracts/api.schema.json` covering at least these DTOs: `CurrentUserDTO`, `WorkspaceSummaryDTO`, `EventDTO`, `PublicEventDTO`, `PublicEventSummaryDTO`, `TicketDTO`, `TicketReservationDTO`, `PaidReservationDTO`.

Use backend JSON names as canonical. Include `ticketUrl` in `PaidReservationDTO` because backend returns it and mobile expects it.

- [ ] **Step 2: Align web DTOs**

Update `web/src/domain.ts` so `PaidReservationDTO` includes:

```ts
export interface PaidReservationDTO {
  ticketId: string;
  ticketCode: string;
  ticketUrl: string;
  checkoutSessionId: string;
  checkoutUrl: string;
}
```

Add `imageUrl: string | null` to web `EventDTO` and `PublicEventSummaryDTO` if backend already returns it.

- [ ] **Step 3: Align mobile DTOs**

Constrain mobile `TicketDTO.status` and `TicketDTO.paymentStatus` to match web/backend:

```ts
status: 'reserved' | 'checked_in';
paymentStatus: 'free' | 'pending' | 'paid' | 'cancelled';
```

- [ ] **Step 4: Add a lightweight contract checker**

Create `scripts/check-contracts.mjs` that reads `contracts/api.schema.json`, `web/src/domain.ts`, and `mobile/src/api/types.ts`, then fails if required DTO names are missing from either TypeScript file. Keep this as a guardrail, not a full generator yet.

- [ ] **Step 5: Wire verification**

Add a `make verify` step or package script that runs:

```bash
node scripts/check-contracts.mjs
```

- [ ] **Step 6: Verify and commit**

Run:

```bash
node scripts/check-contracts.mjs
pnpm --dir web run build
pnpm --dir mobile run typecheck
make verify
```

Commit:

```bash
git add contracts/api.schema.json scripts/check-contracts.mjs web/src/domain.ts mobile/src/api/types.ts Makefile
git commit -m "chore: add shared api contract guardrails"
```

---

## Task 4: Deepen auth/session Seam

**Findings covered:** 5.

**Files:**

- Create: `backend/internal/app/session_exchange.go`
- Create: `backend/internal/app/session_exchange_test.go`
- Modify: `backend/internal/app/auth.go:23-334`
- Modify: `backend/internal/app/app.go:145-153`, `189-197`
- Create: `mobile/src/modules/session/sessionAdapter.ts`
- Modify: `mobile/src/api/client.ts:4-113`
- Modify: `mobile/src/auth/AuthContext.tsx`
- Modify: `mobile/src/auth/sessionCookieStore.ts`

- [ ] **Step 1: Add backend session exchange tests**

Test priority order: `X-Subcult-Session-Token`, `Authorization: Bearer`, `X-Subcult-Session`, cookie. Test response emission sets cookie and `X-Subcult-Session`.

- [ ] **Step 2: Implement backend session exchange Module**

Move transport parsing from `sessionTokenFromRequest` into `session_exchange.go` while preserving current compatibility.

Target backend Interface:

```go
func sessionTokenFromRequest(r *http.Request) string
func (a *App) setSession(w http.ResponseWriter, token string, expiresAt time.Time)
```

Keep these existing names to minimize caller churn; move Implementation behind them.

- [ ] **Step 3: Add mobile session Adapter**

Create a mobile Module with this Interface:

```ts
export type SessionHeaders = Record<string, string>;
export async function loadSessionHeaders(): Promise<SessionHeaders>;
export async function absorbSessionHeaders(headers: Headers): Promise<void>;
export async function clearSession(): Promise<void>;
```

Its Implementation can still store `subcult_session=...`, but callers should not parse it.

- [ ] **Step 4: Update mobile API client**

Replace manual cookie/header construction in `mobile/src/api/client.ts` with `loadSessionHeaders()` and `absorbSessionHeaders()`.

- [ ] **Step 5: Verify and commit**

Run:

```bash
go test ./... -run Session
pnpm --dir mobile run typecheck
make verify
```

Commit:

```bash
git add backend/internal/app/session_exchange.go backend/internal/app/session_exchange_test.go backend/internal/app/auth.go backend/internal/app/app.go mobile/src/modules/session/sessionAdapter.ts mobile/src/api/client.ts mobile/src/auth/AuthContext.tsx mobile/src/auth/sessionCookieStore.ts
git commit -m "refactor: deepen session exchange seam"
```

---

## Task 5: Deepen Event lifecycle Module

**Findings covered:** 1.

**Files:**

- Create: `backend/internal/app/event_lifecycle.go`
- Create: `backend/internal/app/event_lifecycle_test.go`
- Modify: `backend/internal/app/events.go`
- Create: `web/src/modules/eventLifecycle/eventLifecycle.ts`
- Modify: `web/src/views/EventEditorView.tsx`
- Create: `mobile/src/modules/events/eventLifecycle.ts`
- Modify: `mobile/app/event-edit.tsx`

- [ ] **Step 1: Write backend lifecycle tests**

Cover these cases:

- Draft can publish when required Public Event Page fields exist.
- Draft cannot End of Night.
- Published can End of Night.
- End of Night cannot be edited in ways current code forbids.
- Ticket Allocation cannot drop below reserved Ticket count.

- [ ] **Step 2: Implement backend lifecycle Module**

Create constants and helpers:

```go
type eventStatus string

const (
    eventStatusDraft      eventStatus = "draft"
    eventStatusPublished  eventStatus = "published"
    eventStatusEndOfNight eventStatus = "end_of_night"
)

type eventLifecycleView struct {
    CanPublish       bool `json:"canPublish"`
    CanEndOfNight    bool `json:"canEndOfNight"`
    CanEditPricing   bool `json:"canEditPricing"`
    CanEditCapacity  bool `json:"canEditCapacity"`
    IsClosedOut      bool `json:"isClosedOut"`
}
```

Use it internally first; expose derived fields later only if needed.

- [ ] **Step 3: Delegate backend handlers**

Replace inline status checks in `handlePublishEvent`, `handleUpdateEvent`, and `handleEndOfNight` with lifecycle helpers.

- [ ] **Step 4: Add client display helpers**

Move status copy and derived flags out of web/mobile screens into client Modules. Do not duplicate backend invariants; use helpers for display only.

- [ ] **Step 5: Verify and commit**

Run:

```bash
go test ./... -run EventLifecycle
pnpm --dir web run test
pnpm --dir mobile run typecheck
make verify
```

Commit:

```bash
git add backend/internal/app/event_lifecycle.go backend/internal/app/event_lifecycle_test.go backend/internal/app/events.go web/src/modules/eventLifecycle web/src/views/EventEditorView.tsx mobile/src/modules/events mobile/app/event-edit.tsx
git commit -m "refactor: deepen event lifecycle module"
```

---

## Task 6: Deepen Ticket / Payment / Door Record Module

**Findings covered:** 6.

**Files:**

- Create: `backend/internal/app/ticket_journey.go`
- Create: `backend/internal/app/ticket_journey_test.go`
- Modify: `backend/internal/app/tickets.go`
- Modify: `backend/internal/app/payments.go`
- Create: `web/src/modules/tickets/ticketJourney.ts`
- Modify: `web/src/views/PublicEventView.tsx`, `web/src/views/TicketView.tsx`, `web/src/views/DoorView.tsx`
- Create: `mobile/src/modules/tickets/ticketJourney.ts`
- Modify: `mobile/app/ticket.tsx`, `mobile/app/door.tsx`, `mobile/src/tickets/walletStore.ts`

- [ ] **Step 1: Write backend Ticket journey tests**

Cover free reservation, paid pending reservation, paid webhook fulfillment, capacity, Door lookup, Door check-in, and checked-in Ticket display.

- [ ] **Step 2: Implement backend Ticket journey helpers**

Create a Module that derives Ticket journey state without naming Stripe in the domain state:

```go
type ticketJourneyState string

const (
    ticketJourneyFreeReserved ticketJourneyState = "free_reserved"
    ticketJourneyPaymentPending ticketJourneyState = "payment_pending"
    ticketJourneyReady ticketJourneyState = "ready"
    ticketJourneyCheckedIn ticketJourneyState = "checked_in"
    ticketJourneyCancelled ticketJourneyState = "cancelled"
)
```

- [ ] **Step 3: Keep Stripe as Adapter**

Ensure Stripe-specific checkout/session/webhook code remains in `payments.go` or a Stripe-specific Adapter, while `tickets.go` talks in Payment, Ticket, and Door Record terms.

- [ ] **Step 4: Add web/mobile display helpers**

Move repeated status labels, pending copy, checked-in copy, and code display into `ticketJourney.ts` Modules.

- [ ] **Step 5: Verify and commit**

Run:

```bash
go test ./... -run Ticket
pnpm --dir web run test
pnpm --dir mobile run typecheck
make verify
```

Commit:

```bash
git add backend/internal/app/ticket_journey.go backend/internal/app/ticket_journey_test.go backend/internal/app/tickets.go backend/internal/app/payments.go web/src/modules/tickets web/src/views/PublicEventView.tsx web/src/views/TicketView.tsx web/src/views/DoorView.tsx mobile/src/modules/tickets mobile/app/ticket.tsx mobile/app/door.tsx mobile/src/tickets/walletStore.ts
git commit -m "refactor: deepen ticket door journey"
```

---

## Task 7: Deepen media storage Adapter

**Findings covered:** 8.

**Files:**

- Create: `backend/internal/app/media_storage.go`
- Create: `backend/internal/app/media_storage_test.go`
- Modify: `backend/internal/app/app.go:24-35`
- Modify: `backend/internal/app/media.go`
- Modify: `docs/stacks.md:98-113` if config names or behaviour change

- [ ] **Step 1: Write media storage tests**

Use a fake Adapter to test:

- missing config returns service unavailable for upload;
- successful upload returns Event with `imageUrl`;
- failed Adapter upload returns stable error.

- [ ] **Step 2: Introduce Adapter Interface**

Create:

```go
type mediaStorage interface {
    UploadEventImage(ctx context.Context, eventID string, filename string, contentType string, body io.Reader) (publicURL string, err error)
}
```

- [ ] **Step 3: Construct Adapter at startup**

Add `media mediaStorage` to `App`. Initialize it in `New` from config. Do not rebuild storage on every request.

- [ ] **Step 4: Update upload handler**

Make `handleUploadEventImage` call `a.media.UploadEventImage(...)` and persist returned URL.

- [ ] **Step 5: Verify and commit**

Run:

```bash
go test ./... -run Media
make verify
```

Commit:

```bash
git add backend/internal/app/media_storage.go backend/internal/app/media_storage_test.go backend/internal/app/app.go backend/internal/app/media.go docs/stacks.md
git commit -m "refactor: deepen event media storage adapter"
```

---

## Task 8: Deepen Run of Show Module

**Findings covered:** 9.

**Files:**

- Create: `mobile/src/modules/runOfShow/runOfShowModel.ts`
- Modify: `mobile/app/run-of-show.tsx`
- Create: `web/src/modules/runOfShow/runOfShowModel.ts`
- Modify: `web/src/views/EventEditorView.tsx`
- Optionally create backend tests if backend status transition rules are currently untested.

- [ ] **Step 1: Add Run of Show model tests**

Test sorting, time parsing, invalid end-before-start, edit payload generation, and status label generation.

- [ ] **Step 2: Implement mobile Run of Show model**

Move these out of `mobile/app/run-of-show.tsx`:

- `parseOptionalDateTime`
- `toLocalInput`
- `sortStaffingItems`
- create/update payload building
- status display mapping

- [ ] **Step 3: Update mobile screen**

Keep `mobile/app/run-of-show.tsx` focused on rendering and calling commands. It should not know date parsing details beyond displaying errors from the model.

- [ ] **Step 4: Share the same display rules on web**

Move staffing grouping/count display helpers from `EventEditorView.tsx` into `web/src/modules/runOfShow/runOfShowModel.ts`.

- [ ] **Step 5: Verify and commit**

Run:

```bash
pnpm --dir mobile run typecheck
pnpm --dir web run test
make verify
```

Commit:

```bash
git add mobile/src/modules/runOfShow mobile/app/run-of-show.tsx web/src/modules/runOfShow web/src/views/EventEditorView.tsx
git commit -m "refactor: deepen run of show module"
```

---

## Task 9: Deepen Event Operations Record editor shell

**Findings covered:** 2.

**Files:**

- Create: `web/src/modules/eventEditor/eventEditorModel.ts`
- Create: `web/src/modules/eventEditor/eventEditorLoaders.ts`
- Modify: `web/src/views/EventEditorView.tsx`

- [ ] **Step 1: Extract pure editor model functions**

Move `formFromEvent`, `buildPayload`, `formsMatch`, settlement adjustment form helpers, archive display helpers, role/application grouping, staffing grouping, and commitment counts into focused Modules.

- [ ] **Step 2: Extract loader orchestration**

Create loader functions for:

- base Event and optional Event Report;
- Archive;
- Settlement;
- Event Roles and Participation Applications;
- staffing/Run of Show;
- Commitments;
- Templates;
- Notifications and Reminders.

- [ ] **Step 3: Keep view stable**

Change `EventEditorView.tsx` to call these Modules without altering routes, headings, or user-visible copy except where duplicate copy is centralized.

- [ ] **Step 4: Verify and commit**

Run:

```bash
pnpm --dir web run test
pnpm --dir web run build
make verify
```

Commit:

```bash
git add web/src/modules/eventEditor web/src/views/EventEditorView.tsx
git commit -m "refactor: deepen event operations editor shell"
```

---

## Task 10: Deepen Workspace home Module

**Findings covered:** 3.

**Files:**

- Create: `web/src/modules/workspace/workspaceModel.ts`
- Create: `web/src/modules/workspace/workspaceLoaders.ts`
- Modify: `web/src/views/WorkspaceView.tsx`
- Modify: `web/src/App.test.tsx`

- [ ] **Step 1: Extract Workspace state model**

Move ordering/counting helpers for Events, Archives, Contacts, Commitments, Templates, and Reminders into `workspaceModel.ts`.

- [ ] **Step 2: Extract Workspace loaders**

Move load functions for current Workspace, fallback Workspace, archive search, Contacts, Commitments, Templates, Reminders, and dev email outbox into `workspaceLoaders.ts`.

- [ ] **Step 3: Keep destructive actions behind one Interface**

Move Event Template deletion and reset behaviour into a function that takes current template state and returns next template state plus notice.

- [ ] **Step 4: Add focused tests**

Test template deletion state, active edit reset, denied private panels, and archive query behaviour.

- [ ] **Step 5: Verify and commit**

Run:

```bash
pnpm --dir web run test
pnpm --dir web run build
make verify
```

Commit:

```bash
git add web/src/modules/workspace web/src/views/WorkspaceView.tsx web/src/App.test.tsx
git commit -m "refactor: deepen workspace home module"
```

---

## Task 11: Deepen mobile Event edit Module

**Findings covered:** 1, 8, and mobile side of 2.

**Files:**

- Create: `mobile/src/modules/events/eventEditModel.ts`
- Modify: `mobile/app/event-edit.tsx`

- [ ] **Step 1: Add Event edit model tests if mobile test harness exists**

Cover default form, form from Event, payload generation, invalid date, invalid Ticket Allocation, fixed-price cents conversion, readiness warnings, and image-selected notice state.

- [ ] **Step 2: Extract Event edit model**

Move from `mobile/app/event-edit.tsx`:

- `FormState`
- `emptyForm`
- `defaultStartsAtInput`
- `formFromEvent`
- `buildPayload`
- `requirePayload`
- `readinessWarnings`

- [ ] **Step 3: Keep screen as Adapter**

The screen should orchestrate Expo ImagePicker, navigation, and calls to `createEvent`, `updateEvent`, `uploadEventImage`, and `publishEvent`; domain rules live in the model.

- [ ] **Step 4: Verify and commit**

Run:

```bash
pnpm --dir mobile run typecheck
make verify
```

Commit:

```bash
git add mobile/src/modules/events/eventEditModel.ts mobile/app/event-edit.tsx
git commit -m "refactor: deepen mobile event edit module"
```

---

## Task 12: Deepen shared mobile persistence Adapter

**Findings covered:** supporting 5, 6, and mobile Locality.

**Files:**

- Create: `mobile/src/modules/storage/persistedStore.ts`
- Modify: `mobile/src/auth/sessionCookieStore.ts`
- Modify: `mobile/src/staff/selectionStore.ts`
- Modify: `mobile/src/tickets/walletStore.ts`

- [ ] **Step 1: Create persistence Adapter**

Expose:

```ts
export async function readPersistedValue(key: string): Promise<string | null>;
export async function writePersistedValue(key: string, value: string): Promise<void>;
export async function removePersistedValue(key: string): Promise<void>;
```

Implementation should keep current SecureStore/localStorage branching.

- [ ] **Step 2: Migrate stores**

Update session, staff selection, and ticket wallet stores to use the Adapter. Preserve keys and stored value formats.

- [ ] **Step 3: Verify and commit**

Run:

```bash
pnpm --dir mobile run typecheck
make verify
```

Commit:

```bash
git add mobile/src/modules/storage/persistedStore.ts mobile/src/auth/sessionCookieStore.ts mobile/src/staff/selectionStore.ts mobile/src/tickets/walletStore.ts
git commit -m "refactor: add mobile persistence adapter"
```

---

## Task 13: Isolate Event Discovery as a late-stage Module

**Findings covered:** 7.

**Files:**

- Create: `backend/internal/app/discovery.go`
- Modify: `backend/internal/app/public_discovery.go`
- Create: `web/src/modules/discovery/discoveryModel.ts`
- Modify: `web/src/views/DiscoverView.tsx`
- Create: `mobile/src/modules/discovery/discoveryModel.ts`
- Modify: `mobile/app/index.tsx`
- Optional: create `docs/adr/0004-event-discovery-ahead-of-first-cut.md` if product decision is to keep discovery active.

- [ ] **Step 1: Decide discovery mode**

Choose one:

1. Keep discovery active but explicitly mark it as alpha-ahead-of-v1.
2. Hide discovery from default navigation while keeping direct routes for testing.
3. Disable discovery endpoint except in development.

Recommended: keep active but isolate and document, because README already advertises `/discover`.

- [ ] **Step 2: Add backend discovery policy Module**

Create a small Module that answers whether Event Discovery is enabled and how Public Event Pages are listed. Do not mix it with direct-link `handlePublicEvent`.

- [ ] **Step 3: Extract web/mobile discovery models**

Move query parsing, empty state copy, and result display helpers out of `DiscoverView.tsx` and `mobile/app/index.tsx`.

- [ ] **Step 4: Add ADR if keeping discovery active**

Create ADR 0004 with:

```markdown
# ADR 0004: Keep Alpha Event Discovery Isolated Ahead of First Discovery Cut

## Status

Accepted

## Context

CONTEXT.md says the First Discovery Cut is direct-link only, but the alpha already exposes `/discover` and mobile discovery for rehearsal.

## Decision

Keep Event Discovery active in alpha, but isolate it as a late-stage Module so direct-link Public Event Page behaviour remains the v1 Interface.

## Consequences

- Discovery can be disabled, narrowed, or expanded without changing publication.
- Future search and moderation work has a clear Seam.
- Architecture reviews should not re-suggest deleting discovery solely because it is ahead of the First Discovery Cut.
```

- [ ] **Step 5: Verify and commit**

Run:

```bash
go test ./... -run Discovery
pnpm --dir web run test
pnpm --dir mobile run typecheck
make verify
```

Commit:

```bash
git add backend/internal/app/discovery.go backend/internal/app/public_discovery.go web/src/modules/discovery web/src/views/DiscoverView.tsx mobile/src/modules/discovery mobile/app/index.tsx docs/adr/0004-event-discovery-ahead-of-first-cut.md
git commit -m "refactor: isolate alpha event discovery"
```

---

## Task 14: Final integration review and documentation

**Findings covered:** all.

**Files:**

- Modify: `README.md` if user-facing routes, verification, or architecture conventions changed.
- Modify: `docs/stacks.md` if commands, contract checks, Expo testing, or schema rules changed.
- Modify: `AGENTS.md` only if stack conventions changed.

- [ ] **Step 1: Run full verification**

Run:

```bash
make verify
```

Expected: PASS.

- [ ] **Step 2: Inspect changed architecture**

Run:

```bash
git status
git diff --stat
git diff -- backend/internal/app web/src mobile/src mobile/app docs README.md AGENTS.md
```

Check that no generated cache, secrets, or local data are included.

- [ ] **Step 3: Run review-quality**

Use `review-quality` on the final branch. Ask the reviewer specifically:

- Are the new Modules deeper than the old ones?
- Did any new Interface become too wide?
- Are Adapter seams real, or still hypothetical with one Adapter and no tests?
- Did tests move to the new Interfaces?

- [ ] **Step 4: Fix review findings**

Apply only concrete fixes. If a recommendation requires a product decision, record it as a follow-up issue instead of expanding this refactor.

- [ ] **Step 5: Final commit**

```bash
git add README.md docs/stacks.md AGENTS.md
git commit -m "docs: record architecture deepening conventions"
```

Skip this commit if there are no documentation changes.

---

## Acceptance criteria

- Event lifecycle rules are concentrated in a backend Module with tests.
- Web/mobile status copy and derived display decisions no longer duplicate backend invariants.
- Workspace home and Event Operations Record editor have smaller orchestration Modules and reduced view-file complexity.
- Ticket, Payment, and Door Record behaviour has one tested journey Module; Stripe remains an Adapter.
- Auth/session transport has one backend exchange Module and one mobile Adapter.
- Shared DTO drift is guarded by a contract check.
- Event media storage is constructed at startup and fakeable in tests.
- Run of Show ordering, time parsing, and status rules are outside screen files.
- Event Discovery is explicitly isolated and, if kept ahead of `CONTEXT.md` First Discovery Cut, documented in an ADR.
- Schema authority is explicit for alpha.
- `make verify` passes.

## Risks and guardrails

- **Risk:** refactor becomes a rewrite. **Guardrail:** keep HTTP routes and visible UI stable until each Module has tests.
- **Risk:** generated/shared contracts become heavy too early. **Guardrail:** start with a checker, not a generator.
- **Risk:** one Adapter equals a hypothetical Seam. **Guardrail:** every Adapter must have a fake or test double in tests.
- **Risk:** frontend Modules duplicate backend domain rules. **Guardrail:** frontend Modules may format or present; backend remains source of truth for invariants.
- **Risk:** Event Discovery scope churn. **Guardrail:** isolate first, product decision second.
