alter table events add column ends_at timestamptz;
alter table events add constraint events_interval_check check (ends_at is null or ends_at > starts_at);

create table event_availability_polls (
 id uuid primary key default gen_random_uuid(),
 event_id uuid not null references events(id) on delete cascade,
 prompt text not null check (char_length(prompt) between 1 and 500),
 token_hash text not null unique,
 status text not null default 'open' check (status in ('open','closed')),
 show_names boolean not null default false,
 closes_at timestamptz,
 chosen_option_id uuid,
 created_by_person_id uuid not null references people(id),
 created_at timestamptz not null default now(),
 rate_window timestamptz not null default now(),
 rate_count integer not null default 0
);
create index event_availability_polls_event_idx on event_availability_polls(event_id);
create table event_availability_options (
 id uuid primary key default gen_random_uuid(),
 poll_id uuid not null references event_availability_polls(id) on delete cascade,
 sort_order integer not null,
 starts_at timestamptz not null,
 ends_at timestamptz,
 timezone text not null check (char_length(timezone) between 1 and 64),
 check (ends_at is null or ends_at > starts_at),
 unique(id,poll_id),unique(poll_id,sort_order)
);
alter table event_availability_polls add constraint poll_chosen_option_fk
 foreign key (chosen_option_id,id) references event_availability_options(id,poll_id) deferrable initially deferred;
create table event_availability_responses (
 id uuid primary key default gen_random_uuid(),
 poll_id uuid not null references event_availability_polls(id) on delete cascade,
 edit_token_hash text not null unique,
 display_name text not null check (char_length(display_name) between 1 and 80),
 private_note text not null default '' check (char_length(private_note) <= 1000),
 updated_at timestamptz not null default now(),
 unique(id,poll_id)
);
create index event_availability_responses_poll_idx on event_availability_responses(poll_id);
create table event_availability_answers (
 response_id uuid not null,
 poll_id uuid not null,
 option_id uuid not null,
 answer text not null check (answer in ('yes','if_need_be','no')),
 primary key(response_id,option_id),
 foreign key(response_id,poll_id) references event_availability_responses(id,poll_id) on delete cascade,
 foreign key(option_id,poll_id) references event_availability_options(id,poll_id) on delete cascade
);
