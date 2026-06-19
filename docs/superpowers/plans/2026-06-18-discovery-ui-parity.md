# Discovery and UI Parity Implementation Plan

> **For agentic workers:** Execute this plan task-by-task. Recommended path:
> dispatch a fresh subagent per task, review each result with `review-quality`,
> then continue. For complex multi-agent splits, use
> `parallel-feature-development`, `team-composition-patterns`, and
> `team-communication-protocols`. Steps use checkbox (`- [ ]`) syntax for
> tracking.

**Goal:** Productize alpha Event Discovery while making the public web journey visually and behaviorally match the mobile app across Discovery, Public Event Page, Ticket, and Door.

**Architecture:** Keep Discovery isolated as the alpha-ahead-of-v1 Module from ADR 0004. Treat mobile as the visual reference for the public journey, but do not redesign authenticated Workspace/Event Operations Record surfaces in this sprint. Use shared copy/view-model helpers where practical; do not share React components across web and mobile.

**Tech Stack:** Go API with `net/http`, Vite React TypeScript with Tailwind-style utility classes and Vitest, Expo SDK 54 React Native mobile with pure-module Vitest tests, shared contract checker, Makefile verification.

---

## Scope and design decision

The user direction is: **“we definitely need to make the web pages match the app.”** For this sprint, “web pages” means the public attendee/operator journey:

1. `web/src/views/DiscoverView.tsx`
2. `web/src/views/PublicEventView.tsx`
3. `web/src/views/TicketView.tsx`
4. `web/src/views/DoorView.tsx`

The mobile app remains the visual reference:

- light surfaces
- rounded cards
- simple black/neutral primary actions
- full-bleed media where useful
- concise mobile-style copy
- clear loading, empty, and error states

Out of scope:

- Global rewrite of Workspace/EventEditor dark operator UI.
- Shared React component library across web and mobile.
- Algorithmic feeds, recommendations, moderation queues, public profiles, or geography filters.
- Changing payment provider language or Ticket code semantics.

## File structure target

### QA / docs

- Create: `docs/qa/discovery-ui-parity-2026-06.md` — baseline/final UI parity QA matrix.
- Modify: `docs/adr/0004-event-discovery-ahead-of-first-cut.md` only if Discovery behavior expands beyond existing alpha browsing/search.

### Backend

- Modify: `backend/internal/app/public_discovery.go` only if mobile search exposes an existing query parameter more consistently.
- Modify: `backend/internal/app/discovery.go` and `backend/internal/app/discovery_test.go` only if policy/copy around active alpha Discovery needs tests.
- Avoid backend DTO changes unless a missing field blocks parity.

### Web

- Create: `web/src/modules/publicUi/publicUi.ts` — public journey style/copy helpers for light app-parity web surfaces.
- Create: `web/src/modules/publicUi/publicUi.test.ts` — exact copy/style helper tests.
- Modify: `web/src/modules/discovery/discoveryModel.ts` and tests — align Discovery copy/query helpers with mobile.
- Modify: `web/src/views/DiscoverView.tsx` — light app-parity Discovery page.
- Modify: `web/src/views/PublicEventView.tsx` — mobile-inspired public Event detail hierarchy while preserving role applications.
- Modify: `web/src/views/TicketView.tsx` — add web QR display and align Ticket state treatment with mobile.
- Modify: `web/src/views/DoorView.tsx` — align Door lookup/check-in layout and auth/empty/error affordances with mobile.

### Mobile

- Modify: `mobile/src/modules/discovery/discoveryModel.ts` and tests — add search/empty/error helpers matching web copy.
- Modify: `mobile/app/index.tsx` — add mobile Discovery search while preserving vertical feed behavior.
- Modify: `mobile/app/event-detail.tsx` only if role/application parity is pulled into this sprint after Tasks 1–4 are stable.

---

## Task 1: Record Discovery + UI parity baseline

**Files:**

- Create: `docs/qa/discovery-ui-parity-2026-06.md`

- [ ] **Step 1: Create the QA document**

Create `docs/qa/discovery-ui-parity-2026-06.md`:

