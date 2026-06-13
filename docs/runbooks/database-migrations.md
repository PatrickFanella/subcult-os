# Database Migrations Runbook

`subcult-os` currently uses one embedded idempotent schema file:

- Source: `backend/internal/app/schema.sql`
- Embed/wiring: `backend/internal/app/db.go`
- Startup call: `backend/cmd/app/main.go` calls `RunMigrations(ctx, db)` before serving traffic.

## Current behavior

`RunMigrations` reads `schema.sql` and executes it as one SQL batch. The schema uses `create table if not exists`, constraints, and indexes that are safe for the current alpha database shape.

This is acceptable while the app is in alpha and schema changes are additive.

## Current limitation

There is no ordered migration history yet. Do not use the current approach for destructive changes, backfills, long-running data rewrites, or any production database that needs reversible deploys.

## Before external production data

Before running against external production data, add a real migration tool or repository-local migration runner that provides:

1. ordered migration files,
2. a migration history table,
3. repeatable local/CI execution,
4. explicit rollback or forward-fix guidance,
5. backup instructions before destructive changes.

## Safe alpha change rules

- Additive tables and nullable columns are acceptable in `schema.sql`.
- New constraints must be compatible with existing local data.
- Data deletion, column renames, type changes, and non-null migrations with existing rows require a migration tool first.
- Run `make verify` after schema edits.
- When local credentials or schema assumptions change, reset local data with `make reset-db` if preserving local rows is not required.

## Local verification

```bash
make up-build
make smoke
make alpha-qa
make verify
```
