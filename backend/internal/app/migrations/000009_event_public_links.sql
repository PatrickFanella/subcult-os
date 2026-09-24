-- LINK-01: private explicit relationship between a local operator event and
-- a public tv.subcult.event.occurrence AT record it corresponds to.
--
-- This relationship is operator-private: it is never projected into public
-- records or the anonymous public endpoints, and public/anonymous read
-- paths never join this table. It belongs to a local `events` row (not an
-- `event_occurrences` row) inside a workspace, and carries the exact
-- public_uri/observed_cid seen at attach or refresh time, the authority
-- that asserted the relationship (the repo DID of the public record; see
-- docs/development/public-links.md), and freshness fields (observed_at,
-- last_checked_at, status, last_error) that a refresh operation updates in
-- place. Event editing, occurrences, tickets and staffing never read this
-- table on their hot path; only this table's own status/freshness columns
-- change when the public record changes, becomes unavailable, or is
-- deleted, per data-boundaries.md's requirement that an external record
-- change never silently rewrites operational facts.
--
-- Composite (event_id, workspace_id) foreign key back to `events`, matching
-- migration 000008's pattern, so a cross-workspace reference is rejected by
-- PostgreSQL itself. unique (event_id, public_uri) makes repeated attach of
-- the same URI to the same event idempotent at the database level too.

create table event_public_links (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references workspaces(id) on delete cascade,
  event_id uuid not null,
  public_uri text not null check (length(public_uri) between 1 and 2000),
  observed_cid text not null check (length(observed_cid) between 1 and 256),
  authority_did text not null check (length(authority_did) between 1 and 2000),
  status text not null default 'fresh' check (status in ('fresh', 'changed', 'invalid', 'unavailable', 'deleted')),
  last_error text,
  observed_at timestamptz not null default now(),
  last_checked_at timestamptz not null default now(),
  created_by_person_id uuid not null references people(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  foreign key (event_id, workspace_id) references events (id, workspace_id) on delete cascade,
  unique (id, workspace_id),
  unique (event_id, public_uri)
);
create index event_public_links_event_idx on event_public_links (event_id, created_at desc);
create index event_public_links_workspace_idx on event_public_links (workspace_id, created_at desc);
