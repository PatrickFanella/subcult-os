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
    <section className="overflow-hidden rounded-[1.75rem] border border-sky-300/15 bg-zinc-950/85 shadow-xl shadow-black/30" aria-labelledby="atproto-identity-title">
      <div className="grid gap-6 p-6 lg:grid-cols-[0.9fr_1.1fr]">
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-sky-300">Portable identity</p>
          <h2 id="atproto-identity-title" className="mt-2 text-2xl font-semibold tracking-tight text-white">Link an AT Protocol account</h2>
          <p className="mt-3 max-w-xl text-sm leading-6 text-zinc-400">
            Prove control of a handle or DID without handing Subcult OS repository permissions. A link identifies you; it never adds workspace membership or publishing authority.
          </p>
          <div className="mt-4 flex flex-wrap gap-2 text-[11px] uppercase tracking-[0.2em] text-zinc-400">
            <span className="rounded-full border border-sky-300/20 bg-sky-300/5 px-3 py-1">Identity scope only</span>
            <span className="rounded-full border border-white/10 bg-white/5 px-3 py-1">No automatic merge</span>
          </div>
        </div>

        <form className="rounded-2xl border border-white/10 bg-white/[0.04] p-4" onSubmit={startLink}>
          <label className="block text-sm font-medium text-zinc-200" htmlFor="atproto-identifier">Handle or DID</label>
          <p className="mt-1 text-xs leading-5 text-zinc-500">For example, <code>artist.example.com</code> or <code>did:plc:…</code></p>
          <div className="mt-3 flex flex-col gap-3 sm:flex-row">
            <input
              id="atproto-identifier"
              className="min-w-0 flex-1 rounded-2xl border border-white/10 bg-black/30 px-4 py-3 text-white outline-none transition placeholder:text-zinc-600 focus:border-sky-300/60"
              value={identifier}
              onChange={(event) => setIdentifier(event.target.value)}
              placeholder="handle.example.com"
              autoCapitalize="none"
              autoCorrect="off"
              spellCheck={false}
              disabled={submitting || available === null}
            />
            <button className="rounded-2xl bg-sky-300 px-5 py-3 font-medium text-zinc-950 transition hover:bg-sky-200 disabled:cursor-not-allowed disabled:bg-sky-300/50" type="submit" disabled={submitting || available === null}>
              {submitting ? 'Opening…' : available === null ? 'Checking…' : 'Link account'}
            </button>
          </div>
        </form>
      </div>

      {notice ? <p className="mx-6 mb-4 rounded-2xl border border-emerald-400/20 bg-emerald-400/10 px-4 py-3 text-sm text-emerald-100" role="status">{notice}</p> : null}
      {error ? <p className="mx-6 mb-4 rounded-2xl border border-rose-400/20 bg-rose-400/10 px-4 py-3 text-sm text-rose-100" role="alert">{error}</p> : null}

      {links.length > 0 ? (
        <div className="border-t border-white/10 bg-black/20 px-6 py-5">
          <p className="text-xs uppercase tracking-[0.25em] text-zinc-500">Linked identities</p>
          <div className="mt-3 grid gap-3 md:grid-cols-2">
            {links.map((link) => (
              <article key={link.did} className="rounded-2xl border border-white/10 bg-white/[0.04] p-4">
                <p className="font-medium text-white">{link.handle ?? 'AT Protocol identity'}</p>
                <p className="mt-1 break-all font-mono text-xs leading-5 text-zinc-500">{link.did}</p>
                <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
                  <p className="text-xs text-zinc-500">Linked {new Date(link.verifiedAt).toLocaleDateString()}</p>
                  {confirmingDID === link.did ? (
                    <span className="flex items-center gap-2">
                      <button className="rounded-full border border-white/10 px-3 py-2 text-xs text-zinc-300 hover:bg-white/10" type="button" onClick={() => setConfirmingDID(null)}>Keep linked</button>
                      <button className="rounded-full bg-rose-300 px-3 py-2 text-xs font-medium text-zinc-950 hover:bg-rose-200 disabled:opacity-50" type="button" onClick={() => void unlink(link.did)} disabled={unlinkingDID === link.did}>
                        {unlinkingDID === link.did ? 'Unlinking…' : 'Confirm unlink'}
                      </button>
                    </span>
                  ) : (
                    <button className="rounded-full border border-rose-300/20 px-3 py-2 text-xs text-rose-200 hover:bg-rose-300/10" type="button" onClick={() => setConfirmingDID(link.did)}>Unlink</button>
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
