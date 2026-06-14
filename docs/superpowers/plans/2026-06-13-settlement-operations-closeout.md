# Settlement Operations Closeout Implementation Plan

> **For agentic workers:** Execute this plan task-by-task. Recommended path:
> dispatch a fresh subagent per task, review each result with `review-quality`,
> then continue. For complex multi-agent splits, use
> `parallel-feature-development`, `team-composition-patterns`, and
> `team-communication-protocols`. Steps use checkbox (`- [ ]`) syntax for
> tracking.

**Goal:** Turn end-of-night settlement summaries into a small operational closeout system with persisted settlements, review API, manual adjustments, UI, and finalization guards.

**Architecture:** Keep the current immutable event report as the operator-facing snapshot, but add `event_settlements` as durable money state and `event_settlement_adjustments` as append-only manual money lines. End-of-night creates the settlement idempotently; later API/UI calls review, adjust, and finalize it without changing ticket/payment fulfillment behavior.

**Tech Stack:** Go `net/http` backend, pgx/Postgres embedded `schema.sql`, React TypeScript SPA, Vitest, `make verify`.

---

### Task 1: Settlement Record Foundation

**Files:**
- Modify: `backend/internal/app/schema.sql`
- Modify: `backend/internal/app/events.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`

- [ ] Add `event_settlements` with one row per event and fields matching the existing report settlement summary.
- [ ] Make end-of-night insert the settlement row idempotently in the same transaction as the report.
- [ ] Keep existing report JSON shape compatible.
- [ ] Add tests proving row creation and repeated end-of-night idempotency.
- [ ] Run `gofmt` and `make verify`.
- [ ] Commit as `Add event settlement records`.

### Task 2: Settlement Review API

**Files:**
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/events.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`

- [ ] Add `GET /api/events/{eventID}/settlement`.
- [ ] Allow owners and members who can already view the event report.
- [ ] Return 404 before end-of-night/settlement creation.
- [ ] Return settlement totals, status, timestamps, and empty adjustments array.
- [ ] Run `gofmt` and `make verify`.
- [ ] Commit as `Add settlement review API`.

### Task 3: Manual Settlement Adjustments

**Files:**
- Modify: `backend/internal/app/schema.sql`
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/events.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`

- [ ] Add append-only `event_settlement_adjustments` with amount, label, reason, creator, and created time.
- [ ] Add `POST /api/events/{eventID}/settlement/adjustments` owner-only.
- [ ] Validate non-zero amount and non-empty label.
- [ ] Return settlement with `netTotalCents = grossPaidRevenueCents + sum(adjustments)`.
- [ ] Audit adjustment creation.
- [ ] Run `gofmt` and `make verify`.
- [ ] Commit as `Add settlement adjustments`.

### Task 4: Settlement UI Panel

**Files:**
- Modify: `web/src/api.ts` if helpers are needed
- Modify: `web/src/domain.ts`
- Modify: `web/src/views/EventEditorView.tsx`
- Modify: `web/src/App.test.tsx`

- [ ] Load settlement details when an event is end-of-night.
- [ ] Show gross revenue, adjustments, net total, status, and adjustment rows.
- [ ] Add a small owner-oriented adjustment form.
- [ ] Refresh settlement after adding an adjustment.
- [ ] Add frontend regression coverage.
- [ ] Run `make verify`.
- [ ] Commit as `Add settlement review UI`.

### Task 5: Settlement Finalization Guards

**Files:**
- Modify: `backend/internal/app/schema.sql`
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/events.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`
- Modify: `web/src/views/EventEditorView.tsx`
- Modify: `web/src/App.test.tsx`

- [ ] Add `POST /api/events/{eventID}/settlement/finalize` owner-only.
- [ ] Mark settlement `finalized` with timestamp/person.
- [ ] Reject new adjustments once finalized.
- [ ] Keep existing report immutable.
- [ ] Show finalized/locked state in UI.
- [ ] Run `make verify`.
- [ ] Commit as `Finalize event settlements`.
