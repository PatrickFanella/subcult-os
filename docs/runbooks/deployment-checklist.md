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

## Database safety

Read `docs/runbooks/database-migrations.md` before changing `backend/internal/app/schema.sql`.

Before any non-additive schema change:

1. take a database backup,
2. test the migration against a copy of production-like data,
3. define rollback or forward-fix steps,
4. keep the previous deploy artifact available.

## Rollback notes

The current alpha uses a single Go API and static web build. Rollback means redeploying the previous API/web image or artifact. If schema changed, rollback may require restoring a database backup unless the change was additive and backward-compatible.
