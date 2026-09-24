-- AT-01 recovery: approved backfill authorities and a run ledger for
-- backfill/rebuild/reconcile operations against the AT record projection.
-- See docs/development/projection.md. Additive only; no existing table,
-- column or row changes.

-- An authority (DID) an operator has explicitly approved as a backfill
-- source. Backfill only ever lists records from a DID present here with no
-- revoked_at; this is the allowlist gate for the otherwise-unbounded
-- "fetch someone else's repo" capability.
create table at_projection_authorities (
  did text primary key,
  approved_by_person_id uuid not null references people(id),
  approved_at timestamptz not null default now(),
  revoked_at timestamptz,
  note text not null default ''
);

-- One row per backfill/rebuild/reconcile invocation, for aggregate,
-- secret-free operator status (last run per kind, outcome, counts).
-- authority is null for a rebuild/reconcile pass across every approved
-- authority; counts is a small JSON object of integers only (record/URI
-- counts), never record bodies.
create table at_projection_runs (
  id uuid primary key default gen_random_uuid(),
  kind text not null check (kind in ('backfill', 'rebuild', 'reconcile')),
  authority text,
  started_at timestamptz not null default now(),
  finished_at timestamptz,
  outcome text not null default 'running' check (outcome in ('running', 'completed', 'failed', 'gap', 'bounded')),
  counts jsonb not null default '{}'::jsonb,
  error text not null default ''
);
create index at_projection_runs_kind_idx on at_projection_runs (kind, started_at desc);
