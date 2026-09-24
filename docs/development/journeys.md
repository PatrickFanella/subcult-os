# User journeys and acceptance
Status: proposed integrated acceptance scenarios. Use synthetic records in disposable environments.

## J1 — Unlinked event remains usable
Owner creates workspace and event without an AT account. Guest reserves a free ticket. Member checks in. Owner runs End of Night and reads the private report.
Pass: the clean platform supports the complete local lifecycle with no AT connection. Prototype IDs, routes and ticket URLs need not survive redesign.
Check: replacement lifecycle tests plus browser/mobile observation.
Failure injection: disable protocol connectivity entirely.

## J2 — Attach an existing public occurrence
Authorized operator selects one validated public event by AT URI inside the unified API. UI previews title, time, public venue/place and source identity before attaching it.
Pass: one mapping; exact observed CID/freshness available privately; duplicate action does not create another local event.
Negative: another workspace cannot read or modify the mapping.

## J3 — Publish an approved public edit
Authorized creator/operator previews only allowlisted fields. Submit once; simulate a lost PDS response.
Pass: pending/unknown outcome is visible; reconciliation confirms exact revision without duplicate creation. Local save and public publication are visibly distinct.
Negative: revoked creator authority blocks execution, including queued work.

## J4 — Discover and reserve
Anonymous participant discovers the validated occurrence through Subcult.tv and follows its approved reservation destination.
Pass: correct operator event, safe link handling, existing free reservation and ticket access; no private staffing, contact or financial data in the public payload.
Negative: missing, stale or invalid mapping does not route to another event silently.
Implemented 2026-09-24 over the DISC-01 projection: `GET /api/public/discovery/occurrences`
and `.../occurrences/{uri...}` (anonymous, no session) list/detail the
projected occurrence, resolve reservation handoff through `event_public_links`
(local reservation only in this slice — the admitted Lexicon has no ticket-URL
field yet, so external handoff is not reachable), and never fall back to a
different event on a missing/stale mapping. See
[`discovery-ux.md`](discovery-ux.md) for the API, web UI and test evidence.

## J5 — External edit conflict
Another authorized AT client changes the public occurrence before the OS editor submits.
Pass: stale CID conflicts; UI shows the changed public fields and explicit resolution. Private tickets and operations remain unchanged.
Never automatically reschedule a ticketed local event based on an external public edit.

## J6 — Public deletion and recovery
Delete a disposable public record; rebuild the public projection from the appropriate authority.
Pass: public record no longer appears as current; OS retains authorized private operational history and flags the unavailable public reference.
A source deletion does not prove every third-party copy has been erased.

## J7 — Event-night outage
Pause public-protocol connectivity during the local free-ticket rehearsal.
Pass: authorized local ticket lookup/check-in continues; public publication waits; no fabricated synchronization success.
This proves isolation, not offline mobile support or readiness under real door pressure.

## UI requirements across journeys
Keyboard operation, named controls, predictable focus after mutation, announced asynchronous status, non-color-only errors, narrow-screen layout and readable long event/venue names.
Display times with explicit local event-zone context after the time-model decision. Do not silently reinterpret daylight-saving transitions.
Test all loading, empty, denied, unavailable and conflict states. Record browser/device and served revision for actual journey evidence.
