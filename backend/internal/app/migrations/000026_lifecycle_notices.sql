-- Listing-only notices. Queue acceptance is distinct from provider delivery.
create table lifecycle_notices (
 id uuid primary key default gen_random_uuid(),
 change_id uuid not null unique references event_lifecycle_changes(id) on delete cascade,
 action_id uuid not null unique references event_lifecycle_actions(id) on delete cascade,
 workspace_id uuid not null references workspaces(id) on delete cascade,
 event_id uuid not null,
 occurrence_id uuid not null references event_occurrences(id) on delete cascade,
 occurrence_updated_at timestamptz not null,
 public_cid text not null,
 subject text not null check (octet_length(subject) between 1 and 1000),
 body text not null check (octet_length(body) between 1 and 16384),
 audiences jsonb not null check (jsonb_typeof(audiences)='array'),
 preview_hash text not null check (length(preview_hash)=64),
 request_key uuid not null,
 approved_by_person_id uuid not null references people(id),
 queued_at timestamptz not null default now(),
 foreign key (event_id,workspace_id) references events(id,workspace_id) on delete cascade,
 unique(workspace_id,request_key),
 unique(workspace_id,occurrence_id,occurrence_updated_at)
);
create table lifecycle_notice_recipients (
 id uuid primary key default gen_random_uuid(),
 notice_id uuid not null references lifecycle_notices(id) on delete cascade,
 recipient_email text not null check (length(recipient_email) between 1 and 320),
 source_type text not null check (source_type in ('ticket','crew_person','crew_application')),
 source_id uuid not null,
 outbox_id uuid unique references email_outbox(id),
 withheld_reason text check (withheld_reason='recipient_suppressed'),
 unique(notice_id,recipient_email),
 check ((outbox_id is null)=(withheld_reason is not null))
);
alter table email_outbox drop constraint email_outbox_delivery_status_check;
alter table email_outbox add constraint email_outbox_delivery_status_check check
 (delivery_status in ('held','pending','leased','accepted','failed','quarantined','suppressed','withheld_consent','withheld_authority'));
