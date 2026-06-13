# subcult-os

Full-stack SUBCULT OS project with a Go backend, Vite React TypeScript Tailwind frontend, Postgres, Docker Compose, and Open Pilot conventions.

Bootstrapped from `subculture-collective/project-template`.

## Quick start

```bash
make verify
```

Useful local commands:

```bash
make help
make quick
make up-build
make urls
make smoke
make logs
make down
make reset-db
```

## First lifecycle slice

The first product slice proves:

1. Owner signs up and creates a Workspace.
2. Owner invites a Member by email; development stores invitation emails in `email_outbox`.
3. Owner creates and publishes a direct-link Public Event Page.
4. Guest reserves a free Ticket with email and optional display name.
5. Member runs mobile-friendly Door Check-In by manual lookup/code.
6. Owner runs End of Night and views the private Event Report.

Run locally:

```bash
make up-build
```

Then open:

- Web: http://localhost:38079
- API health: http://localhost:38080/api/health

The default host ports are intentionally high to avoid common local conflicts: `WEB_PORT=38079`, `API_PORT=38080`, and `POSTGRES_PORT=35432`.

Use `make urls` to print the actual published ports, `make smoke` to check the running Docker stack, and `make reset-db` to delete local Postgres data and rebuild if an old volume has stale credentials.

## Open Pilot

This repository keeps the base Open Pilot issue template and PR template.
Bootstrap labels after creating a repo from this template:

```bash
open-pilot labels bootstrap PatrickFanella/subcult-os
```

## License

GPL-3.0-only. See `LICENSE`.
