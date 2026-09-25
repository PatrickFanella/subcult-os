-- EXPORT-01 (#54): manual, append-only event finance lines. This is separate from
-- tickets, settlement, payment providers, and any bank-verification claim.
create table event_finance_lines (
 id uuid primary key default gen_random_uuid(), workspace_id uuid not null references workspaces(id) on delete cascade,
 event_id uuid not null, entry_type text not null check(entry_type in ('budget','payable','actual_payment')),
 direction text not null check(direction in ('income','expense')),
 amount_cents bigint not null check(amount_cents between 0 and 1000000000), currency text not null check(currency ~ '^[a-z]{3}$'),
 label text not null check(length(label) between 1 and 240), reason text not null check(length(reason) between 1 and 2000),
 due_at timestamptz, occurred_at timestamptz, payable_line_id uuid, corrects_line_id uuid,
 request_key uuid not null, request_fingerprint text not null check(length(request_fingerprint)=64),
 created_by_person_id uuid not null references people(id), created_at timestamptz not null default clock_timestamp(),
 foreign key(event_id,workspace_id) references events(id,workspace_id) on delete cascade,
 unique(id,event_id,workspace_id),
 foreign key(payable_line_id,event_id,workspace_id) references event_finance_lines(id,event_id,workspace_id),
 foreign key(corrects_line_id,event_id,workspace_id) references event_finance_lines(id,event_id,workspace_id),
 check ((corrects_line_id is not null) or amount_cents > 0),
 check (entry_type <> 'payable' or direction = 'expense'),
 check (due_at is null or entry_type = 'payable'),
 check ((entry_type = 'actual_payment') = (occurred_at is not null)),
 check (payable_line_id is null or (entry_type = 'actual_payment' and direction = 'expense')),
 unique(event_id,created_by_person_id,request_key)
);
create index event_finance_lines_event_created_idx on event_finance_lines(event_id,created_at,id);
create index event_finance_lines_corrects_idx on event_finance_lines(corrects_line_id);
create unique index event_finance_lines_one_successor_idx on event_finance_lines(corrects_line_id) where corrects_line_id is not null;
