# Public Conversion and Mobile Role Applications Implementation Plan

> **For agentic workers:** Execute this plan task-by-task. Recommended path:
> dispatch a fresh subagent per task, review each result with `review-quality`,
> then continue. For complex multi-agent splits, use
> `parallel-feature-development`, `team-composition-patterns`, and
> `team-communication-protocols`. Steps use checkbox (`- [ ]`) syntax for
> tracking.

**Goal:** Improve attendee conversion on public Event surfaces and polish mobile public role applications without changing Ticket code semantics, payment-provider language, or authenticated operator workflows.

**Architecture:** Keep the public journey in focused view-model Modules instead of adding more inline screen logic. Web and mobile may duplicate platform-specific presentation, but conversion copy, sold-out/readiness labels, and role application state should live in pure helpers with tests. Backend routes and DTOs should stay stable unless a behavior gap cannot be solved at the view-model layer.

**Tech Stack:** Go API with `net/http`, Vite React TypeScript with Vitest, Expo SDK 54 React Native with pure-module Vitest tests, existing `make verify`, existing `make test-db` for optional DB-backed checks.

---

## Scope and product decision

This sprint covers the user priorities:

1. **Mobile Role Applications Polish** — improve the minimal mobile public role application section added in the prior Discovery/UI parity sprint.
2. **Public Event Page Conversion** — make web/mobile public Event detail pages clearer for attendee reservation, paid checkout, sold-out states, and role participation.

Out of scope:

- Discovery product expansion beyond the current search/feed behavior.
- Operator web UI parity or WorkspaceView redesign.
- Shared React component libraries across web/mobile.
- Changing Ticket code semantics, QR value semantics, backend Ticket state names, or payment-provider language.
- Adding auth/session requirements to public Event or public role application flows.

Guardrails:

- Public Event Page must remain no-auth.
- Web role application behavior must remain compatible with current endpoints.
- Mobile role application helper tests must stay pure; do not runtime-import Expo or React Native in `mobile/src/modules/**/*.test.ts`.
- If visual QA cannot be run, document it honestly in QA docs.

## File structure target

### QA / docs

- Create: `docs/qa/public-conversion-mobile-roles-2026-06.md` — command evidence, public conversion QA matrix, manual QA notes.

### Web

- Modify: `web/src/views/PublicEventView.tsx` — consume conversion helpers and tune page hierarchy/copy.
- Modify: `web/src/modules/publicEvent/publicEventConversion.ts` — new pure helper Module for CTA labels, sold-out copy, reservation success copy, role section copy.
- Modify: `web/src/modules/publicEvent/publicEventConversion.test.ts` — exact helper tests.
- Modify: `web/src/App.test.tsx` — route/page assertions for conversion copy and behavior.

### Mobile

- Modify: `mobile/app/event-detail.tsx` — consume helper Modules for conversion and role application polish.
- Modify: `mobile/src/modules/discovery/publicEventRolesModel.ts` — expand pure role helper state/copy.
- Modify: `mobile/src/modules/discovery/publicEventRolesModel.test.ts` — role application helper coverage.
- Create: `mobile/src/modules/events/publicEventConversionModel.ts` — mobile public Event conversion helpers.
- Create: `mobile/src/modules/events/publicEventConversionModel.test.ts` — pure conversion helper tests.

### Backend

- Avoid backend changes by default.
- Only inspect `backend/internal/app/event_roles.go` and `backend/internal/app/tickets.go` if frontend tests reveal a mismatch in existing endpoint behavior.

---

## Task 1: Record baseline QA and conversion matrix

**Files:**

- Create: `docs/qa/public-conversion-mobile-roles-2026-06.md`

- [ ] **Step 1: Create the QA document**

Create `docs/qa/public-conversion-mobile-roles-2026-06.md`:

