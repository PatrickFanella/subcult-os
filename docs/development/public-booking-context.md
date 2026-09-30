# Public booking context

## 2026-09-30 source repair

At PR #178's qualified head `702d2f8`, changing the public event component's
slug from synthetic A to synthetic B and failing B's read left A's title and an
enabled reservation form visible. The component accepted the new slug while
retaining the previous event's guest state. Comparing only the current slug in
write callbacks also cannot distinguish A → B → a fresh A.

`PublicEventView` now keys its state-owning child by slug. Pricing, availability
and forms require a completed read whose `publicSlug` matches that child. Loading
or failed reads show neutral unavailable states. A mismatched read is rejected.
Guest fields, ticket links, role drafts and purchase intent belong to one mounted
child. Layout-effect cleanup marks that child inactive in its removal commit;
late reservation, application and paid-checkout callbacks cannot update a fresh
child or redirect after removal. Synchronous ticket and role submission guards
also prevent repeated submissions before React renders disabled controls.

This follows React's documented [keyed state lifetime](https://react.dev/learn/preserving-and-resetting-state)
and [layout-effect cleanup timing](https://react.dev/reference/react/useLayoutEffect).
The layout effect only changes a ref; it performs no layout measurement or state
update. Existing paid purchase-intent reuse and pending-checkout status fencing
remain in place. An issued free ticket remains visible if the availability refresh
fails or returns another slug, with availability explicitly unknown.

## Verification

The final `bash scripts/dev-env.sh verify` passed 286 web and 34 mobile tests,
backend checks/builds and the complete disposable database gate: 421 top-level
tests, 599 including nested tests, no failures or skips. The disposable database
was removed. An initial run exposed two old SSR tests that expected a free-ticket
placeholder before any event read; their expectations now match the neutral
loading state. Two new SSR cases check that mismatched or still-loading event
data cannot expose the previous guest, ticket or forms.

The actual exported component was exercised in React Strict Mode with runtime
React 19.2.6, Linux Chrome 152/Electron 44, and a 1402 × 876 CSS-pixel viewport.
Synthetic fetch responses held and released requests to verify:

- A → B loading and failed reads hide A and all forms, without claiming free pricing.
- A → B → A discards old free-ticket and role-application successes and old guest fields.
- A ready paid response released after unmount causes no redirect; an old expired
  checkout response cannot replace the current view's state or ticket link.
- Repeated submits while ticket/application writes are held create one request;
  a completed application and a pending paid checkout remain fenced.
- Matching current free and role successes still confirm. An issued free ticket
  survives an unusable refresh. Current paid pending status retains its link
  and disables purchasing.
- A wrong-slug event read shows the explicit mismatch error and no forms.

All seven POST responses were intercepted synthetic fixtures. No reservation,
application, mail or payment provider was contacted. A read-only check of the
development event's public inventory returned nine remaining tickets both
before and after the fixture. No outstanding fixture writes remained at cleanup.
The fixture root, fetch interception and globals were removed; the original
archive page and light appearance were restored.

Inspected pending-checkout screenshots:
`browser-screenshot-127-0-0-1-munx68nw-d4c09225.png` (light) and
`browser-screenshot-127-0-0-1-munx68t5-cbee600e.png` (dark). Detailed local
receipts remain in ignored `.cache/dev-env/public-booking-context-*` files.

## Remaining gates

This is component state and callback proof, not a browser-history routing test.
The current app chooses its view from the initial pathname; this slice adds no
SPA router. Real backend reservation-to-ticket-to-door journeys, provider
checkout, native/physical devices and screen-reader speech remain separate.
Public booking long-content reflow, input focus contrast and legacy event-clock
semantics are not qualified by this slice. Hosted results belong to the exact
published head and must be recorded separately from the local gate.
