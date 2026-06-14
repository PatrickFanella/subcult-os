# Public Event Discovery Implementation Plan

> **For agentic workers:** Execute this plan task-by-task. Recommended path:
> dispatch a fresh subagent per task, review each result with `review-quality`,
> then continue. For complex multi-agent splits, use
> `parallel-feature-development`, `team-composition-patterns`, and
> `team-communication-protocols`. Steps use checkbox (`- [ ]`) syntax for
> tracking.

**Goal:** Make published public events discoverable beyond direct links without exposing private operator, archive, staffing, settlement, application, or workspace-member data.

**Architecture:** Add a read-only public discovery endpoint backed by published `events` only, with a purpose-built summary DTO. The frontend gets a `/discover` page with cards, search-lite, and safety-focused copy; public event detail pages and private workspace/editor routes stay unchanged.

**Tech Stack:** Go `net/http` backend, pgx/Postgres embedded `schema.sql`, React TypeScript SPA, Vitest, `make verify`.

---

## Current State

- Public event detail exists at `/e/{slug}` and `/api/public/events/{slug}`.
- Published events have `public_slug`/`public_url`; drafts and end-of-night events must stay out of discovery.
- Ticket counts, pricing mode, roles, applications, staffing, settlement, and archive data exist, but most of it is private operator state.
- There is no public `/discover` route or `GET /api/public/events` index yet.

## File Structure

- Create: `backend/internal/app/public_discovery.go` for public discovery DTO/query handlers.
- Modify: `backend/internal/app/app.go` to register discovery route.
- Modify: `backend/internal/app/lifecycle_test.go` for backend visibility/search/count tests.
- Modify: `web/src/domain.ts` for discovery DTOs.
- Create: `web/src/views/DiscoverView.tsx` for the public browse/search page.
- Modify: `web/src/App.tsx` to route `/discover`.
- Modify: `web/src/views/PublicEventView.tsx` only for a small discover back-link if useful.
- Modify: `web/src/App.test.tsx` for frontend discovery coverage.
- Modify: `README.md` only after the feature works, to mention `/discover`.

---

### Task 1: Public Discovery Read Model

**Files:**
- Create: `backend/internal/app/public_discovery.go`
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`
- Modify: `docs/superpowers/plans/2026-06-14-public-event-discovery.md`

- [x] **Step 1: Write failing backend coverage**

Add `TestPublicEventDiscoveryAPI` to `backend/internal/app/lifecycle_test.go`:

```go
func TestPublicEventDiscoveryAPI(t *testing.T) {
    fx := newLifecycleFixture(t)
    draft := createEvent(t, fx, "Draft Night", 20)
    published := createEventWithPricing(t, fx, "Published Market", 40, "fixed", 1500, "usd")
    closed := createEvent(t, fx, "Closed Night", 30)

    publishedID := mustString(t, published, "id")
    closedID := mustString(t, closed, "id")
    publishedAfterPublish := publishEvent(t, fx, publishedID)
    publishEvent(t, fx, closedID)
    postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+closedID+"/end-of-night", map[string]any{}, http.StatusOK)

    publishedSlug := mustString(t, publishedAfterPublish, "publicSlug")
    postJSON(t, fx.app, nil, "/api/public/events/"+publishedSlug+"/reservations", map[string]any{
        "name": "Ada",
        "email": "ada@example.com",
    }, http.StatusOK)

    resp := getJSON(t, fx.app, nil, "/api/public/events", http.StatusOK)
    events := resp.JSON.([]any)
    if len(events) != 1 {
        t.Fatalf("expected one discoverable event, got %#v", events)
    }
    event := mustObject(t, events[0])
    if event["title"] != "Published Market" || event["status"] != "published" || event["publicUrl"] == nil {
        t.Fatalf("unexpected public event summary: %#v", event)
    }
    if event["workspaceId"] != nil || event["staffingOpenCount"] != nil || event["settlementSummary"] != nil || event["archive"] != nil {
        t.Fatalf("discovery leaked private fields: %#v", event)
    }
    if int(event["remainingTickets"].(float64)) != 39 || event["isFull"] != false {
        t.Fatalf("unexpected availability: %#v", event)
    }

    titles := []string{mustString(t, draft, "title"), "Closed Night"}
    for _, title := range titles {
        if strings.Contains(fmt.Sprintf("%#v", events), title) {
            t.Fatalf("hidden event %q leaked in discovery: %#v", title, events)
        }
    }
}
```

- [x] **Step 2: Add backend DTO and route**

In `backend/internal/app/app.go`, register before slug detail routes:

```go
a.mux.HandleFunc("GET /api/public/events", a.handleListPublicEvents)
```

Create `backend/internal/app/public_discovery.go`:

```go
package app

import (
    "net/http"
    "time"
)

type publicEventSummaryDTO struct {
    ID               string  `json:"id"`
    Title            string  `json:"title"`
    StartsAt         string  `json:"startsAt"`
    PublicDescription string `json:"publicDescription"`
    LocationDisplay  string  `json:"locationDisplay"`
    PricingMode      string  `json:"pricingMode"`
    TicketPriceCents int     `json:"ticketPriceCents"`
    TicketCurrency   string  `json:"ticketCurrency"`
    RemainingTickets int     `json:"remainingTickets"`
    IsFull           bool    `json:"isFull"`
    Status           string  `json:"status"`
    PublicSlug       string  `json:"publicSlug"`
    PublicURL        string  `json:"publicUrl"`
}