```markdown
# Discovery and UI Parity QA — 2026-06

## Goal

Verify that Event Discovery stays active in alpha and that the public web journey visually matches the mobile app across Discovery, Public Event Page, Ticket, and Door.

## Design reference

Mobile app is the visual reference for public attendee/operator surfaces:

- light surfaces
- rounded cards
- black/neutral primary actions
- concise copy
- full-bleed media where useful
- clear empty/loading/error states

## Environment

- Host: Linux
- Browser: Not run
- Expo/device: Not run
- API URL used by mobile: Not recorded

## Required commands

```bash
make verify
```

## Baseline command results

| Command | Result | Notes |
| --- | --- | --- |
| `make verify` | Not run | |
| `make up-build` | Not run | |
| `make smoke` | Not run | |
| `make down` | Not run | |

## UI parity matrix

| Area | Web path | Mobile path | Expected parity | Baseline | Final | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| Discovery copy | `/discover` | Expo `/` | Loading, empty, error, and CTA copy match | Not run | Not run | |
| Discovery search | `/discover?q=...` | Expo `/` | Both support attendee search with contextual empty state | Not run | Not run | |
| Discovery visual style | `/discover` | Expo `/` | Web uses mobile-inspired light cards/surfaces | Not run | Not run | |
| Public Event detail | `/e/:slug` | Expo `/event-detail` | Hero, pricing, reserve CTA, and role/application sections feel related | Not run | Not run | |
| Ticket view | `/tickets/:code` | Expo `/ticket` | Ticket code, payment state, and QR/door affordance are clear | Not run | Not run | |
| Door view | `/door` | Expo `/door` | Lookup, result, and check-in states use matching structure/copy | Not run | Not run | |

## Regression log

| ID | Area | Symptom | Fix | Verification |
| --- | --- | --- | --- | --- |

## Automated test additions

| Test file | Behavior covered | Command |
| --- | --- | --- |
```

- [ ] **Step 2: Run baseline verification**

Run:

```bash
make verify
```

Expected: PASS.

- [ ] **Step 3: Record baseline honestly**

If browser/Expo are unavailable, set the baseline/final notes to:

```markdown
browser unavailable in agent environment
Expo/device unavailable in agent environment
```

Do not fabricate visual QA.

---

## Task 2: Add public journey web UI helpers

**Files:**

- Create: `web/src/modules/publicUi/publicUi.ts`
- Create: `web/src/modules/publicUi/publicUi.test.ts`

- [ ] **Step 1: Add light public UI helper module**

Create `web/src/modules/publicUi/publicUi.ts`:

```ts
export const publicPageShellClass = 'min-h-screen bg-[#f5f5f5] text-[#171717]';

export const publicPageInnerClass = 'mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-6 sm:px-6 lg:px-8';

export const publicCardClass = 'rounded-[28px] border border-neutral-200 bg-white p-5 shadow-sm';

export const publicHeroCardClass = 'overflow-hidden rounded-[32px] border border-neutral-200 bg-white shadow-sm';

export const publicPrimaryButtonClass = 'inline-flex items-center justify-center rounded-full bg-[#171717] px-5 py-3 text-sm font-bold text-white transition hover:bg-black disabled:cursor-not-allowed disabled:bg-neutral-300';

export const publicSecondaryButtonClass = 'inline-flex items-center justify-center rounded-full border border-neutral-300 bg-white px-5 py-3 text-sm font-bold text-[#171717] transition hover:bg-neutral-50';

export const publicMutedTextClass = 'text-sm text-neutral-600';

export const publicEyebrowClass = 'text-xs font-black uppercase tracking-[0.24em] text-neutral-500';

export function publicStatusPillClass(tone: 'neutral' | 'success' | 'warning' | 'danger' = 'neutral') {
  switch (tone) {
    case 'success':
      return 'rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 text-xs font-bold text-emerald-700';
    case 'warning':
      return 'rounded-full border border-amber-200 bg-amber-50 px-3 py-1 text-xs font-bold text-amber-700';
    case 'danger':
      return 'rounded-full border border-rose-200 bg-rose-50 px-3 py-1 text-xs font-bold text-rose-700';
    default:
      return 'rounded-full border border-neutral-200 bg-neutral-100 px-3 py-1 text-xs font-bold text-neutral-700';
  }
}
```

- [ ] **Step 2: Add exact helper tests**

Create `web/src/modules/publicUi/publicUi.test.ts`:

```ts
import { describe, expect, it } from 'vitest';

import { publicCardClass, publicPageShellClass, publicPrimaryButtonClass, publicStatusPillClass } from './publicUi';

describe('publicUi', () => {
  it('uses app-parity light public surfaces', () => {
    expect(publicPageShellClass).toContain('bg-[#f5f5f5]');
    expect(publicPageShellClass).toContain('text-[#171717]');
    expect(publicCardClass).toContain('bg-white');
    expect(publicPrimaryButtonClass).toContain('bg-[#171717]');
  });

  it('provides stable status pill tones', () => {
    expect(publicStatusPillClass('success')).toContain('emerald');
    expect(publicStatusPillClass('warning')).toContain('amber');
    expect(publicStatusPillClass('danger')).toContain('rose');
    expect(publicStatusPillClass()).toContain('neutral');
  });
});
```

- [ ] **Step 3: Validate**

Run:

```bash
pnpm --dir web run test -- publicUi
pnpm --dir web run format
```

Expected: PASS.

---

## Task 3: Align Discovery copy and mobile search

**Files:**

- Modify: `web/src/modules/discovery/discoveryModel.ts`
- Modify: `web/src/modules/discovery/discoveryModel.test.ts`
- Modify: `mobile/src/modules/discovery/discoveryModel.ts`
- Modify: `mobile/src/modules/discovery/discoveryModel.test.ts`
- Modify: `mobile/app/index.tsx`

- [ ] **Step 1: Add shared Discovery copy helpers on both platforms**

In both `web/src/modules/discovery/discoveryModel.ts` and `mobile/src/modules/discovery/discoveryModel.ts`, expose equivalent helpers:

```ts
export const discoveryLoadingCopy = 'Loading published Events…';
export const discoverySearchPlaceholder = 'Search published Events';
export const discoveryViewEventLabel = 'View Event';

export function discoveryEmptyTitle(query: string) {
  return query.trim() ? 'No Events match that search yet' : 'No published Events yet';
}

export function discoveryEmptyBody(query: string) {
  return query.trim()
    ? `No published Events matched “${query.trim()}”. Try another search.`
    : 'Published Events will appear here when Hosts share them.';
}

export function discoveryErrorCopy(message?: string | null) {
  return message ? `Could not load published Events: ${message}` : 'Could not load published Events.';
}
```

Preserve existing date/currency formatting helpers unless tests prove drift.

- [ ] **Step 2: Add exact tests on both platforms**

Update web and mobile discovery tests with exact assertions:

```ts
expect(discoveryLoadingCopy).toBe('Loading published Events…');
expect(discoverySearchPlaceholder).toBe('Search published Events');
expect(discoveryViewEventLabel).toBe('View Event');
expect(discoveryEmptyTitle('')).toBe('No published Events yet');
expect(discoveryEmptyTitle('noise')).toBe('No Events match that search yet');
expect(discoveryEmptyBody('noise')).toBe('No published Events matched “noise”. Try another search.');
expect(discoveryErrorCopy('offline')).toBe('Could not load published Events: offline');
```

- [ ] **Step 3: Add mobile search state and API query**

In `mobile/app/index.tsx`, add a search input above the feed using current mobile styling. Use existing public Event listing API support if available. The behavior should match web:

```ts
const [searchQuery, setSearchQuery] = useState('');

useEffect(() => {
  let cancelled = false;
  setLoading(true);
  listPublicEvents(searchQuery.trim())
    .then((events) => {
      if (!cancelled) setEvents(events);
    })
    .catch((error) => {
      if (!cancelled) setError(error instanceof Error ? error.message : 'Unable to load events');
    })
    .finally(() => {
      if (!cancelled) setLoading(false);
    });
  return () => {
    cancelled = true;
  };
}, [searchQuery]);
```

If the API function is named differently, adapt to the existing `mobile/src/api/**` function. Do not change backend SQL semantics.

- [ ] **Step 4: Validate**

Run:

```bash
pnpm --dir mobile run test -- src/modules/discovery/discoveryModel.test.ts
pnpm --dir mobile run typecheck
pnpm --dir web run test -- discovery
make verify
```

Expected: PASS.

---

## Task 4: Restyle web Discovery to match the app

**Files:**

- Modify: `web/src/views/DiscoverView.tsx`
- Modify: `web/src/modules/discovery/discoveryModel.test.ts` only if copy helpers changed.

- [ ] **Step 1: Apply public UI helpers to `DiscoverView`**

Import:

```ts
import { publicCardClass, publicPageInnerClass, publicPageShellClass, publicPrimaryButtonClass, publicMutedTextClass, publicEyebrowClass } from '../modules/publicUi/publicUi';
```

Update the page shell from dark `zinc-*` styling to the public UI helper classes. Keep route `/discover`, URL query sync, API query behavior, result ordering, and click paths unchanged.

