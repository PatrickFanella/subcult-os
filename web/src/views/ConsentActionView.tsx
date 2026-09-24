import { useRef, useState } from 'react';

import { postJSON } from '../api';

type ConsentAction = 'confirm' | 'withdraw';

function tokenFromLocation() {
  return new URLSearchParams(window.location.search).get('token') ?? '';
}

// The emailed consent links open this page instead of hitting the API
// directly, so a link-prefetching mail scanner cannot confirm or withdraw
// on the recipient's behalf: the state change happens only on the button.
export function ConsentActionView({ action }: { action: ConsentAction }) {
  const [loading, setLoading] = useState(false);
  const submitting = useRef(false);
  const [notice, setNotice] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function submit() {
    if (submitting.current) return;
    submitting.current = true;
    setError(null);
    setLoading(true);
    try {
      const token = tokenFromLocation();
      if (!token) throw new Error('This link is missing its token.');
      await postJSON<{ status: string }>(`/api/public/consent/${encodeURIComponent(token)}/${action}`, {});
      window.history.replaceState(null, '', window.location.pathname);
      setNotice(action === 'confirm' ? 'Thanks, your consent is confirmed.' : 'You have been unsubscribed. No further announcements will be sent to this address.');
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to continue');
    } finally {
      submitting.current = false;
      setLoading(false);
    }
  }

  const title = action === 'confirm' ? 'Confirm announcement emails' : 'Unsubscribe from announcements';
  const explanation =
    action === 'confirm'
      ? 'A workspace asked to send you announcement emails. Nothing is sent until you confirm below, and you can withdraw at any time.'
      : 'Confirm below to withdraw your consent. Transactional messages you request yourself, such as ticket confirmations, are unaffected.';
  return (
    <main className="min-h-screen px-4 py-6 text-zinc-100">
      <section className="mx-auto flex min-h-[calc(100vh-3rem)] w-full max-w-md flex-col justify-center">
        <div className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6 shadow-2xl shadow-black/40">
          <p className="text-xs uppercase tracking-[0.3em] text-fuchsia-300">Email preferences</p>
          <h1 className="mt-3 text-3xl font-semibold text-white">{title}</h1>
          <p className="mt-3 text-sm text-zinc-400">{explanation}</p>
          {!notice ? (
            <button className="mt-6 w-full rounded-2xl bg-amber-300 px-4 py-3 font-medium text-zinc-950 disabled:opacity-60" disabled={loading} type="button" onClick={() => void submit()}>
              {loading ? 'Working…' : action === 'confirm' ? 'Confirm consent' : 'Unsubscribe'}
            </button>
          ) : null}
          {error ? <p role="alert" className="mt-4 rounded-2xl bg-rose-500/10 px-4 py-3 text-sm text-rose-200">{error}</p> : null}
          {notice ? <p role="status" className="mt-4 rounded-2xl bg-emerald-500/10 px-4 py-3 text-sm text-emerald-100">{notice}</p> : null}
        </div>
      </section>
    </main>
  );
}
