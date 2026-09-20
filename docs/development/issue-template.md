# Development issue template
Draft locally; use the repository's existing Open Pilot template when creating a hosted issue. Creating this document does not authorize queuing.

## Goal
One user-visible result or independently reusable boundary.

## Context
Backlog ID:
Repository and exact base revision:
Relevant accepted ADRs:
Source paths:
Known failing baseline:
Dependencies and unresolved decision IDs:

## Acceptance Criteria
- Given [state], when [action], then [observable result].
- Unauthorized and cross-workspace behavior:
- Error/retry/conflict behavior:
- Public/private data boundary:
- Required retained-data or external-consumer compatibility (or `none evidenced`):

## Test Command
Existing deterministic command:
New test required (clearly marked not yet implemented):
Database/provider/browser prerequisites:
Full repository gate and expected artifacts:

## Out of Scope
No unrelated refactor, new provider, deployment, user-data migration, external messages or changed product policy.

## Migration and Rollback
Fresh installation:
Existing-data inventory and retention classification:
Conditional retained-data migration (or `not applicable`):
Failure halfway through:
Rollback or forward-fix:
Public-write compensation if applicable:

## Queue Checklist
- [ ] Required decisions resolved.
- [ ] Source baseline inspected and user changes preserved.
- [ ] Acceptance can be disproved by the named checks.
- [ ] New tests distinguished from existing commands.
- [ ] Sensitive data and credentials excluded from artifacts.
- [ ] Human approves ready-for-agent and agent:queued handling.

## Completion record
Revision/diff:
Checks run and exact results:
Not run and why:
User-visible evidence:
Remaining risks:
Next dependency:
