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

export function DoorView({ eventId }: { eventId: string }) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<TicketDTO[]>([]);
  const [loading, setLoading] = useState(false);
  const [checkingIn, setCheckingIn] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  async function handleSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError(null);
    setMessage(null);

    try {
      const loaded = await api<TicketDTO[]>(`/api/events/${eventId}/door/tickets?query=${encodeURIComponent(query.trim())}`);
      setResults(loaded);
      setMessage(loaded.length === 0 ? 'No matches' : `${loaded.length} match${loaded.length === 1 ? '' : 'es'}`);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to search tickets');
    } finally {
      setLoading(false);
    }
  }

  async function handleCheckIn(code: string) {
    setCheckingIn(code);
    setError(null);
    setMessage(null);

    try {
      const updated = await postJSON<TicketDTO>(`/api/events/${eventId}/door/check-ins`, { code });
      setResults((current) => current.map((ticket) => (ticket.code === code ? updated : ticket)));
      setMessage(`${ticketLabel(updated)} checked in`);
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
          <p className="mt-2 text-sm leading-6 text-zinc-400">Search by email, display name, or code. Keep this screen open on a phone.</p>

          <div className="mt-4 rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm text-zinc-300">Event ID: {eventId}</div>
        </header>

        <form className="rounded-[1.75rem] border border-white/10 bg-zinc-950/90 p-4 shadow-xl shadow-black/20 sm:p-6" onSubmit={handleSearch}>
          <label className="block space-y-2 text-sm">
            <span className="text-zinc-300">Lookup</span>
            <input
              className="door-input w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-lg text-white outline-none transition placeholder:text-zinc-500 focus:border-amber-300/60 focus:bg-white/8"
              type="search"
              autoComplete="off"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Email, name, or code"
            />
          </label>

          <button
            className="door-action mt-3 w-full rounded-2xl bg-amber-300 px-4 py-4 text-lg font-semibold text-zinc-950 transition hover:bg-amber-200 disabled:cursor-not-allowed disabled:bg-amber-300/60"
            type="submit"
            disabled={loading}
          >
            {loading ? 'Searching…' : 'Search tickets'}
          </button>
        </form>

        {error ? <p className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">{error}</p> : null}
        {message ? <p className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">{message}</p> : null}

        <div className="space-y-3">
          {results.length === 0 ? <p className="rounded-[1.75rem] border border-white/10 bg-white/5 p-6 text-sm text-zinc-400">Search results will appear here.</p> : null}

          {results.map((ticket) => (
            <article key={ticket.id} className="rounded-[1.75rem] border border-white/10 bg-zinc-950/90 p-5 shadow-xl shadow-black/20">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <p className="text-lg font-medium text-white">{ticketLabel(ticket)}</p>
                  <p className="mt-1 text-sm text-zinc-400">{ticket.email}</p>
                </div>
                <span className="rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs uppercase tracking-[0.25em] text-zinc-400">{statusLabel(ticket)}</span>
              </div>

              <div className="mt-4 grid gap-2 text-sm text-zinc-400 sm:grid-cols-2">
                <div className="rounded-2xl border border-white/10 bg-white/5 px-4 py-3">Code: {ticket.code}</div>
                <div className="rounded-2xl border border-white/10 bg-white/5 px-4 py-3">Checked in: {ticket.checkedInAt ?? 'No'}</div>
              </div>

              <button
                className="door-action mt-4 w-full rounded-2xl bg-white px-4 py-4 text-left text-base font-semibold text-zinc-950 transition hover:bg-zinc-200 disabled:cursor-not-allowed disabled:bg-white/70"
                type="button"
                onClick={() => handleCheckIn(ticket.code)}
                disabled={checkingIn === ticket.code || ticket.status === 'checked_in'}
              >
                {ticket.status === 'checked_in' ? 'Already checked in' : checkingIn === ticket.code ? 'Checking in…' : 'Check in'}
              </button>
            </article>
          ))}
        </div>
      </section>
    </main>
  );
}
