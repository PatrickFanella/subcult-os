import { loadPersistedValue, removePersistedValue, savePersistedValue } from '@/modules/storage/persistedStore';

const accessTokenKey = 'subcult_os_access_token';
const refreshTokenKey = 'subcult_os_refresh_token';
const legacySessionCookieKey = 'subcult_os_session_cookie';
const accessTokenHeader = 'x-subcult-access-token';
const refreshTokenHeader = 'x-subcult-refresh-token';

export type SessionHeaders = Record<string, string>;

let accessToken: string | null = null;
let refreshToken: string | null = null;
let loadedStoredTokens = false;

async function ensureSessionLoaded() {
  if (loadedStoredTokens) return;
  [accessToken, refreshToken] = await Promise.all([
    loadPersistedValue(accessTokenKey),
    loadPersistedValue(refreshTokenKey),
  ]);
  await removePersistedValue(legacySessionCookieKey);
  loadedStoredTokens = true;
}

async function persistToken(key: string, value: string | null) {
  if (value) await savePersistedValue(key, value);
  else await removePersistedValue(key);
}

export async function loadSessionHeaders(): Promise<SessionHeaders> {
  await ensureSessionLoaded();
  if (!accessToken) return {};
  return { Authorization: `Bearer ${accessToken}`, 'X-Subcult-Access-Token': accessToken };
}

export async function loadRefreshHeaders(): Promise<SessionHeaders> {
  await ensureSessionLoaded();
  return refreshToken ? { 'X-Subcult-Refresh-Token': refreshToken } : {};
}

export async function absorbSessionHeaders(headers: Headers) {
  const nextAccess = headers.get(accessTokenHeader);
  const nextRefresh = headers.get(refreshTokenHeader);
  if (nextAccess) {
    accessToken = nextAccess;
    await persistToken(accessTokenKey, nextAccess);
  }
  if (nextRefresh) {
    refreshToken = nextRefresh;
    await persistToken(refreshTokenKey, nextRefresh);
  }
  loadedStoredTokens = true;
}

export async function clearSession() {
  accessToken = null;
  refreshToken = null;
  loadedStoredTokens = true;
  await Promise.all([persistToken(accessTokenKey, null), persistToken(refreshTokenKey, null)]);
}

export async function getSessionDebugState() {
  await ensureSessionLoaded();
  return { loadedStoredTokens, hasAccessToken: Boolean(accessToken), hasRefreshToken: Boolean(refreshToken) };
}

// Temporary compatibility read for the settings screen during the token-store cutover.
export async function loadStoredSessionCookie() {
  await ensureSessionLoaded();
  return accessToken;
}

export async function removeStoredSessionCookie() {
  await clearSession();
}
