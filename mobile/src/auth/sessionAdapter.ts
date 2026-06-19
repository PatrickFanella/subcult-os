import { loadPersistedValue, removePersistedValue, savePersistedValue } from '@/modules/storage/persistedStore';

const sessionCookieKey = 'subcult_os_session_cookie';
const sessionCookieName = 'subcult_session';

export type SessionHeaders = Record<string, string>;

let sessionCookie: string | null = null;
let loadedStoredCookie = false;

function sessionCookieFromHeader(value: string | null) {
  if (!value) {
    return null;
  }

  const [cookie] = value.split(';');
  if (!cookie.includes(`${sessionCookieName}=`)) {
    return null;
  }

  const trimmed = cookie.trim();
  return trimmed || null;
}

function bearerTokenFromSessionCookie(value: string | null) {
  if (!value?.startsWith(`${sessionCookieName}=`)) {
    return null;
  }

  return value.slice(`${sessionCookieName}=`.length) || null;
}

async function ensureSessionLoaded() {
  if (loadedStoredCookie) {
    return;
  }

  sessionCookie = await loadPersistedValue(sessionCookieKey);
  loadedStoredCookie = true;
}

async function persistSessionCookie(value: string | null) {
  sessionCookie = value;
  loadedStoredCookie = true;
  if (value) {
    await savePersistedValue(sessionCookieKey, value);
    return;
  }

  await removePersistedValue(sessionCookieKey);
}

function sessionHeadersFromCookie(cookie: string | null): SessionHeaders {
  const token = bearerTokenFromSessionCookie(cookie);
  if (!cookie || !token) {
    return {};
  }

  return {
    Cookie: cookie,
    'X-Subcult-Session': cookie,
    'X-Subcult-Session-Token': token,
    Authorization: `Bearer ${token}`,
  };
}

export async function loadStoredSessionCookie() {
  await ensureSessionLoaded();
  return sessionCookie;
}

export async function storeSessionCookie(value: string) {
  await persistSessionCookie(value);
}

export async function removeStoredSessionCookie() {
  await persistSessionCookie(null);
}

export async function loadSessionHeaders() {
  await ensureSessionLoaded();
  return sessionHeadersFromCookie(sessionCookie);
}

export async function absorbSessionHeaders(headers: Headers) {
  const cookie = sessionCookieFromHeader(headers.get('set-cookie')) ?? sessionCookieFromHeader(headers.get('x-subcult-session'));
  if (cookie) {
    await persistSessionCookie(cookie);
  }
}

export async function clearSession() {
  await persistSessionCookie(null);
}

export async function getSessionDebugState() {
  await ensureSessionLoaded();
  const token = bearerTokenFromSessionCookie(sessionCookie) ?? '';
  return {
    loadedStoredCookie,
    hasSessionCookie: Boolean(sessionCookie),
    hasBearerToken: Boolean(token),
    sessionCookiePrefix: sessionCookie ? sessionCookie.slice(0, `${sessionCookieName}=`.length) : '',
  };
}
