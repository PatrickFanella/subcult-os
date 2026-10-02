import { useEffect, useRef, useState } from 'react';
import QRCode from 'qrcode';
import { api } from '../api';
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
  formatTicketCode,
  ticketJourneyCodeCopy,
  ticketJourneyDisplayName,
  ticketJourneyPaymentBadge,
  ticketJourneyPaymentLabel,
  ticketJourneyPaymentSummary,
  ticketJourneyStatusBadge,
  ticketJourneyStatusCopy,
  ticketJourneyStatusLabel,
  ticketPaymentAllowsAdmission,
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

function ticketStatusTone(ticket: TicketDTO) {
  if (!ticketPaymentAllowsAdmission(ticket.paymentStatus)) return ticket.paymentStatus === 'cancelled' ? 'danger' : 'warning';
  return ticket.status === 'checked_in' ? 'success' : 'warning';
}

function ticketPaymentTone(paymentStatus: TicketDTO['paymentStatus']) {
  switch (paymentStatus) {
    case 'free':
    case 'paid':
      return 'success';
    case 'cancelled':
      return 'danger';
    case 'pending':
    default:
      return 'warning';
  }
}

function ticketArrivalNotes(paymentStatus: TicketDTO['paymentStatus']) {
  if (paymentStatus === 'pending') return 'Payment is still pending. Refresh after checkout completes; the door will only accept paid/free tickets.';
  if (!ticketPaymentAllowsAdmission(paymentStatus)) return 'This ticket is not ready for entry. Contact the organizer if you need help with payment.';
  return 'Show this QR code or ticket code at the door. Staff scanners read the ticket code embedded in the QR pass.';
}

