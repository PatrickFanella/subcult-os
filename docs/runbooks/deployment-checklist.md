# Deployment Checklist

Use this checklist before running `subcult-os` outside local development.

## Required verification

Run from the repository root:

```bash
make verify
make build
make compose-config
```

For the local Docker stack, also run:

```bash
make up-build
make smoke
make alpha-qa
```

`make alpha-qa` is a local alpha lifecycle test. It creates throwaway users, a Workspace, an invite, an Event, a Ticket reservation, a Door Check-In, and an End of Night report.

## Required environment

Production must set:

- `APP_ENV=production`
- `DATABASE_URL` with the production Postgres connection string
- `SESSION_SECRET` as a non-default random value with at least 24 characters
- `PUBLIC_WEB_URL` as the HTTPS browser origin
- `API_ADDR` for the bind address, usually `:8080` inside a container

Do not commit real secrets to `.env`, `.env.example`, docs, or compose files.

## Health checks

- `/api/health`: process is serving HTTP.
- `/api/ready`: API can reach the database.

Use `/api/ready` for load balancer or orchestrator readiness checks when available.

## Notification privacy and delivery

- Notification activity is operator-only; public discovery and public event pages never expose notification data.
- `email_outbox` rows contain recipient email addresses and email bodies, so treat outbox content as private workspace data.
- Contacts/commitments may contain sensitive free text; review logs, notification templates, and public routes before production launch.
- Before enabling production delivery, review the mail provider for privacy, transport security, retention, and deliverability behavior.

## Database safety

Read `docs/runbooks/database-migrations.md` before changing `backend/internal/app/schema.sql`.

Before any non-additive schema change:

1. take a database backup,
2. test the migration against a copy of production-like data,
3. define rollback or forward-fix steps,
4. keep the previous deploy artifact available.

## Rollback notes

The current alpha uses a single Go API and static web build. Rollback means redeploying the previous API/web image or artifact. If schema changed, rollback may require restoring a database backup unless the change was additive and backward-compatible.
