# Synthetic database and key recovery

This BACKUP-01 rehearsal checks recovery through the current Subcult OS code.
It uses synthetic identities and keys, with no mail or OAuth provider calls.
Deployed candidate recovery, secret-store recovery, roles/ACLs and service/proxy
configuration recovery remain separate gates under
[the cutover runbook](../runbooks/subcults-cutover.md).

## Run

On a host with the installed T3 development environment, from this checkout:

```sh
rtk proxy python3 scripts/qa-backup-recovery.py
```

The script accepts no arguments, database URLs or key inputs. It imports the
installed launcher and extends its recipe for this invocation only. It takes
the worktree and host test locks, refuses existing containers in its test
project, snapshots the current checkout and installs dependencies. Other
worktrees' test projects and previews are preserved.

Two PostgreSQL services run on the launcher's isolated test network, with no
published ports or retained database volume. Each has tmpfs storage, 768 MiB
memory, one CPU and a 128-PID limit. After dependency setup, the checks container
has no provider access. Database utilities come from the database image itself.

The rehearsal performs these checks:

1. Migrate an empty source database and seed a verified encrypted email, an
   active OAuth session, a pending OAuth request and a queued revocation.
2. Generate a custom-format dump, recording its size and SHA-256. Remove the
   source container and its tmpfs database before attempting restoration.
3. Restore into the second database with `pg_restore --exit-on-error`.
4. Confirm a wrong identity key cannot decrypt the restored email or session.
5. Confirm a wrong signing key fails a challenge against the recorded public
   key and produces a different JWKS identity.
6. Confirm the matching keys recover the current migration version, email,
   session tokens, request verifier, revocation payload and signing identity.
   Revocation uses a fake callback; it never contacts a provider.
7. Remove only the owned test containers/network and compiled fixture binaries.

The identity and client signing keys are generated in memory and passed through
child environments, not command arguments or Compose configuration. The dump
contains only public client-signing identity; per-session DPoP keys are separate
and encrypted with the session payload. No production keys are used. Keys are
not retained when the process exits, so a retained synthetic dump is evidence,
not a standalone recoverable deployment backup.

## Evidence

Each invocation prints its private evidence directory under
`~/.local/state/subcult-os/backup-rehearsal/<UTC timestamp>/`. Directories are
mode `0700` and files are `0600`. Successful and failed runs are retained:

- `receipt.json`: source host, path, revision and dirty state; launcher and
  source-manifest hashes; database/tool image IDs; dump checksum; completed
  phases; source-removal and cleanup results.
- `source-sha256.json`: hashes of the exact copied source files, including
  uncommitted changes admitted by the launcher's snapshot policy.
- `run.log`: setup, aggregate fixture assertions and cleanup output.
- `synthetic.dump`: custom-format database dump, if capture completed.

A pass requires every phase and cleanup to succeed. `make verify` syntax-checks
the Python helper and compiles the tagged Go fixture but does **not** run the
two-database rehearsal. Run the command above explicitly when validating this
recovery boundary. Full `make verify` and disposable `make test-db` remain the
repository gates.

The 2026-09-24 Kvant run `20260924T220704690747Z` passed all five phases
(`seed`, `restore`, `wrong-identity`, `wrong-signing`, `verify`) and cleanup.
It used base revision `fa8edd1e4ef32fa4c9d61b7638ff4ae3c966e008` plus the
uncommitted rehearsal changes recorded in its source manifest. Both databases
used image `sha256:0027bef26712baaee437a4ea48fdf3d2d2e2bc5f0d81615374408ca320f3c7e3`.
The source was removed before restore. The 156,026-byte dump has SHA-256
`0d6bcd22b37130831551a371677dca36de2680ad54e600f382bcbff3df40d8e0`.
The source manifest has SHA-256
`9a2098a40939ffcff0a98bb43e632fdb8404d9b4465d83a4791e8e6a183dafc1`.

This proves database-content recovery with separately supplied synthetic keys.
It does not prove retrieval of real keys from a backup/secret manager, session
cookie continuity, owner/ACL restoration, a deployed image, live OAuth/mail,
or cutover/rollback. BACKUP-01 remains open until its operational gates pass.
