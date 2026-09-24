-- AUTH-01: separate account control, workspace permission, creator
-- delegation and real-world organization claim, per
-- docs/development/authority-model.md.
--
-- Widens workspace_members.role from the two-value ('owner', 'member')
-- check to five permission roles plus the legacy 'member' alias, and adds
-- time-boxed expiry and revocation to membership. 'member' is kept as a
-- valid role value (a legacy alias with the same permissions as 'crew')
-- so every existing row and every existing literal 'owner'/'member' call
-- site in backend/internal/app keeps working unmodified; new code should
-- assign 'crew' rather than 'member' going forward.
--
-- Adds creator_delegations: a cultural profile owner's scoped, time-boxed
-- grant of the right to act for them to a workspace. It carries the same
-- composite (id, workspace_id) foreign key pattern introduced in
-- migration 000008 so a delegation can never reference a cultural profile
-- from a different workspace.

alter table workspace_members drop constraint workspace_members_role_check;
alter table workspace_members add constraint workspace_members_role_check
  check (role in ('owner', 'organizer', 'finance', 'door', 'crew', 'member'));

alter table workspace_members add column expires_at timestamptz;
alter table workspace_members add column revoked_at timestamptz;
alter table workspace_members add column revoked_by_person_id uuid references people(id);

create index workspace_members_active_idx
  on workspace_members (workspace_id, person_id)
  where removed_at is null and revoked_at is null;

create table creator_delegations (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references workspaces(id) on delete cascade,
  cultural_profile_id uuid not null,
  scope text[] not null default '{}',
  granted_by_person_id uuid not null references people(id),
  granted_at timestamptz not null default now(),
  expires_at timestamptz,
  revoked_at timestamptz,
  revoked_by_person_id uuid references people(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  foreign key (cultural_profile_id, workspace_id)
    references cultural_profiles (id, workspace_id) on delete cascade
);
create index creator_delegations_profile_idx
  on creator_delegations (cultural_profile_id, workspace_id);
create index creator_delegations_workspace_idx
  on creator_delegations (workspace_id, created_at desc);
