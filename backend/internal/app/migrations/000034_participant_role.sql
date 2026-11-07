-- Adds the 'participant' workspace role below 'crew' and lets an invitation
-- carry the role it grants. A participant has no baseline operate access:
-- they see their own participant portal, open shifts they are eligible for
-- and their calendar download, but no operator data (events, staffing board,
-- notes, contacts, applications, members, tickets or finance). See
-- docs/development/authority-model.md.
alter table workspace_members drop constraint workspace_members_role_check;
alter table workspace_members add constraint workspace_members_role_check
  check (role in ('owner', 'organizer', 'finance', 'door', 'crew', 'member', 'participant'));

-- Invitations were fixed to the legacy 'member' (crew) role. Owners can now
-- invite at the participant or crew tier; existing rows keep 'member'.
alter table workspace_invitations drop constraint workspace_invitations_role_check;
alter table workspace_invitations add constraint workspace_invitations_role_check
  check (role in ('member', 'crew', 'participant'));
