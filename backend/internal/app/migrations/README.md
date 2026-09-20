# Database migrations

`schema.sql` is the clean version-1 schema. Add later changes here as `NNNNNN_name.sql`, beginning with `000002_...`.

The application embeds these files, validates a gap-free sequence, hashes every migration, serializes runners with a PostgreSQL advisory transaction lock, and records successful applications in `schema_migrations`. Never edit an applied migration; add the next version.

Prototype compatibility is not automatic. Add a data migration only when a read-only inventory identifies real retained data and the migration has explicit acceptance and rollback evidence.