export function TicketView({ code }: { code: string }) {
	const [ticket, setTicket] = useState<TicketDTO | null>(null);
	const [qr, setQr] = useState<{ code: string; dataUrl: string } | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [refreshTick, setRefreshTick] = useState(0);
	const [qrFailedCode, setQrFailedCode] = useState<string | null>(null);
	const loadedCode = useRef<string | null>(null);

  useEffect(() => {
    let cancelled = false;

	async function load() {
		setLoading(true);
		setError(null);
		if (loadedCode.current !== code) {
			setTicket(null);
			setQr(null);
			setQrFailedCode(null);
		}

      try {
        const loaded = await api<TicketDTO>(`/api/tickets/${encodeURIComponent(code)}`);
        if (!cancelled) {
          loadedCode.current = code;
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
  }, [code, refreshTick]);

  useEffect(() => {
    let cancelled = false;

	async function buildQr() {
		setQrFailedCode(null);
		if (!ticket?.code) {
			setQr(null);
			return;
		}

		const qrCode = ticket.code;
		setQr(null);

		try {
			const generated = await QRCode.toDataURL(qrCode, {
				color: { dark: '#171717', light: '#ffffff' },
				errorCorrectionLevel: 'M',
				margin: 1,
				scale: 8,
        });

		if (!cancelled) {
			setQr({ code: qrCode, dataUrl: generated });
		}
	} catch {
		if (!cancelled) {
			setQr(null);
			setQrFailedCode(qrCode);
		}
	}
    }

    void buildQr();

    return () => {
      cancelled = true;
    };
	}, [ticket?.code]);

  return (
    <main className={publicPageShellClass}>
      <section className={`${publicPageInnerClass} max-w-2xl`}>
        <div className="flex items-center justify-between gap-3">
          <div>
            <p className={publicEyebrowClass}>Ticket</p>
            <h1 className="mt-2 text-3xl font-bold tracking-[-0.04em] text-fg-primary sm:text-4xl">Your ticket</h1>
          </div>
          <a className={publicSecondaryButtonClass} href="/">
            Workspace
          </a>
        </div>

        <div className="space-y-4">
          <p className={`${publicMutedTextClass} leading-6`}>Your reservation lives here. Keep this page open or save the code for arrival.</p>

          {loading ? <div className={`${publicCardClass} ${publicMutedTextClass}`}>Loading ticket…</div> : null}
          {error ? <div role="alert" className="border border-status-danger/20 bg-status-surface-danger px-4 py-3 text-sm font-bold text-status-danger">
            <p>{error}</p>
            {ticket ? <p className="mt-2">Showing the last loaded ticket. Its payment and check-in status may have changed.</p> : null}
          </div> : null}
          <button className={publicSecondaryButtonClass} type="button" disabled={loading} onClick={() => setRefreshTick((value) => value + 1)}>
            {loading ? 'Refreshing ticket…' : error ? 'Retry ticket' : 'Refresh ticket status'}
          </button>

          {ticket ? (
            <>
              <div className="overflow-hidden rounded-hero border border-stroke-subtle bg-surface-panel p-6">
                <div className="flex items-start justify-between gap-3">
                  <span className={publicStatusPillClass(ticketStatusTone(ticket))}>{ticketJourneyStatusBadge(ticket)}</span>
                  <span className="bg-surface-panel p-3 text-xl" aria-hidden="true">
                    ↗
                  </span>
                </div>

                <div className="mt-5">
                  <h2 className="break-all text-2xl font-bold text-fg-primary">Ticket {ticket.code}</h2>
                  <p className="mt-1 text-sm text-fg-muted">{ticketJourneyDisplayName(ticket)}</p>
                </div>

                <div className="mt-5 grid gap-3 text-sm font-semibold text-fg-secondary sm:grid-cols-2">
                  <p>Payment: {ticket.paymentStatus}</p>
                  <p>Status: {ticket.status}</p>
                </div>

                <div className="my-6 border-t-2 border-dashed border-stroke-subtle" />

                <div className="flex flex-col items-center text-center">
				<div className="flex h-56 w-56 items-center justify-center border border-stroke-subtle bg-surface-panel p-4" aria-label="Ticket QR code">
					{qr?.code === ticket.code ? (
						<img className="h-full w-full" src={qr.dataUrl} alt={`QR code for ticket ${ticket.code}`} />
					) : qrFailedCode === ticket.code ? (
                      <span role="status" className="text-sm font-bold text-fg-secondary">QR unavailable. Show the ticket code below for manual entry.</span>
                    ) : (
                      <span className="text-xs font-bold uppercase tracking-[0.2em] text-fg-muted">Preparing QR</span>
                    )}
                  </div>
                  <p className="mt-4 max-w-full break-all text-center font-mono text-sm tracking-[0.12em] text-fg-secondary">{ticket.code}</p>
                  <p className="mt-2 text-sm text-fg-muted">{ticketJourneyCodeCopy(ticket)}</p>
                </div>
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                <div className={publicCardClass}>
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className={publicEyebrowClass}>Payment status</p>
                      <p className="mt-2 text-2xl font-bold tracking-[-0.03em] text-fg-primary">{ticketJourneyPaymentLabel(ticket.paymentStatus)}</p>
                    </div>
                    <span className={publicStatusPillClass(ticketPaymentTone(ticket.paymentStatus))}>{ticketJourneyPaymentBadge(ticket)}</span>
                  </div>

                  <p className="mt-3 text-sm leading-6 text-fg-secondary">{ticketJourneyPaymentSummary(ticket)}</p>
                </div>

                <div className={publicCardClass}>
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className={publicEyebrowClass}>Status</p>
                      <p className="mt-2 text-2xl font-bold tracking-[-0.03em] text-fg-primary">{ticketJourneyStatusLabel(ticket.status)}</p>
                    </div>
                    <span className={publicStatusPillClass(ticketStatusTone(ticket))}>{ticketJourneyStatusBadge(ticket)}</span>
                  </div>

                  <p className="mt-3 text-sm leading-6 text-fg-secondary">
                    {ticketJourneyStatusCopy(ticket, formatHumanTime(ticket.checkedInAt))}
                  </p>
                </div>
              </div>

              <div className={publicCardClass}>
                <p className="text-xl font-bold tracking-[-0.03em] text-fg-primary">Arrival notes</p>
                <p className="mt-2 text-sm leading-6 text-fg-secondary">{ticketArrivalNotes(ticket.paymentStatus)}</p>
                <div className="mt-5 grid gap-3 sm:grid-cols-2">
                  <div className="rounded-2xl border border-stroke-subtle bg-surface-inset p-4">
                    <p className={publicEyebrowClass}>Name</p>
                    <p className="mt-2 text-sm font-bold text-fg-primary">{ticketJourneyDisplayName(ticket)}</p>
                  </div>
                  <div className="rounded-2xl border border-stroke-subtle bg-surface-inset p-4">
                    <p className={publicEyebrowClass}>Email</p>
                    <p className="mt-2 break-words text-sm font-bold text-fg-primary">{ticket.email}</p>
                  </div>
                  <div className="rounded-2xl border border-stroke-subtle bg-surface-inset p-4">
                    <p className={publicEyebrowClass}>Checked in</p>
                    <p className="mt-2 text-sm font-bold text-fg-primary">{formatHumanTime(ticket.checkedInAt)}</p>
                  </div>
                  <div className="rounded-2xl border border-stroke-subtle bg-surface-inset p-4">
                    <p className={publicEyebrowClass}>Ticket code</p>
                    <p className="mt-2 text-sm font-bold text-fg-primary">{formatTicketCode(ticket.code)}</p>
                  </div>
                </div>
              </div>

              <div className="flex flex-wrap gap-2 text-sm">
                <a className={publicPrimaryButtonClass} href={`/door/${ticket.eventId}`}>
                  Door
                </a>
                <a className={publicSecondaryButtonClass} href="/">
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
