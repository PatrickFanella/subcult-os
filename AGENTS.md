# Agent Instructions

This repository is `subcult-os`, a Go full-stack project with Vite React TypeScript, Postgres, and Docker, bootstrapped from `subculture-collective/project-template`.

## Priorities

1. Keep the project runnable and minimal.
2. Preserve Open Pilot issue and PR templates.
3. Keep generated caches, secrets, and local data out of git.
4. Update this file and `README.md` when stack conventions change.

## Verification

Run:

```bash
make verify
```

Use narrower commands only when an issue explicitly asks for a smaller check.

## Cloned Dependency Source

Read-only dependency source repositories are available under
`.blacktower/clonedeps/repos/` for inspection. Do not edit these clones.

- `.blacktower/clonedeps/repos/bluesky-social__indigo/` — `bluesky-social/indigo` at `41278964ec8e3253e70d4e919dfb8e34211c543d`; use it to inspect the pinned unstable AT Protocol syntax and OAuth implementation.