func (a *App) handleListPublicEvents(w http.ResponseWriter, r *http.Request) {
    if a.db == nil {
        writeJSON(w, http.StatusOK, []publicEventSummaryDTO{})
        return
    }
    rows, err := a.db.Query(r.Context(), `
        select id, title, starts_at, public_description, location_display,
               ticket_allocation, pricing_mode, ticket_price_cents, ticket_currency,
               (select count(*) from tickets t where t.event_id = events.id and t.payment_status <> 'cancelled') as reserved_count,
               public_slug
        from events
        where status = 'published' and public_slug is not null
        order by starts_at asc, created_at asc
        limit 100
    `)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "could not list public events")
        return
    }
    defer rows.Close()

    events := []publicEventSummaryDTO{}
    for rows.Next() {
        var row struct {
            ID, Title, PublicDescription, LocationDisplay, PricingMode, TicketCurrency, PublicSlug string
            StartsAt time.Time
            TicketAllocation, TicketPriceCents, ReservedCount int
        }
        if err := rows.Scan(&row.ID, &row.Title, &row.StartsAt, &row.PublicDescription, &row.LocationDisplay, &row.TicketAllocation, &row.PricingMode, &row.TicketPriceCents, &row.TicketCurrency, &row.ReservedCount, &row.PublicSlug); err != nil {
            writeError(w, http.StatusInternalServerError, "could not read public event")
            return
        }
        remaining := row.TicketAllocation - row.ReservedCount
        if remaining < 0 {
            remaining = 0
        }
        events = append(events, publicEventSummaryDTO{
            ID: row.ID, Title: row.Title, StartsAt: row.StartsAt.UTC().Format(time.RFC3339Nano),
            PublicDescription: row.PublicDescription, LocationDisplay: row.LocationDisplay,
            PricingMode: row.PricingMode, TicketPriceCents: row.TicketPriceCents,
            TicketCurrency: row.TicketCurrency, RemainingTickets: remaining, IsFull: remaining == 0,
            Status: "published", PublicSlug: row.PublicSlug, PublicURL: a.publicEventURL(row.PublicSlug),
        })
    }
    if err := rows.Err(); err != nil {
        writeError(w, http.StatusInternalServerError, "could not list public events")
        return
    }
    writeJSON(w, http.StatusOK, events)
}
```

- [x] **Step 3: Add TypeScript DTO**

In `web/src/domain.ts`:

```ts
export interface PublicEventSummaryDTO {
  id: string;
  title: string;
  startsAt: string;
  publicDescription: string;
  locationDisplay: string;
  pricingMode: 'free' | 'fixed';
  ticketPriceCents: number;
  ticketCurrency: string;
  remainingTickets: number;
  isFull: boolean;
  status: 'published';
  publicSlug: string;
  publicUrl: string;
}
```

- [x] **Step 4: Verify and commit**

Run: `gofmt -w backend/internal/app/public_discovery.go backend/internal/app/app.go backend/internal/app/lifecycle_test.go && make verify`

Commit:

```bash
git add backend/internal/app/public_discovery.go backend/internal/app/app.go backend/internal/app/lifecycle_test.go web/src/domain.ts docs/superpowers/plans/2026-06-14-public-event-discovery.md
git commit -m "Add public event discovery API"
```

---

### Task 2: Discovery Page Shell

**Files:**
- Create: `web/src/views/DiscoverView.tsx`
- Modify: `web/src/App.tsx`
- Modify: `web/src/App.test.tsx`

- [ ] **Step 1: Add route and smoke test**

In `web/src/App.tsx`, import `DiscoverView` and route `/discover` before workspace fallback:

```tsx
if (pathname === '/discover') {
  return <DiscoverView />;
}
```

In `web/src/App.test.tsx`, add a route smoke test that mocks state/API patterns already used in the file and asserts `Discover events`, event title, date/location, and `View event` link.

- [ ] **Step 2: Implement `DiscoverView`**

Create `web/src/views/DiscoverView.tsx` with:

- state: `events`, `loading`, `error`
- effect: `api<PublicEventSummaryDTO[]>('/api/public/events')`
- empty copy: `No published events are discoverable yet.`
- card fields: title, date, location, description, pricing, remaining tickets
- link: `<a href={event.publicUrl}>View event</a>`

- [ ] **Step 3: Verify and commit**

Run: `make verify`

Commit:

```bash
git add web/src/views/DiscoverView.tsx web/src/App.tsx web/src/App.test.tsx
git commit -m "Add public discovery page"
```

---

### Task 3: Search-Lite and Filters

**Files:**
- Modify: `backend/internal/app/public_discovery.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/views/DiscoverView.tsx`
- Modify: `web/src/App.test.tsx`

- [ ] **Step 1: Add backend search tests**

Extend `TestPublicEventDiscoveryAPI` or add `TestPublicEventDiscoverySearchAPI` covering:

- `?q=market` matches title/description/location case-insensitively
- `?q=missing` returns `[]`
- `%` and `_` are literal search terms, not wildcards. Create one published event whose title or description contains `%` and another whose title contains `_`, then assert `q=%` and `q=_` match only the literal event(s), not every published event.
- query over 120 runes returns `400` with `search query is too long`
- drafts and end-of-night events never match even if text matches

- [ ] **Step 2: Implement safe query filtering**

In `handleListPublicEvents`, read `q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))`. Reject `utf8.RuneCountInString(q) > 120`. Use `position($1 in lower(title)) > 0` style search, not raw `LIKE`, for title, public description, and location. Empty query returns latest/upcoming published list.

- [ ] **Step 3: Add frontend search UI**

In `DiscoverView.tsx`, add search input, submit, reset, URL `?q=` sync, and copy. Use the same cancellation/request sequencing pattern as `PublicEventView.tsx`: each load captures a local `cancelled` flag or monotonically increasing request ID so an older slow response cannot overwrite a newer search/reset result.

- no query empty: `No published events are discoverable yet.`
- query empty: `No events matched your search.`
- reset button clears `q`, reloads default list, and clears stale errors.

- [ ] **Step 4: Verify and commit**

Run: `make verify`

Commit:

```bash
git add backend/internal/app/public_discovery.go backend/internal/app/lifecycle_test.go web/src/views/DiscoverView.tsx web/src/App.test.tsx
git commit -m "Add public event discovery search"
```

---

### Task 4: Public Trust and Availability Context

**Files:**
- Modify: `backend/internal/app/public_discovery.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`
- Modify: `web/src/views/DiscoverView.tsx`
- Modify: `web/src/App.test.tsx`

- [ ] **Step 1: Add host display to DTO safely**

Extend `publicEventSummaryDTO` and `PublicEventSummaryDTO` with:

```go
WorkspaceName string `json:"workspaceName"`
```

Join `workspaces w on w.id = e.workspace_id` and return `w.name` only. Do not expose workspace ID, member emails, or role data.

- [ ] **Step 2: Add application-open context without role details**

Add `ApplicationsOpen bool json:"applicationsOpen"` from an `exists` scalar subquery checking for at least one active public role for the event. Do not expose exact role counts or application counts in discovery.

- [ ] **Step 3: Render trust/context cards**

In `DiscoverView.tsx`, render:

- `Hosted by {workspaceName}`
- `Free` or formatted price with currency
- `{remainingTickets} tickets left` / `Sold out`
- `Applications open` only when `applicationsOpen` is true

- [ ] **Step 4: Verify and commit**

Run: `make verify`

Commit:

```bash
git add backend/internal/app/public_discovery.go backend/internal/app/lifecycle_test.go web/src/domain.ts web/src/views/DiscoverView.tsx web/src/App.test.tsx
git commit -m "Add discovery event context"
```

---

### Task 5: Navigation, Docs, and Safety Regression Pass

**Files:**
- Modify: `web/src/views/PublicEventView.tsx`
- Modify: `web/src/views/WorkspaceView.tsx`
- Modify: `web/src/App.test.tsx`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `README.md`

- [ ] **Step 1: Add navigation affordances**

Add a small `Discover more events` link to `PublicEventView.tsx`. Add a `Public discovery` link in `WorkspaceView.tsx` header/side rail so operators can see the public browse surface.

- [ ] **Step 2: Strengthen safety tests**

Add backend assertions that discovery JSON does not include these substrings/keys: `applicantEmail`, `message`, `staffingItems`, `settlement`, `archive`, `notes`, `workspaceId`. Keep this on the discovery response, not private APIs.

- [ ] **Step 3: Document the public discovery URL**

In `README.md`, add one bullet to the current shipped slice section:

```md
- Public discovery page at `/discover` lists published events without exposing private workspace, archive, staffing, settlement, or application data.
```

- [ ] **Step 4: Final verify and commit**

Run: `make verify`

Commit:

```bash
git add web/src/views/PublicEventView.tsx web/src/views/WorkspaceView.tsx web/src/App.test.tsx backend/internal/app/lifecycle_test.go README.md
git commit -m "Document and link public discovery"
```

---

## Final Verification

- [ ] Run `make verify`.
- [ ] Run `git status --short --branch` and confirm the working tree is clean.
- [ ] Run `git log --oneline -10` and confirm the five public discovery commits are present.

## Out of Scope

- SEO metadata, sitemaps, robots, or structured data.
- Public archive pages.
- Organizer profile pages.
- Geo/radius search.
- Pagination beyond a fixed limit of 100.
- Exposing staffing, settlement, application, archive notes, participant emails, or workspace IDs.

## Self-Review

- Spec coverage: the plan covers public discovery API, page shell, search-lite, context cards, navigation/docs, and privacy regression tests.
- Placeholder scan: no task depends on future unspecified work.
- Type consistency: Go `publicEventSummaryDTO` maps to TypeScript `PublicEventSummaryDTO`; public event detail remains `PublicEventDTO`.
