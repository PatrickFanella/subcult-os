create table atproto_oauth_requests (
  state_hash char(64) primary key,
  person_id uuid not null references people(id) on delete cascade,
  payload_ciphertext bytea not null,
  expires_at timestamptz not null,
  claimed_at timestamptz,
  created_at timestamptz not null default now()
);

create index atproto_oauth_requests_expiry
  on atproto_oauth_requests (expires_at);

create table atproto_oauth_sessions (
  id uuid primary key default gen_random_uuid(),
  person_id uuid not null references people(id) on delete cascade,
  did text not null,
  session_id text not null,
  payload_ciphertext bytea not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (did, session_id)
);

create index atproto_oauth_sessions_person
  on atproto_oauth_sessions (person_id, updated_at desc);

alter table auth_audit_events
  drop constraint auth_audit_events_event_type_check;

alter table auth_audit_events
  add constraint auth_audit_events_event_type_check check (event_type in (
    'signup_requested',
    'email_verified',
    'login_succeeded',
    'login_failed',
    'session_rotated',
    'session_reuse_detected',
    'session_revoked',
    'all_sessions_revoked',
    'recovery_requested',
    'password_recovered',
    'atproto_did_linked',
    'atproto_did_unlinked'
  ));
