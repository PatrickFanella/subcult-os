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

async function request(path: string, options: RequestInit) {
  const headers = new Headers(options.headers);
  if (!headers.has('Content-Type')) headers.set('Content-Type', 'application/json');
  return fetch(path, { ...options, credentials: 'include', headers });
}

async function refreshSession() {
  if (!refreshPromise) {
    refreshPromise = request('/api/auth/refresh', { method: 'POST', body: '{}' })
      .then((response) => response.ok)
      .finally(() => {
        refreshPromise = null;
      });
  }
  return refreshPromise;
}

export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  let response = await request(path, options);
  if (response.status === 401 && !path.startsWith('/api/auth/') && await refreshSession()) {
    response = await request(path, options);
  }

  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new ApiError(response.status, typeof data.error === 'string' ? data.error : `Request failed: ${response.status}`, data);
  }

  return data as T;
}

export function postJSON<T>(path: string, body: unknown): Promise<T> {
  return api<T>(path, { method: 'POST', body: JSON.stringify(body) });
}

export function patchJSON<T>(path: string, body: unknown): Promise<T> {
  return api<T>(path, { method: 'PATCH', body: JSON.stringify(body) });
}

export function deleteJSON<T = void>(path: string): Promise<T> {
  return api<T>(path, { method: 'DELETE' });
}
