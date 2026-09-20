# Continue development safely
## Before editing
1. Read the target repository's current instructions, README, relevant accepted ADRs and nearest module docs.
2. Record root, branch, HEAD and dirty state. Do not treat this pack's September snapshot as current forever.
3. Read [backlog](backlog.md), [decisions](decisions.md) and the relevant track.
4. Reproduce the smallest current failure or baseline behavior.
5. Confirm the requested task authorizes implementation; proposal writing alone does not.

## Work one slice
Select one eligible backlog item with resolved blocking decisions. Keep unrelated user files intact. Use existing stack, repository contracts and test tools.
Write the named new negative checks before claiming the new boundary. Update backend, web and mobile together when the API changes.
Do not invent a passing command for a test that has not been implemented. Record missing provider or environment prerequisites explicitly.

## Before handing off
Run focused checks and required repository gates. Inspect diffs for formatter side effects, secret values, debug data and unintended lockfile changes.
Record source versus runtime evidence separately. Include screenshots or browser observations only when actually captured from the relevant artifact.
Update task status, actual commands, decisions and next dependency. Leave failed evidence intact.

## Suggested next-agent prompt
“Implement DB-01 in Subcult OS using docs/development. Treat Subcults as read-only reference material and preserve all user-owned files. Design the clean platform schema and smallest ordered migration ledger, with fresh, replay, concurrency and failed-migration checks. Inventory any existing database read-only before proposing deletion; add an upgrade path only for explicitly retained real data. Do not import Subcults migrations, deploy, publish, reset data or lower gates. Record exact evidence and update the backlog.”

For INV-01, perform only a file-level source/dependency/provenance audit and update extraction-inventory.md; do not copy implementation code.

## Document maintenance
Keep this directory in Subcult OS as the authoritative task handoff. If copied into an Obsidian vault, mark it as a snapshot with the source revision/date; do not create a competing canonical copy in Subcults.
Update source-baseline.md when relevant files move. Change proposed to implemented only when source and test evidence exist. Add runtime qualification separately.
Research pricing, grant amounts and investor lists are not engineering commitments and should not enter runtime configuration.
