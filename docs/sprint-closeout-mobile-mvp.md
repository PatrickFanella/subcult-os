# Mobile MVP sprint closeout

Date: 2026-06-16

## Scope completed

- Added a new Expo React Native app under `mobile/` for attendee, organizer, and staff flows.
- Wired mobile public discovery, event detail, reservations, ticket lookup, saved ticket wallet, QR pass rendering, and native share actions.
- Added mobile auth/session handling, staff workspace/event selection, door search/check-in, QR scanning, event dashboard, run-of-show, staffing actions, role creation, and application review.
- Added mobile organizer create/edit/publish flow with event image selection and backend-proxied upload.
- Added backend CORS support for local Expo web development.
- Added event `image_url` support, public event image exposure, ticket URL exposure, paid checkout recovery fields, partial ticket-code door search, and media upload endpoint.
- Set up S3-compatible media configuration for local development against the Almaz MinIO bucket `subcult-os-media`.
- Updated root Makefile, README, stack docs, compose env wiring, and ignored generated/native/mobile artifacts.

## Local development notes

- Local API: `http://localhost:38080`
- Local web app: `http://localhost:38079`
- Expo web/dev server: usually `http://localhost:8081`
- Local media uploads require the ignored `.env` keys documented in `.env.example`:
  - `MEDIA_S3_ENDPOINT`
  - `MEDIA_S3_ACCESS_KEY`
  - `MEDIA_S3_SECRET_KEY`
  - `MEDIA_S3_BUCKET`
  - `MEDIA_S3_REGION`
  - `MEDIA_PUBLIC_BASE_URL`
- Do not commit `.env`; it contains local MinIO credentials.
- The imported `Event Management Mobile App/` mockup folder is ignored and should not be committed unless intentionally promoted into source.

## Final verification checklist

Before committing or starting the next sprint:

1. Run `make clean`.
2. Run `make verify`.
3. Confirm `git status --short --ignored` shows only intended source changes plus ignored `.env`, dependency directories, and generated build artifacts.
4. Smoke-test the current local stack:
   - login as seeded organizer
   - discover public events
   - reserve a free ticket and view its QR pass
   - create or edit an event and upload an image
   - search/check in a ticket from Door or Scanner
   - create/update a run-of-show item
   - create a role and review an application when seed data exists

## Suggested commit boundaries

1. **Backend platform/API**
   - CORS middleware and tests
   - media config/upload endpoint
   - event image URL support
   - ticket URL and paid checkout recovery fields
   - door search and staffing lock fixes
2. **Mobile app**
   - Expo scaffold, API/auth clients, navigation, screens, wallet, QR/scanner, organizer/staff flows, assets, lockfile
3. **Project wiring/docs**
   - Makefile, `.gitignore`, `.env.example`, compose env wiring, README, docs
4. **Design-system web styling**
   - `web/src/styles.css` semantic tokens/utilities, if kept separate from app/API work

## Next sprint candidates

- Replace cookie bridge with a deliberate mobile auth/token flow.
- Add offline scanner resilience: event-day ticket snapshot, queued check-ins, conflict handling, and sync status.
- Promote event media from single `events.image_url` to an `event_media` table with object keys, kind, alt text, sort order, and lifecycle cleanup.
- Polish public role application flows and make accepted/confirmed status clearer to attendees.
- Add notifications/reminders for ticket holders, staff assignments, and day-of schedule changes.
- Add attendee saved/favorited events separate from saved tickets.
- Add production deployment configuration for API/mobile/media, including public URL/deep-link strategy.
- Run accessibility and real-device QA pass for mobile forms, camera permissions, share sheets, checkout recovery, and session persistence.
