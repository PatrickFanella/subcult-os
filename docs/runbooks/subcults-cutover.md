# Legacy Subcults replacement runbook

Status: approved cutover target; not approved to execute until all pre-cutover gates pass.

The product owner authorized Subcult OS to replace the existing legacy Subcults application at `https://subcults.subcult.tv`. That authority permits an eventual controlled cutover, not premature downtime, data destruction, secret extraction, or an unqualified in-place upgrade.

## Known public state

On 2026-09-20 the public hostname served the legacy app through Cloudflare and Almaz Caddy. Live read-only host inspection confirmed that the active Caddy route sends `/api/*` and `/health/*` to Dozor `10.0.0.57:3025` and all other paths to Dozor `10.0.0.57:3024`. Historical NUC routes are stale and must not be used for cutover. These routes were observed without authentication:

| Route | Observed behavior | Replacement contract |
| --- | --- | --- |
| `/` | `200` | Browser app must pass the release journey before cutover |
| `/health/live` | `200` JSON | Legacy health evidence only; OS uses `/api/health` and `/api/ready` |
| `/api/v1/auth/atproto/client-metadata` | `200` JSON | Preserve exact client ID URL; narrow scope to `atproto` |
| `/api/v1/auth/atproto/jwks` | `200` JSON | Preserve exact URL and a valid public P-256 JWKS |
| `/api/v1/auth/atproto/callback` | unauthenticated `GET` redirected | Preserve exact callback URL after OS callback handling exists |

The legacy metadata described a confidential web client using `private_key_jwt`, ES256, DPoP, key ID `subcults-1`, and broader repository scopes. The replacement keeps the public URL identity and confidential-client shape but does not inherit repository permissions. Existing grants may require fresh authorization; verify this explicitly rather than weakening the replacement scope.

The live Compose project is `/srv/containers/subcults` on Dozor. It currently runs `subcults-frontend`, `subcults-api`, `subcults-indexer`, `subcults-tap`, `subcults-redis`, and `pg16-postgis`; every container reported running, healthy where configured, and zero restarts. Exact image IDs captured on 2026-09-20 are:

- API `sha256:a9f3748d6e895862b0034bad343b22390cade56d37c81402b20f9702dbb606e9`
- frontend `sha256:dd1976fa4696eedf0d69534eeb0c12e6dff3d991064da1de7366597b954f4257`
- indexer `sha256:15f464f81a78a7726ff181424483643632b1cd4b475555a2810da8ef18106541`
- Tap `sha256:49df1f10e641f1dc53bc5b9b927a29284bd986cb0a61d21af65b909e5309a9a8`
- PostgreSQL/PostGIS `sha256:681931a625df344215e9b8998bf34daf146b6a395ceacee4439eb9c85869239f`
- Redis `sha256:e7723ff73d963f5cc6d9c4643ea3d989527a402a319239054e9472a7fb9219a2`

The legacy `subcults` database reported migration version 46 with `dirty=false` and zero rows in `users`, `events`, `profiles`, `atproto_oauth_links`, `atproto_oauth_sessions`, and `atproto_oauth_requests`. This supports a clean Subcult OS database rather than a retained-user migration, but row counts are not authorization to delete the legacy database. Backup directories were not readable to the unprivileged inspection account, so current backup/restore evidence remains a pre-cutover gate.

## Non-negotiable safety rules

- Do not stop the legacy service to stage or test Subcult OS.
- Do not point the public proxy at Subcult OS until the staged artifact passes every gate below.
- Do not run Subcult OS migrations against the legacy database. Migration 2 intentionally refuses a populated prototype account table, and no retained-account migration has been approved.
- Never print, copy into docs, or commit the legacy OAuth private key, session key, database credentials or user data.
- Preserve the exact previous proxy configuration, immutable image identifiers, service configuration and database backup needed for rollback.
- Keep legacy containers, images and volumes intact until the observation window closes and a separate retirement is approved.
- If any resolved live fact differs from this document, stop and update the plan before mutation.

## Phase 1: capture the live rollback manifest

Perform read-only inventory on the actual host and record the results in a dated, access-controlled deployment artifact:

1. DNS/Cloudflare target and TLS/proxy mode.
2. Caddy source configuration and rendered active route for `subcults.subcult.tv`.
3. Service/container names, image repositories, immutable image digests, command, environment variable names, mounts, networks, restart policies and published ports.
4. Database engine/version, database name, volume or dataset, schema migration level and a fresh row-count-only inventory of account and OAuth tables. Reconfirm the observed zero rows immediately before cutover.
5. Current health responses and a browser screenshot/recording of the public home and one representative public event journey.
6. The legacy OAuth metadata and JWKS public documents, including a fingerprint of each public JWK. Never capture private key material.

The manifest must name the exact one-command or one-file proxy rollback and identify who can execute it.

