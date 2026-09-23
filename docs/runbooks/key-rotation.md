# Key Rotation Runbook

This covers the two secrets that protect verified identity data and AT
Protocol OAuth client assertions:

- `IDENTITY_PROTECTION_KEY` — encrypts verified emails (`email_identities`)
  and, through the same derivation, AT OAuth session/request/revocation
  payloads (`atproto_oauth_sessions`, `atproto_oauth_requests`,
  `atproto_oauth_revocations`). See `backend/internal/app/identity_crypto.go`
  and `backend/internal/atproto/oauth_store.go`.
- `ATPROTO_OAUTH_CLIENT_PRIVATE_KEY` / `ATPROTO_OAUTH_CLIENT_KEY_ID` — the
  confidential OAuth client's P-256 signing key, published in the client
  JWKS. See `backend/internal/atproto/oauth_client.go`.

Both keys are read from process environment variables (`backend/internal/app/config.go`).
This repository does not name a specific secret-manager product; whatever the
deployment's secret store is, it must inject these as environment variables
(see `docs/runbooks/deployment-checklist.md` for the full required set). Do
not commit real key material to `.env`, `.env.example`, docs, compose files,
issues, logs, or screenshots.

## Key ownership

| Key | Owner | Rotation trigger |
| --- | --- | --- |
| `IDENTITY_PROTECTION_KEY` | Identity module (`backend/internal/app`) | Suspected exposure, scheduled rotation policy, offboarding an operator who held the value |
| `ATPROTO_OAUTH_CLIENT_PRIVATE_KEY` | AT Protocol adapter (`backend/internal/atproto`) | Same triggers, or provider-mandated client key rotation |

Only deployment operators with secret-store access should ever hold these
values. Application code never logs, returns, or accepts them over an API
request; they exist only as process environment variables and inside the
secret store.

## Backup and encrypted recovery

- The keys themselves are secrets, not data to back up separately: losing a
  key is equivalent to losing everything it protects (see "Key loss" below).
  Whatever secret store holds `IDENTITY_PROTECTION_KEY` and
  `ATPROTO_OAUTH_CLIENT_PRIVATE_KEY` must itself have durable, access-audited
  backup/recovery, independent of the application database.
- The *data* the keys protect (the Postgres database) is backed up and
  restored per `docs/runbooks/database-migrations.md` and the deployment
  checklist's database safety section. A restored database is only readable
  if the matching key (current, or current+previous during a rotation
  window) is also available — restoring a backup taken before a rotation
  completed requires keeping the previous key until that backup's retention
  window closes.
- Never store a key and its ciphertext in the same backup artifact or the
  same access boundary; that collapses "encrypted" to "obfuscated."

## Rotating `IDENTITY_PROTECTION_KEY`

Rotation keeps both keys live for a bounded transition window so nothing
becomes unreadable mid-rotation:

1. Generate a new 32-byte key and base64-encode it, for example:

   ```bash
   openssl rand -base64 32
   ```

2. In the secret store, move the current `IDENTITY_PROTECTION_KEY` value into
   `IDENTITY_PROTECTION_KEY_PREVIOUS`, and set `IDENTITY_PROTECTION_KEY` to
   the new value. Both variables are read by `backend/internal/app/config.go`;
   `IDENTITY_PROTECTION_KEY_PREVIOUS` is optional and only used to decrypt
   rows sealed under the retiring key. New writes always use the current key.
3. Deploy. Reads keep working during the transition: `identityProtector`
   (`backend/internal/app/identity_crypto.go`) and `atproto.OAuthStore`
   (`backend/internal/atproto/oauth_store.go`) both try the current key first,
   then the previous key. Login/verification/recovery lookups try the current
   key's lookup hash, then the previous key's hash
   (`emailLookupHashCandidates`), so a row that has not been re-encrypted yet
   is still found.
