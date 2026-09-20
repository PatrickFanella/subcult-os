# Development documentation validation
Date: 2026-09-20. The initial checks below are historical; subsequent implementation and verification are recorded in [execution-log.md](execution-log.md). The Gitea roadmap and stacked PRs now track remaining delivery.

## Evidence and scope
Repository paths, revisions, Make targets, package scripts, current DTO checker behavior and accepted domain/AT decisions were inspected directly. Current public-beta status was treated as dated documentation rather than a fresh runtime result.
Protocol overview, Lexicon guidance and OAuth specification were rechecked on their official documentation pages. New identity, protocol, projection and publication fields, states, endpoints and tests are explicitly proposals, not existing capabilities.

## Checks
- OS `scripts/check-contracts.mjs`: passed after adding public-contract forbidden-field checks.
- Required OS `make verify`: initially timed out because `pnpm` resolved to a hanging PATH wrapper. With `/usr/bin/pnpm`, web install succeeded. Mobile install populated dependencies but exited nonzero because the repository's supply-chain policy does not approve the `esbuild` build script. That policy was preserved, so the aggregate gate is not claimed.
- Focused API-01 Go serializer and disposable-PostgreSQL endpoint tests passed. Non-DB Go tests, vet, backend build, web typecheck/lint/tests/build, direct mobile typecheck/tests, and Compose rendering passed. Exact commands and limitations are in `execution-log.md`.
- `git diff --check`: passed for tracked changes.
- `validate-docs.mjs`: checks every local Markdown link, titles, whitespace and all 15 backlog items' acceptance/verification/rollback sections. Run it after edits.
- Subcults remained clean and read-only. INV-01 records file-level adapt/rewrite/fixture/reference/reject dispositions and the missing root-license evidence; no source was copied. DB-01 changed only Subcult OS migration machinery and exercised disposable local PostgreSQL; no external database, provider or deployment was changed.
- The current OS-core revision passed documentation validation with 21 documents, 55 internal links and 15 ordered backlog items. ADRs 0005/0006, the extraction inventory and manifest establish planning authority and provenance boundaries; they do not establish implemented consolidation.
- ADR 0006 removes assumed prototype API/schema compatibility. DB-01 applied the clean version-1 baseline only to a named disposable PostgreSQL container, then proved replay, concurrent-runner serialization, tamper rejection, failed-transaction rollback, database-ahead rejection and a dump/restore row-count match. Retained-data migration remains conditional on a future read-only inventory of an actual retained database.

## Artifact boundary
The ZIP is a portable snapshot of this development directory plus the repository runbooks and upstream contribution packages it links to. The repository's separate execution-order plan links to the directory; the full backlog, order, deployment safety rules and legacy-service cutover procedure remain available inside the snapshot without that external plan.
The initial documentation pass did not exercise runtime behavior. Subsequent execution entries record disposable migration/restore tests, aggregate verification and a synthetic browser identity journey; Gitea roadmap publication is complete. Real AT provider interoperability, native-device qualification and production replacement remain unqualified. Never treat an earlier documentation-only statement as the current implementation status.
