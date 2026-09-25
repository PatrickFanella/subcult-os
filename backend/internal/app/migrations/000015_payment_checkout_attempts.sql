-- COMMERCE-01: retain the local record that authorizes a provider checkout
-- before the provider is contacted.  A pending ticket is deliberately kept
-- reserved when creation is ambiguous; an operator can reconcile it from the
-- durable attempt rather than silently selling the same capacity twice.
create table payment_checkout_attempts (
  id uuid primary key default gen_random_uuid(),
  ticket_id uuid not null unique references tickets(id) on delete cascade,
  provider text not null check (provider = 'stripe'),
  provider_idempotency_key text not null unique check (trim(provider_idempotency_key) <> ''),
  status text not null default 'creating' check (status in ('creating', 'ready', 'unknown', 'fulfilled', 'expired', 'anomalous')),
  provider_session_id text unique,
  last_error_code text check (last_error_code is null or last_error_code in ('provider_unknown', 'provider_rejected', 'persistence_failed')),
  created_at timestamptz not null default now(),
  ready_at timestamptz,
  unknown_at timestamptz,
  terminal_at timestamptz,
  updated_at timestamptz not null default now()
);

create index payment_checkout_attempts_status_idx
  on payment_checkout_attempts (status, created_at);

-- The event ledger remains idempotent by provider event ID.  These fields
-- retain the outcome of accepted-but-anomalous callbacks without storing raw
-- provider errors or trusting callback metadata as the source of record.
alter table payment_webhook_events
  add column outcome text not null default 'received' check (outcome in ('received', 'fulfilled', 'expired', 'ignored', 'anomalous')),
  add column outcome_reason text,
  add column checkout_attempt_id uuid references payment_checkout_attempts(id) on delete set null,
  add column ticket_id uuid references tickets(id) on delete set null;