## Phase 2: make recoverable backups

Before any production mutation:

1. Create a consistent legacy database backup using the database's supported snapshot or logical-dump procedure.
2. Store it outside the live volume and record time, size, checksum, database version and restore command.
3. Restore the backup into an isolated database and verify schema version plus non-sensitive table counts.
4. archive the active Caddy configuration and service definition with checksums;
5. prove the previous image digest can still be pulled or is present locally;
6. record the current public-key fingerprints and decide whether the existing confidential-client key can be securely reused or whether cutover intentionally rotates it.

A backup without a successful isolated restore is not a qualified rollback artifact.

## Phase 3: stage Subcult OS independently

Deploy the candidate under distinct service names, ports, database name/volume and an internal or access-controlled hostname. Do not share the legacy database or writable storage.

The candidate configuration must include:

- `APP_ENV=production`
- a new Subcult OS `DATABASE_URL`
- independent `SESSION_SECRET` and `IDENTITY_PROTECTION_KEY` values from the deployment secret store
- `PUBLIC_WEB_URL=https://subcults.subcult.tv`
- the exact production client-metadata, callback and JWKS URLs documented in the AT kernel
- a secret-store-backed P-256 client key and intentional key ID

Keep `ATPROTO_OAUTH_ENABLED=false` until the authorization/callback implementation exists. When testing the complete OAuth flow on staging, use a separate qualified client identity or an access-controlled routing mechanism that does not change public production traffic.

## Phase 4: qualify the replacement

Record candidate Git SHA, image digests, migration ledger and configuration-key names. Then pass:

1. `CI=true make verify` at the candidate SHA.
2. Fresh-database migration and replay checks plus the applicable database-backed suite.
3. Candidate `/api/health` and `/api/ready` checks.
4. A real browser journey for signup, email verification, login, workspace creation, event creation/publish, public discovery/detail, free reservation, ticket display, door check-in and end-of-night report.
5. Mobile/API journeys required for the intended launch surface.
6. Anonymous-response checks confirming operator, identity and attendance data are not exposed.
7. Exact metadata/JWKS comparison: client ID, callback and JWKS URLs match production; scope is exactly `atproto`; no private key field or key material is present.
8. Once implemented, a bounded live AT OAuth link, refresh, unlink/revocation and replay-negative journey with a test identity. Confirm no repository permission and no local-account merge.
9. Backup/restore rehearsal for the new Subcult OS database.

Health checks alone do not qualify the replacement. If the launch intentionally omits a legacy feature, record the product decision and user impact before cutover.

## Phase 5: atomic proxy cutover

Schedule a monitored window only after every prior gate is evidenced.

1. Re-read live state and ensure the rollback manifest still matches.
2. Take a final consistent legacy database backup and verify its checksum.
3. Prevent only those legacy writes that cannot safely overlap the proxy switch; do not destroy the legacy service.
4. Change the smallest possible Caddy upstream mapping from the legacy ports to the staged Subcult OS services.
5. Validate and atomically reload Caddy.
6. Verify externally through Cloudflare: TLS, home, health/readiness behavior, public discovery/detail, authentication, and exact OAuth metadata/JWKS.
7. Run one bounded synthetic end-to-end journey and inspect candidate logs/metrics for errors without exposing personal data.
8. Record cutover time, configuration checksum, old/new image digests and verification evidence.

Do not change DNS when a local upstream switch is sufficient. Do not delete the legacy route, services, images, database or volume during cutover.

## Rollback triggers and procedure

Rollback immediately for unavailable home/API, failed readiness, authentication/session failure, migration errors, public data leakage, materially broken launch journeys, invalid OAuth metadata/JWKS, elevated server errors, or any unexplained write divergence.

1. Restore the saved Caddy upstream mapping to the exact legacy services.
2. Validate and atomically reload Caddy.
3. Confirm the legacy home, representative public journey and public OAuth documents externally.
4. Stop new Subcult OS writes if needed, preserving its database and logs for diagnosis.
5. Record the failed candidate SHA/images, timestamps, symptoms and rollback verification. Do not rewrite failed evidence.

Restoring the legacy database is not normally part of proxy rollback because the systems use independent databases. If the legacy database was mutated despite this runbook, stop and use the tested restore procedure with explicit owner approval.

## Post-cutover observation and retirement

Keep the rollback path intact through an agreed observation window. Monitor availability, readiness, server errors, signup/verification/login, public event reads, reservation/check-in, email outbox/provider behavior, database growth, and AT OAuth errors once enabled. Redact tokens, email bodies and protected identity material.

Legacy retirement is a separate destructive action. It requires a final retained-data/export decision, confirmed backup retention, explicit approval, and a documented removal list. A successful cutover alone does not authorize deleting legacy containers, images, volumes, databases or backups.
