import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';

import { postJSON } from '../api';
import type { CurrentUserDTO } from '../domain';

type IdentityAction = 'verify' | 'request-recovery' | 'complete-recovery';

function tokenFromLocation() {
  return new URLSearchParams(window.location.search).get('token') ?? '';
}

export function IdentityActionView({ action }: { action: IdentityAction }) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(action === 'verify');
  const [notice, setNotice] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (action !== 'verify') return;
    const token = tokenFromLocation();
    if (!token) {
      setError('This verification link is missing its token.');
      setLoading(false);
      return;
    }
    void postJSON<CurrentUserDTO>('/api/auth/verify-email', { token })
      .then(() => {
        window.location.href = '/';
      })
      .catch((caught) => setError(caught instanceof Error ? caught.message : 'Unable to verify email'))
      .finally(() => setLoading(false));
  }, [action]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setNotice(null);
    setLoading(true);
    try {
      if (action === 'request-recovery') {
        await postJSON<{ ok: boolean }>('/api/auth/recovery/request', { email: email.trim() });
        setNotice('If that verified account exists, a recovery link has been sent.');
      } else {
        const token = tokenFromLocation();
        if (!token) throw new Error('This recovery link is missing its token.');
        if (password.length < 8) throw new Error('Password must be at least 8 characters.');
        await postJSON<{ ok: boolean }>('/api/auth/recovery/complete', { token, newPassword: password });
        setNotice('Password updated. You can now sign in.');
      }
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to continue');
    } finally {
      setLoading(false);
    }
  }

  const title = action === 'verify' ? 'Verify your email' : action === 'request-recovery' ? 'Recover your account' : 'Choose a new password';
  return (
    <main className="min-h-screen px-4 py-6 text-zinc-100">
      <section className="mx-auto flex min-h-[calc(100vh-3rem)] w-full max-w-md flex-col justify-center">
        <div className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6 shadow-2xl shadow-black/40">
          <p className="text-xs uppercase tracking-[0.3em] text-fuchsia-300">Account security</p>
          <h1 className="mt-3 text-3xl font-semibold text-white">{title}</h1>
          {action === 'verify' ? <p className="mt-3 text-sm text-zinc-400">{loading ? 'Verifying…' : error ?? 'Verified.'}</p> : (
            <form className="mt-6 space-y-4" onSubmit={submit}>
              {action === 'request-recovery' ? (
                <input className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3" type="email" autoComplete="email" required value={email} onChange={(event) => setEmail(event.target.value)} placeholder="Email address" />
              ) : (
                <input className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3" type="password" autoComplete="new-password" minLength={8} required value={password} onChange={(event) => setPassword(event.target.value)} placeholder="New password" />
              )}
              <button className="w-full rounded-2xl bg-amber-300 px-4 py-3 font-medium text-zinc-950 disabled:opacity-60" disabled={loading} type="submit">{loading ? 'Working…' : 'Continue'}</button>
            </form>
          )}
          {error && action !== 'verify' ? <p className="mt-4 rounded-2xl bg-rose-500/10 px-4 py-3 text-sm text-rose-200">{error}</p> : null}
          {notice ? <p className="mt-4 rounded-2xl bg-emerald-500/10 px-4 py-3 text-sm text-emerald-100">{notice}</p> : null}
          <a className="mt-5 inline-block text-sm text-amber-300" href="/login">Return to sign in</a>
        </div>
      </section>
    </main>
  );
}
