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

Current frontend layout:

- `web/package.json`
- `web/pnpm-lock.yaml`
- `web/tsconfig.json`
- `web/src/`

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
See `docs/runbooks/database-migrations.md` for the current migration runbook.

While the app is in alpha, additive schema edits in `schema.sql` are acceptable.
Destructive changes, backfills, long-running rewrites, or production databases
that need reversible deploys require a real ordered migration tool first.

## Docker

Keep Docker defaults boring:

- Build repeatably from the repo root.
- Keep secrets in `.env` or secret managers, never in images.
- Make `docker compose config --quiet` pass in CI.
- Prefer profiles for optional local infrastructure.
