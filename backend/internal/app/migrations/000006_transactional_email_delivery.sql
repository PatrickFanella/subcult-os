-- Existing and old-binary inserts remain held: enabling a worker never drains
-- legacy development mail. Only the new explicitly enabled API opts rows in.
alter table email_outbox
  add column delivery_status text not null default 'held'
    check (delivery_status in ('held','pending','leased','accepted','failed','quarantined','suppressed')),
  add column sender_address text not null default '',
  add column reply_to_address text not null default '',
  add column attempts integer not null default 0 check (attempts >= 0),
  add column first_attempt_at timestamptz,
  add column next_attempt_at timestamptz not null default now(),
  add column expires_at timestamptz not null default (now() + interval '23 hours'),
  add column lease_token uuid,
  add column lease_until timestamptz,
  add column provider_message_id uuid,
  add column accepted_at timestamptz,
  add column last_error_code text;
create index email_outbox_delivery_due on email_outbox (next_attempt_at, id)
  where delivery_status in ('pending','leased');
create unique index email_outbox_provider_message on email_outbox (provider_message_id)
  where provider_message_id is not null;
