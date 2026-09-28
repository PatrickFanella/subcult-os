import { api, postJSON } from '@/api/client';
import { clearSession } from '@/auth/sessionAdapter';
import type { CurrentUserDTO, SignupResultDTO } from '@/api/types';

export function getMe() {
  return api<CurrentUserDTO>('/api/me');
}

export interface MobileAuthDebugDTO {
  hasAccessToken: boolean;
  hasRefreshToken: boolean;
  recognized: boolean;
  personId: string;
}

export function getMobileAuthDebug() {
  return api<MobileAuthDebugDTO>('/api/debug/mobile-auth');
}

export async function login(body: { email: string; password: string }) {
  return postJSON<CurrentUserDTO>('/api/mobile/auth/login', body);
}

export async function signup(body: { email: string; password: string; displayName?: string }) {
  return postJSON<SignupResultDTO>('/api/mobile/auth/signup', body);
}

export function verifyEmail(token: string) {
  return postJSON<CurrentUserDTO>('/api/mobile/auth/verify-email', { token });
}

export function requestRecovery(email: string) {
  return postJSON<{ ok: boolean }>('/api/mobile/auth/recovery/request', { email });
}

// A completed recovery revokes every session for the account, including this
// device's. A rejected token revokes nothing, so the session is kept.
export async function completeRecovery(token: string, newPassword: string) {
  await postJSON<{ ok: boolean }>('/api/mobile/auth/recovery/complete', { token, newPassword });
  await clearSession();
}

export async function logout() {
  try {
    await postJSON<{ ok: boolean }>('/api/mobile/auth/logout', {});
  } finally {
    await clearSession();
  }
}
