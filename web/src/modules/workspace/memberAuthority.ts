import type { WorkspaceRole } from '../../domain';

const labels: Record<WorkspaceRole, string> = {
  owner: 'Owner', organizer: 'Organizer', finance: 'Finance', door: 'Door', crew: 'Crew', member: 'Crew',
};
const hints: Record<WorkspaceRole, string> = {
  owner: 'Manage members, events, door operations and finances.',
  organizer: 'Publish events and manage day-to-day coordination.',
  finance: 'Manage settlement and payment records.',
  door: 'Look up tickets and check in guests.',
  crew: 'Help with event work; no door or finance permissions.',
  member: 'Help with event work; no door or finance permissions.',
};
export function roleLabel(role: string) {
  return Object.hasOwn(labels, role) ? labels[role as WorkspaceRole] : 'Unknown role';
}
export function roleHint(role: string) {
  return Object.hasOwn(hints, role) ? hints[role as WorkspaceRole] : 'Reload the workspace to check this role.';
}
