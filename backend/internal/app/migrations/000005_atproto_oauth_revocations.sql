-- Credentials survive local unlink only in this encrypted, bounded outbox.
create table atproto_oauth_revocations (
  id uuid primary key default gen_random_uuid(),
  person_id uuid references people(id) on delete set null,
  did text not null,
  session_id text not null,
  payload_ciphertext bytea,
  status text not null default 'pending'
    check (status in ('pending', 'leased', 'completed', 'quarantined')),
  attempts integer not null default 0 check (attempts >= 0),
  next_attempt_at timestamptz not null default now(),
  lease_until timestamptz,
  lease_token uuid,
  last_error_code text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  expires_at timestamptz not null default now() + interval '7 days',
  unique (did, session_id),
  check (status <> 'leased' or (lease_until is not null and lease_token is not null)),
  check (status not in ('pending', 'leased') or payload_ciphertext is not null)
);

create index atproto_oauth_revocations_due
  on atproto_oauth_revocations (next_attempt_at)
  where status in ('pending', 'leased');
