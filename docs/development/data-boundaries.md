# Data, identity and consent boundaries
Status: acceptance requirements for new work; not a security audit of the full applications. [`privacy-audit-2026-09-23.md`](privacy-audit-2026-09-23.md) is the dated route/export/log/telemetry audit against these rules; [`data-lifecycle.md`](data-lifecycle.md) specifies retention, access export, account deletion, correction and why authoritative deletion cannot erase downstream copies. Re-run the audit when a new anonymous route, export or log statement is added.

## Public payload rule
Use allowlisted projection builders. Test both property names and sentinel values across nested structures. A public schema that accepts a field does not establish permission to publish the field.
Ordinary API visibility is not permission to replicate content onto public AT infrastructure. Independently review the proposed transfer.

## Private classes
Never publish tickets/codes, contact emails or phone numbers, invitation tokens, OAuth tokens, member roles, consent receipts, attendance, staff assignments, settlement details, private archive notes, precise personal location or protected venue details.
Private observability should use opaque request/intent identifiers; avoid bodies, OAuth callback parameters, contact strings and full URLs with credentials. Restrict error details to operators who can access the event.

## Consent
Keep transactional notices distinct from marketing. Reservation, purchase, scene membership, push permission and public follow are not interchangeable.
For any future audience delivery, specify sender, channel, purpose, scope, verification and suppression authority. Recheck suppression immediately before sending. If consent status cannot be established, do not send.
The first consolidation slices transfer no contacts and perform no audience import/export. Consent portability remains separate research, not a shortcut around either legacy ledger.
[`consent.md`](consent.md) (CONSENT-01, migration 000013) implements this section's requirements: the `consent_grants` schema, `checkSendPermission` (which never derives permission from tickets, contacts, role applications, atproto identity links or workspace membership) and its send-time recheck in `processEmailDeliveries`. [`announcements.md`](announcements.md) (SIGNAL-01, migration 000014) is the one implemented announcement send path, built on top of that boundary.

## Authorization checks
Enforce workspace membership and role at each OS boundary. Enforce current creator/PDS authorization at publication. Require both for operator-to-public mutations.
Reject cross-workspace object references even when UUIDs are valid. Avoid automatic account linking based on matching email addresses. Revocation must invalidate queued work before delivery or publication when required.

## Location and time
Keep protected locations behind existing grants. Public maps use only the approved coarse/public projection. Define how time zone and daylight-saving changes are represented before mapping local editor values to public occurrences.
If an external public record changes time or location, require operator review for ticketed events rather than silently rewriting operational facts.

## Threat cases to test
Malicious record text/media URL; private-network resolver targets; oversized payload; revoked authority between queue and execution; lost response followed by retry; cross-tenant ID substitution; sensitive field added to internal DTO; logging of secret-bearing errors; stale public record claiming a misleading location.

## Protocol limits
The current AT overview describes repositories as public and non-public data mechanisms as future work. Do not invent an encrypted private-data layer by putting confidential operational payloads into public records. [Protocol overview](https://atproto.com/specs/atp).
Use the current supported OAuth profile and existing repository implementation, subject to review. [OAuth specification](https://atproto.com/specs/oauth).
Independent security and privacy review remains a release gate for new trust boundaries.
