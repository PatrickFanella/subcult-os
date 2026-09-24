-- CONSENT-01: verified channel consent and suppression semantics, per
-- docs/development/consent.md.
--
-- consent_grants is the only source of announcement-purpose send
-- permission. It never derives permission from any other table: no
-- foreign key or trigger here reads tickets, contacts,
-- event_role_applications, atproto identity links or workspace_members,
-- and application code (checkSendPermission in consent.go) must not
-- either. A grant is private: recipient_address, verification_token_hash
-- and withdrawal_reason are never projected to any public route.
--
-- recipient_address is stored as plaintext, the same convention already
-- used for other private recipient columns in this codebase (for example
-- tickets.email, contacts.email); this is noted, not hidden, in
-- consent.md.
--
-- channel is constrained to 'email' today. Adding 'sms' later only needs
-- a new migration widening this check constraint, matching the pattern
-- migration 000010 used to widen workspace_members.role.
--
-- "unique active grant" means at most one non-withdrawn row per
-- (workspace_id, channel, recipient_address, purpose), whether or not it
-- is yet verified; the partial unique index below enforces this so a
-- second announcement request for the same address must withdraw the
-- existing grant first rather than accumulate duplicates.
create table consent_grants (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references workspaces(id) on delete cascade,
  channel text not null check (channel in ('email')),
  recipient_address text not null check (trim(recipient_address) <> ''),
  purpose text not null check (purpose in ('announcement', 'transactional')),
  scope text not null default '',
  verification_token_hash text not null,
  verified_at timestamptz,
  disclosure_version text not null check (trim(disclosure_version) <> ''),
  granted_at timestamptz not null default now(),
  withdrawn_at timestamptz,
  withdrawal_reason text,
  source text not null check (source in ('explicit_form', 'operator_recorded')),
  created_by_person_id uuid references people(id),
  created_at timestamptz not null default now()
);

create unique index consent_grants_active_idx
  on consent_grants (workspace_id, channel, recipient_address, purpose)
  where withdrawn_at is null;

-- The verification token also serves as the public confirm/withdraw link
-- credential, so it must be unique regardless of withdrawal state.
create unique index consent_grants_token_idx
  on consent_grants (verification_token_hash);

create index consent_grants_workspace_idx
  on consent_grants (workspace_id, created_at desc);

-- purpose defaults every existing and legacy-binary row to 'transactional',
-- so identity verification/recovery, ticket confirmations and workspace
-- invitations keep sending unchanged: checkSendPermission never requires a
-- grant for a transactional message. workspace_id is nullable because none
-- of those existing call sites currently attach one; it is required only
-- so a future announcement-purpose row (no such row is enqueued by this
-- slice) carries the workspace checkSendPermission needs to look up a
-- grant. Both columns are additive and read by no query in this migration.
alter table email_outbox
  add column purpose text not null default 'transactional' check (purpose in ('transactional', 'announcement')),
  add column workspace_id uuid references workspaces(id) on delete set null;

-- Add the terminal "denied by consent/suppression at send time" status
-- alongside the existing set from migration 000006.
alter table email_outbox drop constraint email_outbox_delivery_status_check;
alter table email_outbox add constraint email_outbox_delivery_status_check
  check (delivery_status in ('held', 'pending', 'leased', 'accepted', 'failed', 'quarantined', 'suppressed', 'withheld_consent'));
