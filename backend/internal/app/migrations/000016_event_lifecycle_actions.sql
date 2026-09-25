-- LIFE-01: durable local decisions and independently retryable external action
-- intents. This migration dispatches nothing; refunds, provider calls, notice
-- content and public-record writes remain separate later workflows.
create table event_lifecycle_changes (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references workspaces(id) on delete cascade,
  event_id uuid not null,
  kind text not null check (kind in ('cancellation', 'reschedule')),
  reason text not null check (length(trim(reason)) between 1 and 1000),
  target_revision text not null check (length(trim(target_revision)) between 1 and 256),
  decision_snapshot jsonb not null check (jsonb_typeof(decision_snapshot) = 'object' and octet_length(decision_snapshot::text) <= 16384),
  approved_by_person_id uuid not null references people(id),
  status text not null default 'approved' check (status in ('approved', 'superseded')),
  created_at timestamptz not null default now(),
  superseded_at timestamptz,
  foreign key (event_id, workspace_id) references events(id, workspace_id) on delete cascade
);
create index event_lifecycle_changes_event_idx on event_lifecycle_changes (event_id, created_at desc);

create table event_lifecycle_actions (
  id uuid primary key default gen_random_uuid(),
  change_id uuid not null references event_lifecycle_changes(id) on delete cascade,
  action_kind text not null check (action_kind in ('public_record', 'provider_ticket', 'operational_notice', 'refund')),
  destination text not null check (length(trim(destination)) between 1 and 200),
  idempotency_key text not null unique check (length(trim(idempotency_key)) between 1 and 256),
  payload jsonb not null check (jsonb_typeof(payload) = 'object' and octet_length(payload::text) <= 16384),
  status text not null default 'pending' check (status in ('pending', 'running', 'succeeded', 'retryable', 'unknown', 'failed', 'superseded')),
  attempt_count integer not null default 0 check (attempt_count >= 0),
  next_attempt_at timestamptz,
  lease_token uuid,
  lease_expires_at timestamptz,
  failure_category text check (failure_category is null or failure_category in ('lease_expired', 'transport', 'provider_rejected', 'provider_unavailable', 'validation', 'permission', 'reconciliation_required', 'internal')),
  provider_reference text check (provider_reference is null or length(provider_reference) <= 200),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  finished_at timestamptz,
  unique (change_id, action_kind, destination)
);
create index event_lifecycle_actions_due_idx on event_lifecycle_actions (next_attempt_at, created_at)
  where status in ('pending', 'retryable');
