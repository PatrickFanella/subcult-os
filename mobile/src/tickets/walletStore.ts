import type { TicketDTO } from '@/api/types';
import { buildTicketWalletSnapshot } from '@/modules/tickets/ticketJourney';
import { loadPersistedValue, savePersistedValue } from '@/modules/storage/persistedStore';

const walletKey = 'subcult_os_ticket_wallet';

export interface SavedTicket {
  code: string;
  email: string;
  displayName: string | null;
  status: string;
  paymentStatus: string;
  ticketUrl?: string;
  checkoutSessionId?: string;
  savedAt: string;
}

export interface PendingPaidTicketInput {
  code: string;
  email: string;
  displayName: string | null;
  ticketUrl: string;
  checkoutSessionId: string;
}

export async function loadSavedTickets(): Promise<SavedTicket[]> {
  const raw = await loadPersistedValue(walletKey);
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

export async function saveTicketToWallet(ticket: TicketDTO) {
  const current = await loadSavedTickets();
  const saved: SavedTicket = buildTicketWalletSnapshot({
    code: ticket.code,
    email: ticket.email,
    displayName: ticket.displayName,
    status: ticket.status,
    paymentStatus: ticket.paymentStatus,
    ticketUrl: ticket.ticketUrl,
  });
  const next = [saved, ...current.filter((item) => item.code !== ticket.code)].slice(0, 50);
  await savePersistedValue(walletKey, JSON.stringify(next));
  return next;
}

export async function savePendingPaidTicket(ticket: PendingPaidTicketInput) {
  const current = await loadSavedTickets();
  const saved: SavedTicket = buildTicketWalletSnapshot({
    code: ticket.code,
    email: ticket.email,
    displayName: ticket.displayName,
    status: 'reserved',
    paymentStatus: 'pending',
    ticketUrl: ticket.ticketUrl,
    checkoutSessionId: ticket.checkoutSessionId,
  });
  const next = [saved, ...current.filter((item) => item.code !== ticket.code)].slice(0, 50);
  await savePersistedValue(walletKey, JSON.stringify(next));
  return next;
}
