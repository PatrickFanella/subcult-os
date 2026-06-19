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
| `make verify` | Passed | Baseline non-visual verification |
| `make up-build` | Not run | |
| `make smoke` | Not run | |
| `make down` | Not run | |

## Final command results

| Command | Result | Notes |
| --- | --- | --- |
| `make verify` | Passed | Final non-visual verification after Discovery/UI parity changes |
| `make up-build` | Not run | Browser/Docker smoke unavailable for this sprint pass |
| `make smoke` | Not run | Browser/Docker smoke unavailable for this sprint pass |
| `make down` | Not run | Not needed because Docker stack was not started |

## UI parity matrix

| Area | Web path | Mobile path | Expected parity | Baseline | Final | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| Discovery copy | `/discover` | Expo `/` | Loading, empty, error, and CTA copy match | Not run | Not run | browser unavailable in agent environment; Expo/device unavailable in agent environment |
| Discovery search | `/discover?q=...` | Expo `/` | Both support attendee search with contextual empty state | Not run | Not run | browser unavailable in agent environment; Expo/device unavailable in agent environment |
| Discovery visual style | `/discover` | Expo `/` | Web uses mobile-inspired light cards/surfaces | Not run | Not run | browser unavailable in agent environment; Expo/device unavailable in agent environment |
| Public Event detail | `/e/:slug` | Expo `/event-detail` | Hero, pricing, reserve CTA, and role/application sections feel related | Not run | Not run | browser unavailable in agent environment; Expo/device unavailable in agent environment |
| Ticket view | `/tickets/:code` | Expo `/ticket` | Ticket code, payment state, and QR/door affordance are clear | Not run | Not run | browser unavailable in agent environment; Expo/device unavailable in agent environment |
| Door view | `/door` | Expo `/door` | Lookup, result, and check-in states use matching structure/copy | Not run | Not run | browser unavailable in agent environment; Expo/device unavailable in agent environment |

## Regression log

| ID | Area | Symptom | Fix | Verification |
| --- | --- | --- | --- | --- |

## Automated test additions

| Test file | Behavior covered | Command |
| --- | --- | --- |
| `web/src/modules/publicUi/publicUi.test.ts` | Exact public light-surface helper classes and status tones | `pnpm --dir web run test -- publicUi` |
| `web/src/modules/discovery/discoveryModel.test.ts` | Discovery copy helpers and locale-stable pricing | `pnpm --dir web run test -- discovery` |
| `mobile/src/modules/discovery/discoveryModel.test.ts` | Mobile Discovery copy/search helper parity | `pnpm --dir mobile run test -- src/modules/discovery/discoveryModel.test.ts` |
| `mobile/src/modules/discovery/publicEventRolesModel.test.ts` | Public role capacity/application helper behavior | `pnpm --dir mobile run test -- publicEventRolesModel.test.ts` |
| `web/src/modules/tickets/ticketJourney.test.ts` | Ticket/Door helper copy, including Door check-in labels | `pnpm --dir web run test -- tickets` |
| `web/src/App.test.tsx` | Public routes, Discovery copy, Public Event Page, Ticket, and Door route coverage | `pnpm --dir web run test -- src/App.test.tsx` |
