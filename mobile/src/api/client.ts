import { apiUrl } from '@/config/api';
import { absorbSessionHeaders, loadSessionHeaders } from '@/auth/sessionAdapter';

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

export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers);
  const hasFormDataBody = typeof FormData !== 'undefined' && options.body instanceof FormData;
  if (!hasFormDataBody && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }
  const sessionHeaders = await loadSessionHeaders();
  for (const [key, value] of Object.entries(sessionHeaders)) {
    if (!headers.has(key)) {
      headers.set(key, value);
    }
  }

  const headerObject: Record<string, string> = {};
  headers.forEach((value, key) => {
    headerObject[key] = value;
  });

  const response = await fetch(apiUrl(path), {
    ...options,
    credentials: 'include',
    headers: headerObject,
  });

  await absorbSessionHeaders(response.headers);

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
