-- Private owner observations. Reviews never change mail delivery state.
create table lifecycle_notice_reviews (
 id uuid primary key default gen_random_uuid(),
 notice_id uuid not null references lifecycle_notices(id) on delete cascade,
 recorded_by_person_id uuid not null references people(id),
 request_key uuid not null unique,
 note text not null check (length(note) between 1 and 2000),
 recipients jsonb not null check (jsonb_typeof(recipients)='array'),
 recorded_at timestamptz not null default clock_timestamp()
);
create index lifecycle_notice_reviews_notice_idx on lifecycle_notice_reviews(notice_id,recorded_at,id);
