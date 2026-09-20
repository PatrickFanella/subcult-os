# Subcult.tv platform development handoff
Updated: 2026-09-20. Status: **Subcult OS is the platform core; API privacy, ordered migrations, canonical identity, selective extraction and the AT syntax kernel are implemented**, not a completed protocol integration or release approval.

This directory converts the Subcult research and funding work into an engineering handoff. It is the canonical copy for this task. Subcults was inspected read-only and remains source material rather than the destination repository. The directory can also be copied into an Obsidian vault because internal links are relative Markdown links.

## Start here
1. Read the [development brief](development-brief.md).
2. Check the [source baseline](source-baseline.md) against the checkout you will change.
3. Review the accepted [platform architecture](architecture.md), [selective extraction inventory](extraction-inventory.md), and completed [file-level extraction manifest](subcults-extraction-manifest.md).
4. Select the first eligible task from the [backlog](backlog.md).
5. Use the [verification matrix](verification.md) and [release checklist](release-and-migrations.md).
6. Record unresolved choices in the [decision register](decisions.md).

## Documents
- [Subcults source-extraction track](subcults-track.md)
- [Subcult OS development track](subcult-os-track.md)
- [Selective extraction inventory](extraction-inventory.md)
- [Subcults file-level extraction manifest](subcults-extraction-manifest.md)
- [AT Protocol kernel and OAuth boundary](atproto-kernel.md)
- [Superseded bridge contract](bridge-contract.md)
- [User journeys and UI acceptance](journeys.md)
- [Security, consent and data placement](data-boundaries.md)
- [Upstream contribution plan](upstream.md)
- [Contributor and agent handoff](contributor-handoff.md)
- [Issue template](issue-template.md)
- [Validation record](validation.md)
- [Execution log](execution-log.md)

The execution-order plan also lives in the repository's existing docs/superpowers/plans directory as 2026-09-20-subcults-os-development.md. The plan points here rather than duplicating specifications.

## Authority
Accepted repository ADRs and current code take precedence over proposal prose. [ADR 0005](../adr/0005-subcult-os-platform-core.md) selects OS as the receiving repository, and [ADR 0006](../adr/0006-no-prototype-compatibility-contract.md) permits deliberate prototype-breaking redesign. This pack does not itself migrate or reset databases, replace authentication, copy Subcults source, publish Lexicons, enable providers or authorize deployment.
The adjacent research/funding kits supply rationale, not production requirements. Existing Open Pilot issue and PR templates remain unchanged. No hosted issues have been created or queued.

## First implementation recommendation
BASE-01, API-01, DB-01, INV-01 and IDENT-01 are complete at their documented evidence levels. AT-01 has a verified cross-language syntax foundation; its OAuth, resolver and Lexicon slices remain open. Add publication only after authority and failure behavior are approved.
