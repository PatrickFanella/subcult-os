import { useEffect, useMemo, useState } from 'react';
import { Brand } from '../ui/Brand';

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
    <main className="min-h-screen px-4 py-6 text-fg-primary sm:px-6 lg:px-8">
      <section className="mx-auto flex min-h-[calc(100vh-3rem)] w-full max-w-2xl flex-col justify-center">
        <div className="mb-4 flex flex-wrap items-center justify-between gap-x-4 gap-y-2 text-xs uppercase tracking-[0.2em] text-fg-secondary">
          <Brand />
          <a className="text-fg-primary underline underline-offset-4" href="/workspace">
            Workspace
          </a>
        </div>

        <div className="rounded-panel border border-stroke-subtle bg-surface-panel p-6 sm:p-8">
          <p className="text-xs uppercase tracking-[0.2em] text-fg-muted">Invitation</p>
          <h1 className="mt-3 text-3xl font-bold tracking-tight text-fg-primary">Accept your invite</h1>
          <p className="mt-2 text-sm leading-6 text-fg-secondary">
            We&apos;re checking this invitation, then linking it to the right account path.
          </p>

          <details className="mt-6 rounded-2xl border border-stroke-subtle bg-surface-inset px-4 py-3 text-xs text-fg-muted">
            <summary className="cursor-pointer list-none uppercase tracking-[0.2em] text-fg-secondary">
              Invite token
            </summary>
            <div className="mt-3 space-y-1 font-mono text-[11px] leading-5 text-fg-secondary">
              <div>{preview}</div>
              <div className="break-all text-fg-muted">{token}</div>
            </div>
          </details>

          {state.phase === 'loading' ? (
            <div className="mt-6 rounded-2xl border border-stroke-subtle bg-surface-inset p-4 text-sm text-fg-secondary">
              Checking invitation…
            </div>
          ) : null}

          {state.phase === 'accepted' ? (
            <div className="mt-6 space-y-4">
              <div className="rounded-2xl border border-status-success/20 bg-status-surface-success px-4 py-4 text-sm text-status-success">
                Invitation accepted. You&apos;re in.
              </div>
            </div>
          ) : null}

          {state.phase === 'unauthorized' ? (
            <div className="mt-6 rounded-2xl border border-status-warning/20 bg-status-surface-warning px-4 py-4 text-sm leading-6 text-status-warning">
              This invitation needs the invited email address. Sign in or create the account that matches it.
            </div>
          ) : null}

          {state.phase === 'error' ? (
            <div className="mt-6 rounded-2xl border border-status-danger/20 bg-status-surface-danger px-4 py-4 text-sm leading-6 text-status-danger">
              {state.message}
            </div>
          ) : null}

          <div className="mt-6 border border-stroke-subtle bg-surface-inset p-4">
            <p className="text-xs uppercase tracking-[0.2em] text-fg-muted">Next steps</p>
            <p className="mt-2 text-sm leading-6 text-fg-secondary">
              Keep moving with the workspace, or return here after signing in with the invited email.
            </p>
            <div className="mt-4 flex flex-wrap gap-3 text-sm">
              <a
                className="btn-primary px-4"
                href="/workspace"
              >
                Workspace
              </a>
              <a
                className="btn-secondary px-4"
                href={`/login?next=${encodedNextPath}`}
              >
                Sign in
              </a>
              <a
                className="btn-secondary px-4"
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
