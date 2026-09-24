# Database migrations

`schema.sql` is the clean version-1 schema. The current binary requires version 8: version 2 adds canonical identities and rotating session families, version 3 removes the empty prototype session table, version 4 adds encrypted AT OAuth request/session persistence, version 5 adds durable revocation, version 6 adds opt-in transactional email delivery, version 7 adds signed feedback receipts and suppression, and version 8 adds the minimal cultural model (profiles, places, protected place detail, event occurrences and multi-host occurrence credits; see `docs/development/cultural-model.md`). Add later changes here as gap-free `NNNNNN_name.sql` files.

Version 8 is additive: new tables only, plus one `unique (id, workspace_id)` constraint added to the existing `events` table so occurrence rows can carry a composite foreign key that rejects a cross-workspace reference at the database level. No existing column, row or ticket/staffing/settlement behavior changes. Roll back by rolling back the application only; the new tables stay empty and unreferenced by any older code path.

Version 7 is additive. Disable sending before an application rollback; older workers do not enforce the new suppression ledger. Retain feedback and suppression records. Do not downgrade the database or enable an old worker against it.

Version 6 leaves all existing email rows held. An old binary also inserts held rows via the default. Roll back by disabling mail delivery/stopping the worker and rolling back the application while retaining this additive schema. Do not delete the delivery ledger or reset accepted/ambiguous messages to pending. Test migration from version 5 and full replay in disposable schemas before deployment.

The application embeds these files, validates a gap-free sequence, hashes every migration, serializes runners with a PostgreSQL advisory transaction lock, and records successful applications in `schema_migrations`. Never edit an applied migration; add the next version.

Prototype compatibility is not automatic. Add a data migration only when a read-only inventory identifies real retained data and the migration has explicit acceptance and rollback evidence.

Versions 2 and 3 deliberately stop when prototype accounts or sessions are present. Inventory and explicitly approve any retained data before designing a migration; do not bypass these guards by deleting rows.
