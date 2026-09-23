# Database migrations

`schema.sql` is the clean version-1 schema. The current binary requires version 6: version 2 adds canonical identities and rotating session families, version 3 removes the empty prototype session table, version 4 adds encrypted AT OAuth request/session persistence, version 5 adds durable revocation, and version 6 adds opt-in transactional email delivery. Add later changes here as gap-free `NNNNNN_name.sql` files.

Version 6 leaves all existing email rows held. An old binary also inserts held rows via the default. Roll back by disabling mail delivery/stopping the worker and rolling back the application while retaining this additive schema. Do not delete the delivery ledger or reset accepted/ambiguous messages to pending. Test migration from version 5 and full replay in disposable schemas before deployment.

The application embeds these files, validates a gap-free sequence, hashes every migration, serializes runners with a PostgreSQL advisory transaction lock, and records successful applications in `schema_migrations`. Never edit an applied migration; add the next version.

Prototype compatibility is not automatic. Add a data migration only when a read-only inventory identifies real retained data and the migration has explicit acceptance and rollback evidence.

Versions 2 and 3 deliberately stop when prototype accounts or sessions are present. Inventory and explicitly approve any retained data before designing a migration; do not bypass these guards by deleting rows.
