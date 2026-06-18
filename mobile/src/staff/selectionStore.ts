import * as SecureStore from 'expo-secure-store';
import { Platform } from 'react-native';

const selectedWorkspaceKey = 'subcult_os_selected_workspace_id';
const selectedEventKey = 'subcult_os_selected_event_id';

async function loadValue(key: string) {
  if (Platform.OS === 'web') {
    return globalThis.localStorage?.getItem(key) ?? null;
  }

  return SecureStore.getItemAsync(key);
}

async function storeValue(key: string, value: string) {
  if (Platform.OS === 'web') {
    globalThis.localStorage?.setItem(key, value);
    return;
  }

  await SecureStore.setItemAsync(key, value);
}

async function deleteValue(key: string) {
  if (Platform.OS === 'web') {
    globalThis.localStorage?.removeItem(key);
    return;
  }

  await SecureStore.deleteItemAsync(key);
}

export async function loadStaffSelection() {
  const [workspaceID, eventID] = await Promise.all([
    loadValue(selectedWorkspaceKey),
    loadValue(selectedEventKey),
  ]);

  return { workspaceID, eventID };
}

export async function storeSelectedWorkspaceID(workspaceID: string) {
  await storeValue(selectedWorkspaceKey, workspaceID);
}

export async function storeSelectedEventID(eventID: string) {
  await storeValue(selectedEventKey, eventID);
}

export async function clearSelectedEventID() {
  await deleteValue(selectedEventKey);
}
