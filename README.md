# Subcult OS

**Plan the event. Run the door. Know how the night went.**

Subcult OS is an event workspace for independent venues, collectives, and live
scenes. Bring your event pages, crew, tickets, door operations, and end-of-night
report into one place.

[Explore Subcult OS](https://subcult.tv/products/subcult-os) · [Feedback and feature requests](https://git.subcult.tv/subculture-collective/subcult-os/issues) · [Development guide](DEVELOPMENT.md)

## From the first plan to the final report

| Stage | What you can do |
| --- | --- |
| Plan | Create a workspace, draft an event, reuse event templates, and publish a public event page. |
| Coordinate | Invite collaborators, assign staffing, keep private contacts and commitments, and prepare the run of show. |
| Welcome guests | Let guests reserve free tickets, open their ticket links, and check them in by lookup or ticket code. |
| Run the night | Use the mobile client for scanning, door search, the run of show, and the live dashboard. |
| Close out | Run End of Night and review the private event report. |

Public event discovery gives attendees a place to find published events. Private
workspace notes, staffing, contacts, and reports stay within the organizer
workspace.

## Built for the people doing the work

The web workspace handles planning and public event pages. The Expo mobile app
puts event-time tools on a phone, with organizer, staff, and attendee flows.
Private venue and event access worksheets let owners record observations,
unknowns, and review dates without publishing unverified accessibility claims.

## Current scope

Subcult OS is in active alpha development. The documented lifecycle starts with
workspace creation and free ticket reservations and ends with door check-in and
an event report. Paid ticketing requires a configured Stripe integration;
outbound email requires explicit provider setup. AT Protocol integration remains
outside the qualified live-provider scope.

See the [development guide](DEVELOPMENT.md) for the implemented lifecycle, mobile
setup, integration limits, and production readiness checklist.

## Build with us

The project includes a Go backend, PostgreSQL database, React web client, and
Expo React Native app. Start with the [local setup](DEVELOPMENT.md#quick-start)
and run `make verify` before proposing a change.

- [Platform development](docs/development/README.md)
- [Design system](docs/design-system.md)
- [Deployment checklist](docs/runbooks/deployment-checklist.md)

Built by [Subcult](https://subcult.tv). Licensed under [GPL-3.0-only](LICENSE).