- [ ] **Step 2: Use mobile-style card hierarchy**

Each result card should show:

1. image/hero area when `imageUrl` exists
2. date/time
3. title
4. location
5. pricing / remaining capacity
6. `View Event` primary action

Use the existing `PublicEventSummaryDTO` fields only. Do not add backend fields.

- [ ] **Step 3: Preserve contextual empty/error states**

Use the helpers from Task 3:

```ts
discoveryEmptyTitle(searchQuery)
discoveryEmptyBody(searchQuery)
discoveryErrorCopy(error)
```

- [ ] **Step 4: Validate**

Run:

```bash
pnpm --dir web run test -- discovery
pnpm --dir web run format
make verify
```

Expected: PASS.

---

## Task 5: Restyle web Public Event Page to match the app

**Files:**

- Modify: `web/src/views/PublicEventView.tsx`
- Modify: `web/src/App.test.tsx` only if existing tests need stable copy/style updates.

- [ ] **Step 1: Apply public UI shell and cards**

Use `publicPageShellClass`, `publicPageInnerClass`, `publicHeroCardClass`, `publicCardClass`, and button helpers from `web/src/modules/publicUi/publicUi.ts`.

Preserve:

- `/e/:slug` route
- public reservation behavior
- paid checkout behavior
- role applications on web
- all form validation behavior

- [ ] **Step 2: Match mobile visual hierarchy**

Reorder visible hierarchy if needed to match mobile without changing data:

1. full-bleed hero image or neutral gradient block
2. title and Host Workspace Display Name
3. date/time/location
4. pricing/reservation CTA
5. role/application section

- [ ] **Step 3: Add a focused web test only if copy changes**

If visible copy changes, update or add an App test that asserts the intended public Event Page CTA and reservation path still render.

- [ ] **Step 4: Validate**

Run:

```bash
pnpm --dir web run test -- src/App.test.tsx
pnpm --dir web run format
make verify
```

Expected: PASS.

---

## Task 6: Align Ticket view and add QR parity on web

**Files:**

- Modify: `web/src/views/TicketView.tsx`
- Modify: `web/src/modules/tickets/ticketJourney.ts`
- Modify: `web/src/modules/tickets/ticketJourney.test.ts`
- Modify: `web/package.json` and lockfile only if a QR dependency is required.

- [ ] **Step 1: Prefer no new QR dependency if possible**

Inspect current web dependencies. If a QR library is already available, use it. If not, add a small maintained QR component package only after checking package impact.

Preferred command if needed:

```bash
pnpm --dir web add qrcode.react
```

- [ ] **Step 2: Add Ticket QR to web Ticket page**

Render a QR value matching the mobile QR value. If mobile encodes `ticket.ticketUrl`, web should do the same. If web lacks `ticketUrl`, use the current Ticket URL from `window.location.href` and do not change API shape.

- [ ] **Step 3: Align app-parity Ticket styling**

Use the public UI helpers for light surfaces and black primary actions. Preserve raw/chunked Ticket code behavior already accepted in prior reviews.

- [ ] **Step 4: Test helper behavior**

If adding a helper like `ticketQrValue(ticket, fallbackUrl)`, test it:

```ts
expect(ticketQrValue({ ticketUrl: 'https://example.test/t/ABC' } as never, 'fallback')).toBe('https://example.test/t/ABC');
expect(ticketQrValue({ ticketUrl: '' } as never, 'fallback')).toBe('fallback');
```

- [ ] **Step 5: Validate**

Run:

```bash
pnpm --dir web run test -- tickets
pnpm --dir web run format
make verify
```

Expected: PASS.

---

## Task 7: Align Door page layout and state treatment

**Files:**

- Modify: `web/src/views/DoorView.tsx`
- Modify: `web/src/modules/tickets/ticketJourney.ts`
- Modify: `web/src/modules/tickets/ticketJourney.test.ts`

- [ ] **Step 1: Apply public UI helpers to Door page**

Use public light shell/card/button helpers. Preserve route `/door`, lookup API path, check-in API path, duplicate check-in behavior, and raw Ticket code semantics.

- [ ] **Step 2: Align Door copy with mobile**

If mobile has helper names already, mirror exact strings in web ticket journey helpers:

```ts
export function doorCheckInButtonLabel(checkingIn: boolean, checkedIn: boolean) {
  if (checkingIn) return 'Checking in…';
  return checkedIn ? 'Checked in' : 'Check in';
}
```

