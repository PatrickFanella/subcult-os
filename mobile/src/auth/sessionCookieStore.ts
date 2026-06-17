import * as SecureStore from 'expo-secure-store';
import { Platform } from 'react-native';

const sessionCookieKey = 'subcult_os_session_cookie';

export async function loadStoredSessionCookie() {
  if (Platform.OS === 'web') {
    return globalThis.localStorage?.getItem(sessionCookieKey) ?? null;
  }

  return SecureStore.getItemAsync(sessionCookieKey);
}

export async function storeSessionCookie(value: string) {
  if (Platform.OS === 'web') {
    globalThis.localStorage?.setItem(sessionCookieKey, value);
    return;
  }

  await SecureStore.setItemAsync(sessionCookieKey, value);
}

export async function removeStoredSessionCookie() {
  if (Platform.OS === 'web') {
    globalThis.localStorage?.removeItem(sessionCookieKey);
    return;
  }

  await SecureStore.deleteItemAsync(sessionCookieKey);
}