```markdown
# Public Conversion and Mobile Role Applications QA — 2026-06

## Goal

Verify that public Event pages clearly drive attendee reservations, paid checkout, and role applications across web and mobile without changing public routes or requiring authentication.

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
| `make test-db` | Not run | Optional; requires `TEST_DATABASE_URL` |

## Public conversion matrix

| Area | Web path | Mobile path | Expected behavior | Baseline | Final | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| Free reservation CTA | `/e/:slug` | Expo `/event-detail` | CTA and helper copy make free reservation path obvious | Not run | Not run | |
| Paid ticket CTA | `/e/:slug` | Expo `/event-detail` | Paid checkout copy is provider-neutral and clear | Not run | Not run | |
| Sold-out state | `/e/:slug` | Expo `/event-detail` | Sold-out disables reservation and explains why | Not run | Not run | |
| Reservation success | `/e/:slug` | Expo `/ticket` redirect | Ticket code/next step is clear | Not run | Not run | |
| Role list | `/e/:slug` | Expo `/event-detail` | Roles show capacity and availability clearly | Not run | Not run | |
| Role application | `/e/:slug` | Expo `/event-detail` | Validation, submit, success, and error states are understandable | Not run | Not run | |

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

Update the baseline results table. If browser/Expo are unavailable, leave visual matrix rows as `Not run` with notes:

```markdown
browser unavailable in agent environment; Expo/device unavailable in agent environment
```

---

## Task 2: Add public Event conversion helper Modules

**Files:**

- Create: `web/src/modules/publicEvent/publicEventConversion.ts`
- Create: `web/src/modules/publicEvent/publicEventConversion.test.ts`
- Create: `mobile/src/modules/events/publicEventConversionModel.ts`
- Create: `mobile/src/modules/events/publicEventConversionModel.test.ts`

- [ ] **Step 1: Add web conversion helpers**

Create `web/src/modules/publicEvent/publicEventConversion.ts`:

```ts
import type { PublicEventDTO, TicketReservationDTO } from '../../domain';

export function publicEventPrimaryCtaLabel(event: Pick<PublicEventDTO, 'pricingMode' | 'isFull'> | null, reserving = false) {
  if (reserving) return 'Reserving…';
  if (event?.isFull) return 'Sold out';
  return event?.pricingMode === 'fixed' ? 'Buy ticket' : 'Reserve free ticket';
}

export function publicEventConversionSummary(event: Pick<PublicEventDTO, 'pricingMode' | 'isFull' | 'remainingTickets' | 'ticketPriceCents' | 'ticketCurrency'> | null, priceLabel: string) {
  if (!event) return 'Email required to send the ticket. Display name optional. No account needed.';
  if (event.isFull) return 'This Event is sold out. Check back with the Host for returns or future dates.';
  if (event.pricingMode === 'fixed') return `Secure checkout for ${priceLabel}. Email is required for the ticket link.`;
  return `${event.remainingTickets} ${event.remainingTickets === 1 ? 'spot remains' : 'spots remain'}. Email is required for the ticket link.`;
}

export function publicEventReservationSuccessCopy(ticket: Pick<TicketReservationDTO, 'code'>) {
  return `Ticket reserved. Save code ${ticket.code} and show it at the door.`;
}

export function publicEventRoleSectionIntro(roleCount: number) {
  if (roleCount === 0) return 'No public roles are open right now.';
  return 'Apply for public roles without changing your ticket flow.';
}
```

- [ ] **Step 2: Add web helper tests**

Create `web/src/modules/publicEvent/publicEventConversion.test.ts`:

```ts
import { describe, expect, it } from 'vitest';

import { publicEventConversionSummary, publicEventPrimaryCtaLabel, publicEventReservationSuccessCopy, publicEventRoleSectionIntro } from './publicEventConversion';

describe('publicEventConversion', () => {
  it('labels primary reservation states', () => {
    expect(publicEventPrimaryCtaLabel({ pricingMode: 'free', isFull: false })).toBe('Reserve free ticket');
    expect(publicEventPrimaryCtaLabel({ pricingMode: 'fixed', isFull: false })).toBe('Buy ticket');
    expect(publicEventPrimaryCtaLabel({ pricingMode: 'free', isFull: true })).toBe('Sold out');
    expect(publicEventPrimaryCtaLabel({ pricingMode: 'fixed', isFull: false }, true)).toBe('Reserving…');
  });

  it('summarizes conversion state without provider-specific language', () => {
    expect(publicEventConversionSummary(null, 'Free')).toBe('Email required to send the ticket. Display name optional. No account needed.');
    expect(publicEventConversionSummary({ pricingMode: 'fixed', isFull: false, remainingTickets: 10, ticketPriceCents: 1800, ticketCurrency: 'usd' }, '$18.00')).toBe('Secure checkout for $18.00. Email is required for the ticket link.');
    expect(publicEventConversionSummary({ pricingMode: 'free', isFull: false, remainingTickets: 1, ticketPriceCents: 0, ticketCurrency: 'usd' }, 'Free')).toBe('1 spot remains. Email is required for the ticket link.');
    expect(publicEventConversionSummary({ pricingMode: 'free', isFull: true, remainingTickets: 0, ticketPriceCents: 0, ticketCurrency: 'usd' }, 'Free')).toBe('This Event is sold out. Check back with the Host for returns or future dates.');
  });

  it('describes success and public role section state', () => {
    expect(publicEventReservationSuccessCopy({ code: 'ABCD1234' })).toBe('Ticket reserved. Save code ABCD1234 and show it at the door.');
    expect(publicEventRoleSectionIntro(0)).toBe('No public roles are open right now.');
    expect(publicEventRoleSectionIntro(2)).toBe('Apply for public roles without changing your ticket flow.');
  });
});
```

- [ ] **Step 3: Add mobile conversion helpers**

Create `mobile/src/modules/events/publicEventConversionModel.ts`:

```ts
export type PublicEventConversionState = {
  pricingMode: 'free' | 'fixed' | string;
  isFull: boolean;
  remainingTickets: number;
};

