# Stack Recipes

These are the stack conventions currently used by `subcult-os` and the preferred commands for related changes.

## Go

Current layout:

- `backend/go.mod`
- `backend/cmd/app/main.go`
- `backend/internal/` packages
- `go test ./...` as the default test command

Recommended backend verification coverage:

```bash
gofmt
go test ./...
golangci-lint run
```

## Node / TypeScript

Current TypeScript layout:

- `web/package.json`
- `web/pnpm-lock.yaml`
- `web/tsconfig.json`
- `web/src/`
- `mobile/package.json`
- `mobile/pnpm-lock.yaml`
- `mobile/tsconfig.json`
- `mobile/app/`
- `mobile/src/`

Recommended scripts:

```json
{
  "scripts": {
    "build": "tsc -b",
    "lint": "eslint .",
    "test": "vitest run",
    "format": "prettier --write ."
  }
}
```

## React / Vite / Tailwind

Keep frontend commands explicit:

```bash
pnpm --dir web run build
pnpm --dir web run test
pnpm --dir web run lint
```

## Expo / React Native / Uniwind

The `mobile/` app is the phone-first authenticated client for event-time flows.
Keep public/link-first surfaces in `web/` unless they become app-only flows.

The app is intentionally pinned to Expo SDK 54 for compatibility with the App
Store / Play Store version of Expo Go during real-device rehearsal. Revisit the
latest Expo SDK after moving to development builds.

Recommended commands:

```bash
pnpm --dir mobile run start
pnpm --dir mobile run android
pnpm --dir mobile run ios
pnpm --dir mobile run typecheck
```

When testing against the local backend from a physical device, set
`EXPO_PUBLIC_API_URL` to a LAN-reachable URL, for example:

```bash
EXPO_PUBLIC_API_URL=http://192.168.1.10:38080 pnpm --dir mobile run start
```

Use Expo Router for mobile navigation. Use semantic tokens in
`mobile/src/global.css` and `mobile/src/theme/tokens.ts` instead of hard-coding
one-off colors, spacing, or shape values in screens.

Uniwind is wired through `mobile/metro.config.js`. Import `mobile/src/global.css`
from `mobile/app/_layout.tsx`, and keep mobile component styles in semantic
Tailwind-compatible utilities such as `screen`, `panel`, `field`, `btn-primary`,
`chip`, `heading-1`, and `body-copy`.

Web styling uses the same semantic token names in `web/src/styles.css`; prefer
`bg-surface-*`, `text-fg-*`, `border-stroke-*`, and custom utilities like
`panel`, `field`, `btn-primary`, `btn-secondary`, `chip`, `kicker`, and
`notice-*` over raw color utilities.

## Event media storage

Event hero images are uploaded from mobile to the Go API, then proxied to
S3-compatible storage. The mobile app never receives S3 credentials.

Configure the API with:

- `MEDIA_S3_ENDPOINT` — MinIO/S3 API endpoint reachable by the backend.
- `MEDIA_S3_ACCESS_KEY` / `MEDIA_S3_SECRET_KEY` — scoped media credentials.
- `MEDIA_S3_BUCKET` — bucket for event media.
- `MEDIA_S3_REGION` — defaults to `us-east-1`.
- `MEDIA_PUBLIC_BASE_URL` — public unauthenticated base URL used in event DTOs.

For Almaz MinIO, note that `s3.subcult.tv` has historically been the protected
console route. Use an API route or internal endpoint for `MEDIA_S3_ENDPOINT`,
and expose only the media object base URL needed by attendee clients.

## Python

Add only if the project gains Python services:

- `pyproject.toml`
- `src/<package>/`
- `tests/`

Recommended tools:

```bash
uv run pytest
uv run ruff check .
uv run ruff format .
```

## Rust

Add only if the project gains Rust services:

- `Cargo.toml`
- `src/`
- integration tests when useful

Recommended commands:

```bash
cargo fmt --all
cargo clippy --all-targets -- -D warnings
cargo test --all
```

## Postgres

This project includes local Postgres through Docker Compose:

```bash
docker compose up -d postgres
```

`subcult-os` currently uses one embedded idempotent schema file at
`backend/internal/app/schema.sql`, wired through `backend/internal/app/db.go`.
That embedded schema is the alpha source of truth.

`migrations/` is reserved for the future ordered migration tool once destructive
changes, backfills, long-running rewrites, or production deploys require it.
Until then, additive schema edits belong in `backend/internal/app/schema.sql`.

See `docs/runbooks/database-migrations.md` for the current migration runbook.

## Docker

Keep Docker defaults boring:

- Build repeatably from the repo root.
- Keep secrets in `.env` or secret managers, never in images.
- Make `docker compose config --quiet` pass in CI.
- Prefer profiles for optional local infrastructure.
