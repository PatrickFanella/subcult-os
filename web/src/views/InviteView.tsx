import { useEffect, useMemo, useState } from 'react';

type InviteState =
  | { phase: 'loading' }
  | { phase: 'accepted' }
  | { phase: 'unauthorized' }
  | { phase: 'error'; message: string };

async function acceptInvite(token: string) {
  const response = await fetch(`/api/invitations/${encodeURIComponent(token)}/accept`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
  });

  const data = await response.json().catch(() => ({}));

  if (response.ok) {
    return { ok: true as const };
  }

  const message = typeof data.error === 'string' ? data.error : `Request failed: ${response.status}`;
  return { ok: false as const, status: response.status, message };
}

function tokenPreview(token: string) {
  const normalized = token.trim();

  if (normalized.length <= 8) {
    return normalized;
  }

  return `${normalized.slice(0, 4)}…${normalized.slice(-4)}`;
}

export function InviteView({ token }: { token: string }) {
  const [state, setState] = useState<InviteState>({ phase: 'loading' });

  const nextPath = useMemo(() => `/invite/${token}`, [token]);
  const encodedNextPath = encodeURIComponent(nextPath);
  const preview = useMemo(() => tokenPreview(token), [token]);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setState({ phase: 'loading' });

      try {
        const result = await acceptInvite(token);
        if (cancelled) return;

        if (result.ok) {
          setState({ phase: 'accepted' });
          return;
        }

        if (result.status === 401) {
          setState({ phase: 'unauthorized' });
          return;
        }

        setState({ phase: 'error', message: result.message });
      } catch (caught) {
        if (!cancelled) {
          setState({ phase: 'error', message: caught instanceof Error ? caught.message : 'Unable to accept invitation' });
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [token]);

  return (
    <main className="min-h-screen px-4 py-6 text-zinc-100 sm:px-6 lg:px-8">
      <section className="mx-auto flex min-h-[calc(100vh-3rem)] w-full max-w-2xl flex-col justify-center">
        <div className="mb-4 flex items-center justify-between text-xs uppercase tracking-[0.3em] text-zinc-400">
          <span>subcult-os</span>
          <a className="text-amber-300 transition hover:text-amber-200" href="/workspace">
            Workspace
          </a>
        </div>

        <div className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6 shadow-2xl shadow-black/40 backdrop-blur sm:p-8">
          <p className="text-xs uppercase tracking-[0.3em] text-fuchsia-300">Invitation</p>
          <h1 className="mt-3 text-3xl font-semibold tracking-tight text-white">Accept your invite</h1>
          <p className="mt-2 text-sm leading-6 text-zinc-400">
            We&apos;re checking this invitation, then linking it to the right account path.
          </p>

          <details className="mt-6 rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-xs text-zinc-500">
            <summary className="cursor-pointer list-none uppercase tracking-[0.25em] text-zinc-400">
              Invite token
            </summary>
            <div className="mt-3 space-y-1 font-mono text-[11px] leading-5 text-zinc-300">
              <div>{preview}</div>
              <div className="break-all text-zinc-500">{token}</div>
            </div>
          </details>

          {state.phase === 'loading' ? (
            <div className="mt-6 rounded-2xl border border-white/10 bg-white/5 p-4 text-sm text-zinc-400">
              Checking invitation…
            </div>
          ) : null}

          {state.phase === 'accepted' ? (
            <div className="mt-6 space-y-4">
              <div className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-4 text-sm text-emerald-200">
                Invitation accepted. You&apos;re in.
              </div>
            </div>
          ) : null}

          {state.phase === 'unauthorized' ? (
            <div className="mt-6 rounded-2xl border border-amber-400/30 bg-amber-400/10 px-4 py-4 text-sm leading-6 text-amber-100">
              This invitation needs the invited email address. Sign in or create the account that matches it.
            </div>
          ) : null}

          {state.phase === 'error' ? (
            <div className="mt-6 rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-4 text-sm leading-6 text-rose-200">
              {state.message}
            </div>
          ) : null}

          <div className="mt-6 rounded-[1.5rem] border border-white/10 bg-white/[0.04] p-4">
            <p className="text-xs uppercase tracking-[0.3em] text-zinc-500">Next steps</p>
            <p className="mt-2 text-sm leading-6 text-zinc-400">
              Keep moving with the workspace, or return here after signing in with the invited email.
            </p>
            <div className="mt-4 flex flex-wrap gap-3 text-sm">
              <a
                className="rounded-2xl bg-amber-300 px-4 py-3 font-medium text-zinc-950 transition hover:bg-amber-200"
                href="/workspace"
              >
                Workspace
              </a>
              <a
                className="rounded-2xl border border-white/10 bg-white/5 px-4 py-3 font-medium text-zinc-200 transition hover:bg-white/10"
                href={`/login?next=${encodedNextPath}`}
              >
                Sign in
              </a>
              <a
                className="rounded-2xl border border-white/10 bg-white/5 px-4 py-3 font-medium text-zinc-200 transition hover:bg-white/10"
                href={`/signup?next=${encodedNextPath}`}
              >
                Create account
              </a>
            </div>
            </div>
        </div>
      </section>
    </main>
  );
}