export function publicEventPrimaryActionLabel(event: PublicEventConversionState, reserving = false) {
  if (reserving) return 'Reserving…';
  if (event.isFull) return 'Sold out';
  return event.pricingMode === 'fixed' ? 'Buy ticket' : 'Reserve free ticket';
}

export function publicEventStickyCtaHint(event: PublicEventConversionState, priceLabel: string) {
  if (event.isFull) return 'No tickets remain for this Event.';
  if (event.pricingMode === 'fixed') return `${priceLabel} · secure checkout`;
  return `${event.remainingTickets} ${event.remainingTickets === 1 ? 'spot' : 'spots'} left`;
}
```

- [ ] **Step 4: Add mobile helper tests**

Create `mobile/src/modules/events/publicEventConversionModel.test.ts`:

```ts
import { describe, expect, it } from 'vitest';

import { publicEventPrimaryActionLabel, publicEventStickyCtaHint } from './publicEventConversionModel';

describe('publicEventConversionModel', () => {
  it('labels primary actions', () => {
    expect(publicEventPrimaryActionLabel({ pricingMode: 'free', isFull: false, remainingTickets: 2 })).toBe('Reserve free ticket');
    expect(publicEventPrimaryActionLabel({ pricingMode: 'fixed', isFull: false, remainingTickets: 2 })).toBe('Buy ticket');
    expect(publicEventPrimaryActionLabel({ pricingMode: 'free', isFull: true, remainingTickets: 0 })).toBe('Sold out');
    expect(publicEventPrimaryActionLabel({ pricingMode: 'free', isFull: false, remainingTickets: 2 }, true)).toBe('Reserving…');
  });

  it('describes sticky CTA hints', () => {
    expect(publicEventStickyCtaHint({ pricingMode: 'free', isFull: false, remainingTickets: 1 }, 'Free')).toBe('1 spot left');
    expect(publicEventStickyCtaHint({ pricingMode: 'fixed', isFull: false, remainingTickets: 5 }, '$18.00')).toBe('$18.00 · secure checkout');
    expect(publicEventStickyCtaHint({ pricingMode: 'free', isFull: true, remainingTickets: 0 }, 'Free')).toBe('No tickets remain for this Event.');
  });
});
```

- [ ] **Step 5: Validate**

Run:

```bash
pnpm --dir web run test -- publicEventConversion
pnpm --dir mobile run test -- publicEventConversionModel
pnpm --dir web run format
pnpm --dir mobile run typecheck
```

Expected: PASS.

---

## Task 3: Improve mobile role application polish

**Files:**

- Modify: `mobile/src/modules/discovery/publicEventRolesModel.ts`
- Modify: `mobile/src/modules/discovery/publicEventRolesModel.test.ts`
- Modify: `mobile/app/event-detail.tsx`

- [ ] **Step 1: Expand role helper copy/state**

Modify `mobile/src/modules/discovery/publicEventRolesModel.ts` to add:

```ts
export function publicRoleAvailabilityLabel(capacity: number, filled = 0) {
  if (capacity <= 0) return 'Open application';
  const remaining = Math.max(capacity - filled, 0);
  if (remaining === 0) return 'Role full';
  return `${remaining} ${remaining === 1 ? 'spot' : 'spots'} open`;
}

export function publicRoleApplicationStatusCopy(submitted: boolean, error: string | null) {
  if (error) return error;
  return submitted ? 'Application sent. The Host can review it from the Workspace.' : 'Tell the Host why you are a fit.';
}

