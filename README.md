# Subcult OS

Subcult OS is software for the people running a small event: independent
venues, collectives and crews. The plan, the staffing, the guest list and the
end-of-night report sit on one sheet that the whole team reads. It is in alpha
development, and no real-event operation has been verified on it.

[Explore Subcult OS](https://subcult.tv/products/subcult-os) · [Feedback and feature requests](https://git.subcult.tv/subculture-collective/subcult-os/issues) · [Development guide](DEVELOPMENT.md)

## What it covers

| Stage | What you can do |
| --- | --- |
| Plan | Create a workspace, draft an event, reuse event templates, and publish a public event page. |
| Coordinate | Invite collaborators, assign staffing, keep private contacts and commitments, and prepare the run of show. |
| Welcome guests | Let guests reserve free tickets, open their ticket links, and check them in by lookup or ticket code. |
| Run the night | Use the mobile client for scanning, door search, the run of show, and the live dashboard. |
| Close out | Run End of Night and review the private event report. |

Attendees find published events on the public discovery page. Workspace notes,
staffing, contacts and reports stay private to the organizer workspace.

## Web and phone

Planning and public event pages are on the web. The Expo mobile app carries the
event-time tools: scanner, door search, run of show and the dashboard, with
organizer, staff and attendee flows. It runs in Expo Go for development. App-store
availability is not promised.

Private venue and event access worksheets let owners record observations,
unknowns, and review dates without publishing unverified accessibility claims.

## Current scope

Subcult OS is in alpha development. The documented lifecycle starts with
workspace creation and free ticket reservations and ends with door check-in and
an event report. Qualification so far uses local rehearsal data; no live event operation has
been verified. Paid ticketing code exists but needs a configured Stripe integration and
has not been qualified with a live provider; outbound email requires explicit
provider setup. AT Protocol integration remains outside the qualified
live-provider scope.

See the [development guide](DEVELOPMENT.md) for the implemented lifecycle, mobile
setup, integration limits, and production readiness checklist.

## Development

The project includes a Go backend, PostgreSQL database, React web client, and
Expo React Native app. Start with the [local setup](DEVELOPMENT.md#quick-start)
and run `make verify` before proposing a change.

- [Platform development](docs/development/README.md)
- [Design system](docs/design-system.md)
- [Deployment checklist](docs/runbooks/deployment-checklist.md)

Built by [Subcult](https://subcult.tv). Licensed under [GPL-3.0-only](LICENSE).