4. Run the re-encryption sweep in bounded batches until it reports zero
   remaining work:

   ```bash
   make identity-rekey-status   # counts only; decrypts nothing
   make identity-rekey          # processes one batch (default 500 rows per table)
   ```

   `identity-rekey` (`backend/cmd/identity-rekey`, wired into
   `backend/internal/app/identity_rekey.go` and
   `backend/internal/atproto/oauth_rekey.go`) is resumable and idempotent: it
   claims rows with `FOR UPDATE SKIP LOCKED`, skips anything already sealed
   under the current key, and is safe to rerun (or run under `-watch`) until
   `identity-rekey-status` reports no rows left protected only by the
   previous key. It rewrites `email_identities` (ciphertext and lookup hash
   together, so a row's ciphertext and hash are never inconsistent) and AT
   OAuth `atproto_oauth_sessions` / `atproto_oauth_revocations` payloads. It
   prints aggregate counts only — never an email, DID, session identifier, or
   ciphertext.
5. **Documented limit:** `atproto_oauth_requests` rows are not rekeyed. They
   expire in 10 minutes (`oauthRequestLifetime` in
   `backend/internal/atproto/oauth_store.go`) and are deleted on claim or by
   `DeleteExpiredRequests`, so they age out on their own. Do not remove
   `IDENTITY_PROTECTION_KEY_PREVIOUS` within 10 minutes of setting it, so any
   in-flight OAuth start/callback started just before rotation can still
   complete.
6. Once the status command reports zero previous-key-only rows for
   `email_identities`, `atproto_oauth_sessions`, and
   `atproto_oauth_revocations`, and at least 10 minutes have passed since step
   2, remove `IDENTITY_PROTECTION_KEY_PREVIOUS` from the secret store and
   redeploy.

## Rotating the AT OAuth signing key

1. Generate a new client key:

   ```bash
   make generate-atproto-key
   ```

   Send the printed multibase private key directly to the secret store; never
   commit it.
2. Move the current `ATPROTO_OAUTH_CLIENT_PRIVATE_KEY` /
   `ATPROTO_OAUTH_CLIENT_KEY_ID` into
   `ATPROTO_OAUTH_CLIENT_PRIVATE_KEY_PREVIOUS` /
   `ATPROTO_OAUTH_CLIENT_KEY_ID_PREVIOUS`, and set the current variables to
   the new key and a new, distinct key ID.
3. Deploy. `backend/internal/atproto/oauth_client.go` publishes both the
   current and the previous public key in the JWKS during this window
   (`OAuthClientSettings.PreviousPrivateKey`/`PreviousKeyID`), so client
   assertions or DPoP proofs already issued under the retiring key still
   validate against the JWKS while an AT Protocol resource server checks
   them. The private signing key itself always uses the *current* key for
   new assertions; the previous key's private half is never used to sign
   anything after rotation.
4. Once every session issued under the previous key has completed or expired
   (bounded by the provider's own token lifetime, not by this repository),
   remove `ATPROTO_OAUTH_CLIENT_PRIVATE_KEY_PREVIOUS` /
   `ATPROTO_OAUTH_CLIENT_KEY_ID_PREVIOUS` and redeploy so the JWKS publishes
   only the current key again.

## Rollback

- **`IDENTITY_PROTECTION_KEY`:** if the new key was set but no re-encryption
  has happened yet, rollback is just restoring the old value as
  `IDENTITY_PROTECTION_KEY` (and clearing `_PREVIOUS`) in the secret store and
  redeploying — no data was rewritten. If some rows were already
  re-encrypted, keep both the new key as `IDENTITY_PROTECTION_KEY` and the
  old key as `IDENTITY_PROTECTION_KEY_PREVIOUS` rather than rolling back
  fully; reads still work for both old and new rows during that window. Do
  not delete `IDENTITY_PROTECTION_KEY_PREVIOUS` until the status command
  reports zero remaining rows.
- **AT OAuth signing key:** revert `ATPROTO_OAUTH_CLIENT_PRIVATE_KEY` /
  `ATPROTO_OAUTH_CLIENT_KEY_ID` to the previous values and redeploy. Because
  no persisted data is keyed to the client signing key (it only signs
  outbound assertions), rollback has no re-encryption step.

## Key loss

- **Losing `IDENTITY_PROTECTION_KEY` (and any previous key) entirely:** every
  verified email in `email_identities` and every AT OAuth session/revocation
  payload becomes permanently unreadable. Accounts remain functional for
  login (password hashes are independent), but email display, email-based
  recovery/verification flows, and any queued AT OAuth revocation all break
  until affected rows are cleared and users re-verify/re-link. This is not
  recoverable by any in-repository tool; it requires the secret store's own
  backup/recovery for the key material itself.
- **Using the wrong key:** decryption fails closed with a generic "decrypt
  email" / "decrypt AT OAuth secret" error (see
  `backend/internal/app/identity_crypto.go` and
  `backend/internal/atproto/oauth_store.go`); no plaintext or partial
  plaintext is ever returned.
- **Restoring a database backup:** the restored ciphertext is only readable
  by whichever key (current, or current+previous) was in effect when that
  backup was taken. If the backup predates a rotation that has since
  completed (previous key removed), that backup's protected data is
  unreadable unless the removed key is separately recoverable from the
  secret store's own backup.

## Verification performed for this runbook

The rotation and transition behavior described above is covered by
automated tests that run without network access:

- `backend/internal/app/identity_crypto_test.go` — wrong key rejected,
  tampered ciphertext rejected, reads succeed under current+previous during
  rotation, re-encryption is idempotent.
- `backend/internal/app/app_test.go` — production config validation fails
  closed with a clear message when `IDENTITY_PROTECTION_KEY` is missing or
  malformed, and accepts a well-formed previous key.
- `backend/internal/atproto/oauth_client_test.go` — JWKS includes both keys
  during a signing-key transition and only the current key once the
  transition ends; invalid previous-key settings are rejected.
- `backend/internal/app/identity_rekey_integration_test.go` (requires
  `TEST_DATABASE_URL`, run via `make test-db`) — seeds email and AT OAuth
  rows under a previous key, runs the `identity-rekey` command end to end,
  confirms the previous key can be removed afterward and reads still
  succeed, and confirms a second run is a no-op.

Real secret-store provisioning of these keys in a deployed environment (as
opposed to environment variables in a test/dev process) is out of reach in
this environment; see the execution log entry dated 2026-09-23 in
`docs/development/execution-log.md`.
