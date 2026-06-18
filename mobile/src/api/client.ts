import { apiUrl } from '@/config/api';
import { loadStoredSessionCookie, removeStoredSessionCookie, storeSessionCookie } from '@/auth/sessionCookieStore';

let sessionCookie: string | null = null;
let loadedStoredCookie = false;

async function ensureStoredCookieLoaded() {
  if (loadedStoredCookie) {
    return;
  }

  sessionCookie = await loadStoredSessionCookie();
  loadedStoredCookie = true;
}

export async function setSessionCookieFromHeader(value: string | null) {
  if (!value) {
    return;
  }

  const [cookie] = value.split(';');
  if (cookie.includes('subcult_session=')) {
    sessionCookie = cookie;
    loadedStoredCookie = true;
    await storeSessionCookie(cookie);
  }
}

export async function clearSessionCookie() {
  sessionCookie = null;
  loadedStoredCookie = true;
  await removeStoredSessionCookie();
}

export class ApiError extends Error {
  status: number;
  data: unknown;

  constructor(status: number, message: string, data: unknown) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.data = data;
  }
}

export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  await ensureStoredCookieLoaded();

  const headers = new Headers(options.headers);
  const hasFormDataBody = typeof FormData !== 'undefined' && options.body instanceof FormData;
  if (!hasFormDataBody && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }
  if (sessionCookie && !headers.has('Cookie')) {
    headers.set('Cookie', sessionCookie);
  }
  if (sessionCookie && !headers.has('X-Subcult-Session')) {
    headers.set('X-Subcult-Session', sessionCookie);
  }
  const bearerToken = sessionCookie?.startsWith('subcult_session=') ? sessionCookie.slice('subcult_session='.length) : null;
  if (bearerToken && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${bearerToken}`);
  }

  const response = await fetch(apiUrl(path), {
    ...options,
    credentials: 'include',
    headers,
  });

  await setSessionCookieFromHeader(response.headers.get('set-cookie'));
  await setSessionCookieFromHeader(response.headers.get('x-subcult-session'));

  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    const message = typeof data?.error === 'string' ? data.error : `Request failed: ${response.status}`;
    throw new ApiError(response.status, message, data);
  }

  return data as T;
}

export function postJSON<T>(path: string, body: unknown): Promise<T> {
  return api<T>(path, { method: 'POST', body: JSON.stringify(body) });
}

export function patchJSON<T>(path: string, body: unknown): Promise<T> {
  return api<T>(path, { method: 'PATCH', body: JSON.stringify(body) });
}

export function postForm<T>(path: string, body: FormData): Promise<T> {
  return api<T>(path, { method: 'POST', body });
}
