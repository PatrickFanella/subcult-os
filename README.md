# subcult-os

Full-stack SUBCULT OS project with a Go backend, Vite React TypeScript Tailwind frontend, Postgres, Docker Compose, and Open Pilot conventions.

Bootstrapped from `subculture-collective/project-template`.

## Quick start

```bash
make verify
```

Useful local commands:

```bash
make help
make quick
make up-build
make urls
make smoke
make alpha-qa
make logs
make down
make reset-db
```

## First lifecycle slice

The first product slice proves:

1. Owner signs up and creates a Workspace.
2. Owner invites a Member by email; development stores invitation emails in `email_outbox` and exposes recent messages in the Workspace UI.
3. Invitee opens `/invite/{token}`, signs up or signs in with the invited email, and accepts Workspace membership.
4. Owner creates and publishes a direct-link Public Event Page.
5. Guest reserves a free Ticket with email and optional display name; the page shows an in-place confirmation and Ticket link.
6. Member runs mobile-friendly Door Check-In by manual lookup or exact Ticket code.
7. Owner runs End of Night and views the private Event Report.
8. People can belong to multiple Workspaces and switch the active operator home from the Workspace access panel.

Run locally:

```bash
make up-build
```

Then open:

- Web: http://localhost:38079
- API health: http://localhost:38080/api/health

The default host ports are intentionally high to avoid common local conflicts: `WEB_PORT=38079`, `API_PORT=38080`, and `POSTGRES_PORT=35432`.

Use `make urls` to print the actual published ports, `make smoke` to check the running Docker stack, and `make reset-db` to delete local Postgres data and rebuild if an old volume has stale credentials.

Recommended local QA loop:

```bash
make up-build
make smoke
make urls
make alpha-qa
make verify
```

If the app is already running, use the Workspace page to inspect development invite/ticket emails instead of connecting to Postgres directly.

## Current alpha security notes

- Passwords are stored with bcrypt. Legacy local SHA-256 password hashes are upgraded on successful login.
- Cookie-authenticated mutating API requests with an `Origin` header must come from the same host or `PUBLIC_WEB_URL`.
- Login attempts are lightly throttled per email/IP in process memory.
- Public free ticket reservations remain guest-accessible without account login.

## Production readiness checklist

- Set `APP_ENV=production`.
- Set a real `DATABASE_URL`; production startup fails without it.
- Set a non-default `SESSION_SECRET` with at least 24 characters.
- Set `PUBLIC_WEB_URL` to the HTTPS web origin used by browsers.
- Stripe paid ticketing is optional until an Event uses paid pricing; when enabled, set `STRIPE_SECRET_KEY` and `STRIPE_WEBHOOK_SECRET`.
- Use `/api/health` for process health and `/api/ready` for DB-backed readiness.
- Review `docs/runbooks/database-migrations.md` before changing persisted schema.
- Review `docs/runbooks/deployment-checklist.md` before running outside local development.

## Paid ticketing local setup

For local paid-ticket testing, copy the optional Stripe env vars from `.env.example`, then run Stripe CLI webhook forwarding:

```bash
stripe listen --forward-to localhost:38080/api/stripe/webhook
```

The paid ticketing runbook is `docs/runbooks/stripe-local-testing.md`.

- `make alpha-qa` stays the default free-ticket lifecycle check.
- `make alpha-qa-paid` runs the optional paid API QA when `STRIPE_SECRET_KEY` and `STRIPE_WEBHOOK_SECRET` are set.

Stripe keys are not required unless you configure a paid Event. Use the Stripe test card `4242 4242 4242 4242`, and treat the webhook as the source of truth for fulfillment.

## Open Pilot

This repository keeps the base Open Pilot issue template and PR template.
Bootstrap labels after creating a repo from this template:

```bash
open-pilot labels bootstrap PatrickFanella/subcult-os
```

## License

GPL-3.0-only. See `LICENSE`.
