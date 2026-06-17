# Sprint: organizer fake-event rehearsal

Date: 2026-06-16

## Goal

Make SUBCULT OS usable for an organizer to set up and rehearse a fake event end-to-end.

This sprint is not a real door-pressure test. It is a workflow rehearsal for creating, publishing, previewing, and operationally preparing an event.

## Primary journey

Organizer can:

1. Sign in with a workspace account.
2. Create a workspace from mobile if the account does not already have one.
3. Select a workspace and event in the mobile Staff dashboard.
4. Create or edit an event from mobile.
4. Add public basics: title, date/time, description, location, capacity, pricing, and image.
5. Publish the event.
6. Preview the mobile/public event surface.
7. Reserve at least one test ticket.
8. Create roles and run-of-show/staffing items.
9. Correct run-of-show item title, notes, and times without recreating the item.
10. Open the readiness checklist and see what is complete or missing.
11. Rehearse ticket lookup/check-in with the test ticket.

## Acceptance criteria

- Mobile Staff dashboard exposes a clear event readiness path.
- Mobile Staff dashboard lets a signed-in organizer create a first workspace or add another workspace.
- Event edit screen shows preview/readiness links after an event exists.
- Event edit screen explains what is missing before publish/rehearsal.
- Readiness screen summarizes setup progress and links to the right setup screens.
- Readiness checks cover event details, hero image, ticket settings, publish status, test ticket, roles, run-of-show, and door rehearsal.
- Readiness can create a rehearsal ticket directly, without leaving organizer mode.
- Run-of-show items can be edited for fake-event setup corrections.
- Role creation supports public/private visibility so fake-event roles can be attendee-facing or internal.
- Roles can be edited and deactivated/reactivated during setup without recreating them.
- Existing verification still passes: `make verify` and Expo web export.

## Rehearsal checklist

Use seeded organizer credentials or create a new account locally.

1. Start API/web and Expo.
2. Sign in on mobile.
3. Create a workspace if the account has none.
4. Open Staff dashboard.
5. Create a new fake event.
6. Upload/select a hero image.
7. Save and publish.
8. Open Readiness.
9. Use Readiness to create a rehearsal ticket, or use preview to reserve a public ticket.
10. Return to Readiness and confirm test-ticket/door checks update.
11. Add one role and one run-of-show item.
12. Open Door or Scanner and find the ticket.

## Deferred from this sprint start

- Offline scanner cache/queued check-ins.
- Full `event_media` table/object lifecycle.
- Role delete and staffing delete.
- Production deep links from Stripe/public web back into mobile.
