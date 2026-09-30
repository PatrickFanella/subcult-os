-- Private event information only; no personal accommodation requests or publication.
create table event_access_revisions (
 id uuid primary key default gen_random_uuid(),
 event_id uuid not null references events(id) on delete cascade,
 topic text not null check (topic in ('entry','bathrooms','seating','sensory','transit','contact')),
 revision integer not null check (revision > 0),
 value text not null,
 details text not null default '' check (length(details) <= 1000),
 source_kind text not null check (source_kind in ('unknown','organizer_assertion','event_observation','external_reference')),
 source_reference text not null default '' check (length(source_reference) <= 500),
 reviewed_at timestamptz,
 expires_at timestamptz,
 correction_reason text not null check (length(correction_reason) between 1 and 1000),
 recorded_by_person_id uuid not null references people(id),
 request_key uuid not null unique,
 request_fingerprint char(64) not null,
 recorded_at timestamptz not null default clock_timestamp(),
 unique(event_id,topic,revision),
 check (value='unknown' or (topic in ('entry','bathrooms') and value in ('yes','no')) or (topic='seating' and value in ('available','limited','not_available')) or (topic in ('sensory','transit','contact') and value='known')),
 check ((value='unknown' and source_kind='unknown' and source_reference='' and details='' and reviewed_at is null and expires_at is null) or (value<>'unknown' and source_kind<>'unknown' and source_reference<>'' and reviewed_at is not null)),
 check (value<>'known' or details<>''),
 check (expires_at is null or expires_at > reviewed_at)
);
