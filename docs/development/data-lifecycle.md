# Data lifecycle — retention, export, deletion and correction (Issue #15)

Status: specification for Issue #15 criterion (d). This document describes the intended behavior of a future access-export/deletion pipeline and pins the parts already true of the current system. No export or deletion endpoint exists yet; nothing in this document should be read as a claim that one does. Only the schema/tests actually in this slice back the "current system" statements below.

## Data classes and retention

| Class | Examples (tables) | Retention | Notes |
|---|---|---|---|
| Account/identity | `people`, `email_identities`, `sessions`, `identity_sessions` | Retained while the account is active; sessions expire on their existing `access_expires_at`/`refresh_expires_at` schedule regardless of account status | Deletion is described below; there is no automatic time-based purge of an active account today. |
| Workspace/operational | `workspaces`, `workspace_members`, `events`, `event_occurrences`, `event_occurrence_profiles` | Retained for the life of the workspace; a departing member's `workspace_members.removed_at` is set, the row is kept for audit/history | Removing a member does not delete their authored content (events, contacts, notes); authorship is workspace history, not personal data of the remover alone. |
| Contacts | `contacts` | Retained for the life of the workspace unless an operator deletes the row | Contacts are the workspace's own record of a relationship (e.g., a past collaborator or volunteer), not the contact's self-managed account; see "Whose data this is" below. |
| Staffing/role applications | `event_staffing`, `event_role_applications` | Retained for the life of the event/workspace | Contains applicant name/email/message; same handling as contacts. |
| Reservations/attendance/tickets | `tickets` | Retained for financial/audit reasons (see Finance below); not purged on request alone | A ticket is both the attendee's data and the organizer's settlement record; see Finance retention. |
| Consent | `consent_grants` (migration 000013; see `docs/development/consent.md`) | Retained until withdrawn; a withdrawn grant's row is kept, not deleted, so the withdrawal itself stays auditable | `CONSENT-01` (issue #23) defines grant/verification/suppression semantics only; no announcement send path exists yet (that is SIGNAL-01, issue #24), and this table is not yet wired into export/deletion below. |
| Precise personal/venue location | `cultural_place_protected_details` (street address, access notes) | Retained for the life of the owning `cultural_places` row | Access-restricted per `docs/development/privacy-audit-2026-09-23.md`; never included in export/deletion of a person's own account, because it belongs to the workspace/venue, not to any one person. |
| Incident notes | none exist yet as a distinct table; `event_staffing.notes` and `contacts.notes` are free text and may contain incident-like content | Same as their parent table | No dedicated incident-note table exists in this codebase today; this document does not fabricate one. |
| Finance/settlement | `tickets.amount_cents`/`currency`/`payment_status`, `audit_entries` (`ticket.reserved`, `ticket.payment_started` metadata), Stripe references | Retained for the retention period required by applicable financial/tax recordkeeping obligations (jurisdiction-dependent; this document does not set a specific number of years because the operator, not this codebase, is the data controller who must set that figure) | Never deleted solely because a person requests deletion; anonymized instead (see Deletion below). |
| Logs | process stdout via `log.Printf` in `requestLogger` | Retained per the deployment's own log retention (outside this repository) | Contains only `method`, `route` (server-owned template), `status`, `duration`; see `privacy-audit-2026-09-23.md`. No personal data to retain or purge. |

## Access export

A person may request an export of their own data. The shape of that export, when implemented, must be:

- **Included**: the requester's own `people` row (excluding password/credential hashes), their own `workspace_members` rows (workspace name/role/joined-at, not other members), events/occurrences/contacts/staffing/role-applications rows they *authored* (`created_by_person_id` = requester), their own `tickets` rows (as the ticket holder, matched by verified email or account link — not by an unverified email claim), and their own `event_role_applications` rows (as the applicant).
- **Excluded because it belongs to someone else**: another person's contact details, another workspace member's account data, another attendee's ticket, `cultural_place_protected_details` (belongs to the venue/workspace, not to any one requester — see the access-control test `TestCulturalPlaceProtectedDetailsRequireWorkspaceMembership`), settlement rows for events the requester did not organize, and any workspace-authored content where the requester was only a subject (e.g., they are named in a staffing note written by someone else) rather than the author or the row's own subject-of-record.
- **Excluded because it is a system integrity record, not personal data about the requester specifically**: `schema_migrations`, `audit_entries` rows about *other* people's actions, `email_provider_events`.
- **Format/verification**: out of scope for this slice; whatever is implemented later must verify the requester's identity/account ownership before returning anything above the coarse-public projection, and must never accept an unauthenticated `?email=` parameter as sufficient proof of identity (this is the same anti-pattern the audit's `email-outbox` finding names for a different route).

