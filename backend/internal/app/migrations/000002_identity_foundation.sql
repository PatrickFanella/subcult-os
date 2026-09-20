do $$
begin
  if exists (select 1 from people limit 1) then
    raise exception 'identity foundation requires an empty prototype people table; inventory retained accounts before migration';
  end if;
end
$$;

create table email_identities (
  id uuid primary key default gen_random_uuid(),
  person_id uuid not null references people(id) on delete cascade,
  email_ciphertext bytea not null,
  email_lookup_hash char(64) not null unique,
  verified_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (person_id)
);

create table identity_challenges (
  id uuid primary key default gen_random_uuid(),
  person_id uuid not null references people(id) on delete cascade,
  email_identity_id uuid not null references email_identities(id) on delete cascade,
  purpose text not null check (purpose in ('verify_email', 'recover_password')),
  token_hash char(64) not null unique,
  expires_at timestamptz not null,
  consumed_at timestamptz,
  created_at timestamptz not null default now()
);

create unique index identity_challenges_one_active_purpose
  on identity_challenges (email_identity_id, purpose)
  where consumed_at is null;

create table identity_sessions (
  id uuid primary key default gen_random_uuid(),
  person_id uuid not null references people(id) on delete cascade,
  family_id uuid not null,
  generation integer not null check (generation >= 0),
  access_token_hash char(64) not null unique,
  refresh_token_hash char(64) not null unique,
  access_expires_at timestamptz not null,
  refresh_expires_at timestamptz not null,
  rotated_at timestamptz,
  revoked_at timestamptz,
  reuse_detected_at timestamptz,
  replaced_by_session_id uuid references identity_sessions(id) on delete set null,
  created_at timestamptz not null default now(),
  last_seen_at timestamptz not null default now(),
  unique (family_id, generation)
);

create index identity_sessions_active_access
  on identity_sessions (access_token_hash)
  where rotated_at is null and revoked_at is null;

create index identity_sessions_active_refresh
  on identity_sessions (refresh_token_hash)
  where revoked_at is null;

create index identity_sessions_person_family
  on identity_sessions (person_id, family_id, generation desc);

create table did_links (
  id uuid primary key default gen_random_uuid(),
  person_id uuid not null references people(id) on delete cascade,
  did text not null unique,
  handle text,
  status text not null default 'active' check (status in ('active', 'revoked')),
  verified_at timestamptz not null,
  revoked_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  check ((status = 'active' and revoked_at is null) or (status = 'revoked' and revoked_at is not null))
);

create index did_links_person_status on did_links (person_id, status);

create table auth_audit_events (
  id uuid primary key default gen_random_uuid(),
  person_id uuid references people(id) on delete set null,
  session_family_id uuid,
  event_type text not null check (event_type in (
    'signup_requested',
    'email_verified',
    'login_succeeded',
    'login_failed',
    'session_rotated',
    'session_reuse_detected',
    'session_revoked',
    'all_sessions_revoked',
    'recovery_requested',
    'password_recovered'
  )),
  created_at timestamptz not null default now()
);

create index auth_audit_events_person_created
  on auth_audit_events (person_id, created_at desc);
