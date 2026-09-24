-- AT-01: allowlisted, restart-safe AT Protocol record projection.
--
-- Stores only the three admitted collections (tv.subcult.profile,
-- tv.subcult.place, tv.subcult.event.occurrence) read from an external
-- AT Protocol stream. This is a read-only mirror of remote records, not the
-- private cultural_* operator tables; nothing here is written by the
-- cultural CRUD/public-preview code path and nothing in this table is
-- authoritative for workspace ownership. See docs/development/projection.md.

create table at_projection_records (
  uri text primary key,
  did text not null,
  collection text not null,
  rkey text not null,
  cid text not null default '',
  rev text not null default '',
  record jsonb,
  size_bytes integer not null default 0,
  status text not null default 'active' check (status in ('active', 'deleted', 'unavailable')),
  first_seen_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  source_cursor text not null
);
create index at_projection_records_did_idx on at_projection_records (did);
create index at_projection_records_collection_idx on at_projection_records (collection);

-- Single-row-per-source cursor, committed atomically with every record
-- write (and with every skip/quarantine decision) so a crash between the
-- two never leaves the cursor ahead of what was actually persisted.
create table at_projection_cursor (
  source_name text primary key,
  cursor text not null,
  updated_at timestamptz not null default now()
);

-- Malformed or oversize records, and any other event the processor could
-- not admit. The cursor still advances past a quarantined event; the
-- payload is bounded so an adversarial record cannot grow this table
-- unbounded.
create table at_projection_quarantine (
  id uuid primary key default gen_random_uuid(),
  uri text,
  reason text not null,
  payload text not null default '',
  cursor text not null,
  created_at timestamptz not null default now()
);
create index at_projection_quarantine_created_idx on at_projection_quarantine (created_at desc);
