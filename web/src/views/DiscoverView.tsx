import { useEffect, useState } from 'react';
import { api } from '../api';
import type { PublicEventSummaryDTO } from '../domain';

function formatDateTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat([], { dateStyle: 'medium', timeStyle: 'short' }).format(date);
}

function formatCurrency(cents: number, currency: string) {
  return new Intl.NumberFormat([], { style: 'currency', currency: currency.toUpperCase() }).format(cents / 100);
}

function pricingLabel(event: PublicEventSummaryDTO) {
  return event.pricingMode === 'free' ? 'Free' : `${formatCurrency(event.ticketPriceCents, event.ticketCurrency)} ticket`;
}

function remainingLabel(event: PublicEventSummaryDTO) {
  return event.isFull ? 'Sold out' : `${event.remainingTickets} remaining`;
}

export function DiscoverView() {
  const [events, setEvents] = useState<PublicEventSummaryDTO[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setError(null);

      try {
        const loaded = await api<PublicEventSummaryDTO[]>('/api/public/events');
        if (!cancelled) {
          setEvents(loaded);
        }
      } catch (caught) {
        if (!cancelled) {
          setError(caught instanceof Error ? caught.message : 'Unable to load published events');
          setEvents([]);
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
  }, []);

  return (
    <main className="min-h-screen px-4 py-6 text-zinc-100 sm:px-6 lg:px-8">
      <section className="mx-auto w-full max-w-6xl space-y-6">
        <header className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-6 shadow-2xl shadow-black/40 backdrop-blur">
          <p className="text-xs uppercase tracking-[0.35em] text-amber-300">Public browse</p>
          <div className="mt-3 flex flex-wrap items-center gap-2">
            <span className="rounded-full border border-emerald-400/30 bg-emerald-500/10 px-3 py-1 text-[0.7rem] font-semibold uppercase tracking-[0.28em] text-emerald-200">
              Discover events
            </span>
            <span className="rounded-full border border-white/10 bg-white/5 px-3 py-1 text-[0.7rem] font-semibold uppercase tracking-[0.28em] text-zinc-200">
              Published only
            </span>
          </div>

          <h1 className="mt-5 text-3xl font-semibold tracking-tight text-white sm:text-4xl">Discover events</h1>
          <p className="mt-3 max-w-2xl text-sm leading-7 text-zinc-300 sm:text-base">
            Browse published events without opening private workspace pages.
          </p>
        </header>

        {loading ? <div className="rounded-[1.75rem] border border-white/10 bg-white/5 p-6 text-sm text-zinc-400">Loading published events…</div> : null}

        {error ? (
          <p aria-live="polite" className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
            Could not load published events. {error}
          </p>
        ) : null}

        {!loading && !error && events?.length === 0 ? (
          <div className="rounded-[1.75rem] border border-white/10 bg-white/5 p-6 text-sm text-zinc-400">
            No published events are discoverable yet.
          </div>
        ) : null}

        {!loading && !error && events && events.length > 0 ? (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {events.map((event) => (
              <article key={event.id} className="rounded-[1.75rem] border border-white/10 bg-zinc-950/85 p-5 shadow-lg shadow-black/20">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className="text-xs uppercase tracking-[0.3em] text-amber-300">{event.status}</p>
                    <h2 className="mt-2 text-xl font-semibold text-white">{event.title}</h2>
                  </div>
                  <span className="rounded-full border border-white/10 bg-white/5 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.25em] text-zinc-300">
                    {remainingLabel(event)}
                  </span>
                </div>

                <div className="mt-4 grid gap-3 text-sm text-zinc-300">
                  <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                    <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Date</p>
                    <p className="mt-2 font-medium text-white">{formatDateTime(event.startsAt)}</p>
                  </div>
                  <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                    <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Location</p>
                    <p className="mt-2 font-medium text-white">{event.locationDisplay}</p>
                  </div>
                  <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                    <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Description</p>
                    <p className="mt-2 leading-6 text-zinc-300">{event.publicDescription || 'No public description provided.'}</p>
                  </div>
                  <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                    <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Pricing</p>
                    <p className="mt-2 font-medium text-white">{pricingLabel(event)}</p>
                  </div>
                  <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                    <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Remaining tickets</p>
                    <p className={`mt-2 font-medium ${event.isFull ? 'text-rose-200' : 'text-white'}`}>{event.remainingTickets}</p>
                  </div>
                </div>

                <a
                  className="mt-5 inline-flex w-full items-center justify-center rounded-2xl bg-amber-300 px-4 py-3 font-medium text-zinc-950 transition hover:bg-amber-200"
                  href={event.publicUrl}
                >
                  View event
                </a>
              </article>
            ))}
          </div>
        ) : null}
      </section>
    </main>
  );
}
