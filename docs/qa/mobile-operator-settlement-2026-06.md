# Mobile Operator and Settlement QA — 2026-06

## Goal

Verify mobile operator readiness and End of Night / Settlement behavior before and after the sprint.

## Environment

- Host: Linux
- Browser: Not run
- Expo/device: Not run
- API URL used by mobile: Not recorded

## Required commands

```bash
make verify
```

```bash
make test-db
```

Run `make test-db` only when `TEST_DATABASE_URL` points at a disposable Postgres database.

## Final command results

| Command | Result | Notes |
| --- | --- | --- |
| `make verify` | Passed | Full repo verification completed successfully |
| `make test-db` | Not run | TEST_DATABASE_URL unavailable in agent environment |
| `make up-build` | Passed | Stack built and started successfully |
| `make smoke` | Passed | API and web smoke checks completed successfully |
| `make down` | Passed | Stack shut down cleanly |

## Baseline command results

| Command | Result | Notes |
| --- | --- | --- |
| `make verify` | Passed | Baseline non-DB verification |
| `make test-db` | Guard verified | TEST_DATABASE_URL unavailable; integration coverage not run |
| `make up-build` | Not run | |
| `make smoke` | Not run | |
| `make down` | Not run | |

## Mobile operator QA matrix

| Area | Path | Steps | Expected result | Baseline | Final | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| Staff shell | Expo `/staff` | Open staff mode after login | Selected Workspace and Event are clear; errors are actionable | Not run | Not run | browser unavailable in agent environment; Expo/device unavailable in agent environment |
| Workspace selection | Expo `/staff` | Switch Workspace | Event list and persisted selection update safely | Not run | Not run | browser unavailable in agent environment; Expo/device unavailable in agent environment |
| Event selection | Expo `/staff` | Switch active Event | Door, Run of Show, and dashboard point at the selected Event | Not run | Not run | browser unavailable in agent environment; Expo/device unavailable in agent environment |
| Door lookup | Expo `/door` | Enter valid Ticket code | Ticket details load with raw Ticket code and payment status | Not run | Not run | browser unavailable in agent environment; Expo/device unavailable in agent environment |
| Door check-in | Expo `/door` | Check in the same Ticket twice | First check-in succeeds; second is idempotent | Not run | Not run | browser unavailable in agent environment; Expo/device unavailable in agent environment |
| Run of Show | Expo `/run-of-show` | Change task status | List reorders by status/start/created/id and shows stable copy | Not run | Not run | browser unavailable in agent environment; Expo/device unavailable in agent environment |
| Ticket wallet | Expo `/tickets` | Open wallet with pending and checked-in Tickets | Pending/ready/checked-in states are distinguishable | Not run | Not run | browser unavailable in agent environment; Expo/device unavailable in agent environment |
| Poor network | Expo app | Disable API or use wrong API URL | Errors explain recovery; app does not lose selected Workspace/Event | Not run | Not run | browser unavailable in agent environment; Expo/device unavailable in agent environment |

## Settlement / End of Night QA matrix

| Area | Path | Steps | Expected result | Baseline | Final | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| End of Night | Web `/events/:id` | End a published Event | Report is created once and Event status becomes End of Night | Not run | Not run | browser unavailable in agent environment |
| Report counts | Web `/events/:id` | Compare reserved/check-in/no-show counts | Counts match Tickets and Door records | Not run | Not run | browser unavailable in agent environment |
| Settlement totals | Web `/events/:id` | Review gross/fees/net/manual adjustments | Totals are readable and internally consistent | Not run | Not run | browser unavailable in agent environment |
| Archive notes | Web `/events/:id` | Add closeout/archive notes | Notes persist and appear in Workspace archive | Not run | Not run | browser unavailable in agent environment |
| Mobile closeout visibility | Expo staff flow | View closed Event summary if available | Mobile shows read-only status/report cue or clear web handoff | Not run | Not run | Expo/device unavailable in agent environment |

## Regression log

| ID | Area | Symptom | Fix | Verification |
| --- | --- | --- | --- | --- |

## Automated test additions

| Test file | Behavior covered | Command |
| --- | --- | --- |
| `backend/internal/app/settlement_test.go` | Settlement no-show clamp, summary/report assembly, net total helper | `make verify` |
| `mobile/src/modules/staff/staffOperatorModel.test.ts` | Staff selection labels, readiness, empty-state copy, persisted event selection | `make verify` |
| `mobile/src/modules/settlement/settlementModel.test.ts` | Mobile closeout labels and web-handoff copy | `make verify` |
