-- LIFE-01 operator worklist: a client-provided decision key makes a retried
-- private decision stable without conflating different occurrence revisions.
alter table event_lifecycle_changes
  add column request_key uuid not null default gen_random_uuid();

alter table event_lifecycle_changes
  add constraint event_lifecycle_changes_workspace_request_key_unique
  unique (workspace_id, request_key);
