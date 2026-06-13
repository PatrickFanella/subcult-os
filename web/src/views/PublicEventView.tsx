import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';
import { api, postJSON } from '../api';
import type { PublicEventDTO, TicketReservationDTO } from '../domain';

function formatDateTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat([], { dateStyle: 'medium', timeStyle: 'short' }).format(date);
}

export function PublicEventView({ slug }: { slug: string }) {
  const [event, setEvent] = useState<PublicEventDTO | null>(null);
  const [email, setEmail] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [loading, setLoading] = useState(true);
  const [reserving, setReserving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setError(null);

      try {
        const loaded = await api<PublicEventDTO>(`/api/public/events/${slug}`);
        if (!cancelled) {
          setEvent(loaded);
        }
      } catch (caught) {
        if (!cancelled) {
          setError(caught instanceof Error ? caught.message : 'Unable to load public event');
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [slug]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setReserving(true);
    setError(null);
    setMessage(null);

    try {
      const ticket = await postJSON<TicketReservationDTO>(`/api/public/events/${slug}/reservations`, {
        email: email.trim(),
        displayName: displayName.trim() || undefined,
      });

      setMessage('Reservation confirmed');
      window.location.href = ticket.ticketUrl;
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to reserve ticket');
    } finally {
      setReserving(false);
    }
  }

  return (
    <main className="min-h-screen px-4 py-6 text-zinc-100 sm:px-6 lg:px-8">
      <section className="mx-auto w-full max-w-4xl space-y-6">
        <header className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6 shadow-2xl shadow-black/40 backdrop-blur">
          <p className="text-xs uppercase tracking-[0.3em] text-fuchsia-300">Reserve</p>
          <h1 className="mt-2 text-3xl font-semibold tracking-tight text-white">{event?.title ?? 'Reserve ticket'}</h1>
          <p className="mt-2 text-sm leading-6 text-zinc-400">Direct-link page for guests reserving a free ticket.</p>

          {event ? (
            <div className="mt-6 grid gap-3 sm:grid-cols-4">
              <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Status</p>
                <p className="mt-2 text-sm font-medium text-white">{event.status}</p>
              </div>
              <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Remaining</p>
                <p className="mt-2 text-sm font-medium text-white">{event.remainingTickets}</p>
              </div>
              <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Location</p>
                <p className="mt-2 text-sm font-medium text-white">{event.locationDisplay}</p>
              </div>
              <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Starts</p>
                <p className="mt-2 text-sm font-medium text-white">{formatDateTime(event.startsAt)}</p>
              </div>
            </div>
          ) : null}
        </header>

        {loading ? <div className="rounded-[1.75rem] border border-white/10 bg-white/5 p-6 text-sm text-zinc-400">Loading…</div> : null}
        {error ? <p className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">{error}</p> : null}
        {message ? <p className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">{message}</p> : null}

        {event ? (
          <div className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
            <section className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6">
              <p className="text-xs uppercase tracking-[0.3em] text-amber-300">About</p>
              <p className="mt-3 whitespace-pre-wrap text-sm leading-7 text-zinc-300">{event.publicDescription}</p>
            </section>

            <form className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6" onSubmit={handleSubmit}>
              <p className="text-xs uppercase tracking-[0.3em] text-amber-300">Reserve ticket</p>
              <label className="mt-4 block space-y-2 text-sm">
                <span className="text-zinc-300">Email</span>
                <input
                  className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-white/8"
                  type="email"
                  autoComplete="email"
                  required
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                />
              </label>

              <label className="mt-4 block space-y-2 text-sm">
                <span className="text-zinc-300">Display name</span>
                <input
                  className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-white/8"
                  type="text"
                  autoComplete="name"
                  value={displayName}
                  onChange={(event) => setDisplayName(event.target.value)}
                  placeholder="Optional"
                />
              </label>

              <button
                className="door-action mt-4 w-full rounded-2xl bg-amber-300 px-4 py-3 font-medium text-zinc-950 transition hover:bg-amber-200 disabled:cursor-not-allowed disabled:bg-amber-300/60"
                type="submit"
                disabled={reserving || event.isFull}
              >
                {event.isFull ? 'Sold out' : reserving ? 'Reserving…' : 'Reserve'}
              </button>

              {event.isFull ? <p className="mt-3 text-sm text-zinc-400">This event is full.</p> : null}
            </form>
          </div>
        ) : null}
      </section>
    </main>
  );
}
