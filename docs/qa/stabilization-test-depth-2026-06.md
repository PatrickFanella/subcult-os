# Stabilization and Test Depth QA — 2026-06

## Goal

Verify that the architecture-deepened Modules preserve the current Event lifecycle across web, API, and mobile, then record the automated tests added to prevent regressions.

## Environment

- Host: Linux
- Browser: Not run — browser unavailable
- Expo/device: Not run — Expo/device unavailable
- Stack URLs from `make urls`:
  - Web: `http://localhost:38079`
  - API: `http://localhost:38080/api/health`
  - Postgres: `localhost:35432`

## Command results

- `make up-build` — passed
- `make urls` — passed; URLs above
- `make smoke` — passed after adding bounded health retries that fail after timeout
- `make down` — passed
- `make verify` — passed
- `make test-db` — not run; TEST_DATABASE_URL unavailable

## Required commands

Run before and after fixes:

```bash
make verify
```

Run when `TEST_DATABASE_URL` is available:

```bash
make test-db
```

If no disposable Postgres test database is available, record `Not run —
TEST_DATABASE_URL unavailable` in the manual QA notes instead of silently
treating skipped DB tests as coverage.

`make verify` runs non-DB backend tests by clearing `TEST_DATABASE_URL`; use
`make test-db` for the DB-backed lifecycle tests.

## Manual QA matrix

| Area | Path | Steps | Expected result | Result | Notes |
| --- | --- | --- | --- | --- | --- |
| Web auth | `/login` | Sign up, log out, log in | Current Workspace loads without session errors | Not run | browser unavailable |
| Workspace | `/workspace` | Switch Workspace, view Events, open Event editor | Event cards keep old status copy: Draft/Live/Closed | Not run | browser unavailable |
| Event create | `/events/new` | Create draft with title, description, location, allocation | Draft Event saves and returns to editor | Not run | browser unavailable |
| Event publish | `/events/:id` | Publish complete draft | Public Event Page link works | Not run | browser unavailable |
| Public Event Page | `/e/:slug` | Reserve a free Ticket | Ticket page opens with raw Ticket code | Not run | browser unavailable |
| Door Record | `/door` | Look up Ticket and check in twice | Second check-in is idempotent | Not run | browser unavailable |
| End of Night | `/events/:id` | End the night after check-in | Event Report, Archive, Settlement sections load | Not run | browser unavailable |
| Discovery | `/discover` | Search published Events | Event Discovery remains active and copy is unchanged | Not run | browser unavailable |
| Mobile attendee | Expo app | Open discovery, Event detail, reserve/check Ticket | Attendee flow works without session header regressions | Not run | Expo/device unavailable |
| Mobile staff | Expo app | Select Workspace/Event, use Door and Run of Show | Operator flow works and ordering is stable | Not run | Expo/device unavailable |

## Regression log

| ID | Found in area | Symptom | Fix commit | Verification |
| --- | --- | --- | --- | --- |
| 1 | smoke target | `make smoke` failed on the web health probe during startup | not committed | `make smoke` passed after adding bounded retries that fail after timeout |

## Automated test additions

| Test file | Behavior covered | Command |
| --- | --- | --- |
| backend/internal/app/event_lifecycle_test.go | Always-on lifecycle helper coverage for draft/published/closed edit and publish rules | `cd backend && go test ./internal/app -run 'TestEventLifecycleHelpers|TestEventLifecyclePublishedWithoutReservationsAllowsOperationalChanges|TestEventLifecyclePublishReadinessTrimsRequiredCopy' -count=1 -v` |
| backend/internal/app/ticket_journey_test.go | Capacity, availability, and door check-in precedence rules | `cd backend && go test ./internal/app -run 'TestTicketJourneyCapacityAndAvailability|TestTicketJourneyStateAndLabels|TestTicketJourneyDoorCheckInRules|TestTicketJourneyCapacityPrecedenceForPaidEvents' -count=1 -v` |
| backend/internal/app/media_storage_test.go | Media storage config guards, upload flow, and public URL cleanup | `cd backend && go test ./internal/app -run 'TestNewMediaStorage|TestAppNewInitializesMediaStorageOnce|TestHandleUploadEventImage|TestS3MediaStorage' -count=1 -v` |
| mobile/src/modules/events/eventEditModel.test.ts | Event edit payload normalization, readiness warnings, and image selection state | `pnpm --dir mobile run test` |
| mobile/src/modules/runOfShow/runOfShowModel.test.ts | Run of show datetime parsing, sorting, payload building, and labels | `pnpm --dir mobile run test` |
| mobile/src/modules/tickets/ticketJourney.test.ts | Ticket code formatting, wallet snapshot, and arrival/badge copy | `pnpm --dir mobile run test` |
| mobile/src/modules/discovery/discoveryModel.test.ts | Discovery copy, pricing labels, and subtitle formatting | `pnpm --dir mobile run test` |
| Makefile smoke target | Retry transient API/web startup timing during smoke without masking failed probes | `make smoke` |
