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
| `make verify` | PASS | Completed in agent environment on 2026-06-19; dependency install warnings noted for ignored `esbuild@0.27.7` build scripts. |
| `make test-db` | Not run | Optional; requires `TEST_DATABASE_URL` |

## Final command results

| Command | Result | Notes |
| --- | --- | --- |
| `make verify` | Passed | Final non-visual verification after public conversion/mobile role changes; dependency install warnings noted for ignored `esbuild@0.27.7` build scripts. |
| `make test-db` | Not run | TEST_DATABASE_URL unavailable |

## Public conversion matrix

| Area | Web path | Mobile path | Expected behavior | Baseline | Final | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| Free reservation CTA | `/e/:slug` | Expo `/event-detail` | CTA and helper copy make free reservation path obvious | Not run | Not run | browser unavailable in agent environment; Expo/device unavailable in agent environment |
| Paid ticket CTA | `/e/:slug` | Expo `/event-detail` | Paid checkout copy is provider-neutral and clear | Not run | Not run | browser unavailable in agent environment; Expo/device unavailable in agent environment |
| Sold-out state | `/e/:slug` | Expo `/event-detail` | Sold-out disables reservation and explains why | Not run | Not run | browser unavailable in agent environment; Expo/device unavailable in agent environment |
| Reservation success | `/e/:slug` | Expo `/ticket` redirect | Ticket code/next step is clear | Not run | Not run | browser unavailable in agent environment; Expo/device unavailable in agent environment |
| Role list | `/e/:slug` | Expo `/event-detail` | Roles show capacity and availability clearly | Not run | Not run | browser unavailable in agent environment; Expo/device unavailable in agent environment |
| Role application | `/e/:slug` | Expo `/event-detail` | Validation, submit, success, and error states are understandable | Not run | Not run | browser unavailable in agent environment; Expo/device unavailable in agent environment |

## Regression log

| ID | Area | Symptom | Fix | Verification |
| --- | --- | --- | --- | --- |

## Automated test additions

| Test file | Behavior covered | Command |
| --- | --- | --- |
| `web/src/modules/publicEvent/publicEventConversion.test.ts` | Web CTA, sold-out, secure checkout, reservation success, and role intro helper copy | `pnpm --dir web run test -- publicEventConversion` |
| `mobile/src/modules/events/publicEventConversionModel.test.ts` | Mobile sticky CTA labels/hints, sold-out state, zero remaining, fixed-price copy | `pnpm --dir mobile run test -- publicEventConversionModel` |
| `mobile/src/modules/discovery/publicEventRolesModel.test.ts` | Mobile public role capacity, availability, validation, submit state, and success/error copy | `pnpm --dir mobile run test -- publicEventRolesModel.test.ts` |
| `web/src/App.test.tsx` | Public Event conversion rendering, sold-out regression, role fields, and provider-neutral copy | `pnpm --dir web run test -- src/App.test.tsx` |
