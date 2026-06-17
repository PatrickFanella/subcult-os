import { api, clearSessionCookie, postJSON } from '@/api/client';
import type { CurrentUserDTO } from '@/api/types';

export function getMe() {
  return api<CurrentUserDTO>('/api/me');
}

export function login(body: { email: string; password: string }) {
  return postJSON<CurrentUserDTO>('/api/auth/login', body);
}

export function signup(body: { email: string; password: string; displayName?: string }) {
  return postJSON<CurrentUserDTO>('/api/auth/signup', body);
}

export async function logout() {
  try {
    await postJSON<{ ok: boolean }>('/api/auth/logout', {});
  } finally {
    await clearSessionCookie();
  }
}
