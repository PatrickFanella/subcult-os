import { useRef, useState } from 'react';
import { Brand } from '../ui/Brand';
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
  const [loading, setLoading] = useState(false);
  const submitting = useRef(false);
  const [notice, setNotice] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    // One-use challenges must not run in a mount effect (Strict Mode replays it).
    // The ref also guards repeated submissions before React commits disabled UI.
    if (submitting.current) return;
    submitting.current = true;
    setError(null);
    setNotice(null);
    setLoading(true);
    try {
      if (action === 'verify') {
        const token = tokenFromLocation();
        if (!token) throw new Error('This verification link is missing its token.');
        await postJSON<CurrentUserDTO>('/api/auth/verify-email', { token });
        window.location.replace('/');
      } else if (action === 'request-recovery') {
        await postJSON<{ ok: boolean }>('/api/auth/recovery/request', { email: email.trim() });
        setNotice('If that address belongs to a verified account, check its email for a recovery link. Email delivery is not confirmed.');
      } else {
        const token = tokenFromLocation();
        if (!token) throw new Error('This recovery link is missing its token.');
        if (password.length < 8) throw new Error('Password must be at least 8 characters.');
        await postJSON<{ ok: boolean }>('/api/auth/recovery/complete', { token, newPassword: password });
        window.history.replaceState(null, '', window.location.pathname);
        setPassword('');
        setNotice('Password updated. You can now sign in.');
      }
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to continue');
    } finally {
      submitting.current = false;
      setLoading(false);
    }
  }

  const title = action === 'verify' ? 'Verify your email' : action === 'request-recovery' ? 'Recover your account' : 'Choose a new password';
  return (
    <main className="min-h-screen px-4 py-6 text-fg-primary sm:px-6 lg:px-8">
      <section className="mx-auto flex min-h-[calc(100vh-3rem)] w-full max-w-md flex-col justify-center">
        <Brand className="mb-4" />
        <div className="rounded-panel border border-stroke-subtle bg-surface-panel p-6">
          <p className="text-xs uppercase tracking-[0.2em] text-fg-muted">Account security</p>
          <h1 className="mt-3 text-3xl font-bold text-fg-primary">{title}</h1>
          {action === 'verify' ? <p className="mt-3 text-sm text-fg-secondary">Confirm below to verify your email and sign in.</p> : null}
          {!notice ? (
            <form className="mt-6 space-y-4" onSubmit={submit}>
              {action === 'request-recovery' ? (
                <input className="field py-3" type="email" aria-label="Email address" autoComplete="email" required value={email} onChange={(event) => setEmail(event.target.value)} placeholder="Email address" />
              ) : action === 'complete-recovery' ? (
                <input className="field py-3" type="password" aria-label="New password" autoComplete="new-password" minLength={8} required value={password} onChange={(event) => setPassword(event.target.value)} placeholder="New password" />
              ) : null}
              <button className="btn-primary w-full px-4" disabled={loading} type="submit">{loading ? 'Working…' : action === 'verify' ? 'Verify email and sign in' : 'Continue'}</button>
            </form>
          ) : null}
          {error ? <p role="alert" className="mt-4 rounded-2xl bg-status-surface-danger px-4 py-3 text-sm text-status-danger">{error}</p> : null}
          {notice ? <p role="status" className="mt-4 rounded-2xl bg-status-surface-success px-4 py-3 text-sm text-status-success">{notice}</p> : null}
          <a className="mt-5 inline-block text-sm text-fg-primary underline underline-offset-4" href="/login">Return to sign in</a>
        </div>
      </section>
    </main>
  );
}
