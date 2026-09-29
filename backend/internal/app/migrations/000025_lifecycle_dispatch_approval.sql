-- Existing worklist entries remain draft-only. Only a later destination-specific
-- approval flow may create a dispatch-approved action with complete payload.
alter table event_lifecycle_actions
  add column dispatch_approved boolean not null default false;

create index event_lifecycle_actions_dispatch_due_idx
  on event_lifecycle_actions (action_kind, destination, next_attempt_at, created_at)
  where dispatch_approved and status in ('pending', 'retryable');
