create extension if not exists pgcrypto;

create table if not exists people (
  id uuid primary key default gen_random_uuid(),
  email text not null unique,
  display_name text,
  password_hash text not null,
  created_at timestamptz not null default now()
);

create table if not exists sessions (
  id uuid primary key default gen_random_uuid(),
  person_id uuid not null references people(id) on delete cascade,
  token_hash text not null unique,
  expires_at timestamptz not null,
  created_at timestamptz not null default now()
);

create table if not exists workspaces (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  created_at timestamptz not null default now()
);

create table if not exists workspace_members (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references workspaces(id) on delete cascade,
  person_id uuid not null references people(id) on delete cascade,
  role text not null check (role in ('owner', 'member')),
  removed_at timestamptz,
  created_at timestamptz not null default now(),
  unique (workspace_id, person_id)
);

create table if not exists workspace_invitations (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references workspaces(id) on delete cascade,
  email text not null,
  role text not null default 'member' check (role = 'member'),
  token_hash text not null unique,
  accepted_at timestamptz,
  invited_by_person_id uuid not null references people(id),
  created_at timestamptz not null default now()
);

create table if not exists events (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references workspaces(id) on delete cascade,
  title text not null,
  starts_at timestamptz not null,
  public_description text not null,
  location_display text not null,
  ticket_allocation integer not null check (ticket_allocation >= 0),
  status text not null default 'draft' check (status in ('draft', 'published', 'end_of_night')),
  public_slug text unique,
  published_at timestamptz,
  ended_at timestamptz,
  created_by_person_id uuid not null references people(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists tickets (
  id uuid primary key default gen_random_uuid(),
  event_id uuid not null references events(id) on delete cascade,
  email text not null,
  display_name text,
  code text not null unique,
  status text not null default 'reserved' check (status in ('reserved', 'checked_in')),
  checked_in_at timestamptz,
  checked_in_by_person_id uuid references people(id),
  created_at timestamptz not null default now()
);

create table if not exists event_reports (
  id uuid primary key default gen_random_uuid(),
  event_id uuid not null unique references events(id) on delete cascade,
  generated_by_person_id uuid not null references people(id),
  generated_at timestamptz not null default now(),
  snapshot jsonb not null
);

create table if not exists audit_entries (
  id uuid primary key default gen_random_uuid(),
  actor_person_id uuid references people(id),
  action text not null,
  subject_type text not null,
  subject_id uuid,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create table if not exists email_outbox (
  id uuid primary key default gen_random_uuid(),
  recipient_email text not null,
  subject text not null,
  body text not null,
  related_type text not null,
  related_id uuid,
  created_at timestamptz not null default now()
);

alter table events add column if not exists pricing_mode text not null default 'free' check (pricing_mode in ('free', 'fixed'));
alter table events add column if not exists ticket_price_cents integer not null default 0 check (ticket_price_cents >= 0);
alter table events add column if not exists ticket_currency text not null default 'usd';

alter table tickets add column if not exists payment_status text not null default 'free' check (payment_status in ('free', 'pending', 'paid', 'cancelled'));
alter table tickets add column if not exists amount_cents integer not null default 0 check (amount_cents >= 0);
alter table tickets add column if not exists currency text not null default 'usd';
alter table tickets add column if not exists stripe_checkout_session_id text unique;
alter table tickets add column if not exists paid_at timestamptz;

create table if not exists payment_webhook_events (
  id text primary key,
  provider text not null default 'stripe',
  event_type text not null,
  processed_at timestamptz not null default now()
);
