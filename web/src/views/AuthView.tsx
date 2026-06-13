import { useMemo, useState } from 'react';
import type { FormEvent } from 'react';
import { postJSON } from '../api';
import type { CurrentUserDTO } from '../domain';

type Mode = 'login' | 'signup';

function getMode(): Mode {
  if (typeof window !== 'undefined' && window.location.pathname === '/signup') {
    return 'signup';
  }

  return 'login';
}

function getNextPath() {
  if (typeof window === 'undefined') {
    return '/';
  }

  const next = new URLSearchParams(window.location.search).get('next');
  if (!next || next.trim().length === 0 || !next.startsWith('/') || next.startsWith('//')) {
    return '/';
  }

  return next;
}

function authHref(mode: Mode) {
  const next = getNextPath();
  return next === '/' ? `/${mode}` : `/${mode}?next=${encodeURIComponent(next)}`;
}

function goToNext(nextPath: string) {
  window.location.href = nextPath;
}

export function AuthView() {
  const [mode, setMode] = useState<Mode>(getMode);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const nextPath = useMemo(getNextPath, []);
  const title = useMemo(() => (mode === 'signup' ? 'Create account' : 'Sign in'), [mode]);
  const eyebrow = mode === 'signup' ? 'Join the room' : 'Operator access';
  const description =
    mode === 'signup'
      ? 'Create your account, keep the invited email if you arrived from a handoff, and step into the workspace.'
      : 'Sign in to resume the workspace. If you were sent here from an invite, use the same email that received it.';
  const invitePrompt = nextPath.startsWith('/invite/')
    ? 'Accepting an invitation? Sign in/sign up with the invited email.'
    : null;

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);

    const trimmedEmail = email.trim();
    if (!trimmedEmail) {
      setError('Enter your email address.');
      return;
    }

    if (password.length < 8) {
      setError('Password must be at least 8 characters.');
      return;
    }

    setLoading(true);

    try {
      const body = {
        email: trimmedEmail,
        password,
        ...(mode === 'signup' ? { displayName: displayName.trim() || undefined } : {}),
      };

      await postJSON<CurrentUserDTO>(mode === 'signup' ? '/api/auth/signup' : '/api/auth/login', body);
      goToNext(nextPath);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to continue');
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="min-h-screen px-4 py-6 text-zinc-100 sm:px-6 lg:px-8">
      <section className="mx-auto flex min-h-[calc(100vh-3rem)] w-full max-w-md flex-col justify-center">
        <div className="mb-4 flex items-center justify-between text-xs uppercase tracking-[0.3em] text-zinc-400">
          <span>subcult-os</span>
          <a className="text-amber-300 transition hover:text-amber-200" href="/">
            Workspace
          </a>
        </div>

        <div className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6 shadow-2xl shadow-black/40 backdrop-blur">
          <p className="text-xs uppercase tracking-[0.3em] text-fuchsia-300">{eyebrow}</p>
          <h1 className="mt-3 text-3xl font-semibold tracking-tight text-white">{title}</h1>
          <p className="mt-2 text-sm leading-6 text-zinc-400">{description}</p>

          {invitePrompt ? (
            <div className="mt-5 rounded-2xl border border-amber-400/20 bg-amber-400/[0.08] px-4 py-3 text-sm leading-6 text-amber-100">
              {invitePrompt}
            </div>
          ) : null}

          <div className="mt-6 flex gap-2 text-sm">
            <a
              className={`rounded-full px-3 py-2 transition ${mode === 'login' ? 'bg-white text-zinc-950' : 'bg-white/5 text-zinc-300 hover:bg-white/10'}`}
              href={authHref('login')}
              onClick={(event) => {
                event.preventDefault();
                setMode('login');
              }}
            >
              Sign in
            </a>
            <a
              className={`rounded-full px-3 py-2 transition ${mode === 'signup' ? 'bg-white text-zinc-950' : 'bg-white/5 text-zinc-300 hover:bg-white/10'}`}
              href={authHref('signup')}
              onClick={(event) => {
                event.preventDefault();
                setMode('signup');
              }}
            >
              Create account
            </a>
          </div>

          <form className="mt-6 space-y-4" noValidate onSubmit={handleSubmit}>
            <label className="block space-y-2 text-sm">
              <span className="text-zinc-300">Email</span>
              <input
                className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none transition placeholder:text-zinc-500 focus:border-amber-300/60 focus:bg-white/8"
                type="email"
                autoComplete="email"
                required
                value={email}
                onChange={(event) => setEmail(event.target.value)}
              />
            </label>

            {mode === 'signup' ? (
              <label className="block space-y-2 text-sm">
                <span className="text-zinc-300">Display name</span>
                <input
                  className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none transition placeholder:text-zinc-500 focus:border-amber-300/60 focus:bg-white/8"
                  type="text"
                  autoComplete="name"
                  value={displayName}
                  onChange={(event) => setDisplayName(event.target.value)}
                  placeholder="Optional"
                />
              </label>
            ) : null}

            <label className="block space-y-2 text-sm">
              <span className="text-zinc-300">Password</span>
              <input
                className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none transition placeholder:text-zinc-500 focus:border-amber-300/60 focus:bg-white/8"
                type="password"
                autoComplete={mode === 'signup' ? 'new-password' : 'current-password'}
                minLength={8}
                required
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                aria-describedby="password-rules"
              />
              <span id="password-rules" className="block text-xs text-zinc-500">
                8+ characters
              </span>
            </label>

            {error ? <p className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">{error}</p> : null}

            <button
              className="w-full rounded-2xl bg-amber-300 px-4 py-3 font-medium text-zinc-950 transition hover:bg-amber-200 disabled:cursor-not-allowed disabled:bg-amber-300/60"
              type="submit"
              disabled={loading}
            >
              {loading ? 'Working…' : title}
            </button>
          </form>
        </div>
      </section>
    </main>
  );
}
