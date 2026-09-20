---
type: repository-evidence
status: proposed
created: 2026-09-19
research_as_of: 2026-09-19
tags:
  - subcult-research
---

# Subcult Repository Evidence

## Inspection scope

The proposal research inspected source and documentation at these local revisions:

- Subcults: revision `3cf88ec`, branch `fix/main-regression-recovery`, directory `/home/onnwee/Work/subcult/subcults`. Revision rechecked during packaging.
- Subcult OS: base revision `abf3f50`, research worktree `/home/onnwee/.t3/worktrees/subcult-os/t3code-e6dba5a4`.

Paths below are repository-relative provenance text. They are intentionally not links to machine-specific files so this dossier remains portable.

## Subcults assets

| Evidence path | Source-observed capability |
| --- | --- |
| `README.md` | Scene/music product, stack and privacy principles |
| `docs/adr/0007-scene-signals-touring-relationship-model.md` | Profiles/Acts, Places/Venues, Scenes, Events, Appearances, Tours and relationship distinctions |
| `internal/touring/sql_repository.go` | Durable touring repository implementation |
| `internal/audience/service.go` | Contact/consent/suppression service foundation |
| `internal/signal/delivery.go` | Delivery eligibility check near send and delivery state handling |
| `lexicons/README.md` | `tv.subcult.*` emit policy; legacy intake policy |
| `internal/atprotocol/lexicon.go` | Public validation and named private-field rejection |
| `internal/atprotocol/publication.go` | Publication mapping, revision/concurrency and reconciliation foundations |
| `docs/operations/ATPROTO_PDS.md` | OAuth/PDS/sync operations and qualification requirements |

Nine public lexicons cover profile, act, place, venue, scene, event, tour, appearance and assertion. Their existence does not prove interoperability with a second application.

The delivery implementation is not proof of exactly-once provider side effects. An external send and local status update have a failure boundary that needs explicit testing. A field reject-list is not proof that every possible private datum is excluded.

## Subcult OS assets

| Evidence path | Source-observed capability |
| --- | --- |
| `backend/internal/app/app.go` | Workspace, event, commitment, staffing, reservation, door, settlement and archive routes |
| `backend/internal/app/schema.sql` | Operations schema, state and ownership relationships |
| `README.md` | Alpha product and web/mobile scope |
| `docs/qa/public-conversion-mobile-roles-2026-06.md` | Historical verification and explicit browser/device/DB gaps |
| `CONTEXT.md` | Private/public boundaries and product constraints |

Existing code supports an integration starting point. It does not establish complete refunds, robust offline scanning, accounting compliance or production-grade granular permissions.

## Reconcile historical documents

The August 8 Subcults audience plan contains older missing-adapter/login statements. Later source and release material supersede those implementation claims. Do not repeat them as current defects.

The August 9 public-beta release contract records failed checks and unqualified provider/PDS operation. That is historical gate evidence, not a fresh failing-test count. Preserve the gate, rerun the checks and record a current result before estimating repair.

## Evidence tiers

Source exists → local tests → configured integration → staging/browser/device journey → production observation → repeated event qualification.

No step implies the next. This dossier performs research and packaging, not live release qualification. See [[Subcult Release Qualification]] and [[Subcult Research Methods and Refresh]].

