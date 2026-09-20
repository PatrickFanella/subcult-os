# Database Migrations Runbook

`subcult-os` uses an embedded, ordered, forward-only migration history:

- Version 1 baseline: `backend/internal/app/schema.sql`
- Later migrations: `backend/internal/app/migrations/NNNNNN_name.sql`
- Embed/runner: `backend/internal/app/db.go`
- Explicit command: `backend/cmd/migrate/main.go`
- Startup gate: `backend/cmd/app/main.go` runs `RunMigrations` before serving traffic

## Guarantees

The runner opens one PostgreSQL transaction, takes a transaction-scoped advisory lock and applies every unapplied version in order. `schema_migrations` records the version, name, SHA-256 checksum and application time. Replay is safe; an applied file whose name or bytes changed is rejected. A failed migration rolls back its schema changes and ledger entry. A database ahead of the binary fails closed.

Version 1 is the accepted clean OS baseline. Do not modify `schema.sql` after it has been used outside disposable development. Add the next gap-free file under `migrations/` instead. Never rewrite a recorded migration to make a check pass.

## Local commands

```bash
make migrate         # start Postgres, build the migration binary, apply pending versions
make migrate-status  # show the immutable ledger, public tables and indexes
make migrate-reset   # destructive: recreate local Docker data and apply ordered migrations
make test-db         # run lifecycle and migration integration tests against TEST_DATABASE_URL
```

`make migrate` uses the same image, configuration and embedded runner as application startup. It does not feed SQL directly to `psql`.

## Adding a migration

1. Inventory the target database read-only and classify its data as disposable, reproducible or retained.
2. Add exactly the next six-digit version, for example `000002_identity_foundation.sql`.
3. Prefer additive, short transactions. For consequential backfills, document batching, compatibility and forward-fix behavior before implementation.
4. Add focused fresh/replay/failure tests and any specifically required populated-upgrade fixture.
5. Run `make test-db` against disposable PostgreSQL, then the broader repository gates.
6. Back up and restore representative retained data before an authorized deployment.

Do not add a legacy upgrade path merely to preserve unused prototype data. If retained real accounts, tickets or external identifiers are found, their inventory and acceptance rules must precede the migration.

## Rollback and recovery

Migrations are forward-only. A source rollback is safe only while the older binary supports the current database version; the runner otherwise rejects the mismatch. Prefer a reviewed forward fix. Before destructive or shape-changing work, capture an independently restorable backup and exact schema/application versions.

A database restore can lose writes made after the backup. Once non-synthetic data exists, reconcile those writes and all external side effects before recovery. Database rollback cannot undo PDS publication, payment, sent mail or another provider action.

For disposable local data only, `make migrate-reset` is available. It deletes the Compose volume and must not be used against an unresolved or retained database.

## Verification matrix

- Fresh database: every version applies and the ledger reaches the binary version.
- Replay: no duplicate ledger or schema change.
- Concurrency: advisory locking serializes runners.
- Tamper: changed applied migration is rejected.
- Failure: schema and ledger changes in the failed transaction disappear.
- Ahead-of-binary: startup and explicit migration fail closed.
- Backup/restore: schema version and representative row counts match after restore.

For a local application rehearsal:

```bash
make up-build
make migrate-status
make smoke
make alpha-qa
make verify
```
