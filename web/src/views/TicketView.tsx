import { useEffect, useState } from 'react';
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

function ticketStatusTone(status: TicketDTO['status']) {
  return status === 'checked_in' ? 'success' : 'warning';
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
  return paymentStatus === 'pending'
    ? 'Payment is still pending. Refresh after checkout completes; the door will only accept paid/free tickets.'
    : 'Show this QR code or ticket code at the door. Staff scanners read the ticket code embedded in the QR pass.';
}

export function TicketView({ code }: { code: string }) {
	const [ticket, setTicket] = useState<TicketDTO | null>(null);
	const [qr, setQr] = useState<{ code: string; dataUrl: string } | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

	async function load() {
		setLoading(true);
		setError(null);
		setTicket(null);
		setQr(null);

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

  useEffect(() => {
    let cancelled = false;

	async function buildQr() {
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
            <h1 className="mt-2 text-3xl font-black tracking-[-0.04em] text-[#171717] sm:text-4xl">Show this at the door</h1>
          </div>
          <a className={publicSecondaryButtonClass} href="/">
            Workspace
          </a>
        </div>

        <div className="space-y-4">
          <p className={`${publicMutedTextClass} leading-6`}>Your reservation lives here. Keep this page open or save the code for arrival.</p>

          {loading ? <div className={`${publicCardClass} ${publicMutedTextClass}`}>Loading ticket…</div> : null}
          {error ? <p className="rounded-[22px] border border-rose-200 bg-rose-50 px-4 py-3 text-sm font-bold text-rose-700">{error}</p> : null}

          {ticket ? (
            <>
              <div className="overflow-hidden rounded-[32px] border border-neutral-200 bg-white p-6 shadow-sm">
                <div className="flex items-start justify-between gap-3">
                  <span className={publicStatusPillClass(ticketStatusTone(ticket.status))}>{ticketJourneyStatusBadge(ticket.status)}</span>
                  <span className="rounded-full bg-white p-3 text-xl shadow-sm" aria-hidden="true">
                    ↗
                  </span>
                </div>

                <div className="mt-5">
                  <h2 className="text-2xl font-black tracking-[-0.03em] text-[#171717]">Ticket {ticket.code}</h2>
                  <p className="mt-1 text-sm text-neutral-500">{ticketJourneyDisplayName(ticket)}</p>
                </div>

                <div className="mt-5 grid gap-3 text-sm font-semibold text-neutral-700 sm:grid-cols-2">
                  <p>Payment: {ticket.paymentStatus}</p>
                  <p>Status: {ticket.status}</p>
                </div>

                <div className="my-6 border-t-2 border-dashed border-neutral-200" />

                <div className="flex flex-col items-center text-center">
				<div className="flex h-56 w-56 items-center justify-center rounded-[24px] border border-neutral-200 bg-white p-4" aria-label="Ticket QR code">
					{qr?.code === ticket.code ? (
						<img className="h-full w-full" src={qr.dataUrl} alt={`QR code for ticket ${ticket.code}`} />
					) : (
                      <span className="text-xs font-black uppercase tracking-[0.28em] text-neutral-500">Preparing QR</span>
                    )}
                  </div>
                  <p className="mt-4 break-words font-mono text-sm tracking-[0.28em] text-neutral-600">{ticket.code}</p>
                  <p className="mt-2 text-sm text-neutral-500">{ticketJourneyCodeCopy(ticket.status)}</p>
                </div>
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                <div className={publicCardClass}>
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className={publicEyebrowClass}>Payment status</p>
                      <p className="mt-2 text-2xl font-black tracking-[-0.03em] text-[#171717]">{ticketJourneyPaymentLabel(ticket.paymentStatus)}</p>
                    </div>
                    <span className={publicStatusPillClass(ticketPaymentTone(ticket.paymentStatus))}>{ticketJourneyPaymentBadge(ticket)}</span>
                  </div>

                  <p className="mt-3 text-sm leading-6 text-neutral-600">{ticketJourneyPaymentSummary(ticket)}</p>
                </div>

                <div className={publicCardClass}>
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className={publicEyebrowClass}>Status</p>
                      <p className="mt-2 text-2xl font-black tracking-[-0.03em] text-[#171717]">{ticketJourneyStatusLabel(ticket.status)}</p>
                    </div>
                    <span className={publicStatusPillClass(ticketStatusTone(ticket.status))}>{ticketJourneyStatusBadge(ticket.status)}</span>
                  </div>

                  <p className="mt-3 text-sm leading-6 text-neutral-600">
                    {ticketJourneyStatusCopy(ticket.status, formatHumanTime(ticket.checkedInAt))}
                  </p>
                </div>
              </div>

              <div className={publicCardClass}>
                <p className="text-xl font-black tracking-[-0.03em] text-[#171717]">Arrival notes</p>
                <p className="mt-2 text-sm leading-6 text-neutral-600">{ticketArrivalNotes(ticket.paymentStatus)}</p>
                <div className="mt-5 grid gap-3 sm:grid-cols-2">
                  <div className="rounded-2xl border border-neutral-200 bg-neutral-50 p-4">
                    <p className={publicEyebrowClass}>Name</p>
                    <p className="mt-2 text-sm font-bold text-[#171717]">{ticketJourneyDisplayName(ticket)}</p>
                  </div>
                  <div className="rounded-2xl border border-neutral-200 bg-neutral-50 p-4">
                    <p className={publicEyebrowClass}>Email</p>
                    <p className="mt-2 break-words text-sm font-bold text-[#171717]">{ticket.email}</p>
                  </div>
                  <div className="rounded-2xl border border-neutral-200 bg-neutral-50 p-4">
                    <p className={publicEyebrowClass}>Checked in</p>
                    <p className="mt-2 text-sm font-bold text-[#171717]">{formatHumanTime(ticket.checkedInAt)}</p>
                  </div>
                  <div className="rounded-2xl border border-neutral-200 bg-neutral-50 p-4">
                    <p className={publicEyebrowClass}>Ticket code</p>
                    <p className="mt-2 text-sm font-bold text-[#171717]">{formatTicketCode(ticket.code)}</p>
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
