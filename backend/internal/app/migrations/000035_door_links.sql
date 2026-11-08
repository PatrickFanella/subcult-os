create table event_door_links (
  id uuid primary key default gen_random_uuid(),
  event_id uuid not null references events(id) on delete cascade,
  label text not null check (char_length(label) between 1 and 80),
  token_hash text not null unique,
  valid_from timestamptz not null,
  valid_until timestamptz not null,
  created_by_person_id uuid not null references people(id),
  created_at timestamptz not null default now(),
  revoked_at timestamptz,
  rate_window timestamptz not null default now(),
  rate_count integer not null default 0,
  check (valid_until > valid_from and valid_until <= valid_from + interval '7 days'),
  unique (id, event_id)
);
create index event_door_links_event_idx on event_door_links(event_id);
alter table tickets add column checked_in_by_door_link_id uuid;
alter table tickets add constraint tickets_door_link_event_fk
  foreign key (checked_in_by_door_link_id, event_id) references event_door_links(id, event_id);
-- Historical fixtures may have no actor. Every new link admission must have
-- exactly its link actor, and can never also claim a person actor.
alter table tickets add constraint tickets_check_in_actor_check check (
  checked_in_by_door_link_id is null or
  (checked_in_by_person_id is null and checked_in_at is not null and status = 'checked_in')
);