export function publicRoleCanSubmit(submitting: boolean, submitted: boolean) {
  return !submitting && !submitted;
}
```

- [ ] **Step 2: Add exact role helper tests**

Append to `mobile/src/modules/discovery/publicEventRolesModel.test.ts`:

```ts
it('describes public role availability and submit state', () => {
  expect(publicRoleAvailabilityLabel(0, 0)).toBe('Open application');
  expect(publicRoleAvailabilityLabel(2, 1)).toBe('1 spot open');
  expect(publicRoleAvailabilityLabel(2, 2)).toBe('Role full');
  expect(publicRoleApplicationStatusCopy(true, null)).toBe('Application sent. The Host can review it from the Workspace.');
  expect(publicRoleApplicationStatusCopy(false, 'Please enter your name.')).toBe('Please enter your name.');
  expect(publicRoleCanSubmit(false, false)).toBe(true);
  expect(publicRoleCanSubmit(true, false)).toBe(false);
  expect(publicRoleCanSubmit(false, true)).toBe(false);
});
```

Import the new helpers in that test file.

- [ ] **Step 3: Refactor mobile role cards to use helpers**

In `mobile/app/event-detail.tsx`:

- Replace inline capacity copy with `publicRoleAvailabilityLabel(role.capacity, role.filledCount ?? 0)` if `filledCount` exists; otherwise pass `0` for filled.
- Replace inline submitted/error helper copy with `publicRoleApplicationStatusCopy(draft.submitted, draft.error)`.
- Disable the submit button with `!publicRoleCanSubmit(draft.submitting, draft.submitted)`.
- Keep the existing endpoint calls, payload trimming, no-auth public path, and reservation CTA untouched.

- [ ] **Step 4: Validate**

Run:

```bash
pnpm --dir mobile run test -- publicEventRolesModel.test.ts
pnpm --dir mobile run typecheck
make verify
```

Expected: PASS.

---

## Task 4: Improve mobile public Event conversion CTA and sold-out state

**Files:**

- Modify: `mobile/app/event-detail.tsx`
- Modify: `mobile/src/modules/events/publicEventConversionModel.ts`
- Modify: `mobile/src/modules/events/publicEventConversionModel.test.ts`

- [ ] **Step 1: Use conversion helpers in sticky CTA**

In `mobile/app/event-detail.tsx`:

- Import `publicEventPrimaryActionLabel` and `publicEventStickyCtaHint`.
- Replace inline bottom button text with `publicEventPrimaryActionLabel(event, reserving)`.
- Replace inline price/remaining helper text with `publicEventStickyCtaHint(event, price)`.
- Disable the button when `event.isFull || reserving`.
- Keep `handleReserve()` unchanged so API behavior stays stable.

- [ ] **Step 2: Make sold-out message explicit**

Add a small non-blocking copy row near the sticky CTA:

```tsx
{soldOut ? <Text style={styles.stickyWarning}>This Event is sold out.</Text> : null}
```

Add `stickyWarning` to the existing StyleSheet:

```ts
stickyWarning: {
  marginBottom: 8,
  color: '#b91c1c',
  fontSize: 13,
  fontWeight: '700',
},
```

- [ ] **Step 3: Validate**

Run:

```bash
pnpm --dir mobile run test -- publicEventConversionModel
pnpm --dir mobile run typecheck
make verify
```

Expected: PASS.

---

## Task 5: Improve web Public Event conversion hierarchy

**Files:**

- Modify: `web/src/views/PublicEventView.tsx`
- Modify: `web/src/modules/publicEvent/publicEventConversion.ts`
- Modify: `web/src/modules/publicEvent/publicEventConversion.test.ts`
- Modify: `web/src/App.test.tsx`

- [ ] **Step 1: Use conversion helpers in web reservation card**

In `web/src/views/PublicEventView.tsx`:

- Import `publicEventPrimaryCtaLabel`, `publicEventConversionSummary`, `publicEventReservationSuccessCopy`, and `publicEventRoleSectionIntro`.
- Replace local `ctaLabel()` with `publicEventPrimaryCtaLabel(event, reserving)`.
- Replace local `heroSummary()` with `publicEventConversionSummary(event, pricingLabel(event))`.
- Replace reservation success paragraph with `publicEventReservationSuccessCopy(reservation)`.
- Replace role section intro with `publicEventRoleSectionIntro(roles?.length ?? 0)`.

- [ ] **Step 2: Preserve paid checkout behavior and provider-neutral copy**

Verify the paid form still posts to:

```ts
`/api/public/events/${slug}/paid-reservations`
```

and still redirects with:

```ts
window.location.href = checkout.checkoutUrl;
```

Do not reintroduce `Stripe Checkout` copy.

- [ ] **Step 3: Add/adjust App assertions**

In `web/src/App.test.tsx`, update public Event assertions to include:

```ts
expect(rendered).toContain('No account needed');
expect(rendered).toContain('Secure checkout');
expect(rendered).toContain('Apply for public roles without changing your ticket flow.');
```

Keep existing assertions that prove:

- role application fields render
- paid CTA renders as `Buy ticket`
- provider-specific `Stripe Checkout` does not appear

- [ ] **Step 4: Validate**

Run:

```bash
pnpm --dir web run test -- publicEventConversion
pnpm --dir web run test -- src/App.test.tsx
pnpm --dir web run format
make verify
```

Expected: PASS.

---

## Task 6: Add conversion-focused regression tests

**Files:**

- Modify: `web/src/App.test.tsx`
- Modify: `mobile/src/modules/events/publicEventConversionModel.test.ts`
- Modify: `mobile/src/modules/discovery/publicEventRolesModel.test.ts`

- [ ] **Step 1: Add web sold-out regression assertion**

In `web/src/App.test.tsx`, add a public Event test state where the mocked Event has:

```ts
isFull: true,
remainingTickets: 0,
pricingMode: 'free',
```

Assert:

```ts
expect(rendered).toContain('Sold out');
expect(rendered).toContain('This Event is sold out. Check back with the Host for returns or future dates.');
```

- [ ] **Step 2: Add mobile helper edge cases**

In `mobile/src/modules/events/publicEventConversionModel.test.ts`, add assertions for zero remaining non-full Events and fixed-price copy.

In `mobile/src/modules/discovery/publicEventRolesModel.test.ts`, add assertions for whitespace validation:

```ts
expect(validatePublicRoleApplicationDraft({ applicantName: '  ', applicantEmail: 'guest@example.com', message: '' })).toBe('Please enter your name.');
expect(validatePublicRoleApplicationDraft({ applicantName: 'Guest', applicantEmail: 'guest', message: '' })).toBe('Please enter a valid email address.');
```

- [ ] **Step 3: Validate**

Run:

```bash
pnpm --dir web run test -- src/App.test.tsx
pnpm --dir mobile run test -- publicEventConversionModel publicEventRolesModel
make verify
```

Expected: PASS.

---

## Task 7: Final QA evidence and review

**Files:**

- Modify: `docs/qa/public-conversion-mobile-roles-2026-06.md`

- [ ] **Step 1: Run final verification**

Run:

```bash
make verify
```

Expected: PASS.

- [ ] **Step 2: Optionally run DB tests if a disposable DB is available**

If `TEST_DATABASE_URL` is available, run:

```bash
make test-db
```

If unavailable, record:

```markdown
Not run — TEST_DATABASE_URL unavailable
```

- [ ] **Step 3: Update QA doc**

Add final command results and automated test additions:

```markdown
## Final command results

