import * as SecureStore from 'expo-secure-store';
import { Platform } from 'react-native';

import type { TicketDTO } from '@/api/types';

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

async function readRaw() {
  if (Platform.OS === 'web') return globalThis.localStorage?.getItem(walletKey) ?? null;
  return SecureStore.getItemAsync(walletKey);
}

async function writeRaw(value: string) {
  if (Platform.OS === 'web') {
    globalThis.localStorage?.setItem(walletKey, value);
    return;
  }
  await SecureStore.setItemAsync(walletKey, value);
}

export async function loadSavedTickets(): Promise<SavedTicket[]> {
  const raw = await readRaw();
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
  const saved: SavedTicket = {
    code: ticket.code,
    email: ticket.email,
    displayName: ticket.displayName,
    status: ticket.status,
    paymentStatus: ticket.paymentStatus,
    ticketUrl: ticket.ticketUrl,
    savedAt: new Date().toISOString(),
  };
  const next = [saved, ...current.filter((item) => item.code !== ticket.code)].slice(0, 50);
  await writeRaw(JSON.stringify(next));
  return next;
}

export async function savePendingPaidTicket(ticket: PendingPaidTicketInput) {
  const current = await loadSavedTickets();
  const saved: SavedTicket = {
    code: ticket.code,
    email: ticket.email,
    displayName: ticket.displayName,
    status: 'reserved',
    paymentStatus: 'pending',
    ticketUrl: ticket.ticketUrl,
    checkoutSessionId: ticket.checkoutSessionId,
    savedAt: new Date().toISOString(),
  };
  const next = [saved, ...current.filter((item) => item.code !== ticket.code)].slice(0, 50);
  await writeRaw(JSON.stringify(next));
  return next;
}
