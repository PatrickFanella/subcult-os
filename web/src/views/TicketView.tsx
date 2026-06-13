import { useEffect, useState } from 'react';
import { api } from '../api';
import type { TicketDTO } from '../domain';

function statusLabel(ticket: TicketDTO) {
  return ticket.status === 'checked_in' ? 'Checked in' : 'Reserved';
}

function ticketName(ticket: TicketDTO) {
  return ticket.displayName ?? ticket.email;
}

export function TicketView({ code }: { code: string }) {
  const [ticket, setTicket] = useState<TicketDTO | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setError(null);

      try {
        const loaded = await api<TicketDTO>(`/api/tickets/${code}`);
        if (!cancelled) {
          setTicket(loaded);
        }
      } catch (caught) {
        if (!cancelled) {
          setError(caught instanceof Error ? caught.message : 'Unable to load ticket');
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
  }, [code]);

  return (
    <main className="min-h-screen px-4 py-6 text-zinc-100 sm:px-6 lg:px-8">
      <section className="mx-auto flex min-h-[calc(100vh-3rem)] w-full max-w-xl items-center">
        <div className="w-full space-y-4 rounded-[1.75rem] border border-white/10 bg-zinc-950/90 p-6 shadow-2xl shadow-black/40 backdrop-blur">
          <p className="text-xs uppercase tracking-[0.3em] text-fuchsia-300">Ticket</p>
          <h1 className="text-3xl font-semibold tracking-tight text-white">{ticket ? ticketName(ticket) : 'Loading ticket'}</h1>
          <p className="text-sm leading-6 text-zinc-400">Confirmation page for a single free reservation.</p>

          {loading ? <div className="rounded-2xl border border-white/10 bg-white/5 p-4 text-sm text-zinc-400">Loading…</div> : null}
          {error ? <p className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">{error}</p> : null}

          {ticket ? (
            <>
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                  <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Code</p>
                  <p className="mt-2 break-all text-sm font-medium text-white">{ticket.code}</p>
                </div>
                <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                  <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Status</p>
                  <p className="mt-2 text-sm font-medium text-white">{statusLabel(ticket)}</p>
                </div>
                <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                  <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Email</p>
                  <p className="mt-2 text-sm font-medium text-white">{ticket.email}</p>
                </div>
                <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                  <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Checked in</p>
                  <p className="mt-2 text-sm font-medium text-white">{ticket.checkedInAt ?? 'No'}</p>
                </div>
              </div>

              <div className="flex flex-wrap gap-2 text-sm">
                <a className="rounded-full bg-amber-300 px-4 py-2 font-medium text-zinc-950 transition hover:bg-amber-200" href={`/door/${ticket.eventId}`}>
                  Door
                </a>
                <a className="rounded-full border border-white/10 bg-white/5 px-4 py-2 text-zinc-200 transition hover:bg-white/10" href="/">
                  Workspace
                </a>
              </div>
            </>
          ) : null}
        </div>
      </section>
    </main>
  );
}
