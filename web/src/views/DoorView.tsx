import { useState } from 'react';
import type { FormEvent } from 'react';
import { api, postJSON } from '../api';
import type { TicketDTO } from '../domain';
import {
  publicCardClass,
  publicEyebrowClass,
  publicMutedTextClass,
  publicPageInnerClass,
  publicPageShellClass,
  publicPrimaryButtonClass,
  publicSecondaryButtonClass,
  publicStatusPillClass,
} from '../modules/publicUi/publicUi';
import {
  doorCheckInButtonLabel,
  ticketJourneyDisplayName,
  ticketJourneyDoorStatusBadge,
  ticketJourneyDoorStatusCopy,
  ticketJourneyStatusLabel,
} from '../modules/tickets/ticketJourney';

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
    return 'border-emerald-200 bg-emerald-50 text-emerald-700';
  }

  if (kind === 'error') {
    return 'border-rose-200 bg-rose-50 text-rose-700';
  }

  return 'border-neutral-200 bg-white text-neutral-600';
}

function doorStatusPillTone(status: TicketDTO['status']) {
  return status === 'checked_in' ? 'success' : 'neutral';
}

function doorStatusCardClass(status: TicketDTO['status']) {
  return status === 'checked_in'
    ? 'border-emerald-200 bg-emerald-50 text-emerald-900'
    : 'border-neutral-200 bg-neutral-50 text-[#171717]';
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
      setNotice({ kind: 'neutral', text: 'Type an email, name, or exact ticket code.' });
      return;
    }

    if (!eventId) {
      setLoading(false);
      setError('Missing event ID. Open Door from Staff mode.');
      return;
    }

    setQuery(trimmedQuery);
    setLoading(true);

    try {
      const loaded = await api<TicketDTO[]>(`/api/events/${eventId}/door/tickets?query=${encodeURIComponent(trimmedQuery)}`);
      setResults(loaded);
      setNotice(loaded.length === 0 ? { kind: 'neutral', text: `No matches for “${trimmedQuery}”.` } : { kind: 'success', text: `${loaded.length} ticket${loaded.length === 1 ? '' : 's'} ready.` });
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to search tickets');
    } finally {
      setLoading(false);
    }
  }

  async function handleCheckIn(ticket: TicketDTO) {
    if (!eventId) {
      setError('Missing event ID. Open Door from Staff mode.');
      return;
    }

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
      setNotice({ kind: 'success', text: wasAlreadyCheckedIn ? `Already checked in — ${ticketJourneyDisplayName(updated)}` : `Checked in — ${ticketJourneyDisplayName(updated)}` });
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to check in ticket');
    } finally {
      setCheckingIn(null);
    }
  }

  return (
    <main className={publicPageShellClass}>
      <section className={`${publicPageInnerClass} max-w-3xl`}>
        <header className="rounded-[32px] border border-neutral-200 bg-white p-6 shadow-sm">
          <p className={`${publicEyebrowClass} text-blue-600`}>Door Mode</p>
          <h1 className="mt-2 text-4xl font-black tracking-[-0.04em] text-[#171717]">Guest List</h1>
          <p className={`mt-2 ${publicMutedTextClass}`}>Search by name, email, or exact ticket code. Paste a full code and press Search to jump straight to check-in.</p>

          <div className="mt-4 rounded-2xl bg-neutral-100 px-4 py-3 text-sm font-medium text-neutral-700">Event ID: {eventId || 'Missing'}</div>
        </header>

        <form className={publicCardClass} onSubmit={handleSearch}>
          <label className="block space-y-2 text-sm">
            <span className="font-bold text-[#171717]">Lookup or exact code</span>
            <input
              className="door-input w-full rounded-[18px] border border-neutral-200 bg-neutral-100 px-4 py-4 text-base font-medium text-[#171717] outline-none transition placeholder:text-neutral-400 focus:border-neutral-400 focus:bg-white"
              type="search"
              autoComplete="off"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Name, email, or ticket code"
            />
          </label>

          <div className="mt-3 grid gap-2 sm:grid-cols-2">
            <button
              className={`door-action ${publicPrimaryButtonClass} min-h-14 rounded-[18px] text-base`}
              type="submit"
              disabled={loading}
            >
              {loading ? 'Searching…' : 'Search'}
            </button>
            <button
              className={`door-action ${publicSecondaryButtonClass} min-h-14 rounded-[18px] text-base`}
              type="button"
              onClick={handleClear}
            >
              Reset
            </button>
          </div>

          <p className="mt-3 text-xs leading-5 text-neutral-500">Exact code works. Search by email, name, or the full ticket code to pull up a single result.</p>
        </form>

        {error ? <p className="rounded-[18px] border border-rose-200 bg-rose-50 px-4 py-3 text-sm font-bold text-rose-700">{error}</p> : null}
        {notice ? (
          <p className={`rounded-[18px] border px-4 py-3 text-sm ${noticeClassName(notice.kind)}`} aria-live="polite">
            {notice.text}
          </p>
        ) : null}

        <div className="space-y-3">
          {results.length === 0 ? <p className={`${publicCardClass} ${publicMutedTextClass}`}>Search results will appear here.</p> : null}

          {results.map((ticket) => (
            <article key={ticket.id} className={publicCardClass}>
              <div className={`rounded-[24px] border px-4 py-4 ${doorStatusCardClass(ticket.status)}`}>
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className={publicEyebrowClass}>Status</p>
                    <p className="mt-2 text-2xl font-black tracking-[-0.03em]">{ticketJourneyStatusLabel(ticket.status)}</p>
                  </div>
                  <span className={publicStatusPillClass(doorStatusPillTone(ticket.status))}>{ticketJourneyDoorStatusBadge(ticket.status)}</span>
                </div>

                <p className="mt-3 text-sm text-neutral-700">{ticketJourneyDoorStatusCopy(ticket.status, formatHumanTime(ticket.checkedInAt))}</p>
              </div>

              <div className="mt-4 space-y-3">
                <div>
                  <p className="text-2xl font-black tracking-[-0.03em] text-[#171717]">{ticketJourneyDisplayName(ticket)}</p>
                  <p className="mt-1 text-sm text-neutral-500">{ticket.email}</p>
                </div>

                <div className="rounded-2xl bg-neutral-100 px-4 py-4">
                  <p className={publicEyebrowClass}>Code</p>
                  <p className="mt-2 break-words font-mono text-lg font-bold tracking-[0.08em] text-[#171717] sm:text-xl">{ticket.code}</p>
                </div>

                <div className="grid gap-2 sm:grid-cols-2">
                  <div className="rounded-2xl bg-neutral-100 px-4 py-3 text-sm text-neutral-600">
                    <p className={publicEyebrowClass}>Checked in</p>
                    <p className="mt-2 font-medium text-[#171717]">{formatHumanTime(ticket.checkedInAt)}</p>
                  </div>
                  <div className="rounded-2xl bg-neutral-100 px-4 py-3 text-sm text-neutral-600">
                    <p className={publicEyebrowClass}>Payment</p>
                    <p className="mt-2 font-medium text-[#171717]">{ticket.paymentStatus}</p>
                  </div>
                </div>
              </div>

              <button
                className={`door-action mt-4 w-full ${publicPrimaryButtonClass} min-h-14 rounded-[18px] text-base`}
                type="button"
                onClick={() => handleCheckIn(ticket)}
                disabled={checkingIn === ticket.code || ticket.status === 'checked_in'}
              >
                {doorCheckInButtonLabel(checkingIn === ticket.code, ticket.status === 'checked_in')}
              </button>
            </article>
          ))}
        </div>
      </section>
    </main>
  );
}
