-- IMPORT-01: workspace-scoped, review-only cultural import previews.
--
-- This ledger stores an operator's source assertion, parser provenance and
-- normalized allowlisted rows. It deliberately stores neither raw CSV bytes
-- nor any apply decision. Canonical matches are bounded review hints only;
-- no trigger or foreign key can turn one into a canonical update.
create table cultural_import_previews (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references workspaces(id) on delete cascade,
  schema_name text not null check (schema_name = 'subcult-occurrence-csv/v1'),
  source_id text not null check (length(trim(source_id)) between 1 and 200),
  source_name text not null check (length(trim(source_name)) between 1 and 400),
  source_assertion text not null check (length(trim(source_assertion)) between 1 and 1000),
  content_sha256 char(64) not null check (content_sha256 ~ '^[0-9a-f]{64}$'),
  created_by_person_id uuid not null references people(id),
  created_at timestamptz not null default now(),
  unique (id, workspace_id)
);
create index cultural_import_previews_workspace_idx
  on cultural_import_previews (workspace_id, created_at desc);

create table cultural_import_candidates (
  id uuid primary key default gen_random_uuid(),
  import_id uuid not null,
  workspace_id uuid not null references workspaces(id) on delete cascade,
  row_number integer not null check (row_number >= 2),
  source_record_id text not null check (length(trim(source_record_id)) between 1 and 200),
  title text not null check (length(trim(title)) between 1 and 1000),
  description text not null default '' check (length(description) <= 12000),
  starts_at timestamptz not null,
  ends_at timestamptz,
  timezone text not null check (length(trim(timezone)) between 1 and 64),
  status text not null check (status in ('scheduled', 'rescheduled', 'postponed', 'cancelled')),
  venue_name text not null default '' check (length(venue_name) <= 600),
  locality text not null default '' check (length(locality) <= 400),
  region text not null default '' check (length(region) <= 400),
  country text not null default '' check (country = '' or country ~ '^[A-Z]{2}$'),
  match_count integer not null default 0 check (match_count >= 0),
  matches_truncated boolean not null default false,
  created_at timestamptz not null default now(),
  unique (id, workspace_id),
  unique (import_id, row_number),
  foreign key (import_id, workspace_id) references cultural_import_previews(id, workspace_id) on delete cascade,
  check (ends_at is null or ends_at > starts_at)
);
create index cultural_import_candidates_import_idx
  on cultural_import_candidates (import_id, row_number);

create table cultural_import_candidate_matches (
  id uuid primary key default gen_random_uuid(),
  candidate_id uuid not null,
  workspace_id uuid not null references workspaces(id) on delete cascade,
  occurrence_id uuid,
  occurrence_id_snapshot uuid not null,
  event_id_snapshot uuid not null,
  name_snapshot text not null check (length(trim(name_snapshot)) between 1 and 1000),
  starts_at_snapshot timestamptz not null,
  status_snapshot text not null check (status_snapshot in ('scheduled', 'rescheduled', 'postponed', 'cancelled')),
  updated_at_snapshot timestamptz not null,
  created_at timestamptz not null default now(),
  foreign key (candidate_id, workspace_id) references cultural_import_candidates(id, workspace_id) on delete cascade,
  foreign key (occurrence_id, workspace_id) references event_occurrences(id, workspace_id) on delete set null (occurrence_id),
  unique (candidate_id, occurrence_id_snapshot)
);
create index cultural_import_candidate_matches_occurrence_idx
  on cultural_import_candidate_matches (occurrence_id);

-- Error text is intentionally represented only by stable, parser-owned codes
-- and locations. Neither source assertion nor CSV cell text is repeated here.
create table cultural_import_preview_errors (
  id uuid primary key default gen_random_uuid(),
  import_id uuid not null references cultural_import_previews(id) on delete cascade,
  row_number integer not null check (row_number >= 0),
  field_name text not null default '' check (length(field_name) <= 64),
  code text not null check (length(code) between 1 and 100),
  created_at timestamptz not null default now(),
  unique (import_id, row_number, field_name, code)
);
create index cultural_import_preview_errors_import_idx
  on cultural_import_preview_errors (import_id, row_number, field_name, code);
