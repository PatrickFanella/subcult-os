import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';
import { api, postJSON } from '../api';
import type { PublicEventDTO, TicketReservationDTO } from '../domain';

function formatDateTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat([], { dateStyle: 'medium', timeStyle: 'short' }).format(date);
}

function chunkCode(value: string) {
  return value.match(/.{1,4}/g) ?? [value];
}

export function PublicEventView({ slug }: { slug: string }) {
  const [event, setEvent] = useState<PublicEventDTO | null>(null);
  const [email, setEmail] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [loading, setLoading] = useState(true);
  const [reserving, setReserving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [reservation, setReservation] = useState<TicketReservationDTO | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setError(null);
      setReservation(null);
      setEmail('');
      setDisplayName('');
      setReserving(false);

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

    const trimmedEmail = email.trim();
    const trimmedDisplayName = displayName.trim();

    if (!trimmedEmail) {
      setError('Please enter the email address where we should send the ticket.');
      return;
    }

    setReserving(true);
    setError(null);

    try {
      const ticket = await postJSON<TicketReservationDTO>(`/api/public/events/${slug}/reservations`, {
        email: trimmedEmail,
        displayName: trimmedDisplayName || undefined,
      });

      setReservation(ticket);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to reserve ticket');
    } finally {
      setReserving(false);
    }
  }

  return (
    <main className="min-h-screen px-4 py-6 text-zinc-100 sm:px-6 lg:px-8">
      <section className="mx-auto w-full max-w-4xl space-y-6">
        <header className="relative overflow-hidden rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6 shadow-2xl shadow-black/40 backdrop-blur">
          <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_right,_rgba(251,191,36,0.16),_transparent_38%),radial-gradient(circle_at_bottom_left,_rgba(217,70,239,0.14),_transparent_34%)]" />
          <div className="relative">
            <p className="text-xs uppercase tracking-[0.35em] text-fuchsia-300">Free guest reservation</p>
            <div className="mt-3 flex flex-wrap items-center gap-2">
              <span className="rounded-full border border-amber-300/30 bg-amber-300/10 px-3 py-1 text-[0.7rem] font-semibold uppercase tracking-[0.28em] text-amber-200">
                No account needed
              </span>
              <span className={`rounded-full border px-3 py-1 text-[0.7rem] font-semibold uppercase tracking-[0.28em] ${event?.isFull ? 'border-rose-400/30 bg-rose-500/10 text-rose-200' : 'border-emerald-400/30 bg-emerald-500/10 text-emerald-200'}`}>
                {event?.isFull ? 'Sold out' : `${event?.remainingTickets ?? '—'} remaining`}
              </span>
            </div>

            <div className="mt-5 grid gap-6 lg:grid-cols-[1.15fr_0.85fr] lg:items-end">
              <div>
                <h1 className="text-3xl font-semibold tracking-tight text-white sm:text-4xl">{event?.title ?? 'Reserve your free ticket'}</h1>
                <p className="mt-3 max-w-2xl text-sm leading-7 text-zinc-300 sm:text-base">
                  Grab a free spot. Email required to send the ticket. Display name optional. No account needed.
                </p>
              </div>

              <div className="grid gap-3 sm:grid-cols-3 lg:grid-cols-1">
                <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                  <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Date & time</p>
                  <p className="mt-2 text-sm font-medium text-white">{event ? formatDateTime(event.startsAt) : 'Loading event…'}</p>
                </div>
                <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                  <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Location</p>
                  <p className="mt-2 text-sm font-medium text-white">{event?.locationDisplay ?? 'Loading location…'}</p>
                </div>
                <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                  <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Tickets</p>
                  <p className={`mt-2 text-sm font-medium ${event?.isFull ? 'text-rose-200' : 'text-white'}`}>
                    {event ? (event.isFull ? 'Sold out' : `${event.remainingTickets} left`) : 'Loading availability…'}
                  </p>
                </div>
              </div>
            </div>
          </div>

          {event ? (
            <div className="relative mt-6 grid gap-3 sm:grid-cols-4">
              <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Status</p>
                <p className="mt-2 text-sm font-medium text-white">{event.status}</p>
              </div>
              <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Remaining</p>
                <p className={`mt-2 text-sm font-medium ${event.isFull ? 'text-rose-200' : 'text-white'}`}>{event.isFull ? 'Sold out' : event.remainingTickets}</p>
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
        {error ? (
          <p aria-live="polite" className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
            {error}
          </p>
        ) : null}

        {event ? (
          <div className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
            <section className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6">
              <p className="text-xs uppercase tracking-[0.3em] text-amber-300">About</p>
              <p className="mt-3 whitespace-pre-wrap text-sm leading-7 text-zinc-300">{event.publicDescription}</p>
            </section>

            {reservation ? (
              <section className="rounded-[1.75rem] border border-emerald-500/20 bg-emerald-500/10 p-6">
                <p className="text-xs uppercase tracking-[0.3em] text-emerald-200">Reservation confirmed</p>
                <h2 className="mt-3 text-xl font-semibold text-white">Your ticket is ready</h2>

                <div className="mt-5 grid gap-3 text-sm text-zinc-200">
                  <div className="rounded-2xl border border-white/10 bg-black/20 p-4">
                    <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Ticket holder</p>
                    <p className="mt-2 font-medium text-white">{reservation.displayName ?? '—'}</p>
                    <p className="mt-1 text-zinc-300">{reservation.email}</p>
                  </div>

                  <div className="rounded-2xl border border-white/10 bg-black/20 p-4">
                    <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Access code</p>
                    <div className="mt-3 flex flex-wrap gap-2 text-sm font-semibold tracking-[0.35em] text-white">
                      {chunkCode(reservation.code).map((part, index) => (
                        <span key={`${part}-${index}`} className="rounded-xl border border-white/10 bg-white/5 px-3 py-2 font-mono">
                          {part}
                        </span>
                      ))}
                    </div>
                  </div>
                </div>

                <a
                  className="mt-5 inline-flex w-full items-center justify-center rounded-2xl bg-emerald-300 px-4 py-3 font-medium text-zinc-950 transition hover:bg-emerald-200"
                  href={reservation.ticketUrl}
                >
                  Open ticket
                </a>

                <p className="mt-3 text-sm text-emerald-100/80">Ticket link recorded and sent by the dev outbox.</p>
              </section>
            ) : (
              <form className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6" onSubmit={handleSubmit}>
                <p className="text-xs uppercase tracking-[0.3em] text-amber-300">Reserve ticket</p>
                <p className="mt-2 text-sm leading-6 text-zinc-400">Email is required so we can send the ticket. Display name is optional.</p>

                <label className="mt-4 block space-y-2 text-sm">
                  <span className="text-zinc-300">Email <span className="text-rose-300">required</span></span>
                  <input
                    className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-white/8"
                    type="email"
                    autoComplete="email"
                    required
                    value={email}
                    onChange={(event) => setEmail(event.target.value)}
                    disabled={event.isFull || reserving}
                  />
                </label>

                <label className="mt-4 block space-y-2 text-sm">
                  <span className="text-zinc-300">Display name <span className="text-zinc-500">optional</span></span>
                  <input
                    className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-white/8"
                    type="text"
                    autoComplete="name"
                    value={displayName}
                    onChange={(event) => setDisplayName(event.target.value)}
                    placeholder="Optional"
                    disabled={event.isFull || reserving}
                  />
                </label>

                <button
                  className="door-action mt-4 w-full rounded-2xl bg-amber-300 px-4 py-3 font-medium text-zinc-950 transition hover:bg-amber-200 disabled:cursor-not-allowed disabled:bg-amber-300/60"
                  type="submit"
                  disabled={reserving || event.isFull}
                >
                  {event.isFull ? 'Sold out' : reserving ? 'Reserving…' : 'Reserve free ticket'}
                </button>

                {event.isFull ? <p className="mt-3 text-sm text-rose-200">This event is sold out. Reservations are closed.</p> : <p className="mt-3 text-sm text-zinc-400">No account needed — just your email.</p>}
              </form>
            )}
          </div>
        ) : null}
      </section>
    </main>
  );
}
