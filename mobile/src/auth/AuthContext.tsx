import type { PropsWithChildren } from 'react';
import { createContext, useContext, useEffect, useMemo, useState } from 'react';

import * as authAPI from '@/api/auth';
import type { CurrentUserDTO, SignupResultDTO } from '@/api/types';

type AuthContextValue = {
  user: CurrentUserDTO | null;
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string, displayName?: string) => Promise<SignupResultDTO>;
  verifyEmail: (token: string) => Promise<void>;
  signOut: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: PropsWithChildren) {
  const [user, setUser] = useState<CurrentUserDTO | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function refresh() {
    setLoading(true);
    setError(null);
    try {
      const current = await authAPI.getMe();
      setUser(current);
    } catch (caught) {
      setUser(null);
      if (caught instanceof Error && !caught.message.toLowerCase().includes('unauthorized')) {
        setError(caught.message);
      }
    } finally {
      setLoading(false);
    }
  }

  async function signIn(email: string, password: string) {
    setLoading(true);
    setError(null);
    try {
      setUser(await authAPI.login({ email, password }));
    } catch (caught) {
      const message = caught instanceof Error ? caught.message : 'Unable to sign in';
      setError(message);
      throw new Error(message);
    } finally {
      setLoading(false);
    }
  }

  async function signUp(email: string, password: string, displayName?: string) {
    setLoading(true);
    setError(null);
    try {
      return await authAPI.signup({ email, password, displayName });
    } catch (caught) {
      const message = caught instanceof Error ? caught.message : 'Unable to create account';
      setError(message);
      throw new Error(message);
    } finally {
      setLoading(false);
    }
  }

  async function verifyEmail(token: string) {
    setLoading(true);
    setError(null);
    try {
      setUser(await authAPI.verifyEmail(token));
    } catch (caught) {
      const message = caught instanceof Error ? caught.message : 'Unable to verify email';
      setError(message);
      throw new Error(message);
    } finally {
      setLoading(false);
    }
  }

  async function signOut() {
    setLoading(true);
    setError(null);
    try {
      await authAPI.logout();
      setUser(null);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void refresh();
  }, []);

  const value = useMemo(() => ({ user, loading, error, refresh, signIn, signUp, verifyEmail, signOut }), [user, loading, error]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) {
    throw new Error('useAuth must be used inside AuthProvider');
  }
  return value;
}
