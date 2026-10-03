import { useState } from 'react';
import type { FormEvent } from 'react';
import { api, postJSON } from '../api';
import type { DoorTicketDTO } from '../domain';
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
    return 'border-status-success/20 bg-status-surface-success text-status-success';
  }

  if (kind === 'error') {
    return 'border-status-danger/20 bg-status-surface-danger text-status-danger';
  }

  return 'border-stroke-subtle bg-surface-panel text-fg-secondary';
}

function doorStatusPillTone(status: DoorTicketDTO['status']) {
  return status === 'checked_in' ? 'success' : 'neutral';
}

function doorStatusCardClass(status: DoorTicketDTO['status']) {
  return status === 'checked_in'
    ? 'border-status-success/20 bg-status-surface-success text-status-success'
    : 'border-stroke-subtle bg-surface-inset text-fg-primary';
}

export function DoorView({ eventId }: { eventId: string }) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<DoorTicketDTO[]>([]);
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
    setNotice({ kind: 'neutral', text: 'Ready for the next guest.' });
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
      const loaded = await api<DoorTicketDTO[]>(`/api/events/${eventId}/door/tickets?query=${encodeURIComponent(trimmedQuery)}`);
      setResults(loaded);
      setNotice(loaded.length === 0 ? { kind: 'neutral', text: `No matches for “${trimmedQuery}”.` } : { kind: 'success', text: `${loaded.length} ticket${loaded.length === 1 ? '' : 's'} ready.` });
    } catch (caught) {
      setResults([]);
      setError(caught instanceof Error ? caught.message : 'Unable to search tickets');
    } finally {
      setLoading(false);
    }
  }

  async function handleCheckIn(ticket: DoorTicketDTO) {
    if (!eventId) {
      setError('Missing event ID. Open Door from Staff mode.');
      return;
    }

    const wasAlreadyCheckedIn = ticket.status === 'checked_in';
    setCheckingIn(ticket.code);
    setError(null);
    setNotice(null);

    try {
      const updated = await postJSON<DoorTicketDTO>(`/api/events/${eventId}/door/check-ins`, { code: ticket.code });
      setResults((current) => {
        const next = current.map((currentTicket) => (currentTicket.code === updated.code ? updated : currentTicket));
        return next.some((currentTicket) => currentTicket.code === updated.code) ? next : [updated, ...next];
      });
      const attendee = updated.displayName ?? 'Guest';
      setNotice({ kind: 'success', text: wasAlreadyCheckedIn ? `Already checked in — ${attendee}` : `Checked in — ${attendee}` });
    } catch (caught) {
      setResults([]);
      setError(caught instanceof Error ? caught.message : 'Unable to check in ticket');
    } finally {
      setCheckingIn(null);
    }
  }

  return (
    <main className={publicPageShellClass}>
      <section className={`${publicPageInnerClass} max-w-3xl`}>
        <header className="rounded-hero border border-stroke-subtle bg-surface-panel p-6">
          <p className={publicEyebrowClass}>Door Mode</p>
          <h1 className="mt-2 text-4xl font-bold text-fg-primary">Guest List</h1>
          <p className={`mt-2 ${publicMutedTextClass}`}>Search by name, email or ticket code. A full code goes straight to check-in.</p>

          <div className="mt-4 rounded-2xl bg-surface-inset px-4 py-3 text-sm font-medium text-fg-secondary">Event ID: {eventId || 'Missing'}</div>
        </header>

        <form className={publicCardClass} onSubmit={handleSearch}>
          <label className="block space-y-2 text-sm">
            <span className="font-bold text-fg-primary">Lookup or exact code</span>
            <input
              className="field door-input py-4 text-base"
              type="search"
              autoComplete="off"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Name, email, or ticket code"
            />
          </label>

          <div className="mt-3 grid gap-2 sm:grid-cols-2">
            <button
              className={`door-action ${publicPrimaryButtonClass} min-h-14 text-base`}
              type="submit"
              disabled={loading}
            >
              {loading ? 'Searching…' : 'Search'}
            </button>
            <button
              className={`door-action ${publicSecondaryButtonClass} min-h-14 text-base`}
              type="button"
              onClick={handleClear}
            >
              Reset
            </button>
          </div>

          <p className="mt-3 text-xs leading-5 text-fg-muted">A full ticket code returns one guest. A name or email may return several.</p>
        </form>

        {error ? <p className="border border-status-danger/20 bg-status-surface-danger px-4 py-3 text-sm font-bold text-status-danger">{error}</p> : null}
        {notice ? (
          <p className={`border px-4 py-3 text-sm ${noticeClassName(notice.kind)}`} aria-live="polite">
            {notice.text}
          </p>
        ) : null}

        <div className="space-y-3">
          {results.length === 0 ? <p className={`${publicCardClass} ${publicMutedTextClass}`}>Results show here.</p> : null}

          {results.map((ticket) => (
            <article key={ticket.id} className={publicCardClass}>
              <div className={`border px-4 py-4 ${doorStatusCardClass(ticket.status)}`}>
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className={publicEyebrowClass}>Status</p>
                    <p className="mt-2 text-2xl font-bold tracking-[-0.03em]">{ticketJourneyStatusLabel(ticket.status)}</p>
                  </div>
                  <span className={publicStatusPillClass(doorStatusPillTone(ticket.status))}>{ticketJourneyDoorStatusBadge(ticket.status)}</span>
                </div>

                <p className="mt-3 text-sm text-fg-secondary">{ticketJourneyDoorStatusCopy(ticket.status, formatHumanTime(ticket.checkedInAt))}</p>
              </div>

              <div className="mt-4 space-y-3">
                <div>
                  <p className="text-2xl font-bold tracking-[-0.03em] text-fg-primary">{ticket.displayName ?? 'Guest'}</p>
                </div>

                <div className="rounded-2xl bg-surface-inset px-4 py-4">
                  <p className={publicEyebrowClass}>Code</p>
                  <p className="mt-2 break-words font-mono text-lg font-bold tracking-[0.08em] text-fg-primary sm:text-xl">{ticket.code}</p>
                </div>

                <div className="grid gap-2 sm:grid-cols-2">
                  <div className="rounded-2xl bg-surface-inset px-4 py-3 text-sm text-fg-secondary">
                    <p className={publicEyebrowClass}>Checked in</p>
                    <p className="mt-2 font-medium text-fg-primary">{formatHumanTime(ticket.checkedInAt)}</p>
                  </div>
                  <div className="rounded-2xl bg-surface-inset px-4 py-3 text-sm text-fg-secondary">
                    <p className={publicEyebrowClass}>Admission</p>
                    <p className="mt-2 font-medium text-fg-primary">{ticket.admissionEligible ? 'Eligible' : 'Not eligible'}</p>
                  </div>
                </div>
              </div>

              <button
                className={`door-action mt-4 w-full ${publicPrimaryButtonClass} min-h-14 text-base`}
                type="button"
                onClick={() => handleCheckIn(ticket)}
                disabled={checkingIn === ticket.code || ticket.status === 'checked_in' || !ticket.admissionEligible}
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
