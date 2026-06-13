import { useState } from 'react';
import type { FormEvent } from 'react';
import { api, postJSON } from '../api';
import type { TicketDTO } from '../domain';

function ticketLabel(ticket: TicketDTO) {
  return ticket.displayName ?? ticket.email;
}

function statusLabel(ticket: TicketDTO) {
  return ticket.status === 'checked_in' ? 'Checked in' : 'Reserved';
}

function chunkCode(code: string) {
  const chunks: string[] = [];

  for (let index = 0; index < code.length; index += 4) {
    chunks.push(code.slice(index, index + 4));
  }

  return chunks.join(' ');
}

function formatHumanTime(value: string | null) {
  if (!value) {
    return 'Not checked in yet';
  }

  const timestamp = new Date(value);

  if (Number.isNaN(timestamp.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(timestamp);
}

function noticeClassName(kind: 'neutral' | 'success' | 'error') {
  if (kind === 'success') {
    return 'border-emerald-500/30 bg-emerald-500/10 text-emerald-200';
  }

  if (kind === 'error') {
    return 'border-rose-500/30 bg-rose-500/10 text-rose-200';
  }

  return 'border-white/10 bg-white/5 text-zinc-300';
}

function statusSurface(ticket: TicketDTO) {
  return ticket.status === 'checked_in'
    ? 'border-emerald-400/30 bg-emerald-500/10 text-emerald-50'
    : 'border-amber-300/30 bg-amber-300/10 text-amber-50';
}

export function DoorView({ eventId }: { eventId: string }) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<TicketDTO[]>([]);
  const [loading, setLoading] = useState(false);
  const [checkingIn, setCheckingIn] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<{ kind: 'neutral' | 'success' | 'error'; text: string } | null>(null);

  function handleClear() {
    setQuery('');
    setResults([]);
    setLoading(false);
    setCheckingIn(null);
    setError(null);
    setNotice({ kind: 'neutral', text: 'Ready for a new lookup.' });
  }

  async function handleSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const trimmedQuery = query.trim();

    setError(null);
    setNotice(null);

    if (!trimmedQuery) {
      setLoading(false);
      setNotice({ kind: 'neutral', text: 'Type an email, name, or exact code to search.' });
      return;
    }

    setQuery(trimmedQuery);
    setLoading(true);

    try {
      const loaded = await api<TicketDTO[]>(`/api/events/${eventId}/door/tickets?query=${encodeURIComponent(trimmedQuery)}`);
      setResults(loaded);
      setNotice(
        loaded.length === 0
          ? { kind: 'neutral', text: `No matches for “${trimmedQuery}”.` }
          : { kind: 'success', text: `${loaded.length} ticket${loaded.length === 1 ? '' : 's'} ready.` },
      );
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to search tickets');
    } finally {
      setLoading(false);
    }
  }

  async function handleCheckIn(ticket: TicketDTO) {
    const wasAlreadyCheckedIn = ticket.status === 'checked_in';
    setCheckingIn(ticket.code);
    setError(null);
    setNotice(null);

    try {
      const updated = await postJSON<TicketDTO>(`/api/events/${eventId}/door/check-ins`, { code: ticket.code });
      setResults((current) => {
        const next = current.map((currentTicket) => (currentTicket.code === updated.code ? updated : currentTicket));
        return next.some((currentTicket) => currentTicket.code === updated.code) ? next : [updated, ...next];
      });
      setNotice({
        kind: 'success',
        text: wasAlreadyCheckedIn ? `Already checked in — ${ticketLabel(updated)}` : `Checked in — ${ticketLabel(updated)}`,
      });
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to check in ticket');
    } finally {
      setCheckingIn(null);
    }
  }

  return (
    <main className="min-h-screen px-4 py-4 text-zinc-100 sm:px-6 lg:px-8">
      <section className="mx-auto w-full max-w-3xl space-y-4">
        <header className="rounded-[1.75rem] border border-white/10 bg-zinc-950/90 p-5 shadow-2xl shadow-black/40 backdrop-blur sm:p-6">
          <p className="text-xs uppercase tracking-[0.3em] text-fuchsia-300">Door</p>
          <h1 className="mt-2 text-3xl font-semibold tracking-tight text-white">Mobile check-in</h1>
          <p className="mt-2 text-sm leading-6 text-zinc-400">Search by email, display name, or exact code. Paste a full code and press Search to jump straight to check-in.</p>

          <div className="mt-4 rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm text-zinc-300">Event ID: {eventId}</div>
        </header>

        <form className="rounded-[1.75rem] border border-white/10 bg-zinc-950/90 p-4 shadow-xl shadow-black/20 sm:p-6" onSubmit={handleSearch}>
          <label className="block space-y-2 text-sm">
            <span className="text-zinc-300">Lookup or exact code</span>
            <input
              className="door-input w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-lg text-white outline-none transition placeholder:text-zinc-500 focus:border-amber-300/60 focus:bg-white/8"
              type="search"
              autoComplete="off"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Email, name, or full code"
            />
          </label>

          <div className="mt-3 grid gap-2 sm:grid-cols-2">
            <button
              className="door-action rounded-2xl bg-amber-300 px-4 py-4 text-lg font-semibold text-zinc-950 transition hover:bg-amber-200 disabled:cursor-not-allowed disabled:bg-amber-300/60"
              type="submit"
              disabled={loading}
            >
              {loading ? 'Searching…' : 'Search'}
            </button>
            <button
              className="door-action rounded-2xl border border-white/10 bg-white/5 px-4 py-4 text-lg font-semibold text-zinc-100 transition hover:bg-white/10"
              type="button"
              onClick={handleClear}
            >
              Reset
            </button>
          </div>

          <p className="mt-3 text-xs leading-5 text-zinc-500">Exact code works. Search by email, name, or the full ticket code to pull up a single result.</p>
        </form>

        {error ? <p className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">{error}</p> : null}
        {notice ? (
          <p className={`rounded-2xl border px-4 py-3 text-sm ${noticeClassName(notice.kind)}`} aria-live="polite">
            {notice.text}
          </p>
        ) : null}

        <div className="space-y-3">
          {results.length === 0 ? <p className="rounded-[1.75rem] border border-white/10 bg-white/5 p-6 text-sm text-zinc-400">Search results will appear here.</p> : null}

          {results.map((ticket) => (
            <article key={ticket.id} className="rounded-[1.75rem] border border-white/10 bg-zinc-950/90 p-4 shadow-xl shadow-black/20 sm:p-5">
              <div className={`rounded-[1.4rem] border px-4 py-4 ${statusSurface(ticket)}`}>
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className="text-xs uppercase tracking-[0.28em] text-white/70">Status</p>
                    <p className="mt-2 text-2xl font-semibold tracking-tight text-white">{statusLabel(ticket)}</p>
                  </div>
                  <span className="rounded-full border border-white/15 bg-black/15 px-3 py-1 text-xs uppercase tracking-[0.25em] text-white/80">{ticket.status === 'checked_in' ? 'Door ready' : 'Needs check-in'}</span>
                </div>

                <p className="mt-3 text-sm text-white/80">{ticket.status === 'checked_in' ? `Checked in at ${formatHumanTime(ticket.checkedInAt)}` : 'Awaiting check-in'}</p>
              </div>

              <div className="mt-4 space-y-3">
                <div>
                  <p className="text-lg font-medium text-white">{ticketLabel(ticket)}</p>
                  <p className="mt-1 text-sm text-zinc-400">{ticket.email}</p>
                </div>

                <div className="rounded-2xl border border-white/10 bg-white/5 px-4 py-4">
                  <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Code</p>
                  <p className="mt-2 break-words font-mono text-xl tracking-[0.24em] text-white sm:text-2xl">{chunkCode(ticket.code)}</p>
                </div>

                <div className="grid gap-2 sm:grid-cols-2">
                  <div className="rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm text-zinc-300">
                    <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Checked in</p>
                    <p className="mt-2 text-zinc-100">{formatHumanTime(ticket.checkedInAt)}</p>
                  </div>
                  <div className="rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm text-zinc-300">
                    <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Ticket holder</p>
                    <p className="mt-2 text-zinc-100">{ticketLabel(ticket)}</p>
                  </div>
                </div>
              </div>

              <button
                className="door-action mt-4 w-full rounded-2xl bg-white px-4 py-4 text-left text-base font-semibold text-zinc-950 transition hover:bg-zinc-200 disabled:cursor-not-allowed disabled:bg-white/70"
                type="button"
                onClick={() => handleCheckIn(ticket)}
                disabled={checkingIn === ticket.code || ticket.status === 'checked_in'}
              >
                {ticket.status === 'checked_in' ? 'Already checked in' : checkingIn === ticket.code ? 'Checking in…' : 'Check in this code'}
              </button>
            </article>
          ))}
        </div>
      </section>
    </main>
  );
}