- [ ] **Step 3: Add tests**

Add exact helper tests to `web/src/modules/tickets/ticketJourney.test.ts`.

- [ ] **Step 4: Validate**

Run:

```bash
pnpm --dir web run test -- tickets
pnpm --dir web run format
make verify
```

Expected: PASS.

---

## Task 8: Optional mobile role application parity spike

**Files:**

- Modify: `mobile/app/event-detail.tsx`
- Modify: `mobile/src/api/events.ts` or equivalent public Event API client.
- Create: `mobile/src/modules/discovery/publicEventRolesModel.ts`
- Create: `mobile/src/modules/discovery/publicEventRolesModel.test.ts`

- [ ] **Step 1: Decide if role applications fit this sprint**

Before coding, inspect current mobile `event-detail.tsx` and public role/application API. Proceed only if the implementation can stay small and route/API-compatible.

Proceed if all are true:

- backend already exposes public role listing/application endpoints
- mobile can add a small section without navigation redesign
- no auth/session changes are required

If not, add this to the QA doc under `## Follow-up candidates`:

```markdown
- Mobile role applications: web supports public role applications, but mobile needs a dedicated design pass before adding forms.
```

- [ ] **Step 2: If proceeding, add pure role helpers**

Create helper functions for role capacity labels and application button labels, then test them.

- [ ] **Step 3: If proceeding, add mobile section**

Add a compact `Participation` section to `mobile/app/event-detail.tsx` below core Event details. Preserve reservation CTA behavior.

- [ ] **Step 4: Validate**

Run:

```bash
pnpm --dir mobile run test
pnpm --dir mobile run typecheck
make verify
```

Expected: PASS.

---

## Task 9: Final QA and review

**Files:**

- Modify: `docs/qa/discovery-ui-parity-2026-06.md`

- [ ] **Step 1: Run verification**

Run:

```bash
make verify
```

Expected: PASS.

- [ ] **Step 2: Run stack smoke**

Run:

```bash
make up-build
make smoke
make down
```

Expected: PASS.

- [ ] **Step 3: Update QA evidence**

Fill the QA matrix `Final` column only for checks actually run. If browser/Expo are unavailable, leave visual rows as `Not run` with the unavailable reason.

Add automated test rows for new test files:

```markdown
| `web/src/modules/publicUi/publicUi.test.ts` | Public app-parity web UI helper classes | `pnpm --dir web run test -- publicUi` |
| `mobile/src/modules/discovery/discoveryModel.test.ts` | Mobile Discovery copy/search helper parity | `pnpm --dir mobile run test -- src/modules/discovery/discoveryModel.test.ts` |
```

- [ ] **Step 4: Final design review**

Ask `designer` to review the changed public web/mobile files for:

```text
Does the public web journey now match the mobile app closely enough for this sprint, without broad redesign or Workspace/EventEditor scope creep?
```

- [ ] **Step 5: Final code review**

Ask `oracle` to review:

```text
Review Discovery + UI parity sprint. Verify Event Discovery remains isolated per ADR 0004, routes/API shapes are stable, Ticket/Door behavior and raw Ticket code semantics are preserved, tests cover Module Interfaces, and no broad Discovery/payment-provider scope slipped in.
```

Fix only concrete findings.

---

## Acceptance criteria

- `make verify` passes.
- Event Discovery remains active and isolated as alpha-ahead-of-v1.
- Mobile Discovery supports search with contextual empty/error copy.
- Web Discovery, Public Event Page, Ticket, and Door use mobile-inspired public UI surfaces.
- Ticket QR is available on web, or the QA doc records why it was deferred.
- Door/Ticket raw code semantics remain unchanged.
- Role application parity is either implemented in mobile or explicitly recorded as a follow-up candidate.
- QA doc records actual browser/Expo availability honestly.

## Risks and guardrails

- **Risk:** Full theme rewrite. **Guardrail:** only public attendee/operator pages; leave Workspace/EventEditor dark operator UI alone.
- **Risk:** Shared component abstraction. **Guardrail:** share copy/helper concepts, not React components.
- **Risk:** Discovery scope explosion. **Guardrail:** search and public parity only; no ranking/moderation/geography filters.
- **Risk:** Ticket lookup breakage. **Guardrail:** preserve raw Ticket code semantics and test helpers.
- **Risk:** Mobile role application scope creep. **Guardrail:** optional spike with explicit defer path.
