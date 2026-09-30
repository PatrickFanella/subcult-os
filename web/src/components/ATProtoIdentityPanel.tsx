import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';
import { ApiError, api, deleteJSON, postJSON } from '../api';

type ATProtoLink = {
  did: string;
  handle: string | null;
  verifiedAt: string;
};

type ATProtoLinksResponse = {
  links: ATProtoLink[];
};

type ATProtoStartResponse = {
  authorizationUrl: string;
};

export function atprotoCallbackNotice(search: string) {
  const status = new URLSearchParams(search).get('atproto');
  if (status === 'linked') return 'AT Protocol identity linked. It proves account control only; it does not grant workspace or publishing access.';
  if (status === 'cancelled') return 'AT Protocol authorization was cancelled. Nothing was linked.';
  if (status === 'error') return 'AT Protocol linking could not be completed. You can safely try again.';
  return null;
}

export function ATProtoIdentityPanel() {
  const [available, setAvailable] = useState<boolean | null>(null);
  const [links, setLinks] = useState<ATProtoLink[]>([]);
  const [identifier, setIdentifier] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [unlinkingDID, setUnlinkingDID] = useState<string | null>(null);
  const [confirmingDID, setConfirmingDID] = useState<string | null>(null);
  const [notice, setNotice] = useState(() => atprotoCallbackNotice(typeof window === 'undefined' ? '' : window.location.search));
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    api<ATProtoLinksResponse>('/api/v1/auth/atproto/links')
      .then((response) => {
        if (!cancelled) {
          setLinks(response.links);
          setAvailable(true);
        }
      })
      .catch((caught: unknown) => {
        if (cancelled) return;
        if (caught instanceof ApiError && caught.status === 404) {
          setAvailable(false);
          return;
        }
        setAvailable(true);
        setError('AT Protocol identity status is temporarily unavailable.');
      });
    return () => {
      cancelled = true;
    };
  }, []);

  async function startLink(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalized = identifier.trim();
    if (!normalized) {
      setError('Enter an AT Protocol handle or DID.');
      return;
    }
    setSubmitting(true);
    setError(null);
    setNotice(null);
    try {
      const response = await postJSON<ATProtoStartResponse>('/api/v1/auth/atproto/start', { identifier: normalized });
      window.location.assign(response.authorizationUrl);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Could not start AT Protocol authorization.');
      setSubmitting(false);
    }
  }

  async function unlink(did: string) {
    setUnlinkingDID(did);
    setError(null);
    setNotice(null);
    try {
      await deleteJSON(`/api/v1/auth/atproto/links/${encodeURIComponent(did)}`);
      setLinks((current) => current.filter((link) => link.did !== did));
      setConfirmingDID(null);
      setNotice('AT Protocol identity unlinked locally. Stored Subcult OS OAuth sessions were deleted.');
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Could not unlink AT Protocol identity.');
    } finally {
      setUnlinkingDID(null);
    }
  }

  if (available === false) return null;

  return (
    <section className="overflow-hidden rounded-panel border border-status-info/20 bg-surface-panel shadow-panel" aria-labelledby="atproto-identity-title">
      <div className="grid gap-6 p-6 lg:grid-cols-[0.9fr_1.1fr]">
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-fg-muted">Portable identity</p>
          <h2 id="atproto-identity-title" className="mt-2 text-2xl font-extrabold tracking-tight text-fg-primary">Link an AT Protocol account</h2>
          <p className="mt-3 max-w-xl text-sm leading-6 text-fg-secondary">
            Prove control of a handle or DID without handing Subcult OS repository permissions. A link identifies you; it never adds workspace membership or publishing authority.
          </p>
          <div className="mt-4 flex flex-wrap gap-2 text-[11px] uppercase tracking-[0.2em] text-fg-secondary">
            <span className="rounded-full border border-status-info/20 bg-action-disabled px-3 py-1">Identity scope only</span>
            <span className="rounded-full border border-stroke-subtle bg-surface-inset px-3 py-1">No automatic merge</span>
          </div>
        </div>

        <form className="rounded-2xl border border-stroke-subtle bg-surface-inset p-4" onSubmit={startLink}>
          <label className="block text-sm font-medium text-fg-primary" htmlFor="atproto-identifier">Handle or DID</label>
          <p className="mt-1 text-xs leading-5 text-fg-muted">For example, <code>artist.example.com</code> or <code>did:plc:…</code></p>
          <div className="mt-3 flex flex-col gap-3 sm:flex-row">
            <input
              id="atproto-identifier"
              className="min-w-0 flex-1 rounded-2xl border border-stroke-subtle bg-surface-inset px-4 py-3 text-fg-primary outline-none transition placeholder:text-fg-muted focus:border-stroke-focus"
              value={identifier}
              onChange={(event) => setIdentifier(event.target.value)}
              placeholder="handle.example.com"
              autoCapitalize="none"
              autoCorrect="off"
              spellCheck={false}
              disabled={submitting || available === null}
            />
            <button className="rounded-2xl bg-action-primary px-5 py-3 font-medium text-fg-inverse transition hover:bg-action-hover disabled:cursor-not-allowed disabled:bg-action-disabled" type="submit" disabled={submitting || available === null}>
              {submitting ? 'Opening…' : available === null ? 'Checking…' : 'Link account'}
            </button>
          </div>
        </form>
      </div>

      {notice ? <p className="mx-6 mb-4 rounded-2xl border border-status-success/20 bg-status-surface-success px-4 py-3 text-sm text-status-success" role="status">{notice}</p> : null}
      {error ? <p className="mx-6 mb-4 rounded-2xl border border-status-danger/20 bg-status-surface-danger px-4 py-3 text-sm text-status-danger" role="alert">{error}</p> : null}

      {links.length > 0 ? (
        <div className="border-t border-stroke-subtle bg-surface-inset px-6 py-5">
          <p className="text-xs uppercase tracking-[0.25em] text-fg-muted">Linked identities</p>
          <div className="mt-3 grid gap-3 md:grid-cols-2">
            {links.map((link) => (
              <article key={link.did} className="rounded-2xl border border-stroke-subtle bg-surface-inset p-4">
                <p className="font-medium text-fg-primary">{link.handle ?? 'AT Protocol identity'}</p>
                <p className="mt-1 break-all font-mono text-xs leading-5 text-fg-muted">{link.did}</p>
                <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
                  <p className="text-xs text-fg-muted">Linked {new Date(link.verifiedAt).toLocaleDateString()}</p>
                  {confirmingDID === link.did ? (
                    <span className="flex items-center gap-2">
                      <button className="rounded-full border border-stroke-subtle px-3 py-2 text-xs text-fg-secondary hover:bg-surface-inset" type="button" onClick={() => setConfirmingDID(null)}>Keep linked</button>
                      <button className="rounded-full bg-action-primary px-3 py-2 text-xs font-medium text-fg-inverse hover:bg-action-hover disabled:opacity-50" type="button" onClick={() => void unlink(link.did)} disabled={unlinkingDID === link.did}>
                        {unlinkingDID === link.did ? 'Unlinking…' : 'Confirm unlink'}
                      </button>
                    </span>
                  ) : (
                    <button className="rounded-full border border-status-danger/20 px-3 py-2 text-xs text-status-danger hover:bg-action-disabled" type="button" onClick={() => setConfirmingDID(link.did)}>Unlink</button>
                  )}
                </div>
              </article>
            ))}
          </div>
        </div>
      ) : null}
    </section>
  );
}
