-- Preserve event history and extend the private ledger to distinct venue assertions.
alter table event_access_revisions add column place_id uuid references cultural_places(id) on delete cascade;
alter table event_access_revisions alter column event_id drop not null;
alter table event_access_revisions add constraint access_revision_one_scope check ((event_id is null) <> (place_id is null));
alter table event_access_revisions add constraint place_access_topic_revision_key unique(place_id,topic,revision);
alter table event_access_revisions drop constraint event_access_revisions_source_kind_check;
alter table event_access_revisions add constraint access_revision_source_kind check (source_kind in ('unknown','organizer_assertion','event_observation','venue_observation','external_reference'));
alter table event_access_revisions add constraint access_revision_source_scope check ((event_id is not null and source_kind<>'venue_observation') or (place_id is not null and source_kind<>'event_observation'));

-- Owner-created venue references recover uncertain responses without duplication.
create table venue_access_place_requests (
 request_key uuid primary key,
 workspace_id uuid not null references workspaces(id) on delete cascade,
 created_by_person_id uuid not null references people(id),
 place_id uuid not null,
 name text not null check(length(name) between 1 and 600),
 foreign key(place_id,workspace_id) references cultural_places(id,workspace_id) on delete cascade
);