## Account deletion

When a person requests deletion of their account:

- **Deleted**: the `people` row's credential material (password hash, recovery tokens), all of that person's `identity_sessions`, and any `contacts`/staffing/role-application rows where they are the *sole* subject and no other party has a legitimate retained interest (e.g., a contact card the person created for themselves and never shared operationally).
- **Anonymized, not deleted**: `tickets` rows where the person was the buyer/attendee of a *past* event — the row is retained for the organizer's settlement/financial history, but `email` and `display_name` are overwritten with a placeholder (e.g., `deleted-person@invalid.example`, `"Deleted person"`); `event_role_applications.applicant_email`/`applicant_name` for a past application are anonymized the same way; `audit_entries.metadata` values that embedded a plaintext email for that person are not proactively rewritten (their audit purpose is to record what happened, not to remain queryable by email) but any future audit read path must not resolve an anonymized ticket back to a live email.
- **Retained unmodified for legal/financial reasons**: settlement totals, amounts, currencies and payment-processor references stay bound to the anonymized ticket row so an organizer's financial history and any tax/dispute record stays accurate; this is retained by amount/date, not by identity.
- **Not deleted merely by request**: content the person authored on behalf of a workspace (an event, a role description, a public event listing) is workspace history, not that person's personal data alone; deletion removes the *personal* fields above, not the workspace's operational record. If the workspace itself is deleted, that is a separate, workspace-scoped operation, not implied by any one member's account deletion.
- Nothing in this slice implements the deletion job described above. This section specifies the target behavior a future migration/worker must satisfy; see "Remaining limits" in `privacy-audit-2026-09-23.md` for what is out of scope here.

## Correction

A person may request correction of their own `people` row (display name, email — subject to the existing verification flow) and of any row where they are the sole authoring subject (e.g., their own submitted role application, before it is reviewed). Correction of a workspace-authored operational record (an event, a contact card another person maintains about them) is the workspace's decision, not an automatic right of the data subject, because the workspace — not the subject — is the author of record for that field; a person who disputes workspace-authored content about them should raise it with that workspace, and if unresolved, the operator adjudicates.

## Authoritative deletion cannot erase downstream copies

This system's local database, and any AT Protocol record it later publishes (per `docs/development/decisions.md` D6–D10, none published yet), are each an *authoritative source*, not the only copy. Deleting or anonymizing the authoritative row here does not and cannot:

- retract a copy already fetched by an AT Protocol relay, another PDS, or an app view that mirrored the record before deletion;
- retract a copy a third party took (screenshot, cache, export) of a public event page or public preview before it changed;
- force a federated consumer to re-fetch and notice the deletion on any particular timeline.

What this system does instead, once the described pipeline exists:

- **Tombstone/status, not silent removal**: an authoritative record that is deleted is marked deleted (a status field or an AT Protocol tombstone record, per the protocol's own delete-record mechanism) rather than having its row silently vanish, so any consumer that re-checks the authoritative source gets an unambiguous "this was deleted" signal instead of a 404 that could be mistaken for "never existed" or a transient error.
- **Stop projecting**: the public projection (`cultural_public_projection.go`, `handleListPublicEvents`, `handlePublicEvent`) stops serving the deleted/anonymized content from the moment the authoritative row changes; this system does not continue to originate the content once its author has deleted it, even though it cannot reach into other operators' stores.
- **Notify, where the protocol/architecture supports it**: for any published AT Protocol record, deletion is expressed as the protocol's own record-deletion operation so that relays and app views that follow the account's repository *can* observe and honor it; this system does not claim that all consumers *will* honor it, because that is outside this system's control by the protocol's own design (`docs/development/data-boundaries.md`'s "Protocol limits" section already states repositories are public).
- This is a limitation of the underlying protocol and of any system that publishes to it, not a defect introduced by this codebase; documenting it here is required so no future feature is built on the false premise that authoritative deletion is instantaneous and total.

## Remaining limits

- No export or deletion endpoint exists; this document specifies the target shape only, per the Issue #15 brief's instruction not to implement the full pipeline in this slice.
- `consent_grants` (migration 000013, `docs/development/consent.md`) defines grant/verification/suppression semantics and a send-time recheck, not export/deletion: a recipient's own grants are not yet included in the access export or account deletion behavior specified above, and no announcement send path exists yet to generate audience data beyond the grant itself.
- The specific financial-record retention period is a jurisdiction- and operator-dependent legal question this codebase does not decide; whoever operates a deployment must set and document that figure.
- No dedicated incident-note table exists; free-text fields (`contacts.notes`, `event_staffing.notes`) may incidentally contain incident-like content and are handled under those tables' rules above.