| Command | Result | Notes |
| --- | --- | --- |
| `make verify` | Passed | Final non-visual verification |
| `make test-db` | Not run | TEST_DATABASE_URL unavailable |
```

Add test rows for:

- `web/src/modules/publicEvent/publicEventConversion.test.ts`
- `mobile/src/modules/events/publicEventConversionModel.test.ts`
- `mobile/src/modules/discovery/publicEventRolesModel.test.ts`
- `web/src/App.test.tsx`

- [ ] **Step 4: Final review**

Run a final reviewer against the changed files. Review must check:

- no auth requirement introduced for public Event or role application
- no provider-specific payment copy
- sold-out state disables conversion actions
- Ticket code semantics unchanged
- mobile helper tests remain pure
- `make verify` passed

Expected: PASS.

---

## Acceptance criteria

- Mobile public role application section has clearer availability, submission, and success/error states.
- Web and mobile public Event pages use tested conversion helper copy for free/paid/sold-out states.
- Web Public Event Page keeps role applications, reservations, and paid checkout behavior stable.
- Mobile Event Detail keeps no-auth public flow and reservation behavior stable.
- Provider-neutral language is preserved; no `Stripe Checkout` copy is introduced.
- `make verify` passes.
- QA doc records visual/manual limitations honestly.

## Execution notes

- Commit only when explicitly requested.
- Prefer exact helper tests over loose `toContain` copy assertions.
- Do not edit `WorkspaceView` or authenticated operator surfaces in this sprint.
- If an attempted UI polish requires backend changes, stop and ask before expanding scope.
