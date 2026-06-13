import { useEffect, useState } from 'react';
import { api } from '../api';
import type { TicketDTO } from '../domain';

function statusLabel(ticket: TicketDTO) {
  return ticket.status === 'checked_in' ? 'Checked in' : 'Reserved';
}

function ticketName(ticket: TicketDTO) {
  return ticket.displayName ?? ticket.email;
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

function statusSurface(ticket: TicketDTO) {
  return ticket.status === 'checked_in'
    ? 'border-emerald-400/30 bg-emerald-500/10 text-emerald-50'
    : 'border-amber-300/30 bg-amber-300/10 text-amber-50';
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
          <h1 className="text-3xl font-semibold tracking-tight text-white">Show this at the door</h1>
          <p className="text-sm leading-6 text-zinc-400">Your reservation lives here. Keep this page open or save the code for arrival.</p>

          {loading ? <div className="rounded-2xl border border-white/10 bg-white/5 p-4 text-sm text-zinc-400">Loading…</div> : null}
          {error ? <p className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">{error}</p> : null}

          {ticket ? (
            <>
              <div className={`rounded-[1.5rem] border px-4 py-4 ${statusSurface(ticket)}`}>
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className="text-xs uppercase tracking-[0.28em] text-white/70">Status</p>
                    <p className="mt-2 text-2xl font-semibold tracking-tight text-white">{statusLabel(ticket)}</p>
                  </div>
                  <span className="rounded-full border border-white/15 bg-black/15 px-3 py-1 text-xs uppercase tracking-[0.25em] text-white/80">
                    {ticket.status === 'checked_in' ? 'Access granted' : 'Bring to door'}
                  </span>
                </div>

                <p className="mt-3 text-sm text-white/80">
                  {ticket.status === 'checked_in'
                    ? `Checked in at ${formatHumanTime(ticket.checkedInAt)}`
                    : 'Reserved and ready. Show the code below at the door.'}
                </p>
              </div>

              <div className="rounded-[1.5rem] border border-white/10 bg-white/5 p-5">
                <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Show this at the door</p>
                <p className="mt-3 break-words font-mono text-2xl tracking-[0.28em] text-white sm:text-3xl">{chunkCode(ticket.code)}</p>
                <p className="mt-3 text-sm leading-6 text-zinc-400">
                  {ticket.status === 'checked_in'
                    ? 'This reservation has already been scanned.'
                    : 'This code is what the door team needs to check you in.'}
                </p>
              </div>

              <div className="grid gap-3 sm:grid-cols-2">
                <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                  <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Name</p>
                  <p className="mt-2 text-sm font-medium text-white">{ticketName(ticket)}</p>
                </div>
                <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                  <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Email</p>
                  <p className="mt-2 text-sm font-medium text-white">{ticket.email}</p>
                </div>
                <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                  <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Checked in</p>
                  <p className="mt-2 text-sm font-medium text-white">{formatHumanTime(ticket.checkedInAt)}</p>
                </div>
                <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                  <p className="text-xs uppercase tracking-[0.2em] text-zinc-500">Ticket code</p>
                  <p className="mt-2 text-sm font-medium text-white">{chunkCode(ticket.code)}</p>
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
