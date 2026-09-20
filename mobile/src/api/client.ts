import { absorbSessionHeaders, clearSession, loadRefreshHeaders, loadSessionHeaders } from '@/auth/sessionAdapter';
import { apiUrl } from '@/config/api';

export { getSessionDebugState } from '@/auth/sessionAdapter';

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

let refreshPromise: Promise<boolean> | null = null;

async function request(path: string, options: RequestInit, refresh = false) {
  const headers = new Headers(options.headers);
  const hasFormDataBody = typeof FormData !== 'undefined' && options.body instanceof FormData;
  if (!hasFormDataBody && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json');
  const sessionHeaders = refresh ? await loadRefreshHeaders() : await loadSessionHeaders();
  for (const [key, value] of Object.entries(sessionHeaders)) if (!headers.has(key)) headers.set(key, value);

  const response = await fetch(apiUrl(path), { ...options, headers });
  await absorbSessionHeaders(response.headers);
  return response;
}

async function refreshSession() {
  if (!refreshPromise) {
    refreshPromise = (async () => {
      const response = await request('/api/mobile/auth/refresh', { method: 'POST', body: '{}' }, true);
      if (!response.ok) await clearSession();
      return response.ok;
    })().finally(() => {
      refreshPromise = null;
    });
  }
  return refreshPromise;
}

export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  let response = await request(path, options);
  if (response.status === 401 && !path.startsWith('/api/mobile/auth/') && await refreshSession()) response = await request(path, options);

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
