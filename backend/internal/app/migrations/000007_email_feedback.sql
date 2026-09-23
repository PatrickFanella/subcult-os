-- Store only authenticated receipt identifiers and known-message outcomes.
create table email_provider_events (
 event_id text primary key check (length(event_id) between 1 and 256),
 provider_message_id uuid not null,
 event_type text not null check (event_type in ('email.delivered','email.failed','email.suppressed','email.bounced','email.complained')),
 received_at timestamptz not null default now(),
 processed_at timestamptz
);
create index email_provider_events_pending on email_provider_events(provider_message_id,received_at) where processed_at is null;
create table email_suppressions (
 recipient_email text primary key,
 reason text not null check (reason in ('email.suppressed','email.bounced','email.complained')),
 created_at timestamptz not null default now()
);
alter table email_outbox add column feedback_rank integer not null default 0 check (feedback_rank between 0 and 5);
-- Rank is monotonic: unknown, delivered, failed, suppressed, bounced, complained.
