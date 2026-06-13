# Project Maintenance

Use this guide when changing `subcult-os` shared project conventions or operational defaults.

## Maintenance principles

- Keep the project runnable with one canonical verification command.
- Prefer reusable docs, Makefile targets, and Open Pilot conventions over ad hoc workflow notes.
- Keep local-only data, generated caches, and secrets out of git.
- Add tools only when they are useful for `subcult-os`, not as unrelated experiments.

## Safe project changes

Good project-level changes include:

- issue and PR template improvements
- docs structure improvements
- generic security and contribution guidance
- Makefile targets that keep verification deterministic
- `.gitignore`, `.gitattributes`, and editor defaults
- Open Pilot workflow documentation

Avoid changes such as:

- adding production deployment assumptions
- adding unrelated stack scaffolds
- committing generated outputs or local runtime state

## Update checklist

1. Create an issue describing why `subcult-os` should change.
2. Make the change in a branch.
3. Run:

   ```bash
   make verify
   docker compose config --quiet
   ```

4. Update docs if the workflow changes.
5. Open a PR and link the issue.
6. After merge, decide whether any downstream projects need the same change.

## Propagating to downstream projects

For each downstream project:

1. Review the `subcult-os` change for relevance.
2. Apply only the useful parts.
3. Preserve project-specific verification commands.
4. Run the downstream project's canonical verification command.
5. Note any intentional divergence in downstream docs.

## Versioning convention

Use git tags when a `subcult-os` update is meaningful enough for downstream projects to target:

```bash
git tag vYYYY.MM.DD
git push origin vYYYY.MM.DD
```

Do not tag trivial typo fixes unless a derived template needs a stable reference.

## Open Pilot smoke test

After substantial template maintenance, create a small Open Pilot task that changes a docs-only file and uses a narrow test command such as:

```bash
test -f docs/reference/example.md
```

Queue it only after labels are bootstrapped and the issue body is complete.
