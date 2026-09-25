-- TICKET-EXT: a private, operator-configured external purchase-link handoff.
-- This table is deliberately attached to the canonical local occurrence, and
-- no anonymous endpoint reads it in this slice. A later public renderer must
-- make its own publication decision; it may not use this table to discover or
-- expose private occurrences.
create table occurrence_external_ticket_handoffs (
  occurrence_id uuid primary key,
  workspace_id uuid not null references workspaces(id) on delete cascade,
  event_id uuid not null,
  purchase_url text not null check (length(purchase_url) between 1 and 2000),
  provider_label text not null check (length(provider_label) between 1 and 120),
  created_by_person_id uuid not null references people(id),
  updated_by_person_id uuid not null references people(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  foreign key (occurrence_id, workspace_id) references event_occurrences(id, workspace_id) on delete cascade,
  foreign key (event_id, workspace_id) references events(id, workspace_id) on delete cascade
);
create index occurrence_external_ticket_handoffs_event_idx on occurrence_external_ticket_handoffs (event_id, occurrence_id);
