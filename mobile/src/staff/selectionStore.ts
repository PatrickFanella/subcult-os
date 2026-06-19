import { loadPersistedValue, removePersistedValue, savePersistedValue } from '@/modules/storage/persistedStore';

const selectedWorkspaceKey = 'subcult_os_selected_workspace_id';
const selectedEventKey = 'subcult_os_selected_event_id';

export async function loadStaffSelection() {
  const [workspaceID, eventID] = await Promise.all([
    loadPersistedValue(selectedWorkspaceKey),
    loadPersistedValue(selectedEventKey),
  ]);

  return { workspaceID, eventID };
}

export async function storeSelectedWorkspaceID(workspaceID: string) {
  await savePersistedValue(selectedWorkspaceKey, workspaceID);
}

export async function storeSelectedEventID(eventID: string) {
  await savePersistedValue(selectedEventKey, eventID);
}

export async function clearSelectedEventID() {
  await removePersistedValue(selectedEventKey);
}
