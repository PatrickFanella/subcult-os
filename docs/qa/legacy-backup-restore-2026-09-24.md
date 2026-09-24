# Legacy database restore rehearsal — 2026-09-24

Status: **PASS for the legacy database restore slice of BACKUP-01 (#29)**.
Candidate database/key recovery and complete cutover qualification remain open.

## Source and isolation

Read-only inspection matched the deployment recorded in the
[cutover runbook](../runbooks/subcults-cutover.md): the active proxy routes,
legacy image IDs, database migration version and six recorded aggregate counts
were unchanged. The running database was PostgreSQL 16.4 with PostGIS.

The existing backup directory was not readable by the inspection account.
Its permissions were left intact. A fresh custom-format logical dump was
created under that account's private state directory, outside the live volume.
The directory is mode `0700`; the dump is mode `0600`. The dump and detailed
logs remain on the source host and are not repository artifacts.

| Backup property | Observed value |
| --- | --- |
| Capture ID (UTC) | `20260924T214210Z` |
| Format | PostgreSQL custom format, `pg_dump -Fc --no-owner --no-acl` |
| Size | 73,831,011 bytes |
| SHA-256 | `17fcc8d9f37aab90e62d28345de291bf3260863dcac9e90010e6c7340fdbc860` |
| Source and restore image ID | `sha256:681931a625df344215e9b8998bf34daf146b6a395ceacee4439eb9c85869239f` |

Restoration used the exact locally available source database image in a new
container with no network, no published ports and no live mounts. Its database
directory was disposable tmpfs. Limits were 768 MiB memory, one CPU and 128 PIDs.
No application or worker was connected to the restored database.

## Restore result

The successful attempt created the target database with `createdb -T template0`
and ran `pg_restore --no-owner --no-acl --exit-on-error`. Restore exited zero.
Queries used `psql -X -v ON_ERROR_STOP=1` and compared these aggregates with the
live source:

| Check | Source | Restored |
| --- | --- | --- |
| Migration version | 46 | 46 |
| Dirty migration flag | false | false |
| Public base tables | 73 | 73 |
| `users` rows | 0 | 0 |
| `events` rows | 0 | 0 |
| `profiles` rows | 0 | 0 |
| `atproto_oauth_links` rows | 0 | 0 |
| `atproto_oauth_sessions` rows | 0 | 0 |
| `atproto_oauth_requests` rows | 0 | 0 |

The dump checksum and file permissions were rechecked after restoration.
All rehearsal containers were removed, including failed attempts. The legacy
services remained running; the API, Tap and Redis still reported healthy.
No legacy service, proxy route, key, volume or database was changed or removed.

## Failed attempts retained

1. `pg_isready` observed the image's temporary initialization server before the
   target database existed. Restore failed with `database "subcults" does not
   exist`. Subsequent attempts waited for a successful SQL query through the
   final TCP listener inside the isolated container.
2. The PostGIS image initialized its configured database with a `tiger` schema.
   Restoring into that database failed with `schema "tiger" already exists`.
   The successful attempt used a separate database created from `template0`.

Both error logs and receipts remain alongside the backup. No restore error was
ignored, and no live schema was dropped to make restoration pass.

## Evidence limits and remaining gates

- This is a database-content restore, not a complete cluster restore. The dump
  excludes ownership and ACLs; role/permission recovery still needs evidence.
- Six domain-table counts and the public-table count were compared. This does
  not establish row-by-row equivalence across all tables or legacy application
  behavior against the restored database.
- Candidate schema-14 database recovery with its encryption/signing keys has
  not been rehearsed. No production keys were extracted for this test.
- A complete protected service/proxy configuration archive, secret-store
  recovery and an executable proxy rollback receipt remain required.
- Zero rows in the checked tables do not authorize deletion. Other tables are
  present in the dump; all legacy data, images and volumes remain retained.
- No live mail, AT OAuth, browser/device, deployment or cutover qualification
  is claimed. The protected pilot retains its own prerequisites.

SIGNAL-01 source delivery is separately complete through
[PR #137](https://git.subcult.tv/subculture-collective/subcult-os/pulls/137),
merged as `4fd09e7704ef2e7a2b12d55db99c3f457504d3cc`.
[Hosted run 9903](https://git.subcult.tv/subculture-collective/subcult-os/actions/runs/9903)
passed on that merge commit. Those source checks do not qualify a deployed
candidate or close BACKUP-01.
