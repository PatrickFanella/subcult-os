-- MODEL-01: minimal cultural record model (D6/D7 recommended starting points).
--
-- Adds a workspace-scoped Profile (creator or collective; "act" is a kind,
-- not a table), a Place with public fields split from a protected detail
-- table, and a public Event occurrence that relates to the existing private
-- operator `events` row without owning its ticketing/staffing/settlement
-- state. Publication columns (public_uri/public_cid) are nullable and left
-- unset in this slice; a later PUB-01 slice fills them.
--
-- Every new table carries workspace_id and a composite foreign key back to
-- its parent's (id, workspace_id) pair so a cross-workspace reference is
-- rejected by the database itself, not only by the application layer.

alter table events add constraint events_id_workspace_id_key unique (id, workspace_id);

create table cultural_profiles (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references workspaces(id) on delete cascade,
  kind text not null default 'creator' check (kind in ('creator', 'collective', 'act')),
  display_name text not null check (length(display_name) between 1 and 500),
  description text not null default '' check (length(description) <= 3000),
  public_uri text,
  public_cid text,
  created_by_person_id uuid not null references people(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (id, workspace_id)
);
create index cultural_profiles_workspace_idx on cultural_profiles (workspace_id, created_at desc);

create table cultural_places (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references workspaces(id) on delete cascade,
  name text not null check (length(name) between 1 and 600),
  locality text not null default '' check (length(locality) <= 400),
  region text not null default '' check (length(region) <= 400),
  country text check (country is null or length(country) = 2),
  coordinates_public boolean not null default false,
  public_latitude text check (public_latitude is null or length(public_latitude) between 1 and 20),
  public_longitude text check (public_longitude is null or length(public_longitude) between 1 and 20),
  public_uri text,
  public_cid text,
  created_by_person_id uuid not null references people(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (id, workspace_id),
  check (
    (not coordinates_public and public_latitude is null and public_longitude is null)
    or (coordinates_public and public_latitude is not null and public_longitude is not null)
  )
);
create index cultural_places_workspace_idx on cultural_places (workspace_id, created_at desc);

-- Protected location detail. Never selected by the public serializer; kept
-- in its own table so a stray "select * from cultural_places" cannot leak
-- it and so the public projection code has nothing to accidentally read.
create table cultural_place_protected_details (
  place_id uuid primary key references cultural_places(id) on delete cascade,
  street_address text not null default '',
  access_notes text not null default '',
  updated_at timestamptz not null default now()
);

-- The public event occurrence. One private operator event may relate to
-- more than one occurrence (for example the same show cross-listed under
-- two collectives); one occurrence relates to exactly one operator event.
create table event_occurrences (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references workspaces(id) on delete cascade,
  event_id uuid not null,
  place_id uuid,
  name text not null check (length(name) between 1 and 1000),
  description text not null default '' check (length(description) <= 12000),
  starts_at timestamptz not null,
  ends_at timestamptz,
  all_day boolean not null default false,
  timezone text check (timezone is null or length(timezone) <= 64),
  status text not null default 'scheduled'
    check (status in ('scheduled', 'rescheduled', 'postponed', 'cancelled')),
  public_uri text,
  public_cid text,
  created_by_person_id uuid not null references people(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  foreign key (event_id, workspace_id) references events (id, workspace_id) on delete cascade,
  foreign key (place_id, workspace_id) references cultural_places (id, workspace_id) on delete set null,
  unique (id, workspace_id)
);
create index event_occurrences_event_idx on event_occurrences (event_id, starts_at);
create index event_occurrences_workspace_idx on event_occurrences (workspace_id, starts_at);
create index event_occurrences_place_idx on event_occurrences (place_id);

-- Multi-host attribution: one occurrence may credit more than one profile,
-- with a role and an explicit display order.
create table event_occurrence_profiles (
  occurrence_id uuid not null,
  workspace_id uuid not null references workspaces(id) on delete cascade,
  profile_id uuid not null,
  role text not null default 'host' check (role in ('host', 'performer', 'collective')),
  sort_order integer not null default 0,
  created_at timestamptz not null default now(),
  primary key (occurrence_id, profile_id),
  foreign key (occurrence_id, workspace_id) references event_occurrences (id, workspace_id) on delete cascade,
  foreign key (profile_id, workspace_id) references cultural_profiles (id, workspace_id) on delete cascade
);
create index event_occurrence_profiles_occurrence_idx
  on event_occurrence_profiles (occurrence_id, sort_order);
