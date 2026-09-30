import { api, patchJSON } from '../../api';
import type { CurrentWorkspaceDTO, WorkspaceRole } from '../../domain';

export const assignableMemberRoles = ['owner', 'organizer', 'finance', 'door', 'crew'] as const;
export type AssignableMemberRole = typeof assignableMemberRoles[number];
export interface MemberRoleReceipt {
  id: string;
  role: AssignableMemberRole;
  expiresAt?: string;
  revokedAt?: string;
}

export function isAssignableMemberRole(role: string): role is AssignableMemberRole {
  return assignableMemberRoles.some(candidate => candidate === role);
}

export function editableMemberRole(role: WorkspaceRole): AssignableMemberRole | null {
  return role === 'member' ? 'crew' : isAssignableMemberRole(role) ? role : null;
}

export async function loadMemberRoleWorkspace(workspaceId: string) {
  const workspace = await api<CurrentWorkspaceDTO>(`/api/workspaces/${encodeURIComponent(workspaceId)}`);
  if (workspace.id !== workspaceId || !Array.isArray(workspace.members)) {
    throw new Error('Workspace response did not match this role-management page.');
  }
  return workspace;
}

export async function changeWorkspaceMemberRole(workspaceId: string, memberId: string, role: string) {
  if (!isAssignableMemberRole(role)) throw new Error('Select an assignable workspace role.');
  // Role-only: do not clear expiry or imply that revoked access is restored.
  const receipt = await patchJSON<MemberRoleReceipt>(`/api/workspaces/${encodeURIComponent(workspaceId)}/members/${encodeURIComponent(memberId)}`, { role });
  if (receipt.id !== memberId || receipt.role !== role) {
    throw new Error('Role response did not match this membership and requested role.');
  }
  return receipt;
}
