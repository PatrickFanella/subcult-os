import { api, postJSON } from '@/api/client';
import { absorbSessionHeaders, clearSession } from '@/auth/sessionAdapter';
import type { CurrentUserDTO } from '@/api/types';

export function getMe() {
  return api<CurrentUserDTO>('/api/me');
}

export interface MobileAuthDebugDTO {
  hasCookie: boolean;
  hasAuthorization: boolean;
  hasSessionHeader: boolean;
  hasTokenHeader: boolean;
  hasToken: boolean;
  recognized: boolean;
  personId: string;
}

export function getMobileAuthDebug() {
  return api<MobileAuthDebugDTO>('/api/debug/mobile-auth');
}

export async function login(body: { email: string; password: string }) {
  const user = await postJSON<CurrentUserDTO>('/api/auth/login', body);
  await absorbSessionHeaders(new Headers(user.sessionCookie ? { 'x-subcult-session': user.sessionCookie } : {}));
  return user;
}

export async function signup(body: { email: string; password: string; displayName?: string }) {
  const user = await postJSON<CurrentUserDTO>('/api/auth/signup', body);
  await absorbSessionHeaders(new Headers(user.sessionCookie ? { 'x-subcult-session': user.sessionCookie } : {}));
  return user;
}

export async function logout() {
  try {
    await postJSON<{ ok: boolean }>('/api/auth/logout', {});
  } finally {
    await clearSession();
  }
}
